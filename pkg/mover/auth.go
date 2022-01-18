// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import "github.com/google/go-containerregistry/pkg/authn"

// Resolve implements an authn.KeyChain
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Keychain
//
// Returns a custom credentials authn.Authenticator if the given resource
// RegistryStr() matches the Repository, otherwise it returns annonymous access
func (repo ContainerRepository) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if repo.Server == resource.RegistryStr() {
		return repo, nil
	}

	// if no credentials are provided we return annon authentication
	return authn.Anonymous, nil
}

// Authorization implements an authn.Authenticator
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Authenticator
//
// Returns an authn.AuthConfig with a user / password pair to be used for authentication
func (repo ContainerRepository) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{Username: repo.Username, Password: repo.Password}, nil
}

// Define a container images keychain based on the settings provided via the containers struct
// If useDefaultKeychain is set, use config/docker.json otherwise load the provided creds (if any)
func getContainersKeychain(c Containers) authn.Keychain {
	if c.UseDefaultLocalKeychain {
		return authn.DefaultKeychain
	}

	return c.ContainerRepository
}
