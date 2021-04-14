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
	featuresDirectory, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())

	define.Given(`^a directory based helm chart`, func() {
		ChartPath = path.Join(featuresDirectory, "fixtures", "sample-chart")
	})

	define.Given(`^a tgz based helm chart`, func() {
		ChartPath = path.Join(featuresDirectory, "fixtures", "sample-chart-0.1.0.tgz")
	})

	define.Given(`^an image template list file$`, func() {
		ImageTemplateFile = path.Join(featuresDirectory, "fixtures", "sample-chart-images.yaml")
	})

	define.Given("^no image template list file$", func() {
		ImageTemplateFile = ""
	})

	define.Given(`^a rules file that rewrites the registry$`, func() {
		RewriteRulesFile = path.Join(featuresDirectory, "fixtures", "rules", "replace-registry.yaml")
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

	define.Then(`^it prints the usage$`, func() {
		Expect(CommandSession.Out).To(Say("Usage:"))
		Expect(CommandSession.Out).To(Say("chart-mover.*? <chart> \\[flags\\]"))
	})
}
