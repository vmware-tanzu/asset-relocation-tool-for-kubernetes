package rewrite

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

type Rules struct {
	Registry         string
	RepositoryPrefix string
	Digest           string
}

func (r *Rules) IsEmpty() bool {
	return *r == Rules{}
}
