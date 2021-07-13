package common_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommonPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common Package Suite")
}
