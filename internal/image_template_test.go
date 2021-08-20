package internal_test

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/internal"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/test"
)

var _ = Describe("internal.NewFromString", func() {
	Context("Empty string", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"\": missing repo or a registry fragment"))
		})
	})
	Context("Invalid template", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("not a template")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"not a template\": missing repo or a registry fragment"))
		})
	})
	Context("Single template", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .image }}")
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
		imageTemplate, err := internal.NewFromString("{{ .image }}:{{ .tag }}")
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
			_, err := internal.NewFromString("{{ .image }}:{{ .tag1 }}:{{ .tag2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}:{{ .tag1 }}:{{ .tag2 }}\": too many tag template matches"))
		})
	})
	Context("Image and digest", func() {
		imageTemplate, err := internal.NewFromString("{{ .image }}@{{ .digest }}")
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
			_, err := internal.NewFromString("{{ .image }}@{{ .digest1 }}@{{ .digest2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}@{{ .digest1 }}@{{ .digest2 }}\": too many digest template matches"))
		})

	})
	Context("registry, image, and tag", func() {
		imageTemplate, err := internal.NewFromString("{{ .registry }}/{{ .image }}:{{ .tag }}")
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
			_, err := internal.NewFromString("{{ .a }}/{{ .b }}/{{ .c }}/{{ .d }}")
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
	Actions        []*internal.RewriteAction
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

	registryRule          = &internal.OCIImageLocation{Registry: "registry.vmware.com"}
	repositoryPrefixRule  = &internal.OCIImageLocation{RepositoryPrefix: "my-company"}
	digestRule            = &internal.OCIImageLocation{Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	registryAndPrefixRule = &internal.OCIImageLocation{Registry: "registry.vmware.com", RepositoryPrefix: "my-company"}
	registryAndDigestRule = &internal.OCIImageLocation{Registry: "registry.vmware.com", Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
)

var _ = DescribeTable("Rewrite Actions",
	func(input *TableInput, rules *internal.OCIImageLocation, expected *TableOutput) {
		var (
			err           error
			chart         = test.MakeChart(input.ParentChart)
			template      *internal.ImageTemplate
			originalImage name.Reference
			actions       []*internal.RewriteAction
		)

		By("parsing the template string", func() {
			template, err = internal.NewFromString(input.Template)
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
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, repository prefix only", imageAlone, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/my-company/ubuntu:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, digest only", imageAlone, digestRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "index.docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image alone, registry and prefix", imageAlone, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/my-company/ubuntu:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/ubuntu:latest",
			},
		},
	}),
	Entry("image alone, registry and digest", imageAlone, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/library/ubuntu@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and tag, registry only", imageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository prefix only", imageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/my-company/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, digest only", imageAndTag, digestRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/petewall/amazingapp:latest",
		Actions:        []*internal.RewriteAction{},
	}),
	Entry("image and tag, registry and prefix", imageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/my-company/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and digest", imageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("registry and image, registry only", registryAndImage, registryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry and image, repository prefix only", registryAndImage, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/my-company/nginx:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "my-company/nginx:latest",
			},
		},
	}),
	Entry("registry and image, digest only", registryAndImage, digestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "quay.io/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("registry and image, registry and prefix", registryAndImage, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/my-company/nginx:latest",
		Actions: []*internal.RewriteAction{
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
	Entry("registry and image, registry and digest", registryAndImage, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: "registry.vmware.com/proxy/nginx@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
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
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, repository prefix only", registryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/my-company/busybox:busiest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "my-company/busybox",
			},
		},
	}),
	Entry("registry, image, and tag, digest only", registryImageAndTag, digestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:busiest",
		Actions:        []*internal.RewriteAction{},
	}),
	Entry("registry, image, and tag, registry and prefix", registryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/my-company/busybox:busiest",
		Actions: []*internal.RewriteAction{
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
	Entry("registry, image, and tag, registry and digest", registryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("image and digest, registry only", imageAndDigest, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository prefix only", imageAndDigest, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, digest only", imageAndDigest, digestRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".digest",
				Value: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}),
	Entry("image and digest, registry and prefix", imageAndDigest, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, registry and digest", imageAndDigest, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Actions: []*internal.RewriteAction{
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
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("nested values, repository prefix only", nestedValues, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/my-company/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.repository",
				Value: "my-company/wordpress",
			},
		},
	}),
	Entry("nested values, digest only", nestedValues, digestRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/bitnami/wordpress:1.2.3",
		Actions:        []*internal.RewriteAction{},
	}),
	Entry("nested values, registry and prefix", nestedValues, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/my-company/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
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
	Entry("nested values, registry and digest", nestedValues, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, registry only", dependencyRegistryImageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, repository prefix only", dependencyRegistryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/my-company/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy.image",
				Value: "my-company/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, digest only", dependencyRegistryImageAndTag, digestRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/lazycontainers/lazybox:laziest",
		Actions:        []*internal.RewriteAction{},
	}),
	Entry("dependency image and digest, registry and prefix", dependencyRegistryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/my-company/lazybox:laziest",
		Actions: []*internal.RewriteAction{
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
	Entry("dependency image and digest, registry and digest", dependencyRegistryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
)
