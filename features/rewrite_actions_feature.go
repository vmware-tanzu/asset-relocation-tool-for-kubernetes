// +build feature

package features

import (
	"encoding/json"
	"os/exec"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

var _ = Describe("Rewrite Actions command", func() {
	steps := NewSteps()

	Scenario("directory based helm chart", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits without error")
		steps.And("the rewrite actions are printed")
	})

	Scenario("tgz based helm chart", func() {
		steps.Given("a tgz based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits without error")
		steps.And("the rewrite actions are printed")
	})

	Scenario("helm chart with dependencies", func() {
		steps.Given("a helm chart with a chart dependency")
		steps.And("an image template list file for the chart with dependencies")
		steps.And("a rules file that rewrites the registry")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits without error")
		steps.And("the rewritten actions for the parent and dependent chart are printed")
	})

	Scenario("missing helm chart", func() {
		steps.Given("no helm chart")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing helm chart")
		steps.And("it prints the usage")
	})

	Scenario("missing images template list file", func() {
		steps.Given("a directory based helm chart")
		steps.And("no image template list file")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing images template list file")
		steps.And("it prints the usage")
	})

	Scenario("missing rules file", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("no rewrite rules file")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing rules file")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		DefineCommonSteps(define)

		define.When(`^running chart-mover rewrite-actions$`, func() {
			args := []string{"rewrite-actions", ChartPath}
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

		define.Then(`^the rewrite actions are printed$`, func() {
			actions := []lib.RewriteAction{}
			err := json.Unmarshal(CommandSession.Out.Contents(), &actions)
			Expect(err).ToNot(HaveOccurred())

			Expect(actions).To(HaveLen(4))
			Expect(actions).To(ContainElements(
				lib.RewriteAction{
					Path:  ".Values.image.repository",
					Value: "my-registry.example.com/library/nginx",
				},
				lib.RewriteAction{
					Path:  ".Values.wellDefinedImage.registry",
					Value: "my-registry.example.com",
				},
				lib.RewriteAction{
					Path:  ".Values.wellDefinedImage.repository",
					Value: "library/python",
				},
				lib.RewriteAction{
					Path:  ".Values.singleTemplateImage.image",
					Value: "my-registry.example.com/library/busybox:latest",
				},
			))
		})

		define.Then(`^the rewritten actions for the parent and dependent chart are printed$`, func() {
			actions := []lib.RewriteAction{}
			err := json.Unmarshal(CommandSession.Out.Contents(), &actions)
			Expect(err).ToNot(HaveOccurred())

			Expect(actions).To(HaveLen(5))
			Expect(actions).To(ContainElements(
				lib.RewriteAction{
					Path:  ".Values.image.repository",
					Value: "my-registry.example.com/library/nginx",
				},
				lib.RewriteAction{
					Path:  ".Values.samplechart.image.repository",
					Value: "my-registry.example.com/library/nginx",
				},
				lib.RewriteAction{
					Path:  ".Values.samplechart.wellDefinedImage.registry",
					Value: "my-registry.example.com",
				},
				lib.RewriteAction{
					Path:  ".Values.samplechart.wellDefinedImage.repository",
					Value: "library/python",
				},
				lib.RewriteAction{
					Path:  ".Values.samplechart.singleTemplateImage.image",
					Value: "my-registry.example.com/library/busybox:latest",
				},
			))
		})
	})
})
