// +build feature

package features

import (
	"encoding/json"
	"os/exec"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = XDescribe("Rewrite Actions command", func() {
	steps := NewSteps()

	Scenario("directory based helm chart", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits without error")
		steps.And("the rewrite actions are printed")
	})

	Scenario("tgz based helm chart", func() {
		steps.Given("a tgz based helm chart")
		steps.And("an image template list file")
		steps.And("a rules file that rewrites the registry")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits without error")
		steps.And("the rewrite actions are printed")
	})

	Scenario("missing helm chart", func() {
		steps.Given("no helm chart")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing helm chart")
		steps.And("it prints the usage")
	})

	Scenario("missing images template list file", func() {
		steps.Given("a directory based helm chart")
		steps.And("no image template list file")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing images template list file")
		steps.And("it prints the usage")
	})

	Scenario("missing rules file", func() {
		steps.Given("a directory based helm chart")
		steps.And("an image template list file")
		steps.And("no rewrite rules file")
		steps.And("chart-mover has been built")
		steps.When("running chart-mover rewrite-actions")
		steps.Then("the command exits with an error about the missing rules file")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		DefineCommonSteps(define)

		define.When(`^running chart-mover rewrite-actions$`, func() {
			args := []string{"rewrite-actions", ChartPath}
			if ImageTemplateFile != "" {
				args = append(args, "--images", ImageTemplateFile)
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
			var images []string
			err := json.Unmarshal(CommandSession.Out.Contents(), &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(3))
			Expect(images[0]).To(Equal("my-registry.example.com/library/nginx:stable"))
			Expect(images[1]).To(Equal("my-registry.example.com/library/python:stable"))
			Expect(images[2]).To(Equal("my-registry.example.com/library/busybox:latest"))
		})

		define.Then(`^the command exits with an error about the missing helm chart$`, func() {
			Eventually(CommandSession).Should(gexec.Exit(1))
			Expect(CommandSession.Err).To(Say("Error: failed to load helm chart: no Chart.yaml exists in directory"))
			Expect(CommandSession.Err).To(Say("features/fixtures/empty-directory"))
		})

		define.Then(`^the command exits with an error about the missing images template list file$`, func() {
			Eventually(CommandSession).Should(gexec.Exit(1))
			Expect(CommandSession.Err).To(Say("Error: image list file is required"))
		})

		define.Then(`^the command exits with an error about the missing rules file$`, func() {
			Eventually(CommandSession).Should(gexec.Exit(1))
			Expect(CommandSession.Err).To(Say("Error: rewrite rules file is required"))
		})
	})
})
