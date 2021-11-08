// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
)

const (
	Username       = "username"
	Passwd         = "DummyPassword"
	HTPasswdFile   = "/data/htpasswd"
	RegistryDomain = "local-registry.io"
	RegistryConfig = "/data/registry-config.yml"
	CertFile       = "/data/" + RegistryDomain + ".pem"

	TestChart = "/data/wordpress-chart"
	Hints     = "/data/image-hints.yaml"
	Target    = "*-wp-relocated.tgz"
	Prefix    = "relocated/local-example"
)

func run(name string, arg ...string) string {
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		log.Fatalf("failed to run %s: %v\nOutput:\n%s", name, err, out)
	}
	return string(out)
}

func createHtpasswdFile(user, passwd string) {
	run("htpasswd", "-Bbc", HTPasswdFile, user, passwd)
}

func mkcert(domain string) {
	out := run("mkcert", domain)
	log.Printf("%s", out)
}

func fixHosts(domain string) {
	marker := "# Set host " + domain
	out := run("cat", "/etc/hosts")
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if line == marker {
			return
		}
	}
	newHosts := fmt.Sprintf("%s\n%s\n127.0.0.1\t%s", out, marker, domain)
	err := os.WriteFile("/etc/hosts", []byte(newHosts), 0777)
	if err != nil {
		log.Fatalf("failed to update /etc/hosts: %v", err)
	}
	out = run("cat", "/etc/hosts")
	log.Printf("%s", out)
}

func launchRegistry() (*exec.Cmd, io.Reader) {
	cmd := exec.Command("registry", "serve", RegistryConfig)
	cmd.Env = []string{
		"REGISTRY_HTTP_ADDR=0.0.0.0:443",
		"REGISTRY_HTTP_TLS_CERTIFICATE=/data/local-registry.io.pem",
		"REGISTRY_HTTP_TLS_KEY=/data/local-registry.io-key.pem",
		"REGISTRY_AUTH=htpasswd",
		"REGISTRY_AUTH_HTPASSWD_PATH=" + HTPasswdFile,
		"REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm",
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get the stdout pipe: %v", err)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("failed to get the stderr pipe: %v", err)
	}
	outputs := io.MultiReader(outPipe, errPipe)
	if err != nil {
		log.Fatalf("failed to run registry serve: %v", err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalf("failed to run registry serve: %v", err)
	}
	return cmd, outputs
}

func prepareRegistry(domain, username, password string) (*exec.Cmd, io.Reader) {
	createHtpasswdFile(username, password)
	mkcert(domain)
	fixHosts(domain)
	return launchRegistry()
}

func stopRegistry(cmd *exec.Cmd, outputs io.Reader) {
	err := cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Fatalf("failed to terminate subprocess: %v", err)
	}
	slurp, err := io.ReadAll(outputs)
	if err != nil {
		log.Fatalf("failed to get the stderr from the subprocess: %v", err)
	}
	fmt.Printf("%s\n", slurp)
	state, err := cmd.Process.Wait()
	if err != nil {
		log.Fatalf("failed to wait for subprocess: %v", err)
	}
	if !state.Success() {
		log.Fatalf("non zero subprocess exit: %v", state.ExitCode())
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("REGISTRY_TEST") != "true" {
		log.Printf("Skipping registry tests during Unit Tests")
		return
	}
	regCmd, outputs := prepareRegistry(RegistryDomain, Username, Passwd)
	exitVal := m.Run()
	stopRegistry(regCmd, outputs)
	os.Exit(exitVal)
}

func runInTest(t *testing.T, name string, arg ...string) string {
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run %s: %v\nOutput:\n%s", name, err, out)
	}
	return string(out)
}

func prepareTest(t *testing.T, certFile string) {
	caDir := "/etc/docker/certs.d/local-registry.io/"
	runInTest(t, "mkdir", "-p", caDir)
	runInTest(t, "cp", certFile, filepath.Join(caDir, "ca.crt"))
	os.Setenv("SSL_CERT_FILE", certFile)
	os.Setenv("GODEBUG", "http2debug=2")
	t.Logf(runInTest(t, "ls", "-l", caDir))
}

func dockerLogin(t *testing.T, domain, username, password string) {
	t.Logf(runInTest(t, "/bin/docker-login.sh", domain, username, password))
}

func relok8s(t *testing.T, chartPath, hints, target, targetRegistry, targetPrefix string) {
	req := &mover.ChartMoveRequest{
		Source: mover.Source{
			// The Helm Chart can be provided in either tarball or directory form
			Chart: mover.ChartSpec{Local: mover.LocalChart{Path: chartPath}},
			// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
			ImageHintsFile: hints,
		},
		Target: mover.Target{
			Chart: mover.ChartSpec{Local: mover.LocalChart{Path: target}},
			// Where to push and how to rewrite the found images
			// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
			Rules: mover.RewriteRules{
				Registry:         targetRegistry,
				RepositoryPrefix: targetPrefix,
			},
		},
	}
	cm, err := mover.NewChartMover(req, mover.WithLogger(log.Default()))
	if err != nil {
		t.Fatalf("failed to create chart mover: %v", err)
	}
	log.Printf("moving...")
	err = cm.Move()
	if err != nil {
		t.Fatalf("failed to move the chart: %v", err)
	}
}

func TestDockerCredentials(t *testing.T) {
	prepareTest(t, CertFile)
	dockerLogin(t, RegistryDomain, Username, Passwd)
	relok8s(t, TestChart, Hints, Target, RegistryDomain, Prefix)
}
