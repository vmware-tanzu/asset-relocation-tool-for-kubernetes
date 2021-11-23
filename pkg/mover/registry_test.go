// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/avast/retry-go"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
)

const (
	TestChart = "/data/wordpress-chart"
	Hints     = "/data/image-hints.yaml"
	Target    = "*-wp-relocated.tgz"
	Prefix    = "relocated/local-example"

	BadUser   = "notauser"
	BadPasswd = "notapassword"
)

var logger = mover.NoLogger // change to mover.DefaultLogger for debug

func skipUnitTest(t *testing.T) {
	if os.Getenv("LOCAL_REGISTRY_TEST") == "" {
		t.Skip("Skip local-registry tests on unit tests")
	}
}

func run(t *testing.T, name string, arg ...string) string {
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run %s: %v\nOutput:\n%s", name, err, out)
	}
	return string(out)
}

func prepareDockerCA(t *testing.T, certFile string) {
	caDir := "/etc/docker/certs.d/local-registry.io/"
	run(t, "mkdir", "-p", caDir)
	run(t, "cp", certFile, filepath.Join(caDir, "ca.crt"))
	os.Setenv("SSL_CERT_FILE", certFile)
}

func dockerLogin(t *testing.T, domain, username, password string) {
	run(t, "/bin/docker-login.sh", domain, username, password)
}

func dockerLogout(t *testing.T, domain string) {
	dockerLogin(t, domain, "", "")
}

func NewMoveRequest(chartPath, hints, target, targetRegistry, targetPrefix string) *mover.ChartMoveRequest {
	return &mover.ChartMoveRequest{
		Source: mover.Source{
			// The Helm Chart can be provided in either tarball or directory form
			Chart: mover.ChartSpec{Local: &mover.LocalChart{Path: chartPath}},
			// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
			ImageHintsFile: hints,
		},
		Target: mover.Target{
			Chart: mover.ChartSpec{Local: &mover.LocalChart{Path: target}},
			// Where to push and how to rewrite the found images
			// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
			Rules: mover.RewriteRules{
				Registry:         targetRegistry,
				RepositoryPrefix: targetPrefix,
			},
		},
	}
}

func repo(domain, user, passwd string) mover.ContainerRepository {
	return mover.ContainerRepository{
		Server:   domain,
		Username: user,
		Password: passwd,
	}
}

func relok8s(t *testing.T, req *mover.ChartMoveRequest) error {
	cm, err := mover.NewChartMover(req, mover.WithLogger(logger))
	if err != nil {
		t.Fatalf("failed to create chart mover: %v", err)
	}
	return cm.Move()
}

type Params struct {
	certFile, domain, user, passwd string
}

func loadParams() Params {
	return Params{
		certFile: os.Getenv("SSL_CERT_FILE"),
		domain:   os.Getenv("DOMAIN"),
		user:     os.Getenv("USER"),
		passwd:   os.Getenv("PASSWD"),
	}
}

func TestDockerCredentials(t *testing.T) {
	skipUnitTest(t)
	params := loadParams()
	prepareDockerCA(t, params.certFile)
	dockerLogin(t, params.domain, params.user, params.passwd)
	got := relok8s(t, NewMoveRequest(TestChart, Hints, Target, params.domain, Prefix))
	var want error
	if got != want {
		t.Fatalf("want error %v got %v", want, got)
	}
}

func TestCustomCredentials(t *testing.T) {
	skipUnitTest(t)
	params := loadParams()
	prepareDockerCA(t, params.certFile)
	dockerLogout(t, params.domain)
	req := NewMoveRequest(TestChart, Hints, Target, params.domain, Prefix)
	req.Target.Containers.ContainerRepository = repo(params.domain, params.user, params.passwd)
	got := relok8s(t, req)
	var want error
	if got != want {
		t.Fatalf("want error %v got %v", want, got)
	}
}

func TestBadDockerCredentials(t *testing.T) {
	skipUnitTest(t)
	params := loadParams()
	prepareDockerCA(t, params.certFile)
	dockerLogin(t, params.domain, BadUser, BadPasswd)
	got := relok8s(t, NewMoveRequest(TestChart, Hints, Target, params.domain, Prefix))
	// retry.Error is incompatible with errors package, it cannot be unwrapped
	_, ok := got.(retry.Error)
	if !ok {
		t.Fatalf("want error.retry got %v", got)
	}
}

func TestBadCustomCredentials(t *testing.T) {
	skipUnitTest(t)
	params := loadParams()
	prepareDockerCA(t, params.certFile)
	dockerLogout(t, params.domain)
	req := NewMoveRequest(TestChart, Hints, Target, params.domain, Prefix)
	req.Target.Containers.ContainerRepository = repo(params.domain, BadUser, BadPasswd)
	got := relok8s(t, req)
	// retry.Error is incompatible with errors package, it cannot be unwrapped
	_, ok := got.(retry.Error)
	if !ok {
		t.Fatalf("want error.retry got %v", got)
	}
}