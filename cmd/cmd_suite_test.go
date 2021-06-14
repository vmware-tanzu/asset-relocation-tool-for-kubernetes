package cmd_test

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/marketplace-partner-eng/relok8s/v2/cmd/cmdfakes"
)

func TestVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}

//go:generate counterfeiter github.com/google/go-containerregistry/pkg/v1.Image

func MakeImage(digest string) *cmdfakes.FakeImage {
	image := &cmdfakes.FakeImage{}
	image.DigestReturns(v1.NewHash(digest))
	return image
}
