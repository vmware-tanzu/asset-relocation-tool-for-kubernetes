package features

import (
	"os"
	"path"
	"time"

	"github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	ChartPath            string
	ChartMoverBinaryPath string
	CommandSession       *gexec.Session
	ImageTemplateFile    string
	RewriteRulesFile     string
)

var _ = BeforeSuite(func() {
	var err error
	ChartMoverBinaryPath, err = gexec.Build(
		"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2",
		"-ldflags",
		"-X gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/cmd.Version=1.2.3",
	)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func DefineCommonSteps(define goerkin.Definitions) {
	define.Given(`^a directory based helm chart`, func() {
		ChartPath = path.Join("fixtures", "samplechart")
	})

	define.Given(`^a tgz based helm chart`, func() {
		ChartPath = path.Join("fixtures", "samplechart-0.1.0.tgz")
	})

	define.Given(`^a helm chart with a chart dependency$`, func() {
		ChartPath = path.Join("fixtures", "dependentchart")
	})

	define.Given(`^an image template list file$`, func() {
		ImageTemplateFile = path.Join("fixtures", "sample-chart-images.yaml")
	})

	define.Given(`^an image template list file for the chart with dependencies$`, func() {
		ImageTemplateFile = path.Join("fixtures", "dependent-chart-images.yaml")
	})

	define.Given("^no image template list file$", func() {
		ImageTemplateFile = ""
	})

	define.Given(`^a rules file that rewrites the registry$`, func() {
		RewriteRulesFile = path.Join("fixtures", "rules", "replace-registry.yaml")
	})

	define.Given("^no rewrite rules file$", func() {
		RewriteRulesFile = ""
	})

	define.Given(`^no helm chart`, func() {
		wd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		ChartPath = path.Join(wd, "fixtures", "empty-directory")
		ImageTemplateFile = path.Join(wd, "fixtures", "sample-chart-images.yaml")
	})

	define.Then(`^the command exits without error$`, func() {
		Eventually(CommandSession, time.Minute).Should(gexec.Exit(0))
	})

	define.Then(`^the command exits with an error$`, func() {
		Eventually(CommandSession, time.Minute).ShouldNot(gexec.Exit(0))
	})

	define.Then(`^the command exits with an error about the missing helm chart$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: failed to load helm chart at \""))
		// Skipping the first part of the path which would be host-specific
		Expect(CommandSession.Err).To(Say("fixtures/empty-directory\": Chart.yaml file is missing"))
	})

	define.Then(`^the command exits with an error about the missing images template list file$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: image-templates is required. Please try again with '-i <image templates>'"))
	})

	define.Then(`^the command exits with an error about the missing rules file$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: rules-file is required. Please try again with '-r <rules file>'"))
	})

	define.Then(`^it prints the usage$`, func() {
		Expect(CommandSession.Out).To(Say("Usage:"))
		Expect(CommandSession.Out).To(Say("relok8s.*? <chart> \\[flags\\]"))
	})
}
