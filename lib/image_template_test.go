package lib_test

import (
	dockerparser "github.com/novln/docker-parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
	"gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib/libfakes"
	"gopkg.in/yaml.v2"

	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

type TableInput struct {
	Values   map[string]string
	Template string
}
type TableOutput struct {
	Image          string
	RewrittenImage string
	Actions        []*RewriteAction
}

var (
	imageAlone = &TableInput{
		Values: map[string]string{
			"image": "ubuntu:latest",
		},
		Template: "{{ .Values.image }}",
	}
	imageAndTag = &TableInput{
		Values: map[string]string{
			"image": "petewall/amazingapp",
			"tag":   "latest",
		},
		Template: "{{ .Values.image }}:{{ .Values.tag }}",
	}
	registryAndImage = &TableInput{
		Values: map[string]string{
			"registry": "docker.io",
			"image":    "ubuntu",
		},
		Template: "{{ .Values.registry }}/{{ .Values.image }}",
	}
	registryImageAndTag = &TableInput{
		Values: map[string]string{
			"registry": "quay.io",
			"image":    "busycontainers/busybox",
			"tag":      "busiest",
		},
		Template: "{{ .Values.registry }}/{{ .Values.image }}:{{ .Values.tag }}",
	}
	imageAndDigest = &TableInput{
		Values: map[string]string{
			"image":  "petewall/platformio",
			"digest": "sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
		},
		Template: "{{ .Values.image }}@{{ .Values.digest }}",
	}

	registryRule             = &RewriteRules{Registry: "registry.vmware.com"}
	repositoryPrefixRule     = &RewriteRules{RepositoryPrefix: "my-company"}
	repositoryRule           = &RewriteRules{Repository: "owner/name"}
	tagRule                  = &RewriteRules{Tag: "explosive"}
	digestRule               = &RewriteRules{Digest: "sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048"}
	registryAndPrefixRule    = &RewriteRules{Registry: "registry.vmware.com", RepositoryPrefix: "my-company"}
	registryAndTagRule       = &RewriteRules{Registry: "registry.vmware.com", Tag: "explosive"}
	registryAndDigestRule    = &RewriteRules{Registry: "registry.vmware.com", Digest: "sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048"}
	registryTagAndDigestRule = &RewriteRules{Registry: "registry.vmware.com", Tag: "explosive", Digest: "sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048"}
)

func MakeFakeChart(values map[string]string) *libfakes.FakeHelmChart {
	chart := &libfakes.FakeHelmChart{}
	encoded, err := yaml.Marshal(values)
	Expect(err).ToNot(HaveOccurred())
	chart.GetValuesReturns(&helmchart.Config{
		Raw: string(encoded),
	})
	return chart
}

var _ = DescribeTable("Rewrite Actions",
	func(input *TableInput, rules *RewriteRules, expected *TableOutput) {
		var (
			err           error
			chart         = MakeFakeChart(input.Values)
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
		RewrittenImage: "docker.io/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "docker.io/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
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
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
			},
		},
	}),
	Entry("image alone, registry and tag and digest", imageAlone, registryTagAndDigestRule, &TableOutput{
		Image:          "docker.io/library/ubuntu:latest",
		RewrittenImage: "registry.vmware.com/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "registry.vmware.com/library/ubuntu@sha256:7ca36ce7be689526e193a8fd92fb30e63f8b62764677ff21250db6339a459048",
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
	Entry("registry, image and tag, repository prefix only", registryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/my-company/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "my-company/busycontainers/busybox",
			},
		},
	}),
	Entry("registry, image and tag, repository only", registryImageAndTag, repositoryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/owner/name:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.image",
				Value: "owner/name",
			},
		},
	}),
	Entry("registry, image and tag, tag only", registryImageAndTag, tagRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:explosive",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.tag",
				Value: "explosive",
			},
		},
	}),
	Entry("registry, image and tag, digest only", registryImageAndTag, digestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/busycontainers/busybox:busiest",
		Actions:        []*RewriteAction{},
	}),
	Entry("registry, image and tag, registry and prefix", registryImageAndTag, registryAndPrefixRule, &TableOutput{
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
	Entry("registry, image and tag, registry and tag", registryImageAndTag, registryAndTagRule, &TableOutput{
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
	Entry("registry, image and tag, registry and digest", registryImageAndTag, registryAndDigestRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*RewriteAction{
			{
				Path:  ".Values.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image and tag, registry and tag and digest", registryImageAndTag, registryTagAndDigestRule, &TableOutput{
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
)
