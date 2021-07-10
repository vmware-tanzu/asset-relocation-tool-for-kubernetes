package mover

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("targetOutput", func() {
	It("works with default out flag", func() {
		outFmt := "./%s-%s.relocated.tgz"
		target := targetOutput("path", outFmt, "my-chart", "0.1")
		Expect(target).To(Equal("path/my-chart-0.1.relocated.tgz"))
	})
	It("builds custom out input as expected", func() {
		target := targetOutput("path", "%s-%s-wildcardhere.tgz", "my-chart", "0.1")
		Expect(target).To(Equal("path/my-chart-0.1-wildcardhere.tgz"))
	})
})
