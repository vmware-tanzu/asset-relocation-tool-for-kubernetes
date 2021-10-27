// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import "github.com/google/go-containerregistry/pkg/authn"

type authnKeychain Repository

// Resolve implements an authn.KeyChain
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Keychain
//
// Returns a custom credentials authn.Authenticator if the given resource
// RegistryStr() matches the Repository, otherwise it falls back to the default
// KeyChain which may include local docker credentials.
func (kc authnKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if kc.Server == resource.RegistryStr() {
		return &authenticator{Repository(kc)}, nil
	}
	return authn.DefaultKeychain.Resolve(resource)
}

type authenticator struct {
	Repository
}

// Authorization implements an authn.Authenticator
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Authenticator
//
// Returns an authn.AuthConfig with a user / password pair to be used for authentication
func (auth *authenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{Username: auth.Username, Password: auth.Password}, nil
}
