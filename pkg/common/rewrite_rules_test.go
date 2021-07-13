package common_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/common"
)

var _ = Describe("RewriteRules", func() {

	Describe("IsEmpty", func() {
		Context("Empty rules", func() {
			It("returns true", func() {
				empty := &common.RewriteRules{}
				Expect(empty.IsEmpty()).To(BeTrue())
			})
		})

		Context("Not empty rules", func() {
			It("returns false", func() {
				Expect((&common.RewriteRules{Registry: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&common.RewriteRules{RepositoryPrefix: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&common.RewriteRules{Repository: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&common.RewriteRules{Tag: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&common.RewriteRules{Digest: "abc"}).IsEmpty()).To(BeFalse())
			})
		})
	})
})