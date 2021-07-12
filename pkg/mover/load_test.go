package mover_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/mover"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	FixturesRoot = "../../test/fixtures/"
)

var _ = Describe("LoadImagePatterns", func() {
	It("reads from given file first if present", func() {
		imagefile := filepath.Join(FixturesRoot, "testchart.images.yaml")
		contents, err := mover.LoadImagePatterns(imagefile, nil)
		Expect(err).To(BeNil())
		expectedContents, err2 := os.ReadFile(imagefile)
		expected := string(expectedContents)
		Expect(err2).To(BeNil())
		Expect(contents).To(Equal(expected))
	})
	It("reads from chart if file missing", func() {
		chart, err := loader.Load(filepath.Join(FixturesRoot, "self-relok8ing-chart"))
		Expect(err).To(BeNil())
		contents, err2 := mover.LoadImagePatterns("", chart)
		Expect(err2).To(BeNil())
		embeddedPatterns := filepath.Join(FixturesRoot, "self-relok8ing-chart/.relok8s-images.yaml")
		expectedContents, err3 := os.ReadFile(embeddedPatterns)
		expected := string(expectedContents)
		Expect(err3).To(BeNil())
		Expect(contents).To(Equal(expected))
	})
	It("reads nothing when no file and the chart is not self relok8able", func() {
		chart, err := loader.Load(filepath.Join(FixturesRoot, "testchart"))
		Expect(err).To(BeNil())
		contents, err2 := mover.LoadImagePatterns("", chart)
		Expect(err2).To(BeNil())
		Expect(contents).To(BeEmpty())
	})
})
