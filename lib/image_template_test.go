package lib_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib/libfakes"
	"gopkg.in/yaml.v2"

	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

var _ = Describe("ImageTemplate", func() {
	var (
		chart  *libfakes.FakeHelmChart
		rules  *RewriteRules
		values map[string]string
	)

	BeforeEach(func() {
		chart = &libfakes.FakeHelmChart{}
		rules = &RewriteRules{
			Registry:         "internal.vmware.com",
			RepositoryPrefix: "mycompany",
			Tag:              "1.2.3",
		}
	})

	JustBeforeEach(func() {
		encoded, err := yaml.Marshal(values)
		Expect(err).ToNot(HaveOccurred())
		chart.GetValuesReturns(&helmchart.Config{
			Raw: string(encoded),
		})
	})

	Context("image with single template", func() {
		BeforeEach(func() {
			values = map[string]string{
				"singleImageReference": "myimage:latest",
			}
		})

		It("behaves correctly", func() {
			template, err := NewFromString("{{ .Values.singleImageReference }}")

			By("parsing the template string", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(template.Raw).To(Equal("{{ .Values.singleImageReference }}"))
				Expect(template.Template).ToNot(BeNil())
			})

			By("rendering from values", func() {
				err := template.Render(chart)
				Expect(err).ToNot(HaveOccurred())

				Expect(template.OriginalImage).ToNot(BeNil())
				Expect(template.OriginalImage.Remote()).To(Equal("docker.io/library/myimage:latest"))

				By("applying the rewrite rules correctly", func() {
					actions, err := template.Apply(rules)
					Expect(err).ToNot(HaveOccurred())

					Expect(template.NewImage).ToNot(BeNil())
					Expect(template.NewImage.Remote()).To(Equal("internal.vmware.com/mycompany/library/myimage:1.2.3"))

					Expect(actions).To(HaveLen(1))
					Expect(actions[0].Path).To(Equal(".Values.singleImageReference"))
					Expect(actions[0].Value).To(Equal("internal.vmware.com/mycompany/library/myimage:1.2.3"))
				})
			})
		})
	})
})
