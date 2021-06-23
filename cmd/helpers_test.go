package cmd_test

import (
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/cmd"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	FixturesRoot = "../test/fixtures/"
)

var _ = Describe("Helpers", func() {
	Describe("ReadImagePatterns", func() {
		It("reads from given file first if present", func() {
			imagefile := filepath.Join(FixturesRoot, "testchart.images.yaml")
			contents, err := cmd.ReadImagePatterns(imagefile, nil)
			Expect(err).To(BeNil())
			expected, err2 := ioutil.ReadFile(imagefile)
			Expect(err2).To(BeNil())
			Expect(contents).To(Equal(expected))
		})
		It("reads from chart if file missing", func() {
			chart, err := loader.Load(filepath.Join(FixturesRoot, "self-relok8ing-chart"))
			Expect(err).To(BeNil())
			contents, err2 := cmd.ReadImagePatterns("", chart)
			Expect(err2).To(BeNil())
			embeddedPatterns := filepath.Join(FixturesRoot, "self-relok8ing-chart/.relok8s-images.yaml")
			expected, err3 := ioutil.ReadFile(embeddedPatterns)
			Expect(err3).To(BeNil())
			Expect(contents).To(Equal(expected))
		})
		It("reads nothing when no file and the chart is not self relok8able", func() {
			chart, err := loader.Load(filepath.Join(FixturesRoot, "testchart"))
			Expect(err).To(BeNil())
			contents, err2 := cmd.ReadImagePatterns("", chart)
			Expect(err2).To(BeNil())
			Expect(contents).To(BeNil())
		})
	})

	Describe("TargetOutput", func() {
		It("works with default out flag", func() {
			outFmt, err := cmd.ParseOutputFlag(cmd.Output)
			Expect(err).To(BeNil())
			target := cmd.TargetOutput("path", outFmt, "my-chart", "0.1")
			Expect(target).To(Equal("path/my-chart-0.1.relocated.tgz"))
		})
		It("builds custom out input as expected", func() {
			target := cmd.TargetOutput("path", "%s-%s-wildcardhere.tgz", "my-chart", "0.1")
			Expect(target).To(Equal("path/my-chart-0.1-wildcardhere.tgz"))
		})
	})
})
