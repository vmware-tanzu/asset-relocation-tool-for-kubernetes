// +build feature

package features

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("List Images command", func() {
	steps := NewSteps()

	Scenario("directory based helm chart", func() {
		steps.Given("a directory based helm chart")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits without error")
		steps.And("the rendered images are printed")
	})

	//Scenario("tgz based helm chart", func() {
	//	steps.Given("a tgz based helm chart")
	//	steps.And("chart-mover has been built")
	//	steps.When("running chart-mover list-images")
	//	steps.Then("the command exits without error")
	//	steps.And("the rendered images are shown")
	//})

	Scenario("missing helm chart", func() {
		steps.Given("no helm chart")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits with an error about the missing helm chart")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		var (
			chartPath         string
			imageTemplateFile string
			chartMoverPath    string
			commandSession    *gexec.Session
		)

		define.Given(`^chart-mover has been built$`, func() {
			var err error
			chartMoverPath, err = gexec.Build(
				"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2",
			)
			Expect(err).NotTo(HaveOccurred())
		}, func() {
			gexec.CleanupBuildArtifacts()
		})

		define.Given(`^a directory based helm chart`, func() {
			wd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			chartPath = path.Join(wd, "fixtures", "sample-chart")
			imageTemplateFile = path.Join(wd, "fixtures", "sample-chart-images.yaml")
		})

		define.Given(`^no helm chart`, func() {
			wd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			chartPath = path.Join(wd, "fixtures", "empty-directory")
			imageTemplateFile = path.Join(wd, "fixtures", "sample-chart-images.yaml")
		})

		define.When(`^running chart-mover list-images$`, func() {
			listImagesCommand := exec.Command(chartMoverPath,
				"--images", imageTemplateFile,
				"list-images", chartPath)
			var err error
			commandSession, err = gexec.Start(listImagesCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		define.Then(`^the command exits without error$`, func() {
			Eventually(commandSession).Should(gexec.Exit(0))
		})

		define.Then(`^the rendered images are printed$`, func() {
			var images []string
			err := json.Unmarshal(commandSession.Out.Contents(), &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(1))
			Expect(images[0]).To(Equal("docker.io/library/nginx:stable"))
		})

		define.Then(`^the command exits with an error about the missing helm chart$`, func() {
			Eventually(commandSession).Should(gexec.Exit(1))
			Expect(commandSession.Err).To(Say("Error: failed to load helm chart: no Chart.yaml exists in directory"))
			Expect(commandSession.Err).To(Say("features/fixtures/empty-directory"))
		})
		define.Then(`^it prints the usage$`, func() {
			Expect(commandSession.Out).To(Say("Usage:"))
			Expect(commandSession.Out).To(Say("chart-mover list-images <chart> \\[flags\\]"))
		})
	})
})
