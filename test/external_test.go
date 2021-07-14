// +build external

package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/rewrite"
	"helm.sh/helm/v3/pkg/chart/loader"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"gopkg.in/yaml.v2"
)

var _ = Describe("External tests", func() {
	steps := NewSteps()

	Context("Unauthorized", func() {
		Scenario("running chart move", func() {
			steps.Given("no authorization to the remote registry")
			steps.When("running relok8s chart move -y fixtures/testchart --image-patterns fixtures/testchart.images.yaml --repo-prefix tanzu_isv_engineering_private")
			steps.Then("the command exits with an error")
			steps.And("the error message says it failed to pull because it was not authorized")
		})
	})

	Scenario("running chart move", func() {
		steps.Given("a rules file with a custom tag") // This is used for forcing a new tag, ensuring the target is new
		steps.When("running relok8s chart move -y fixtures/testchart --image-patterns fixtures/testchart.images.yaml --repo-prefix tanzu_isv_engineering_private")

		steps.And("the images are pulled")
		steps.And("the rewritten images are checked to see if they need to be pushed")
		steps.Then("the command says that the rewritten image will be pushed")
		steps.And("the command says that the rewritten images will be written to the chart")
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
			bytes, err := yaml.Marshal(&rewrite.Rules{
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

		define.Given(`^no authorization to the remote registry$`, func() {
			homeDir, err := os.UserHomeDir()
			Expect(err).ToNot(HaveOccurred())

			err = os.Rename(
				filepath.Join(homeDir, ".docker", "config.json"),
				filepath.Join(homeDir, ".docker", "totally-not-the-config.json.backup"),
			)
			Expect(err).ToNot(HaveOccurred())
		}, func() {
			homeDir, err := os.UserHomeDir()
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(filepath.Join(homeDir, ".docker", "totally-not-the-config.json.backup"))
			if !os.IsNotExist(err) {
				Expect(err).ToNot(HaveOccurred())

				err = os.Rename(
					filepath.Join(homeDir, ".docker", "totally-not-the-config.json.backup"),
					filepath.Join(homeDir, ".docker", "config.json"),
				)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		define.Then(`^the error message says it failed to pull because it was not authorized$`, func() {
			Eventually(CommandSession.Err, time.Minute).Should(Say("[uU]nauthorized"))
		})

		define.Then(`^the images are pulled$`, func() {
			Eventually(CommandSession.Out, time.Minute).Should(Say("Pulling harbor-repo.vmware.com/tanzu_isv_engineering/tiny:tiniest... Done"))
			Eventually(CommandSession.Out, time.Minute).Should(Say("Pulling harbor-repo.vmware.com/dockerhub-proxy-cache/library/busybox:1.33.1... Done"))
		})

		define.Then(`^the rewritten images are checked to see if they need to be pushed$`, func() {
			Eventually(CommandSession.Out, time.Minute).Should(Say(fmt.Sprintf("Checking harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny:%s \\(sha256:[a-z0-9]*\\)... Push required", customTag)))
			Eventually(CommandSession.Out, time.Minute).Should(Say("Checking harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny@sha256:[a-z0-9]* \\(sha256:[a-z0-9]*\\)... Already exists"))
			Eventually(CommandSession.Out, time.Minute).Should(Say("Checking harbor-repo.vmware.com/tanzu_isv_engineering_private/busybox@sha256:[a-z0-9]* \\(sha256:[a-z0-9]*\\)... Already exists"))
		})

		define.Then(`^the command says that the rewritten image will be pushed$`, func() {
			Eventually(CommandSession.Out, time.Minute).Should(Say("Images to be pushed:"))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny:%s \\(sha256:[a-z0-9]*\\)", customTag)))
		})

		define.Then(`^the command says that the rewritten images will be written to the chart$`, func() {
			Eventually(CommandSession.Out).Should(Say("Changes written to testchart/values.yaml:"))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  .image.tag: %s", customTag)))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  .image.repository: harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny")))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  .sameImageButNoTagRequirement.image: harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny@sha256:[a-z0-9]*")))
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("  .singleImageReference.image: harbor-repo.vmware.com/tanzu_isv_engineering_private/busybox@sha256:[a-z0-9]*")))
		})

		define.Then(`^the rewritten image is pushed$`, func() {
			Eventually(CommandSession.Out).Should(Say(fmt.Sprintf("Pushing harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny:%s... Done", customTag)))
		})

		var modifiedChartPath string
		define.Then(`^the modified chart is written$`, func() {
			cwd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			modifiedChartPath = filepath.Join(cwd, "testchart-0.1.0.relocated.tgz")
			modifiedChart, err := loader.Load(modifiedChartPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(modifiedChart.Values["image"]).To(HaveKeyWithValue("repository", "harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny"))

			Expect(modifiedChart.Values["image"]).To(HaveKeyWithValue("tag", customTag))

			imageMap, ok := modifiedChart.Values["sameImageButNoTagRequirement"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(imageMap).To(HaveKey("image"))
			Expect(imageMap["image"]).To(ContainSubstring("harbor-repo.vmware.com/tanzu_isv_engineering_private/tiny@sha256:"))

			imageMap, ok = modifiedChart.Values["singleImageReference"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(imageMap).To(HaveKey("image"))
			Expect(imageMap["image"]).To(ContainSubstring("harbor-repo.vmware.com/tanzu_isv_engineering_private/busybox@sha256:"))
		}, func() {
			if modifiedChartPath != "" {
				os.Remove(modifiedChartPath)
				modifiedChartPath = ""
			}
		})
	})
})
