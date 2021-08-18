package internal

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type ImageChange struct {
	Pattern            *ImageTemplate
	ImageReference     name.Reference
	RewrittenReference name.Reference
	Image              v1.Image
	Digest             string
	AlreadyPushed      bool
}

func (change *ImageChange) ShouldPush() bool {
	return !change.AlreadyPushed && change.ImageReference.Name() != change.RewrittenReference.Name()
}
