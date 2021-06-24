package pkg_test

import (
	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/test"
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

type TableInput struct {
	ParentChart *test.ChartSeed
	Template    string
}
type TableOutput struct {
	Image          string
	RewrittenImage string
	Actions        []*RewriteAction
}

var (
	imageAlone = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": "ubuntu:latest",
			},
		},
		Template: "{{ .image }}",
	}
	imageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": "petewall/amazingapp",
				"tag":   "latest",
			},
		},
		Template: "{{ .image }}:{{ .tag }}",
	}
	registryAndImage = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "proxy/nginx",
			},
		},
		Template: "{{ .registry }}/{{ .image }}",
	}
	registryImageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "busycontainers/busybox",
				"tag":      "busiest",
			},
		},
		Template: "{{ .registry }}/{{ .image }}:{{ .tag }}",
	}
	imageAndDigest = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image":  "petewall/platformio",
				"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
		Template: "{{ .image }}@{{ .digest }}",
	}

	nestedValues = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": map[string]interface{}{
					"registry":   "docker.io",
					"repository": "bitnami/wordpress",
					"tag":        "1.2.3",
				},
			},
		},
		Template: "{{ .image.registry }}/{{ .image.repository }}:{{ .image.tag }}",
	}

	dependencyRegistryImageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "busycontainers/busybox",
				"tag":      "busiest",
			},
			Dependencies: []*test.ChartSeed{
				{
					Name: "lazy",
					Values: map[string]interface{}{
						"registry": "index.docker.io",
						"image":    "lazycontainers/lazybox",
						"tag":      "laziest",
					},
				},
			},
		},
		Template: "{{ .lazy.registry }}/{{ .lazy.image }}:{{ .lazy.tag }}",
	}

	registryRule             = &RewriteRules{Registry: "registry.vmware.com"}
	repositoryPrefixRule     = &RewriteRules{RepositoryPrefix: "my-company"}
	repositoryRule           = &RewriteRules{Repository: "owner/name"}
	tagRule                  = &RewriteRules{Tag: "explosive"}
	digestRule               = &RewriteRules{Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	registryAndPrefixRule    = &RewriteRules{Registry: "registry.vmware.com", RepositoryPrefix: "my-company"}
	registryAndTagRule       = &RewriteRules{Registry: "registry.vmware.com", Tag: "explosive"}
	registryAndDigestRule    = &RewriteRules{Registry: "registry.vmware.com", Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	registryTagAndDigestRule = &RewriteRules{Registry: "registry.vmware.com", Tag: "explosive", Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
)

var _ = DescribeTable("Rewrite Actions",
	func(input *TableInput, rules *RewriteRules, expected *TableOutput) {
		var (
			err           error
			chart         = test.MakeChart(input.ParentChart)
			template      *ImageTemplate
			originalImage name.Reference
			actions       []*RewriteAction
		)

		By("parsing the template string", func() {
			template, err = NewFromString(input.Template)
			Expect(err).ToNot(HaveOccurred())
			Expect(template.Raw).To(Equal(input.Template))
			Expect(template.Template).ToNot(BeNil())
		})

		By("rendering from values", func() {
			originalImage, err = template.Render(chart)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalImage).ToNot(BeNil())
			Expect(originalImage.Name()).To(Equal(expected.Image))
		})

		By("generating the rewrite rules", func() {
			actions, err = template.Apply(originalImage, rules)
			Expect(err).ToNot(HaveOccurred())
			Expect(actions).To(HaveLen(len(expected.Actions)))
			Expect(actions).To(ContainElements(expected.Actions))
		})

		By("rendering the rewritten image", func() {
			rewrittenImage, err := template.Render(chart, actions...)
			Expect(err).ToNot(HaveOccurred())
			Expect(rewrittenImage).ToNot(BeNil())
			Expect(rewrittenImage.Name()).To(Equal(expected.RewrittenImage))
		})
	},
	Entry("image alone, registry only", imageAlone, registryRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository prefix only", imageAlone, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/my-company/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository only", imageAlone, repositoryRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/owner/name:latest",
			},
		},
	}),
	Entry("image alone, tag only", imageAlone, tagRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/library/ubuntu:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/library/ubuntu:explosive",
			},
		},
	}),
	Entry("image alone, digest only", imageAlone, digestRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image alone, registry and prefix", imageAlone, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/my-company/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, registry and tag", imageAlone, registryAndTagRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu:explosive",
			},
		},
	}),
	Entry("image alone, registry and digest", imageAlone, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image alone, registry and tag and digest", imageAlone, registryTagAndDigestRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),

	Entry("image and tag, registry only", imageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository prefix only", imageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/my-company/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository only", imageAndTag, repositoryRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/owner/name",
			},
		},
	}),
	Entry("image and tag, tag only", imageAndTag, tagRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".tag",
				Value: "explosive",
			},
		},
	}),
	Entry("image and tag, digest only", imageAndTag, digestRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/petewall/amazingapp:latest",
		Actions:        []*RewriteAction{},
	}),
	Entry("image and tag, registry and prefix", imageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/my-company/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and tag", imageAndTag, registryAndTagRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
			{
				Path:  ".tag",
				Value: "explosive",
			},
		},
	}),
	Entry("image and tag, registry and digest", imageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and tag and digest", imageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
			{
				Path:  ".tag",
				Value: "explosive",
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
	Entry("registry and image, repository only", registryAndImage, repositoryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "owner/name:latest",
			},
		},
	}),
	Entry("registry and image, tag only", registryAndImage, tagRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/proxy/nginx:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "proxy/nginx:explosive",
			},
		},
	}),
	Entry("registry and image, digest only", registryAndImage, digestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
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
	Entry("registry and image, registry and tag", registryAndImage, registryAndTagRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "proxy/nginx:explosive",
			},
		},
	}),
	Entry("registry and image, registry and digest", registryAndImage, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("registry and image, registry and tag and digest", registryAndImage, registryTagAndDigestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
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
	Entry("registry, image, and tag, repository only", registryImageAndTag, repositoryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/owner/name:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "owner/name",
			},
		},
	}),
	Entry("registry, image, and tag, tag only", registryImageAndTag, tagRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".tag",
				Value: "explosive",
			},
		},
	}),
	Entry("registry, image, and tag, digest only", registryImageAndTag, digestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:busiest",
		Actions:        []*RewriteAction{},
	}),
	Entry("registry, image, and tag, registry and prefix", registryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/my-company/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "my-company/busybox",
			},
		},
	}),
	Entry("registry, image, and tag, registry and tag", registryImageAndTag, registryAndTagRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".tag",
				Value: "explosive",
			},
		},
	}),
	Entry("registry, image, and tag, registry and digest", registryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, registry and tag and digest", registryImageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".tag",
				Value: "explosive",
			},
		},
	}),

	Entry("image and digest, registry only", imageAndDigest, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository prefix only", imageAndDigest, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, repository only", imageAndDigest, repositoryRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/owner/name@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/owner/name",
			},
		},
	}),
	Entry("image and digest, tag only", imageAndDigest, tagRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions:        []*RewriteAction{},
	}),
	Entry("image and digest, digest only", imageAndDigest, digestRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and digest, registry and prefix", imageAndDigest, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, registry and tag", imageAndDigest, registryAndTagRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, registry and digest", imageAndDigest, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
			{
				Path:  ".digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and digest, registry and tag and digest", imageAndDigest, registryTagAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
			{
				Path:  ".digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),

	Entry("nested values, registry only", nestedValues, registryRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:1.2.3",
		Actions: []*RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("nested values, repository prefix only", nestedValues, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/my-company/wordpress:1.2.3",
		Actions: []*RewriteAction{
			{
				Path:  ".image.repository",
				Value: "my-company/wordpress",
			},
		},
	}),
	Entry("nested values, repository only", nestedValues, repositoryRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/owner/name:1.2.3",
		Actions: []*RewriteAction{
			{
				Path:  ".image.repository",
				Value: "owner/name",
			},
		},
	}),
	Entry("nested values, tag only", nestedValues, tagRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/bitnami/wordpress:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("nested values, digest only", nestedValues, digestRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/bitnami/wordpress:1.2.3",
		Actions:        []*RewriteAction{},
	}),
	Entry("nested values, registry and prefix", nestedValues, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/my-company/wordpress:1.2.3",
		Actions: []*RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image.repository",
				Value: "my-company/wordpress",
			},
		},
	}),
	Entry("nested values, registry and tag", nestedValues, registryAndTagRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("nested values, registry and digest", nestedValues, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:1.2.3",
		Actions: []*RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("nested values, registry and tag and digest", nestedValues, registryTagAndDigestRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image.tag",
				Value: "explosive",
			},
		},
	}),

	Entry("dependency image and digest, registry only", dependencyRegistryImageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, repository prefix only", dependencyRegistryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/my-company/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.image",
				Value: "my-company/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, repository only", dependencyRegistryImageAndTag, repositoryRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/owner/name:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.image",
				Value: "owner/name",
			},
		},
	}),
	Entry("dependency image and digest, tag only", dependencyRegistryImageAndTag, tagRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("dependency image and digest, digest only", dependencyRegistryImageAndTag, digestRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/lazycontainers/lazybox:laziest",
		Actions:        []*RewriteAction{},
	}),
	Entry("dependency image and digest, registry and prefix", dependencyRegistryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
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
	Entry("dependency image and digest, registry and tag", dependencyRegistryImageAndTag, registryAndTagRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".lazy.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("dependency image and digest, registry and digest", dependencyRegistryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, registry and tag and digest", dependencyRegistryImageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".lazy.tag",
				Value: "explosive",
			},
		},
	}),
)
