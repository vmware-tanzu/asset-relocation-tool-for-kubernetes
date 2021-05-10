// +build enemies

package features

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"time"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Enemy tests", func() {
	steps := NewSteps()

	Context("Unauthorized", func() {
		Scenario("Listing and pulling images", func() {
			steps.Given("a helm chart referencing a image in a private registry")
			steps.And("an image template list file for the remote registry helm chart")
			steps.When("running relok8s list-images --pull")
			steps.Then("the command exits with an error")
			steps.And("the error message says it failed to pull because it was not authorized")
		})
	})

	Scenario("Listing and pulling images", func() {
		steps.Given("a helm chart referencing a image in a private registry")
		steps.And("an image template list file for the remote registry helm chart")
		steps.And("credentials to the private registry")
		steps.When("running relok8s list-images --pull")
		steps.Then("the command exits without error")
		steps.And("the remote image is pulled")
		steps.And("the remote image is printed")
	})

	Scenario("Rewritting and pushing images", func() {
		steps.Given("a helm chart referencing a image in a private registry")
		steps.And("an image template list file for the remote registry helm chart")
		steps.And("a rewrite rules file for overwriting the tag")
		steps.And("credentials to the private registry")
		steps.When("running relok8s rewrite-images --push")
		steps.Then("the command exits without error")
		steps.And("the remote image is pulled")
		steps.And("the image is tagged")
		steps.And("the rewritten image is pushed")
		steps.And("the rewritten image is printed")
	})

	steps.Define(func(define Definitions) {
		var registryAuth string
		DefineCommonSteps(define)

		define.Given(`^a helm chart referencing a image in a private registry$`, func() {
			ChartPath = path.Join("fixtures", "remotechart")
		})

		define.Given(`^an image template list file for the remote registry helm chart$`, func() {
			ImageTemplateFile = path.Join("fixtures", "remotechart.yaml")
		})

		define.Given(`^a rewrite rules file for overwriting the tag$`, func() {
			RewriteRulesFile = path.Join("fixtures", "rules", "new-tag.yaml")
		})

		define.Given(`^credentials to the private registry$`, func() {
			registryAuth = os.Getenv("REGISTRY_AUTH")
		}, func() {
			registryAuth = ""
		})

		define.When(`^running relok8s list-images --pull$`, func() {
			args := []string{"list-images", "--pull", ChartPath}
			if ImageTemplateFile != "" {
				args = append(args, "--image-templates", ImageTemplateFile)
			}
			if registryAuth != "" {
				args = append(args, "--registry-auth", registryAuth)
			}
			command := exec.Command(ChartMoverBinaryPath, args...)
			var err error
			CommandSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		define.When(`^running relok8s rewrite-images --push$`, func() {
			args := []string{"rewrite-images", "--push", ChartPath}
			if ImageTemplateFile != "" {
				args = append(args, "--image-templates", ImageTemplateFile)
			}
			if RewriteRulesFile != "" {
				args = append(args, "--rules-file", RewriteRulesFile)
			}
			if registryAuth != "" {
				args = append(args, "--registry-auth", registryAuth)
			}
			command := exec.Command(ChartMoverBinaryPath, args...)
			var err error
			CommandSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		define.Then(`^the error message says it failed to pull because it was not authorized$`, func() {
			Eventually(CommandSession.Err, 5*time.Second).Should(Say("Error response from daemon: unauthorized: unauthorized to access repository"))
		})

		define.Then(`^the remote image is pulled$`, func() {
			Eventually(CommandSession.Err).Should(Say("Pulling harbor-repo.vmware.com/pwall/tiny:tiniest... Done"))
		})

		define.Then(`^the image is tagged$`, func() {
			Eventually(CommandSession.Err).Should(Say("Tagging harbor-repo.vmware.com/pwall/tiny:rewritten... Done"))
		})

		define.Then(`^the remote image is printed$`, func() {
			var images []string
			err := json.Unmarshal(CommandSession.Out.Contents(), &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(1))
			Expect(images).To(ContainElement("harbor-repo.vmware.com/pwall/tiny:tiniest"))
		})

		define.Then(`^the rewritten image is pushed$`, func() {
			Eventually(CommandSession.Err).Should(Say("Pushing harbor-repo.vmware.com/pwall/tiny:rewritten... Done"))
		})

		define.Then(`^the rewritten image is printed$`, func() {
			var images []string
			err := json.Unmarshal(CommandSession.Out.Contents(), &images)
			Expect(err).ToNot(HaveOccurred())

			Expect(images).To(HaveLen(1))
			Expect(images).To(ContainElement("harbor-repo.vmware.com/pwall/tiny:rewritten"))
		})
	})
})
