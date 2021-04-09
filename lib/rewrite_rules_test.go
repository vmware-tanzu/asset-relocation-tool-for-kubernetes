package lib_test

import (
	dockerparser "github.com/novln/docker-parser"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/chart-mover/v2/lib"
)

var _ = Describe("RewriteRules", func() {
	var rules RewriteRules

	Describe("RewriteImage", func() {
		Context("registry only", func() {
			BeforeEach(func() {
				rules = RewriteRules{
					Registry: "internal.vmware.com",
				}
			})

			It("rewrites the images correctly", func() {
				image, _ := dockerparser.Parse("myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/myrepo/myimage:latest"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/myrepo/myimage:latest"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage:1.2.3")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/myrepo/myimage:1.2.3"))
			})
		})

		Context("repository prefix", func() {
			BeforeEach(func() {
				rules = RewriteRules{
					Registry:         "internal.vmware.com",
					RepositoryPrefix: "partnerco",
				}
			})

			It("rewrites the images correctly", func() {
				image, _ := dockerparser.Parse("myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/partnerco/myrepo/myimage:latest"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/partnerco/myrepo/myimage:latest"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage:1.2.3")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("internal.vmware.com/partnerco/myrepo/myimage:1.2.3"))
			})
		})

		Context("tag, no digest", func() {
			BeforeEach(func() {
				rules = RewriteRules{
					Tag: "suspicious",
				}
			})

			It("rewrites the images correctly", func() {
				image, _ := dockerparser.Parse("myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage:suspicious"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage:suspicious"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage:1.2.3")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage:suspicious"))
			})
		})

		Context("digest, no tag", func() {
			BeforeEach(func() {
				rules = RewriteRules{
					Digest: "sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f",
				}
			})

			It("rewrites the images correctly", func() {
				image, _ := dockerparser.Parse("myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage:1.2.3")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))
			})
		})

		Context("tags and digests", func() {
			BeforeEach(func() {
				rules = RewriteRules{
					Digest: "sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f",
					Tag:    "suspicious",
				}
			})

			It("rewrites the images correctly", func() {
				image, _ := dockerparser.Parse("myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))

				image, _ = dockerparser.Parse("docker.io/myrepo/myimage:1.2.3")
				Expect(rules.RewriteImage(image).Remote()).To(Equal("docker.io/myrepo/myimage@sha256:fc92eec5cac70b0c324cec2933cd7db1c0eae7c9e2649e42d02e77eb6da0d15f"))
			})
		})
	})
})
