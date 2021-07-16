package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/cmd"
)

var _ = Describe("Version", func() {
	Context("version is set", func() {
		var (
			stdout          *Buffer
			originalVersion string
		)

		BeforeEach(func() {
			stdout = NewBuffer()

			originalVersion = Version
			Version = "9.9.9"

			VersionCmd.SetOut(stdout)
		})
		AfterEach(func() {
			Version = originalVersion
		})

		It("prints the version", func() {
			VersionCmd.Run(VersionCmd, []string{})
			Expect(stdout).To(Say("relok8s version: 9.9.9"))
		})
	})
})
