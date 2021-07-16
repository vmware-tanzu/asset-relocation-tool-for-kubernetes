SHELL = /bin/bash

default: build


# #### GO Binary Management ####
.PHONY: deps-go-binary deps-goimports deps-counterfeiter deps-ginkgo

GO_VERSION := $(shell go version)
GO_VERSION_REQUIRED = go1.16
GO_VERSION_MATCHED := $(shell go version | grep $(GO_VERSION_REQUIRED))

deps-go-binary:
ifndef GO_VERSION
	$(error Go not installed)
endif
ifndef GO_VERSION_MATCHED
	$(error Required Go version is $(GO_VERSION_REQUIRED), but was $(GO_VERSION))
endif
	@:

HAS_COUNTERFEITER := $(shell command -v counterfeiter;)
HAS_GINKGO := $(shell command -v ginkgo;)
HAS_GO_IMPORTS := $(shell command -v goimports;)

deps-goimports: deps-go-binary
ifndef HAS_GO_IMPORTS
	go get -u golang.org/x/tools/cmd/goimports
endif

deps-counterfeiter: deps-go-binary
ifndef HAS_COUNTERFEITER
	go get -u github.com/maxbrunsfeld/counterfeiter/v6
endif

deps-ginkgo: deps-go-binary
ifndef HAS_GINKGO
	go get -u github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega
endif

# #### CLEAN ####
.PHONY: clean

clean: deps-go-binary 
	rm -rf build/*
	find vendor -d 1 -not -name .gitkeep | xargs rm -rf

# #### DEPS ####
.PHONY: deps

vendor/modules.txt: go.mod
	go mod vendor

deps: vendor/modules.txt deps-goimports deps-counterfeiter deps-ginkgo


# #### BUILD ####
.PHONY: build

SRC = $(shell find . -name "*.go" | grep -v "_test\." )
VERSION := $(or $(VERSION), dev)
LDFLAGS="-X github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/cmd.Version=$(VERSION)"

build/relok8s: $(SRC)
	go build -o build/relok8s -ldflags ${LDFLAGS} ./main.go

build/relok8s-darwin: $(SRC)
	GOARCH=amd64 GOOS=darwin go build -o build/relok8s-darwin -ldflags ${LDFLAGS} ./main.go

build/relok8s-linux: $(SRC)
	GOARCH=amd64 GOOS=linux go build -o build/relok8s-linux -ldflags ${LDFLAGS} ./main.go

build: deps build/relok8s

build-all: build/relok8s-darwin build/relok8s-linux

# #### TESTS ####
.PHONY: lint test test-features test-units

test-units: deps
	ginkgo -r -skipPackage test .

test-fixtures:
	make --directory test/fixtures

test-features: deps test-fixtures
	ginkgo -r -tags=feature test

test-external: deps test-fixtures
	ginkgo -r -tags=external test

test: deps lint test-units test-features

test-all: test test-external

lint: deps-goimports
	git ls-files | grep -v '^vendor/' | grep '.go$$' | xargs goimports -l -w


# #### DEVOPS ####
.PHONY: set-pipeline set-example-pipeline
set-pipeline: ci/pipeline.yaml
	fly -t tie set-pipeline --config ci/pipeline.yaml --pipeline relok8s

set-example-pipeline: docs/example-pipeline.yaml
	fly -t tie set-pipeline --config docs/example-pipeline.yaml --pipeline relok8s-example
