// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import "github.com/google/go-containerregistry/pkg/authn"

type authnKeychain map[string]Credentials

func (kc authnKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if creds, ok := kc[resource.RegistryStr()]; ok {
		return &authenticator{creds}, nil
	}
	return authn.DefaultKeychain.Resolve(resource)
}

type authenticator struct {
	Credentials
}

func (auth *authenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{Username: auth.Username, Password: auth.Password}, nil
}
