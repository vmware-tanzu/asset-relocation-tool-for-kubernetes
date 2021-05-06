package features

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFeatures(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Features Suite")
}
