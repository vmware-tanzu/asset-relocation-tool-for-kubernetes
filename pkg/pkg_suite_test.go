package pkg_test

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/pkgfakes"
)

func TestLib(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pkg Suite")
}

//go:generate counterfeiter github.com/google/go-containerregistry/pkg/v1.Image

func MakeImage(digest string) *pkgfakes.FakeImage {
	image := &pkgfakes.FakeImage{}
	image.DigestReturns(v1.NewHash(digest))
	return image
}
