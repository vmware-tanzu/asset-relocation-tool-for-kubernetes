// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover_test

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
)

const (
	TestChart = "/data/wordpress-chart"
	Hints     = "/data/image-hints.yaml"
	Target    = "*-wp-relocated.tgz"
	Prefix    = "relocated/local-example"
)

func run(t *testing.T, name string, arg ...string) string {
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run %s: %v\nOutput:\n%s", name, err, out)
	}
	return string(out)
}

func prepareTest(t *testing.T, certFile string) {
	caDir := "/etc/docker/certs.d/local-registry.io/"
	run(t, "mkdir", "-p", caDir)
	run(t, "cp", certFile, filepath.Join(caDir, "ca.crt"))
	os.Setenv("SSL_CERT_FILE", certFile)
	t.Logf(run(t, "ls", "-l", caDir))
}

func dockerLogin(t *testing.T, domain, username, password string) {
	t.Logf(run(t, "/bin/docker-login.sh", domain, username, password))
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

func skipUnitTest(t *testing.T) {
	log.Printf("LOCAL_REGISTRY_TEST=%q", os.Getenv("LOCAL_REGISTRY_TEST"))
	if os.Getenv("LOCAL_REGISTRY_TEST") == "" {
		t.Skip("Skip local-registry tests on unit tests")
	}
}

func TestDockerCredentials(t *testing.T) {
	skipUnitTest(t)
	certFile := os.Getenv("SSL_CERT_FILE")
	domain := os.Getenv("DOMAIN")
	user := os.Getenv("USER")
	passwd := os.Getenv("PASSWD")
	prepareTest(t, certFile)
	dockerLogin(t, domain, user, passwd)
	relok8s(t, TestChart, Hints, Target, domain, Prefix)
}

func TestCustomCredentials(t *testing.T) {
	// TODO: add here the custom credentials test
}
