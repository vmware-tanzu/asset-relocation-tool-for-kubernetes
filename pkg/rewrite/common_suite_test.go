package rewrite_test

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommonPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rewrite Package Suite")
}
