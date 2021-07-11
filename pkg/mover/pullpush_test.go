package mover

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/internal"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/internal/internalfakes"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/mover/moverfakes"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/test"
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

const testRetries = 3

var testchart = test.MakeChart(&test.ChartSeed{
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

func NewPattern(input string) *internal.ImageTemplate {
	template, err := internal.NewFromString(input)
	Expect(err).ToNot(HaveOccurred())
	return template
}

func MakeImage(digest string) *moverfakes.FakeImage {
	image := &moverfakes.FakeImage{}
	image.DigestReturns(v1.NewHash(digest))
	return image
}

var _ = Describe("Pull & Push Images", func() {
	var (
		fakeImage     *internalfakes.FakeImageInterface
		originalImage internal.ImageInterface
	)
	BeforeEach(func() {
		originalImage = internal.Image
		fakeImage = &internalfakes.FakeImageInterface{}
		internal.Image = fakeImage
	})
	AfterEach(func() {
		internal.Image = originalImage
	})
	var _ = Describe("pullOriginalImages", func() {
		It("creates a change list for each image in the pattern list", func() {
			digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image1 := MakeImage(digest1)
			digest2 := "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image2 := MakeImage(digest2)
			fakeImage.PullReturnsOnCall(0, image1, digest1, nil)
			fakeImage.PullReturnsOnCall(1, image2, digest2, nil)

			patterns := []*internal.ImageTemplate{
				NewPattern("{{.image.registry}}/{{.image.repository}}"),
				NewPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
			}

			printer := NewPrinter()
			changes, err := pullOriginalImages(testchart, patterns, printer)
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

				patterns := []*internal.ImageTemplate{
					NewPattern("{{.image.registry}}/{{.image.repository}}"),
					NewPattern("{{.secondimage.registry}}/{{.secondimage.repository}}:{{.secondimage.tag}}"),
				}

				printer := NewPrinter()
				changes, err := pullOriginalImages(testchart, patterns, printer)
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
				fakeImage.PullReturns(nil, "", fmt.Errorf("image pull error"))
				patterns := []*internal.ImageTemplate{
					NewPattern("{{.image.registry}}/{{.image.repository}}"),
				}

				printer := NewPrinter()
				_, err := pullOriginalImages(testchart, patterns, printer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("image pull error"))
				Expect(printer.Out).To(Say("Pulling index.docker.io/bitnami/wordpress:1.2.3..."))
			})
		})
	})

	Describe("PushRewrittenImages", func() {
		var images []*internal.ImageChange
		BeforeEach(func() {
			images = []*internal.ImageChange{
				{
					ImageReference:     name.MustParseReference("acme/busybox:1.2.3"),
					RewrittenReference: name.MustParseReference("harbor-repo.vmware.com/pwall/busybox:1.2.3"),
					Image:              MakeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				},
			}
		})

		It("pushes the images", func() {
			printer := NewPrinter()
			err := pushRewrittenImages(images, testRetries, printer)
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
				err := pushRewrittenImages(images, testRetries, printer)
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
				err := pushRewrittenImages(images, testRetries, printer)
				Expect(err).ToNot(HaveOccurred())

				By("not pushing the image", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(0))
				})
			})
		})

		Context("pushing fails once", func() {
			BeforeEach(func() {
				fakeImage.PushReturnsOnCall(0, fmt.Errorf("push failed"))
				fakeImage.PushReturnsOnCall(1, nil)
			})

			It("retries and passes", func() {
				printer := NewPrinter()
				err := pushRewrittenImages(images, testRetries, printer)
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
				fakeImage.PushReturns(fmt.Errorf("push failed"))
			})

			It("returns an error", func() {
				printer := NewPrinter()
				err := pushRewrittenImages(images, testRetries, printer)
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
})
