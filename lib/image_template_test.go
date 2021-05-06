package lib_test

import (
	dockerparser "github.com/novln/docker-parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
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
			imageTemplate, err := NewFromString("{{ .Values.image }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .Values.image }}"))
			Expect(imageTemplate.RegistryTemplate).To(Equal(""))
			Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".Values.image"))
			Expect(imageTemplate.TagTemplate).To(Equal(""))
			Expect(imageTemplate.DigestTemplate).To(Equal(""))
		})

	})
	Context("Image and tag", func() {
		imageTemplate, err := NewFromString("{{ .Values.image }}:{{ .Values.tag }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .Values.image }}:{{ .Values.tag }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(""))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".Values.image"))
		Expect(imageTemplate.TagTemplate).To(Equal(".Values.tag"))
		Expect(imageTemplate.DigestTemplate).To(Equal(""))
	})
	Context("Image and multiple tags", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .Values.image }}:{{ .Values.tag1 }}:{{ .Values.tag2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .Values.image }}:{{ .Values.tag1 }}:{{ .Values.tag2 }}\": too many tag template matches"))
		})
	})
	Context("Image and digest", func() {
		imageTemplate, err := NewFromString("{{ .Values.image }}@{{ .Values.digest }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .Values.image }}@{{ .Values.digest }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(""))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".Values.image"))
		Expect(imageTemplate.TagTemplate).To(Equal(""))
		Expect(imageTemplate.DigestTemplate).To(Equal(".Values.digest"))
	})
	Context("Image and multiple digests", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .Values.image }}@{{ .Values.digest1 }}@{{ .Values.digest2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .Values.image }}@{{ .Values.digest1 }}@{{ .Values.digest2 }}\": too many digest template matches"))
		})

	})
	Context("registry, image, and tag", func() {
		imageTemplate, err := NewFromString("{{ .Values.registry }}/{{ .Values.image }}:{{ .Values.tag }}")
		Expect(err).ToNot(HaveOccurred())
		Expect(imageTemplate.Raw).To(Equal("{{ .Values.registry }}/{{ .Values.image }}:{{ .Values.tag }}"))
		Expect(imageTemplate.RegistryTemplate).To(Equal(".Values.registry"))
		Expect(imageTemplate.RepositoryTemplate).To(Equal(".Values.image"))
		Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(""))
		Expect(imageTemplate.TagTemplate).To(Equal(".Values.tag"))
		Expect(imageTemplate.DigestTemplate).To(Equal(""))
	})
	Context("Too many templates", func() {
		It("returns an error", func() {
			_, err := NewFromString("{{ .Values.a }}/{{ .Values.b }}/{{ .Values.c }}/{{ .Values.d }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .Values.a }}/{{ .Values.b }}/{{ .Values.c }}/{{ .Values.d }}\": more fragments than expected"))
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
		Template: "{{ .Values.image }}",
	}
	imageAndTag = &TableInput{
		Values: map[string]interface{}{
			"image": "petewall/amazingapp",
			"tag":   "latest",
		},
		Template: "{{ .Values.image }}:{{ .Values.tag }}",
	}
	registryAndImage = &TableInput{
		Values: map[string]interface{}{
			"registry": "quay.io",
			"image":    "proxy/nginx",
		},
		Template: "{{ .Values.registry }}/{{ .Values.image }}",
	}
	registryImageAndTag = &TableInput{
		Values: map[string]interface{}{
			"registry": "quay.io",
			"image":    "busycontainers/busybox",
			"tag":      "busiest",
		},
		Template: "{{ .Values.registry }}/{{ .Values.image }}:{{ .Values.tag }}",
	}
	imageAndDigest = &TableInput{
		Values: map[string]interface{}{
			"image":  "petewall/platformio",
			"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Template: "{{ .Values.image }}@{{ .Values.digest }}",
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
		Template: "{{ .Values.lazy.registry }}/{{ .Values.lazy.image }}:{{ .Values.lazy.tag }}",
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
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository prefix only", imageAlone, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "docker.io/my-company/library/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/my-company/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository only", imageAlone, repositoryRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "docker.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/owner/name:latest",
			},
		},
	}),
	Entry("image alone, tag only", imageAlone, tagRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "docker.io/library/ubuntu:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/library/ubuntu:explosive",
			},
		},
	}),
	Entry("image alone, digest only", imageAlone, digestRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image alone, registry and prefix", imageAlone, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/my-company/library/ubuntu:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/my-company/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, registry and tag", imageAlone, registryAndTagRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu:explosive",
			},
		},
	}),
	Entry("image alone, registry and digest", imageAlone, registryAndDigestRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image alone, registry and tag and digest", imageAlone, registryTagAndDigestRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),

	Entry("image and tag, registry only", imageAndTag, registryRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository prefix only", imageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "docker.io/my-company/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/my-company/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository only", imageAndTag, repositoryRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "docker.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/owner/name",
			},
		},
	}),
	Entry("image and tag, tag only", imageAndTag, tagRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "docker.io/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("image and tag, digest only", imageAndTag, digestRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "docker.io/petewall/amazingapp:latest",
		Actions:        []*RewriteAction{},
	}),
	Entry("image and tag, registry and prefix", imageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/my-company/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/my-company/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and tag", imageAndTag, registryAndTagRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("image and tag, registry and digest", imageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and tag and digest", imageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),

	Entry("registry and image, registry only", registryAndImage, registryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry and image, repository prefix only", registryAndImage, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/my-company/proxy/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "my-company/proxy/nginx:latest",
			},
		},
	}),
	Entry("registry and image, repository only", registryAndImage, repositoryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/owner/name:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "owner/name:latest",
			},
		},
	}),
	Entry("registry and image, tag only", registryAndImage, tagRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/proxy/nginx:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "proxy/nginx:explosive",
			},
		},
	}),
	Entry("registry and image, digest only", registryAndImage, digestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("registry and image, registry and prefix", registryAndImage, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/my-company/proxy/nginx:latest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.image",
				Value: "my-company/proxy/nginx:latest",
			},
		},
	}),
	Entry("registry and image, registry and tag", registryAndImage, registryAndTagRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.image",
				Value: "proxy/nginx:explosive",
			},
		},
	}),
	Entry("registry and image, registry and digest", registryAndImage, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("registry and image, registry and tag and digest", registryAndImage, registryTagAndDigestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),

	Entry("registry, image, and tag, registry only", registryImageAndTag, registryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, repository prefix only", registryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/my-company/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "my-company/busycontainers/busybox",
			},
		},
	}),
	Entry("registry, image, and tag, repository only", registryImageAndTag, repositoryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/owner/name:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "owner/name",
			},
		},
	}),
	Entry("registry, image, and tag, tag only", registryImageAndTag, tagRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.tag",
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
		RewrittenImage: "registry.vmware.com/my-company/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.image",
				Value: "my-company/busycontainers/busybox",
			},
		},
	}),
	Entry("registry, image, and tag, registry and tag", registryImageAndTag, registryAndTagRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("registry, image, and tag, registry and digest", registryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, registry and tag and digest", registryImageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),

	Entry("image and digest, registry only", imageAndDigest, registryRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository prefix only", imageAndDigest, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "docker.io/my-company/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/my-company/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository only", imageAndDigest, repositoryRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "docker.io/owner/name@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/owner/name",
			},
		},
	}),
	Entry("image and digest, tag only", imageAndDigest, tagRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions:        []*RewriteAction{},
	}),
	Entry("image and digest, digest only", imageAndDigest, digestRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "docker.io/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and digest, registry and prefix", imageAndDigest, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/my-company/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/my-company/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, registry and tag", imageAndDigest, registryAndTagRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, registry and digest", imageAndDigest, registryAndDigestRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/platformio",
			},
			{
				Path:  ".Values.digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and digest, registry and tag and digest", imageAndDigest, registryTagAndDigestRule, &TableOutput{
		Image:          "docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/petewall/platformio",
			},
			{
				Path:  ".Values.digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),

	Entry("dependency image and digest, registry only", dependencyRegistryImageAndTag, registryRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, repository prefix only", dependencyRegistryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "docker.io/my-company/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.image",
				Value: "my-company/lazycontainers/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, repository only", dependencyRegistryImageAndTag, repositoryRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "docker.io/owner/name:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.image",
				Value: "owner/name",
			},
		},
	}),
	Entry("dependency image and digest, tag only", dependencyRegistryImageAndTag, tagRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "docker.io/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("dependency image and digest, digest only", dependencyRegistryImageAndTag, digestRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "docker.io/lazycontainers/lazybox:laziest",
		Actions:        []*RewriteAction{},
	}),
	Entry("dependency image and digest, registry and prefix", dependencyRegistryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/my-company/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.lazy.image",
				Value: "my-company/lazycontainers/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, registry and tag", dependencyRegistryImageAndTag, registryAndTagRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.lazy.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("dependency image and digest, registry and digest", dependencyRegistryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, registry and tag and digest", dependencyRegistryImageAndTag, registryTagAndDigestRule, &TableOutput{
		Image:          "docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.lazy.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".Values.lazy.tag",
				Value: "explosive",
			},
		},
	}),
)
