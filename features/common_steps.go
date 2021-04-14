// +build feature

package features

import (
	"os"
	"path"

	"github.com/bunniesandbeatings/goerkin"
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

func DefineCommonSteps(define goerkin.Definitions) {
	define.Given(`^a directory based helm chart`, func() {
		ChartPath = path.Join("fixtures", "sample-chart")
	})

	define.Given(`^a tgz based helm chart`, func() {
		ChartPath = path.Join("fixtures", "sample-chart-0.1.0.tgz")
	})

	define.Given(`^an image template list file$`, func() {
		ImageTemplateFile = path.Join("fixtures", "sample-chart-images.yaml")
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

	define.Given(`^chart-mover has been built$`, func() {
		var err error
		ChartMoverBinaryPath, err = gexec.Build(
			"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2",
		)
		Expect(err).NotTo(HaveOccurred())
	}, func() {
		gexec.CleanupBuildArtifacts()
	})

	define.Then(`^the command exits without error$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(0))
	})

	define.Then(`^the command exits with an error about the missing helm chart$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: failed to load helm chart at \""))
		// Skipping the first part of the path which would be host-specific
		Expect(CommandSession.Err).To(Say("fixtures/empty-directory\": Chart.yaml file is missing"))
	})

	define.Then(`^the command exits with an error about the missing images template list file$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: image list file is required"))
	})

	define.Then(`^the command exits with an error about the missing rules file$`, func() {
		Eventually(CommandSession).Should(gexec.Exit(1))
		Expect(CommandSession.Err).To(Say("Error: rewrite rules file is required"))
	})

	define.Then(`^it prints the usage$`, func() {
		Expect(CommandSession.Out).To(Say("Usage:"))
		Expect(CommandSession.Out).To(Say("chart-mover.*? <chart> \\[flags\\]"))
	})
}
