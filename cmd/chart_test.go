package cmd_test

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/cmd"
	. "gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/lib/libfakes"
)

type TestPrinter struct {
	Out *Buffer
	Err *Buffer
}

func NewPrinter() *TestPrinter {
	return &TestPrinter{
		Out: NewBuffer(),
		Err: NewBuffer(),
	}
}

func (c *TestPrinter) Print(i ...interface{}) {
	_, _ = fmt.Fprint(c.Out, i...)
}

func (c *TestPrinter) Println(i ...interface{}) {
	c.Print(fmt.Sprintln(i...))
}

func (c *TestPrinter) Printf(format string, i ...interface{}) {
	c.Print(fmt.Sprintf(format, i...))
}

func (c *TestPrinter) PrintErr(i ...interface{}) {
	_, _ = fmt.Fprint(c.Err, i...)
}

func (c *TestPrinter) PrintErrln(i ...interface{}) {
	c.PrintErr(fmt.Sprintln(i...))
}

func (c *TestPrinter) PrintErrf(format string, i ...interface{}) {
	c.PrintErr(fmt.Sprintf(format, i...))
}

var chart = MakeChart(&ChartSeed{
	Values: map[string]interface{}{
		"image": map[string]interface{}{
			"registry":   "docker.io",
			"repository": "bitnami/wordpress:1.2.3",
		},
		"secondimage": map[string]interface{}{
			"registry":   "docker.io",
			"repository": "bitnami/wordpress",
			"tag":        "1.2.3",
		},
		"observability": map[string]interface{}{
			"image": map[string]interface{}{
				"registry":   "docker.io",
				"repository": "bitnami/wavefront",
				"tag":        "5.6.7",
			},
		},
		"observabilitytoo": map[string]interface{}{
			"image": map[string]interface{}{
				"registry":   "docker.io",
				"repository": "bitnami/wavefront",
				"tag":        "5.6.7",
			},
		},
	},
})

func NewPattern(input string) *ImageTemplate {
	template, err := NewFromString(input)
	Expect(err).ToNot(HaveOccurred())
	return template
}

var _ = Describe("Chart", func() {
	var (
		fakeImage     *libfakes.FakeImageInterface
		originalImage ImageInterface
	)
	BeforeEach(func() {
		originalImage = Image
		fakeImage = &libfakes.FakeImageInterface{}
		Image = fakeImage
	})
	AfterEach(func() {
		Image = originalImage
	})

	Describe("PullOriginalImages", func() {
		It("creates a change list for each image in the pattern list", func() {
			digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image1 := MakeImage(digest1)
			digest2 := "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image2 := MakeImage(digest2)
			fakeImage.PullReturnsOnCall(0, image1, digest1, nil)
			fakeImage.PullReturnsOnCall(1, image2, digest2, nil)

			patterns := []*ImageTemplate{
				NewPattern("{{.image.registry}}/{{.image.repository}}"),
				NewPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
			}

			printer := NewPrinter()
			changes, err := cmd.PullOriginalImages(chart, patterns, printer)
			Expect(err).ToNot(HaveOccurred())

			By("pulling the images", func() {
				Expect(fakeImage.PullCallCount()).To(Equal(2))
				Expect(fakeImage.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				Expect(fakeImage.PullArgsForCall(1).Name()).To(Equal("index.docker.io/bitnami/wavefront:5.6.7"))
			})

			By("returning a list of images", func() {
				Expect(changes).To(HaveLen(2))
				Expect(changes[0].Pattern).To(Equal(patterns[0]))
				Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(changes[1].Pattern).To(Equal(patterns[1]))
				Expect(changes[1].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wavefront:5.6.7"))
				Expect(changes[1].Digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
			})

			By("outputting the progress", func() {
				Expect(printer.Out).To(Say("Pulling index.docker.io/bitnami/wordpress:1.2.3... Done"))
				Expect(printer.Out).To(Say("Pulling index.docker.io/bitnami/wavefront:5.6.7... Done"))
			})
		})

		Context("duplicated image", func() {
			It("only pulls once", func() {
				digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				image := MakeImage(digest)
				fakeImage.PullReturns(image, digest, nil)

				patterns := []*ImageTemplate{
					NewPattern("{{.image.registry}}/{{.image.repository}}"),
					NewPattern("{{.secondimage.registry}}/{{.secondimage.repository}}:{{.secondimage.tag}}"),
				}

				printer := NewPrinter()
				changes, err := cmd.PullOriginalImages(chart, patterns, printer)
				Expect(err).ToNot(HaveOccurred())

				By("pulling the image once", func() {
					Expect(fakeImage.PullCallCount()).To(Equal(1))
					Expect(fakeImage.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				})

				By("returning a list of images", func() {
					Expect(changes).To(HaveLen(2))
					Expect(changes[0].Pattern).To(Equal(patterns[0]))
					Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
					Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[1].Pattern).To(Equal(patterns[1]))
					Expect(changes[1].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
					Expect(changes[1].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				})

				By("outputting the progress", func() {
					Expect(printer.Out).To(Say("Pulling index.docker.io/bitnami/wordpress:1.2.3... Done"))
				})
			})
		})

		Context("error pulling an image", func() {
			It("returns the error", func() {
				fakeImage.PullReturns(nil, "", errors.New("image pull error"))
				patterns := []*ImageTemplate{
					NewPattern("{{.image.registry}}/{{.image.repository}}"),
				}

				printer := NewPrinter()
				_, err := cmd.PullOriginalImages(chart, patterns, printer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("image pull error"))
				Expect(printer.Out).To(Say("Pulling index.docker.io/bitnami/wordpress:1.2.3..."))
			})
		})
	})

	Describe("CheckNewImages", func() {
		It("checks if the rewritten images are present", func() {
			changes := []*cmd.ImageChange{
				{
					Pattern:        NewPattern("{{.image.registry}}/{{.image.repository}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wordpress:1.2.3"),
					Image:          MakeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				{
					Pattern:        NewPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
					Image:          MakeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			}
			rules := &RewriteRules{
				Registry:         "harbor-repo.vmware.com",
				RepositoryPrefix: "pwall",
			}
			printer := NewPrinter()

			fakeImage.CheckReturnsOnCall(0, true, nil)  // Pretend it doesn't exist
			fakeImage.CheckReturnsOnCall(1, false, nil) // Pretend it already exists

			newChanges, actions, err := cmd.CheckNewImages(chart, changes, rules, printer)
			Expect(err).ToNot(HaveOccurred())

			By("checking the existing images on the remote registry", func() {
				Expect(fakeImage.CheckCallCount()).To(Equal(2))
				digest, imageReference := fakeImage.CheckArgsForCall(0)
				Expect(digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				digest, imageReference = fakeImage.CheckArgsForCall(1)
				Expect(digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
			})

			By("updating the image change list", func() {
				Expect(newChanges).To(HaveLen(2))
				Expect(newChanges[0].Pattern).To(Equal(changes[0].Pattern))
				Expect(newChanges[0].ImageReference).To(Equal(changes[0].ImageReference))
				Expect(newChanges[0].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(newChanges[0].Digest).To(Equal(changes[0].Digest))
				Expect(newChanges[0].AlreadyPushed).To(BeFalse())

				Expect(newChanges[1].Pattern).To(Equal(changes[1].Pattern))
				Expect(newChanges[1].ImageReference).To(Equal(changes[1].ImageReference))
				Expect(newChanges[1].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
				Expect(newChanges[1].Digest).To(Equal(changes[1].Digest))
				Expect(newChanges[1].AlreadyPushed).To(BeTrue())
			})

			By("returning a list of changes that would need to be applied to the chart", func() {
				Expect(actions).To(HaveLen(4))
				Expect(actions).To(ContainElements([]*RewriteAction{
					{
						Path:  ".image.registry",
						Value: "harbor-repo.vmware.com",
					},
					{
						Path:  ".image.repository",
						Value: "pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					{
						Path:  ".observability.image.registry",
						Value: "harbor-repo.vmware.com",
					},
					{
						Path:  ".observability.image.repository",
						Value: "pwall/wavefront",
					},
				}))
			})

			By("outputting the progress", func() {
				Expect(printer.Out).To(Say("Checking harbor-repo.vmware.com/pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \\(sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\\)... Push required"))
				Expect(printer.Out).To(Say("Checking harbor-repo.vmware.com/pwall/wavefront:5.6.7 \\(sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\\)... Already exists"))
			})
		})

		Context("two of the same image with different templates", func() {
			It("only checks one image", func() {

				changes := []*cmd.ImageChange{
					{
						Pattern:        NewPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          MakeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					{
						Pattern:        NewPattern("{{.observabilitytoo.image.registry}}/{{.observabilitytoo.image.repository}}:{{.observabilitytoo.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          MakeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				}
				rules := &RewriteRules{
					Registry:         "harbor-repo.vmware.com",
					RepositoryPrefix: "pwall",
				}
				printer := NewPrinter()

				fakeImage.CheckReturns(true, nil) // Pretend it doesn't exist

				newChanges, actions, err := cmd.CheckNewImages(chart, changes, rules, printer)
				Expect(err).ToNot(HaveOccurred())

				By("checking the image once", func() {
					Expect(fakeImage.CheckCallCount()).To(Equal(1))
					digest, imageReference := fakeImage.CheckArgsForCall(0)
					Expect(digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
				})

				By("updating the image change list, but one is marked already pushed", func() {
					Expect(newChanges).To(HaveLen(2))
					Expect(newChanges[0].Pattern).To(Equal(changes[0].Pattern))
					Expect(newChanges[0].ImageReference).To(Equal(changes[0].ImageReference))
					Expect(newChanges[0].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
					Expect(newChanges[0].Digest).To(Equal(changes[0].Digest))
					Expect(newChanges[0].AlreadyPushed).To(BeFalse())

					Expect(newChanges[1].Pattern).To(Equal(changes[1].Pattern))
					Expect(newChanges[1].ImageReference).To(Equal(changes[1].ImageReference))
					Expect(newChanges[1].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
					Expect(newChanges[1].Digest).To(Equal(changes[1].Digest))
					Expect(newChanges[1].AlreadyPushed).To(BeTrue())
				})

				By("returning a list of changes that would need to be applied to the chart", func() {
					Expect(actions).To(HaveLen(4))
					Expect(actions).To(ContainElements([]*RewriteAction{
						{
							Path:  ".observability.image.registry",
							Value: "harbor-repo.vmware.com",
						},
						{
							Path:  ".observability.image.repository",
							Value: "pwall/wavefront",
						},
						{
							Path:  ".observabilitytoo.image.registry",
							Value: "harbor-repo.vmware.com",
						},
						{
							Path:  ".observabilitytoo.image.repository",
							Value: "pwall/wavefront",
						},
					}))
				})

				By("outputting the progress", func() {
					Expect(printer.Out).To(Say("Checking harbor-repo.vmware.com/pwall/wavefront:5.6.7 \\(sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\\)... Push required"))
				})
			})
		})
	})

	Describe("PushRewrittenImages", func() {
		var images []*cmd.ImageChange
		BeforeEach(func() {
			images = []*cmd.ImageChange{
				{
					ImageReference:     name.MustParseReference("acme/busybox:1.2.3"),
					RewrittenReference: name.MustParseReference("harbor-repo.vmware.com/pwall/busybox:1.2.3"),
					Image:              MakeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				},
			}
		})

		It("pushes the images", func() {
			printer := NewPrinter()
			err := cmd.PushRewrittenImages(images, printer)
			Expect(err).ToNot(HaveOccurred())

			By("pushing the image", func() {
				Expect(fakeImage.PushCallCount()).To(Equal(1))
				image, ref := fakeImage.PushArgsForCall(0)
				Expect(image).To(Equal(images[0].Image))
				Expect(ref).To(Equal(images[0].RewrittenReference))
			})

			By("logging the process", func() {
				Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... Done"))
			})
		})

		Context("rewritten image is the same", func() {
			It("does not push the image", func() {
				images[0].RewrittenReference = images[0].ImageReference
				printer := NewPrinter()
				err := cmd.PushRewrittenImages(images, printer)
				Expect(err).ToNot(HaveOccurred())

				By("not pushing the image", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(0))
				})
			})
		})

		Context("image has already been pushed", func() {
			It("does not push the image", func() {
				images[0].AlreadyPushed = true
				printer := NewPrinter()
				err := cmd.PushRewrittenImages(images, printer)
				Expect(err).ToNot(HaveOccurred())

				By("not pushing the image", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(0))
				})
			})
		})

		Context("pushing fails once", func() {
			BeforeEach(func() {
				cmd.Retries = 3
				fakeImage.PushReturnsOnCall(0, errors.New("push failed"))
				fakeImage.PushReturnsOnCall(1, nil)
			})

			It("retries and passes", func() {
				printer := NewPrinter()
				err := cmd.PushRewrittenImages(images, printer)
				Expect(err).ToNot(HaveOccurred())

				By("trying to push the image twice", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(2))
				})

				By("logging the process", func() {
					Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... "))
					Expect(printer.Err).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... Done"))
				})
			})
		})

		Context("pushing fails every time", func() {
			BeforeEach(func() {
				cmd.Retries = 3
				fakeImage.PushReturns(errors.New("push failed"))
			})

			It("returns an error", func() {
				printer := NewPrinter()
				err := cmd.PushRewrittenImages(images, printer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("All attempts fail:\n#1: push failed\n#2: push failed\n#3: push failed"))

				By("trying to push the image", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(3))
				})

				By("logging the process", func() {
					Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... "))
					Expect(printer.Err).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... "))
					Expect(printer.Err).To(Say("Attempt #2 failed: push failed"))
					Expect(printer.Out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3... "))
					Expect(printer.Err).To(Say("Attempt #3 failed: push failed"))
				})
			})
		})
	})

	Describe("ParseOutputFlag", func() {
		It("works with default out flag", func() {
			got, err := cmd.ParseOutputFlag(cmd.Output)
			want := "./%s-%s.relocated.tgz"
			Expect(got).To(Equal(want))
			Expect(err).To(BeNil())
		})
		It("rejects out flag without wildcard *", func() {
			_, err := cmd.ParseOutputFlag("nowildcardhere.tgz")
			Expect(err).Should(MatchError(cmd.ErrorMissingOutPlaceHolder))
		})
		It("rejects out flag without proper extension", func() {
			_, err := cmd.ParseOutputFlag("*-wildcardhere")
			Expect(err).Should(MatchError(cmd.ErrorBadExtension))
		})
		It("accepts out flag with wildcard", func() {
			got, err := cmd.ParseOutputFlag("*-wildcardhere.tgz")
			Expect(got).To(Equal("%s-%s-wildcardhere.tgz"))
			Expect(err).To(BeNil())
		})
	})
})
