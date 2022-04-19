// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

//go:build go1.17

package main

import (
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/cmd"
)

func main() {
	cmd.Execute()
}
