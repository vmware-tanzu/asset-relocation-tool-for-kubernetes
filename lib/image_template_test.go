package lib_test

import (
	dockerparser "github.com/novln/docker-parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"helm.sh/helm/v3/pkg/chart"
)

var _ = Describe("NewFromString", func() {
	Context("Empty string", func() {
		It("returns an error", func() {
			_, err := NewFromString("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"\": missing repo or a registry fragment"))
		})
	})
	Context("Invalid template", func() {
		It("returns an error", func() {
			_, err := NewFromString("not a template")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"not a template\": missing repo or a registry fragment"))
		})
	})
	Context("Single template", func() {
		It("parses successfully", func() {
			imageTemplate, err := NewFromString("{{ .image }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .image }}"))
			Expect(imageTemplate.RegistryTemplate).To(Equal(""))
			Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
			Expect(imageTemplate.TagTemplate).To(Equal(""))
			Expect(imageTemplate.DigestTemplate).To(Equal(""))
		})

	})
	Context("Image and tag", func() {
		imageTemplate, err := NewFromString("{{ .image }}:{{ .tag }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .image }}:{{ .tag }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(""))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
		Expect(imageTemplate.TagTemplate).To(Equal(".tag"))
		Expect(imageTemplate.DigestTemplate).To(Equal(""))
	})
	Context("Image and multiple tags", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .image }}:{{ .tag1 }}:{{ .tag2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}:{{ .tag1 }}:{{ .tag2 }}\": too many tag template matches"))
		})
	})
	Context("Image and digest", func() {
		imageTemplate, err := NewFromString("{{ .image }}@{{ .digest }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .image }}@{{ .digest }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(""))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
		Expect(imageTemplate.TagTemplate).To(Equal(""))
		Expect(imageTemplate.DigestTemplate).To(Equal(".digest"))
	})
	Context("Image and multiple digests", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .image }}@{{ .digest1 }}@{{ .digest2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}@{{ .digest1 }}@{{ .digest2 }}\": too many digest template matches"))
		})

	})
	Context("registry, image, and tag", func() {
		imageTemplate, err := NewFromString("{{ .registry }}/{{ .image }}:{{ .tag }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .registry }}/{{ .image }}:{{ .tag }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(".registry"))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(".image"))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.TagTemplate).To(Equal(".tag"))
		Expect(imageTemplate.DigestTemplate).To(Equal(""))
	})
	Context("Too many templates", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .a }}/{{ .b }}/{{ .c }}/{{ .d }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .a }}/{{ .b }}/{{ .c }}/{{ .d }}\": more fragments than expected"))
		})

	})
})

type ChartInput struct {
	Name   string
	Values map[string]interface{}
}

type TableInput struct {
	Values       map[string]interface{}
	Dependencies []*ChartInput
	Template     string
}
type TableOutput struct {
	Image          string
	RewrittenImage string
	Actions        []*RewriteAction
}

var (
	imageAlone = &TableInput{
		Values: map[string]interface{}{
			"image": "ubuntu:latest",
		},
		Template: "{{ .image }}",
	}
	imageAndTag = &TableInput{
		Values: map[string]interface{}{
			"image": "petewall/amazingapp",
			"tag":   "latest",
		},
		Template: "{{ .image }}:{{ .tag }}",
	}
	registryAndImage = &TableInput{
		Values: map[string]interface{}{
			"registry": "quay.io",
			"image":    "proxy/nginx",
		},
		Template: "{{ .registry }}/{{ .image }}",
	}
	registryImageAndTag = &TableInput{
		Values: map[string]interface{}{
			"registry": "quay.io",
			"image":    "busycontainers/busybox",
			"tag":      "busiest",
		},
		Template: "{{ .registry }}/{{ .image }}:{{ .tag }}",
	}
	imageAndDigest = &TableInput{
		Values: map[string]interface{}{
			"image":  "petewall/platformio",
			"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Template: "{{ .image }}@{{ .digest }}",
	}

	dependencyRegistryImageAndTag = &TableInput{
		Values: map[string]interface{}{
			"registry": "quay.io",
			"image":    "busycontainers/busybox",
			"tag":      "busiest",
		},
		Dependencies: []*ChartInput{
			{
				Name: "lazy",
				Values: map[string]interface{}{
					"registry": "docker.io",
					"image":    "lazycontainers/lazybox",
					"tag":      "laziest",
				},
			},
		},
		Template: "{{ .lazy.registry }}/{{ .lazy.image }}:{{ .lazy.tag }}",
	}

	registryRule          = &RewriteRules{Registry: "registry.vmware.com"}
	repositoryPrefixRule  = &RewriteRules{RepositoryPrefix: "my-company"}
	registryAndPrefixRule = &RewriteRules{Registry: "registry.vmware.com", RepositoryPrefix: "my-company"}
)

func MakeChart(input *TableInput) *chart.Chart {
	newChart := &chart.Chart{
		Values: input.Values,
	}
	for _, dependency := range input.Dependencies {
		newChart.AddDependency(&chart.Chart{
			Metadata: &chart.Metadata{
				Name: dependency.Name,
			},
			Values: dependency.Values,
		})
	}

	return newChart
}

var _ = DescribeTable("Rewrite Actions",
	func(input *TableInput, rules *RewriteRules, expected *TableOutput) {
		var (
			err           error
			chart         = MakeChart(input)
			template      *ImageTemplate
			originalImage *dockerparser.Reference
			actions       []*RewriteAction
		)

		By("parsing the template string", func() {
			template, err = NewFromString(input.Template)
			Expect(err).ToNot(HaveOccurred())
			Expect(template.Raw).To(Equal(input.Template))
			Expect(template.Template).ToNot(BeNil())
		})

		By("rendering from values", func() {
			originalImage, err = template.Render(chart, []*RewriteAction{})
			Expect(err).ToNot(HaveOccurred())
			Expect(originalImage).ToNot(BeNil())
			Expect(originalImage.Remote()).To(Equal(expected.Image))
		})

		By("generating the rewrite rules", func() {
			actions, err = template.Apply(originalImage, rules)
			Expect(err).ToNot(HaveOccurred())
			Expect(actions).To(HaveLen(len(expected.Actions)))
			Expect(actions).To(ContainElements(expected.Actions))
		})

		By("rendering the rewritten image", func() {
			rewrittenImage, err := template.Render(chart, actions)
			Expect(err).ToNot(HaveOccurred())
			Expect(rewrittenImage).ToNot(BeNil())
			Expect(rewrittenImage.Remote()).To(Equal(expected.RewrittenImage))
		})
	},
	Entry("image alone, registry only", imageAlone, registryRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository prefix only", imageAlone, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "docker.io/my-company/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "docker.io/my-company/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, registry and prefix", imageAlone, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/my-company/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/ubuntu:latest",
			},
		},
	}),

	Entry("image and tag, registry only", imageAndTag, registryRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository prefix only", imageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "docker.io/my-company/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "docker.io/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and prefix", imageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/my-company/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/amazingapp",
			},
		},
	}),

	Entry("registry and image, registry only", registryAndImage, registryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry and image, repository prefix only", registryAndImage, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/my-company/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "my-company/nginx:latest",
			},
		},
	}),
	Entry("registry and image, registry and prefix", registryAndImage, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/my-company/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "my-company/nginx:latest",
			},
		},
	}),

	Entry("registry, image, and tag, registry only", registryImageAndTag, registryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, repository prefix only", registryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/my-company/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "my-company/busybox",
			},
		},
	}),

	Entry("image and digest, registry only", imageAndDigest, registryRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository prefix only", imageAndDigest, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "docker.io/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "docker.io/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, registry and prefix", imageAndDigest, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/platformio",
			},
		},
	}),

	Entry("dependency image and digest, registry only", dependencyRegistryImageAndTag, registryRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, repository prefix only", dependencyRegistryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "docker.io/my-company/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.image",
				Value: "my-company/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, registry and prefix", dependencyRegistryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/my-company/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".lazy.image",
				Value: "my-company/lazybox",
			},
		},
	}),
)
