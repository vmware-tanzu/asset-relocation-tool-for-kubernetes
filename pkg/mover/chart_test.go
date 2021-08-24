// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/internalfakes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover/moverfakes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/test"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type testPrinter struct {
	out *Buffer
}

func newLogger() *testPrinter {
	return &testPrinter{
		out: NewBuffer(),
	}
}

func (c *testPrinter) print(i ...interface{}) {
	_, _ = fmt.Fprint(c.out, i...)
}

func (c *testPrinter) Printf(format string, i ...interface{}) {
	c.print(fmt.Sprintf(format, i...))
}

func (c *testPrinter) Println(i ...interface{}) {
	c.print(fmt.Sprintln(i...))
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

func newPattern(input string) *internal.ImageTemplate {
	template, err := internal.NewFromString(input)
	Expect(err).ToNot(HaveOccurred())
	return template
}

//go:generate counterfeiter github.com/google/go-containerregistry/pkg/v1.Image

func makeImage(digest string) *moverfakes.FakeImage {
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

	Describe("ComputeChanges", func() {
		It("checks if the rewritten images are present", func() {
			changes := []*internal.ImageChange{
				{
					Pattern:        newPattern("{{.image.registry}}/{{.image.repository}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wordpress:1.2.3"),
					Image:          makeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				{
					Pattern:        newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
					Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			}
			rules := &RewriteRules{
				Registry:         "harbor-repo.vmware.com",
				RepositoryPrefix: "pwall",
			}

			fakeImage.CheckReturnsOnCall(0, true, nil)  // Pretend it doesn't exist
			fakeImage.CheckReturnsOnCall(1, false, nil) // Pretend it already exists

			newChanges, actions, err := computeChanges(testchart, changes, rules)
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
				Expect(actions).To(ContainElements([]*internal.RewriteAction{
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
		})

		Context("two of the same image with different templates", func() {
			It("only checks one image", func() {

				changes := []*internal.ImageChange{
					{
						Pattern:        newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					{
						Pattern:        newPattern("{{.observabilitytoo.image.registry}}/{{.observabilitytoo.image.repository}}:{{.observabilitytoo.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				}
				rules := &RewriteRules{
					Registry:         "harbor-repo.vmware.com",
					RepositoryPrefix: "pwall",
				}

				fakeImage.CheckReturns(true, nil) // Pretend it doesn't exist

				newChanges, actions, err := computeChanges(testchart, changes, rules)
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
					Expect(actions).To(ContainElements([]*internal.RewriteAction{
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
			})
		})
	})

	Describe("pullOriginalImages", func() {
		It("creates a change list for each image in the pattern list", func() {
			digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image1 := makeImage(digest1)
			digest2 := "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image2 := makeImage(digest2)
			fakeImage.PullReturnsOnCall(0, image1, digest1, nil)
			fakeImage.PullReturnsOnCall(1, image2, digest2, nil)

			patterns := []*internal.ImageTemplate{
				newPattern("{{.image.registry}}/{{.image.repository}}"),
				newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
			}

			changes, err := pullOriginalImages(testchart, patterns)
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
		})

		Context("duplicated image", func() {
			It("only pulls once", func() {
				digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				image := makeImage(digest)
				fakeImage.PullReturns(image, digest, nil)

				patterns := []*internal.ImageTemplate{
					newPattern("{{.image.registry}}/{{.image.repository}}"),
					newPattern("{{.secondimage.registry}}/{{.secondimage.repository}}:{{.secondimage.tag}}"),
				}

				changes, err := pullOriginalImages(testchart, patterns)
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
			})
		})

		Context("error pulling an image", func() {
			It("returns the error", func() {
				fakeImage.PullReturns(nil, "", fmt.Errorf("image pull error"))
				patterns := []*internal.ImageTemplate{
					newPattern("{{.image.registry}}/{{.image.repository}}"),
				}

				_, err := pullOriginalImages(testchart, patterns)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("image pull error"))
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
					Image:              makeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				},
			}
		})

		It("pushes the images", func() {
			printer := newLogger()
			err := pushRewrittenImages(images, testRetries, printer)
			Expect(err).ToNot(HaveOccurred())

			By("pushing the image", func() {
				Expect(fakeImage.PushCallCount()).To(Equal(1))
				image, ref := fakeImage.PushArgsForCall(0)
				Expect(image).To(Equal(images[0].Image))
				Expect(ref).To(Equal(images[0].RewrittenReference))
			})

			By("logging the process", func() {
				Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\nDone"))
			})
		})

		Context("rewritten image is the same", func() {
			It("does not push the image", func() {
				images[0].RewrittenReference = images[0].ImageReference
				printer := newLogger()
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
				printer := newLogger()
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
				printer := newLogger()
				err := pushRewrittenImages(images, testRetries, printer)
				Expect(err).ToNot(HaveOccurred())

				By("trying to push the image twice", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(2))
				})

				By("logging the process", func() {
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\nDone"))
				})
			})
		})

		Context("pushing fails every time", func() {
			BeforeEach(func() {
				fakeImage.PushReturns(fmt.Errorf("push failed"))
			})

			It("returns an error", func() {
				printer := newLogger()
				err := pushRewrittenImages(images, testRetries, printer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("All attempts fail:\n#1: push failed\n#2: push failed\n#3: push failed"))

				By("trying to push the image", func() {
					Expect(fakeImage.PushCallCount()).To(Equal(3))
				})

				By("logging the process", func() {
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #2 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #3 failed: push failed"))
				})
			})
		})
	})
})

const (
	fixturesRoot = "../../test/fixtures/"
)

var _ = Describe("LoadImagePatterns", func() {
	logger := &defaultLogger{}

	It("reads from given file first if present", func() {
		imagefile := filepath.Join(fixturesRoot, "testchart.images.yaml")
		contents, err := loadPatterns(imagefile, nil, logger)
		Expect(err).ToNot(HaveOccurred())

		expected, err := os.ReadFile(imagefile)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal(expected))
	})
	It("reads from chart if file missing", func() {
		chart, err := loader.Load(filepath.Join(fixturesRoot, "self-relok8ing-chart"))
		Expect(err).ToNot(HaveOccurred())

		contents, err := loadPatterns("", chart, logger)
		Expect(err).ToNot(HaveOccurred())

		embeddedPatterns := filepath.Join(fixturesRoot, "self-relok8ing-chart/.relok8s-images.yaml")
		expected, err := os.ReadFile(embeddedPatterns)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal(expected))
	})
	It("reads nothing when no file and the chart is not self relok8able", func() {
		chart, err := loader.Load(filepath.Join(fixturesRoot, "testchart"))
		Expect(err).ToNot(HaveOccurred())

		contents, err := loadPatterns("", chart, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(BeEmpty())
	})
})

func TestNamespacedPath(t *testing.T) {
	tests := []struct {
		inputPath  string
		chartName  string
		outputPath string
	}{
		{".image.registry", "app1", ".image.registry"},
		{".app1.image.registry", "app1", ".image.registry"},
		{".fooapp1.image.registry", "app1", ".fooapp1.image.registry"},
		{".image.app1.registry", "app1", ".image.app1.registry"},
		{".app2.image.registry", "app1", ".app2.image.registry"},
	}

	for _, tc := range tests {
		if got, want := namespacedPath(tc.inputPath, tc.chartName), tc.outputPath; got != want {
			t.Errorf("got=%s; want=%s", got, want)
		}
	}
}
