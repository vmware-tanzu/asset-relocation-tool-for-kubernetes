// +build feature

package features

import (
	"encoding/json"
	"os/exec"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("List Images command", func() {
	steps := NewSteps()

	Scenario("directory based helm chart", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits without error")
		steps.And("the rendered images are printed")
	})

	Scenario("tgz based helm chart", func() {
		steps.Given("a tgz based helm chart")
		steps.And("an image template list file")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits without error")
		steps.And("the rendered images are printed")
	})

	Scenario("missing helm chart", func() {
		steps.Given("no helm chart")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits with an error about the missing helm chart")
		steps.And("it prints the usage")
	})

	Scenario("missing images template list file", func() {
		steps.Given("a directory based helm chart")
		steps.And("no image template list file")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover list-images")
		steps.Then("the command exits with an error about the missing images template list file")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		DefineCommonSteps(define)

		define.When(`^running chart-mover list-images$`, func() {
			args := []string{"list-images", ChartPath}
			if ImageTemplateFile != "" {
				args = append(args, "--image-templates", ImageTemplateFile)
			}
			command := exec.Command(ChartMoverBinaryPath, args...)
			var err error
			CommandSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		define.Then(`^the rendered images are printed$`, func() {
			var images []string
			err := json.Unmarshal(CommandSession.Out.Contents(), &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(3))
			Expect(images[0]).To(Equal("docker.io/library/nginx:stable"))
			Expect(images[1]).To(Equal("docker.io/library/python:3"))
			Expect(images[2]).To(Equal("docker.io/library/busybox:latest"))
		})
	})
})
