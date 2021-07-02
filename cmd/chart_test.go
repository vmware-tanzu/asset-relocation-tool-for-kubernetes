package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/cmd"
)

var _ = Describe("Chart", func() {
	Describe("ParseOutputFlag", func() {
		It("works with default out flag", func() {
			got, err := cmd.ParseOutputFlag(cmd.Output)
			want := "./%s-%s.relocated.tgz"
			Expect(got).To(Equal(want))
			Expect(err).To(BeNil())
		})
		It("rejects out flag without wildcard *", func() {
			_, err := cmd.ParseOutputFlag("nowildcardhere.tgz")
			Expect(err).Should(MatchError(cmd.ErrorMissingOutPlaceHolder))
		})
		It("rejects out flag without proper extension", func() {
			_, err := cmd.ParseOutputFlag("*-wildcardhere")
			Expect(err).Should(MatchError(cmd.ErrorBadExtension))
		})
		It("accepts out flag with wildcard", func() {
			got, err := cmd.ParseOutputFlag("*-wildcardhere.tgz")
			Expect(got).To(Equal("%s-%s-wildcardhere.tgz"))
			Expect(err).To(BeNil())
		})
	})
})
