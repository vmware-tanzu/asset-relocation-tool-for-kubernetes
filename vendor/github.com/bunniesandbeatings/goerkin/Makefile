SHELL = /bin/bash

default: test

HAS_GO_IMPORTS := $(shell command -v goimports;)

deps-goimports:
ifndef HAS_GO_IMPORTS
	go get -u golang.org/x/tools/cmd/goimports
endif

# #### CLEAN ####
clean:
	rm -rf build/*
	go clean --modcache


# #### DEPS ####

deps: deps-goimports
	go mod download

test: deps lint
	ginkgo -r --randomizeAllSpecs --randomizeSuites --race --trace .

lint: deps-goimports
	go vet
	git ls-files | grep '.go$$' | xargs goimports -l -w
