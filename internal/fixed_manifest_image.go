// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type fixedManifestImage struct {
	v1.Image
}

// FixedManifestImage returns a image that produces a manifest that conforms to
// the canonical form found in registries:
// - JSON pretty print of the manifest with 3 spaces indentation.
//
// Sample:
// {
// 	"schemaVersion": 2,
// 	"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
// 	"config": {
// 		 "mediaType": "application/vnd.docker.container.image.v1+json",
// 		 "size": 1456,
// 		 "digest": "sha256:16ea53ea7c652456803632d67517b78a4f9075a10bfdc4fc6b7b4cbf2bc98497"
// 	},
// 	"layers": [
// 		 {
// 				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
// 				"size": 767322,
// 				"digest": "sha256:24fb2886d6f6c5d16481dd7608b47e78a8e92a13d6e64d87d57cb16d5f766d63"
// 		 }
// 	]
// }
func FixedManifestImage(img v1.Image) v1.Image {
	return &fixedManifestImage{Image: img}
}

// RawManifest implements v1.Image's RawManifest wrapping the raw manifest
// in a repo canonized way
// See v1.Image interface:
// https://github.com/google/go-containerregistry/blob/main/pkg/v1/image.go
func (img *fixedManifestImage) RawManifest() ([]byte, error) {
	raw, err := img.Image.RawManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve underlying image raw manifest: %w", err)
	}
	canonized, err := canonizeManifest(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to canonize raw manifest: %w", err)
	}
	return canonized, nil
}

// Digest implements v1.Image's Digest ensuring the canonized raw manigest is
// the one being digested
// See v1.Image interface:
// https://github.com/google/go-containerregistry/blob/main/pkg/v1/image.go
func (img *fixedManifestImage) Digest() (v1.Hash, error) {
	originalDigest, err := img.Image.Digest()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to retrieve underlying image digest: %w", err)
	}
	h, err := hasherFor(originalDigest.Algorithm)
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to prepare digester: %w", err)
	}
	var digest v1.Hash
	digest.Algorithm = originalDigest.Algorithm
	manifest, err := img.RawManifest()
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to read raw manifest: %w", err)
	}
	h.Write(manifest)
	digest.Hex = hex.EncodeToString(h.Sum(nil))
	return digest, nil
}

func canonizeManifest(rawManifest []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := json.Indent(buf, rawManifest, "", "   "); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hasherFor(alg string) (hash.Hash, error) {
	if strings.ToLower(alg) == "sha256" {
		return sha256.New(), nil
	}
	return nil, fmt.Errorf("Digest algorithm %s not supported", alg)
}
