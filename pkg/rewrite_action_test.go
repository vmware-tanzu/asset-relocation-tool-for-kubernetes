package pkg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg"
)

var _ = Describe("RewriteAction", func() {
	Describe("TopLevelKey", func() {
		It("returns the first part of the path", func() {
			action := &pkg.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.TopLevelKey()).To(Equal("alpha"))
		})
	})

	Describe("GetPathToMap", func() {
		It("returns the path without the final key", func() {
			action := &pkg.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetPathToMap()).To(Equal(".alpha.bravo.charlie"))
		})
	})

	Describe("GetSubPathToMap", func() {
		It("returns the path without the final key and the top-level key", func() {
			action := &pkg.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetSubPathToMap()).To(Equal(".bravo.charlie"))
		})
	})

	Describe("GetKey", func() {
		It("returns the last part of the path", func() {
			action := &pkg.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetKey()).To(Equal("delta"))
		})
	})

	Describe("ToMap", func() {
		Context("one key", func() {
			It("becomes a flat map", func() {
				action := &pkg.RewriteAction{
					Path:  ".alpha",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKeyWithValue("alpha", "needle"))
			})
		})

		Context("two keys", func() {
			It("becomes a nested map", func() {
				action := &pkg.RewriteAction{
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
				action := &pkg.RewriteAction{
					Path:  ".alpha.bravo.charlie.delta",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKey("alpha"))

				var ok bool
				haystack, ok = haystack["alpha"].(pkg.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("bravo"))

				haystack, ok = haystack["bravo"].(pkg.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("charlie"))

				haystack, ok = haystack["charlie"].(pkg.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKeyWithValue("delta", "needle"))
			})
		})
	})

	Describe("Apply", func() {

	})

	//Describe("FindChartDestination", func() {
	//	Context("action refers to a chart dependency", func() {
	//		It("returns the dependent chart", func() {
	//		})
	//	})
	//})
})
