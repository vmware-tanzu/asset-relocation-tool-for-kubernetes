package mover_test

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/pkg/mover/moverfakes"
)

func TestLib(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mover Suite")
}

//go:generate counterfeiter github.com/google/go-containerregistry/pkg/v1.Image

func MakeImage(digest string) *moverfakes.FakeImage {
	image := &moverfakes.FakeImage{}
	image.DigestReturns(v1.NewHash(digest))
	return image
}
