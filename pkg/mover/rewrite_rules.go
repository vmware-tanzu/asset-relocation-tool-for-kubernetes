// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"

	"github.com/google/go-containerregistry/pkg/name"
)

// RewriteRules indicate What kind of target registry overrides we want to apply to the found images
type RewriteRules struct {
	// Registry overrides the registry part of the image FQDN, i.e myregistry.io
	Registry string
	// RepositoryPrefix will override the image path by being prepended before the image name
	RepositoryPrefix string
}

func (r *RewriteRules) Validate() error {
	if r.Registry != "" {
		_, err := name.NewRegistry(r.Registry, name.StrictValidation)
		if err != nil {
			return errors.New("registry rule is not valid")
		}
	}

	if r.RepositoryPrefix != "" {
		_, err := name.NewRepository(r.RepositoryPrefix)
		if err != nil {
			return errors.New("repository prefix is not valid")
		}
	}

	return nil
}
