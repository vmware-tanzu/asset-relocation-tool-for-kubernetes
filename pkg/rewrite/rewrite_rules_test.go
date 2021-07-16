package rewrite_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/rewrite"
	common "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/pkg/rewrite"
)

var _ = Describe("RewriteRules", func() {

	Describe("IsEmpty", func() {
		Context("Empty rules", func() {
			It("returns true", func() {
				empty := &common.Rules{}
				Expect(empty.IsEmpty()).To(BeTrue())
			})
		})

		Context("Not empty rules", func() {
			It("returns false", func() {
				Expect((&rewrite.Rules{Registry: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&rewrite.Rules{RepositoryPrefix: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&rewrite.Rules{Repository: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&rewrite.Rules{Tag: "abc"}).IsEmpty()).To(BeFalse())
				Expect((&rewrite.Rules{Digest: "abc"}).IsEmpty()).To(BeFalse())
			})
		})
	})
})
