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

var _ = Describe("Rewrite Images command", func() {
	steps := NewSteps()

	Scenario("directory based helm chart", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits without error")
		steps.And("the rewritten images are printed")
	})

	Scenario("tgz based helm chart", func() {
		steps.Given("a tgz based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits without error")
		steps.And("the rewritten images are printed")
	})

	Scenario("helm chart with dependencies", func() {
		steps.Given("a helm chart with a chart dependency")
		steps.And("an image template list file for the chart with dependencies")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits without error")
		steps.And("the rewritten images from the parent and dependent chart are printed")
	})

	Scenario("missing helm chart", func() {
		steps.Given("no helm chart")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits with an error about the missing helm chart")
		steps.And("it prints the usage")
	})

	Scenario("missing images template list file", func() {
		steps.Given("a directory based helm chart")
		steps.And("no image template list file")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits with an error about the missing images template list file")
		steps.And("it prints the usage")
	})

	Scenario("missing rules file", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("no rewrite rules file")
		steps.When("running chart-mover rewrite-images")
		steps.Then("the command exits with an error about the missing rules file")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		DefineCommonSteps(define)

		define.When(`^running chart-mover rewrite-images$`, func() {
			args := []string{"rewrite-images", ChartPath}
			if ImageTemplateFile != "" {
				args = append(args, "--image-templates", ImageTemplateFile)
			}
			if RewriteRulesFile != "" {
				args = append(args, "--rules-file", RewriteRulesFile)
			}
			command := exec.Command(ChartMoverBinaryPath, args...)
			var err error
			CommandSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		define.Then(`^the rewritten images are printed$`, func() {
			output := CommandSession.Out.Contents()
			errOutput := CommandSession.Err.Contents()
			Expect(string(errOutput)).To(BeEmpty())

			var images []string
			err := json.Unmarshal(output, &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(3))
			Expect(images).To(ContainElements(
				"my-registry.example.com/library/nginx:stable",
				"my-registry.example.com/library/python:3",
				"my-registry.example.com/library/busybox:latest",
			))
		})

		define.Then(`^the rewritten images from the parent and dependent chart are printed$`, func() {
			output := CommandSession.Out.Contents()
			errOutput := CommandSession.Err.Contents()
			Expect(string(errOutput)).To(BeEmpty())

			var images []string
			err := json.Unmarshal(output, &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(4))
			Expect(images).To(ContainElements(
				"my-registry.example.com/library/nginx:latest",
				"my-registry.example.com/library/nginx:stable",
				"my-registry.example.com/library/python:3",
				"my-registry.example.com/library/busybox:latest",
			))
		})
	})
})
