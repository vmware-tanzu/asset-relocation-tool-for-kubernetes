package lib_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
)

var _ = Describe("RewriteAction", func() {
	Describe("ToMap", func() {
		Context("one key", func() {
			It("becomes a flat map", func() {
				action := &RewriteAction{
					Path:  ".alpha",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKeyWithValue("alpha", "needle"))
			})
		})

		Context("two keys", func() {
			It("becomes a nested map", func() {
				action := &RewriteAction{
					Path:  ".alpha.bravo",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKey("alpha"))
				haystackLevelTwo := haystack["alpha"]
				Expect(haystackLevelTwo).To(HaveKeyWithValue("bravo", "needle"))
			})
		})

		Context("multiple keys", func() {
			It("becomes a deeply nested map", func() {
				action := &RewriteAction{
					Path:  ".alpha.bravo.charlie.delta",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKey("alpha"))

				var ok bool
				haystack, ok = haystack["alpha"].(ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("bravo"))

				haystack, ok = haystack["bravo"].(ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("charlie"))

				haystack, ok = haystack["charlie"].(ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKeyWithValue("delta", "needle"))
			})
		})
	})
})
