// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import "github.com/google/go-containerregistry/pkg/authn"

type authnKeychain Repository

func (kc authnKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if kc.Server == resource.RegistryStr() {
		return &authenticator{Repository(kc)}, nil
	}
	return authn.DefaultKeychain.Resolve(resource)
}

type authenticator struct {
	Repository
}

func (auth *authenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{Username: auth.Username, Password: auth.Password}, nil
}
