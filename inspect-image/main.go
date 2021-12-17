// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func mainE() error {
	imageName := "harbor-repo.vmware.com/dockerhub-proxy-cache/library/busybox:1.33.1"
	fmt.Println(imageName)
	imgRef, err := name.NewTag(imageName)
	if err != nil {
		return err
	}
	image, err := remote.Image(imgRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imgRef.Name(), err)
	}
	digest, err := image.Digest()
	if err != nil {
		return err
	}
	fmt.Printf("Digest: %s\n", digest)
	rawManifest, err := image.RawManifest()
	if err != nil {
		return err
	}
	fmt.Printf("Raw manifest:\n%s\n", string(rawManifest))
	h := sha256.New()
	h.Write(rawManifest)
	recomputedDigest := h.Sum(nil)
	hexRecomputedDigest := hex.EncodeToString(recomputedDigest)
	fmt.Printf("Recomputed digest: %s\n", hexRecomputedDigest)
	manifest, err := image.Manifest()
	if err != nil {
		return err
	}
	fmt.Printf("Manifest:\n%v\n", manifest)
	rawJSONManifest, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	fmt.Printf("Recomputed manifest RAW JSON:\n%s\n", string(rawJSONManifest))
	buf := bytes.NewBufferString("")
	if err := json.Indent(buf, rawJSONManifest, "", "   "); err != nil {
		return err
	}
	jsonManifest := buf.Bytes()

	fmt.Printf("Recomputed manifest pretty JSON:\n%s\n", string(jsonManifest))
	h = sha256.New()
	h.Write(jsonManifest)
	reconstructedDigest := h.Sum(nil)
	hexReconstructedDigest := hex.EncodeToString(reconstructedDigest)

	fmt.Printf("Raw manifest hex dump:\n%s\n", hex.Dump(rawManifest))
	fmt.Printf("Recomputed manifest pretty JSON hex dump:\n%s\n", hex.Dump(jsonManifest))

	localImageFilename := "/tmp/busybox-1.33.1-test.tar"
	imgRefs := map[name.Reference]v1.Image{}
	imgRefs[imgRef] = image
	if err := tarball.MultiRefWriteToFile(localImageFilename, imgRefs); err != nil {
		return err
	}
	fmt.Printf("Saved image to %s\n", localImageFilename)

	reloadedImg, err := tarball.ImageFromPath(localImageFilename, &imgRef)
	if err != nil {
		return err
	}
	reloadedDigest, err := reloadedImg.Digest()
	if err != nil {
		return err
	}
	fmt.Printf("Reloaded Digest: %s\n", reloadedDigest.Hex)
	rawReloadedManifest, err := reloadedImg.RawManifest()
	if err != nil {
		return err
	}
	fmt.Printf("Raw Reloaded manifest:\n%s\n", string(rawReloadedManifest))

	buf.Reset()
	if err := json.Indent(buf, rawReloadedManifest, "", "   "); err != nil {
		return err
	}
	rebuiltManifest := buf.Bytes()
	fmt.Printf("Rebuilt Reloaded manifest:\n%s\n", string(rebuiltManifest))

	h = sha256.New()
	h.Write(rebuiltManifest)
	rebuiltDigest := h.Sum(nil)
	hexRebuiltDigest := hex.EncodeToString(rebuiltDigest)
	fmt.Printf("Rebuilt digest: %s\n", string(hexRebuiltDigest))

	fmt.Printf("Digest:               %s\n", digest.Hex)
	fmt.Printf("Recomputed digest:    %s\n", hexRecomputedDigest)
	fmt.Printf("Reconstructed digest: %s\n", hexReconstructedDigest)

	fmt.Printf("Reloaded Digest:      %s\n", reloadedDigest.Hex)
	fmt.Printf("Rebuilt digest:       %s\n", string(hexRebuiltDigest))
	return nil
}

func main() {
	if err := mainE(); err != nil {
		log.Fatal(err)
	}
}
