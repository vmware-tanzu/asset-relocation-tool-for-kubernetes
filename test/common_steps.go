// +build feature external

package test

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	ChartMoverBinaryPath string
	CommandSession       *gexec.Session
	RulesFile            *os.File
)

var _ = BeforeSuite(func() {
	var err error
	ChartMoverBinaryPath, err = gexec.Build(
		"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2",
		"-ldflags",
		"-X github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/cmd.Version=1.2.3",
	)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func DefineCommonSteps(define goerkin.Definitions) {
	define.When(`^running relok8s (.*)$`, func(argString string) {
		args := strings.Split(argString, " ")
		if RulesFile != nil {
			args = append(args, "--rules", RulesFile.Name())
		}
		command := exec.Command(ChartMoverBinaryPath, args...)
		var err error
		CommandSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	define.Then(`^the command exits without error$`, func() {
		Eventually(CommandSession, time.Minute).Should(gexec.Exit(0))
	})

	define.Then(`^the command exits with an error$`, func() {
		Eventually(CommandSession, time.Minute).Should(gexec.Exit(1))
	})

	define.Then(`^it prints the usage$`, func() {
		Expect(CommandSession.Err).To(Say("Usage:"))
		Expect(CommandSession.Err).To(Say("relok8s chart move <chart> \\[flags\\]"))
	})
}
