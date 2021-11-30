// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package external_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/test"
	"helm.sh/helm/v3/pkg/chart/loader"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("External tests", func() {
	var (
		customRepoPrefix string
		tmpDir           string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "external-tests-*")
		if err != nil {
			log.Fatalf("Failed to create temporary dir")
		}
	})

	AfterEach(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Fatalf("failed to close temporary directory %s", tmpDir)
		}
	})

	steps := NewSteps()
	// Using a custom repo prefix in every test run
	customRepoPrefix = fmt.Sprintf("%s/ci-tests/%d", "tanzu_isv_engineering_private", time.Now().Unix())

	Context("Unauthorized", func() {
		Scenario("running chart move", func() {
			steps.Given("no authorization to the remote registry")
			steps.When(fmt.Sprintf("running relok8s chart move -y ../fixtures/testchart --image-patterns ../fixtures/testchart.images.yaml --repo-prefix %s", customRepoPrefix))
			steps.Then("the command exits with an error")
			steps.And("the error message says it failed to pull because it was not authorized")
		})
	})

	Scenario("running chart move", func() {
		steps.When(fmt.Sprintf("running relok8s chart move -y ../fixtures/testchart --image-patterns ../fixtures/testchart.images.yaml --repo-prefix %s", customRepoPrefix))
		steps.And("the move is computed")
		steps.Then("the command says that the rewritten image will be pushed")
		steps.And("the command says that the rewritten images will be written to the chart and subchart")
		steps.And("the command exits without error")
		steps.And("the chart name and version is shown before relocation")
		steps.And("the tagged versions are pushed")
		steps.And("the modified chart is written")
		steps.And("the location of the chart is shown")
		steps.And("the modified chart contains the digest tagged image")
	})

	Scenario("running chart move to intermediate bundle", func() {
		steps.When(fmt.Sprintf("running relok8s chart move -y ../fixtures/testchart --image-patterns ../fixtures/testchart.images.yaml --to-archive %s/testchart-intermediate.tar", tmpDir))
		steps.And("the move is computed")
		steps.Then("the command says it will archive the chart")
		steps.Then("the command says it is writing the hints file")
		steps.Then("the command says it is writing the Helm Chart files")
		steps.Then("the command says it is writing the container images")
		steps.Then("the command says the intermediate bundle is complete")
	})

	Scenario("running chart move from intermediate bundle", func() {
		oldprefix := customRepoPrefix
		customRepoPrefix += "-unbundled"
		steps.When(fmt.Sprintf("running relok8s chart move -y ../fixtures/testchart-intermediate.tar --repo-prefix %s", customRepoPrefix))
		steps.And("the move is computed")
		steps.Then("the command says that the unbundled & rewritten image will be pushed")
		steps.And("the command says that the rewritten images will be written to the chart and subchart")
		steps.And("the command exits without error")
		steps.And("the chart name and version is shown before relocation")
		steps.And("the tagged versions are pushed")
		steps.And("the modified chart is written")
		steps.And("the location of the chart is shown")
		customRepoPrefix = oldprefix
	})

	steps.Define(func(define Definitions) {
		test.DefineCommonSteps(define)

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
			Eventually(test.CommandSession.Err, time.Minute).Should(Say("[uU]nauthorized"))
		})

		define.Then(`^the move is computed$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Computing relocation...\n"))
		})

		define.Then(`^the command says that the rewritten image will be pushed$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Image copies:"))
			Eventually(test.CommandSession.Out).Should(Say("harbor-repo.vmware.com/tanzu_isv_engineering/tiny:tiniest => harbor-repo.vmware.com/%s/tiny:tiniest \\(sha256:[a-z0-9]*\\) \\(push required\\)", customRepoPrefix))
		})

		define.Then(`^the command says that the unbundled & rewritten image will be pushed$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Image copies:"))
			Eventually(test.CommandSession.Out).Should(Say("\\(bundle ../fixtures/testchart-intermediate.tar:harbor-repo.vmware.com/tanzu_isv_engineering/tiny:tiniest\\) => harbor-repo.vmware.com/%s/tiny:tiniest \\(sha256:[a-z0-9]*\\) \\(push required\\)", customRepoPrefix))
			Eventually(test.CommandSession.Out).Should(Say("\\(bundle ../fixtures/testchart-intermediate.tar:harbor-repo.vmware.com/dockerhub-proxy-cache/library/busybox:1.33.1\\) => harbor-repo.vmware.com/%s/busybox@sha256:[a-z0-9]* \\(sha256:[a-z0-9]*\\) \\(push required\\)", customRepoPrefix))
		})

		define.Then(`^the command says that the rewritten images will be written to the chart and subchart$`, func() {
			Eventually(test.CommandSession.Out).Should(Say("Changes to be applied to testchart/values.yaml:"))
			Eventually(test.CommandSession.Out).Should(Say("  .image.repository: harbor-repo.vmware.com/%s/tiny", customRepoPrefix))
			Eventually(test.CommandSession.Out).Should(Say("  .sameImageButNoTagRequirement.image: harbor-repo.vmware.com/%s/tiny@sha256:[a-z0-9]*", customRepoPrefix))
			Eventually(test.CommandSession.Out).Should(Say("  .singleImageReference.image: harbor-repo.vmware.com/%s/busybox@sha256:[a-z0-9]*", customRepoPrefix))
			// Subchart
			Eventually(test.CommandSession.Out).Should(Say("Changes to be applied to testchart/charts/subchart/values.yaml:"))
			Eventually(test.CommandSession.Out).Should(Say("  .image.name: harbor-repo.vmware.com/%s/tiny", customRepoPrefix))
		})

		define.Then(`^the digest version is written to the chart$`, func() {
			Eventually(test.CommandSession.Out).Should(Say("Changes to be applied to testchart/values.yaml:"))
			Eventually(test.CommandSession.Out).Should(Say(fmt.Sprintf("  .sameImageButNoTagRequirement.image: harbor-repo.vmware.com/%s/tiny@sha256:[a-z0-9]*", customRepoPrefix)))
		})

		define.Then(`^the tagged versions are pushed$`, func() {
			Eventually(test.CommandSession.Out).Should(Say(fmt.Sprintf("Pushing harbor-repo.vmware.com/%s/tiny:tiniest...\nDone", customRepoPrefix)))
			Eventually(test.CommandSession.Out).Should(Say(fmt.Sprintf("Pushing harbor-repo.vmware.com/%s/busybox:1.33.1...\nDone", customRepoPrefix)))
		})

		define.Then(`^the chart name and version is shown before relocation$`, func() {
			Eventually(test.CommandSession.Out).Should(Say("Relocating testchart@0.1.0..."))
		})

		define.Then(`^the location of the chart is shown$`, func() {
			Eventually(test.CommandSession.Out).Should(Say("testchart-0.1.0.relocated.tgz"))
		})

		define.Then("^the modified chart contains the digest tagged image$", func() {
			chart, err := loader.Load("testchart-0.1.0.relocated.tgz")
			Expect(err).ToNot(HaveOccurred())
			writtenImage := chart.Values["sameImageButNoTagRequirement"].(map[string]interface{})["image"]
			Expect(writtenImage).To(MatchRegexp("harbor-repo.vmware.com/%s/tiny@sha256:[a-z0-9]*", customRepoPrefix))
		})

		var modifiedChartPath string
		define.Then(`^the modified chart is written$`, func() {
			cwd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			modifiedChartPath = filepath.Join(cwd, "testchart-0.1.0.relocated.tgz")
			modifiedChart, err := loader.Load(modifiedChartPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(modifiedChart.Values["image"]).To(HaveKeyWithValue("repository", fmt.Sprintf("harbor-repo.vmware.com/%s/tiny", customRepoPrefix)))
			// Subchart was rewritten too
			for _, subchart := range modifiedChart.Dependencies() {
				if subchart.Name() == "subchart" {
					Expect(subchart.Values["image"]).To(HaveKeyWithValue("name", fmt.Sprintf("harbor-repo.vmware.com/%s/tiny", customRepoPrefix)))
				}
			}

			imageMap, ok := modifiedChart.Values["sameImageButNoTagRequirement"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(imageMap).To(HaveKey("image"))
			Expect(imageMap["image"]).To(ContainSubstring(fmt.Sprintf("harbor-repo.vmware.com/%s/tiny@sha256:", customRepoPrefix)))

			imageMap, ok = modifiedChart.Values["singleImageReference"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(imageMap).To(HaveKey("image"))
			Expect(imageMap["image"]).To(ContainSubstring(fmt.Sprintf("harbor-repo.vmware.com/%s/busybox@sha256:", customRepoPrefix)))
		}, func() {
			if modifiedChartPath != "" {
				_ = os.Remove(modifiedChartPath)
				modifiedChartPath = ""
			}
		})

		define.Then(`^the command says it will archive the chart$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Will archive Helm Chart testchart@0.1.0, dependent images and hint file to intermediate tarball "))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("2 images detected to be archived"))
		})

		define.Then(`^the command says it is writing the Helm Chart files$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Writing Helm Chart files at original-chart/..."))
		})

		define.Then(`^the command says it is writing the container images$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Processing image harbor-repo.vmware.com/tanzu_isv_engineering/tiny:tiniest\n"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Processing image harbor-repo.vmware.com/dockerhub-proxy-cache/library/busybox:1.33.1\n"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Writing 2 images...\n"))
		})

		define.Then(`^the command says the intermediate bundle is complete$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Intermediate bundle complete at %s/testchart-intermediate.tar\n", tmpDir))
		})

		define.Then(`^the command says it is writing the hints file$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Writing hints.yaml...\n"))
		})

		define.Then(`^remove the archive folder at testchart-intermediate.tar$`, func() {
			err := os.Remove("testchart-intermediate.tar")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
