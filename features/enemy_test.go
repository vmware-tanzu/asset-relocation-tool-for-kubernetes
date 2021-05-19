// +build enemies

package features

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"helm.sh/helm/v3/pkg/chart/loader"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Enemy tests", func() {
	steps := NewSteps()

	Context("Unauthorized", func() {
		Scenario("Listing and pulling images", func() {
			steps.When("running relok8s chart move -y fixtures/testchart --image-patterns fixtures/testchart.images.yaml --rules fixtures/tag-rule.yaml")
			steps.Then("the command exits with an error")
			steps.And("the error message says it failed to pull because it was not authorized")
		})
	})

	Scenario("relocating a chart", func() {
		steps.Given("credentials to the private registry")
		steps.And("a rules file with a custom tag")
		steps.When("running relok8s chart move -y fixtures/testchart --image-patterns fixtures/testchart.images.yaml --repo-prefix tanzu_isv_engineering")

		steps.And("the image is pulled")
		steps.Then("the command says that the rewritten image will be pushed")
		steps.And("the command says that the rewritten image will be written to the chart")
		steps.And("the command exits without error")
		steps.And("the rewritten image is pushed")
		steps.And("the modified chart is written")
	})

	steps.Define(func(define Definitions) {
		DefineCommonSteps(define)

		var customTag string

		define.Given(`^a rules file with a custom tag$`, func() {
			var err error
			RulesFile, err = ioutil.TempFile("", "rulesfile-*.yaml")
			Expect(err).ToNot(HaveOccurred())

			customTag = fmt.Sprintf("%d", time.Now().Unix())
			bytes, err := yaml.Marshal(&lib.RewriteRules{
				Tag: customTag,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = RulesFile.Write(bytes)
			Expect(err).ToNot(HaveOccurred())

			err = RulesFile.Close()
			Expect(err).ToNot(HaveOccurred())
		}, func() {
			if RulesFile != nil {
				os.Remove(RulesFile.Name())
				RulesFile = nil
			}
		})

		define.Given(`^credentials to the private registry$`, func() {
			RegistryAuth = os.Getenv("REGISTRY_AUTH")
			Expect(RegistryAuth).ToNot(BeEmpty())
		}, func() {
			RegistryAuth = ""
		})

		define.Then(`^the error message says it failed to pull because it was not authorized$`, func() {
			Eventually(CommandSession.Err, time.Minute).Should(Say("Error response from daemon: unauthorized: unauthorized to access repository"))
		})

		define.Then(`^the image is pulled$`, func() {
			Eventually(CommandSession.Out, time.Minute).Should(Say("Pulling harbor-repo.vmware.com/pwall/tiny:tiniest... Done"))
		})

		define.Then(`^the command says that the rewritten image will be pushed$`, func() {
			Eventually(CommandSession.Out, time.Minute).Should(Say("Images to be pushed:"))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  harbor-repo.vmware.com/tanzu_isv_engineering/tiny:%s \\(sha256:[a-z0-9]*\\)", customTag)))
		})

		define.Then(`^the command says that the rewritten image will be written to the chart$`, func() {
			Eventually(CommandSession.Out).Should(Say("Changes written to testchart/values.yaml:"))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  .image.tag: %s", customTag)))
			Eventually(CommandSession.Out).Should(Say("  .image.repository: harbor-repo.vmware.com/tanzu_isv_engineering/tiny"))
		})

		define.Then(`^the rewritten image is pushed$`, func() {
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("Pushing harbor-repo.vmware.com/tanzu_isv_engineering/tiny:%s... Done", customTag)))
		})

		var modifiedChartPath string
		define.Then(`^the modified chart is written$`, func() {
			cwd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			modifiedChartPath = filepath.Join(cwd, "testchart-0.1.0.relocated.tgz")
			modifiedChart, err := loader.Load(modifiedChartPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(modifiedChart.Values["image"]).To(HaveKeyWithValue("repository", "harbor-repo.vmware.com/tanzu_isv_engineering/tiny"))
			Expect(modifiedChart.Values["image"]).To(HaveKeyWithValue("tag", customTag))
		}, func() {
			if modifiedChartPath != "" {
				os.Remove(modifiedChartPath)
				modifiedChartPath = ""
			}
		})
	})
})
