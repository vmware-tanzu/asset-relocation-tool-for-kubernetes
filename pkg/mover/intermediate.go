// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	// ErrNotIntermediateBundle when a verified path does not have expected intermediate bundle contents
	ErrNotIntermediateBundle = errors.New("not an intermediate chart bundle")
)

func saveIntermediateBundle(cd *ChartData, tarFile string, log Logger) error {
	tfw, err := newTarFileWriter(tarFile)
	if err != nil {
		return err
	}

	// hints file goes first to be extracted quickly on demand
	log.Printf("Writing %s...\n", IntermediateBundleHintsFilename)
	if err := tfw.WriteFile(IntermediateBundleHintsFilename, cd.rawHints, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write %s: %w", IntermediateBundleHintsFilename, err)
	}

	log.Printf("Writing Helm Chart files at %s/...\n", cd.chart.Metadata.Name)
	if err := tarChart(tfw, cd.chart); err != nil {
		return fmt.Errorf("failed archiving original-chart/: %w", err)
	}

	// Need to give a raw writer to the tarball lib, ready to append tar entries
	w := tfw.ContinueWithRawWriter()
	if err := packImages(w, cd.imageChanges, log); err != nil {
		return fmt.Errorf("failed archiving images: %w", err)
	}

	if err := tfw.Close(); err != nil {
		return fmt.Errorf("failed closing intermediate bundle at %s: %w", tarFile, err)
	}
	log.Printf("Intermediate bundle complete at %s\n", tarFile)
	return nil
}

func tarChart(tfw *tarFileWriter, chart *chart.Chart) error {
	for _, file := range chart.Raw {
		if err := tfw.WriteFile(filepath.Join("original-chart", file.Name), file.Data, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write chart's inner file %s: %v", file.Name, err)
		}
	}
	return nil
}

func packImages(w io.Writer, imageChanges []*internal.ImageChange, logger Logger) error {
	refToImage := map[name.Reference]v1.Image{}
	for _, change := range imageChanges {
		if previousImage, ok := refToImage[change.ImageReference]; ok {
			if err := deduplicateByDigest(change.ImageReference.Name(), change.Image, previousImage); err != nil {
				return err
			}
			continue
		}
		refToImage[change.ImageReference] = change.Image
		logger.Printf("Processing image %s\n", change.ImageReference.Name())
	}
	logger.Printf("Writing all %d images...\n", len(refToImage))
	if err := tarball.MultiRefWrite(refToImage, w); err != nil {
		return err
	}
	return nil
}

// deduplicateByDigest asserts our assumption that, within a given chart,
// a particular fully qualified image tag name uniquely identifies its contents.
// This checks returns an error if the same name is associated to more than one
// digest value. It also fails if the digest values cannot be retrieved.
func deduplicateByDigest(name string, current, previous v1.Image) error {
	previousDigest, err := previous.Digest()
	if err != nil {
		return fmt.Errorf("failed to check previous image digest: %v", err)
	}
	imageDigest, err := current.Digest()
	if err != nil {
		return fmt.Errorf("failed to check current image digest: %v", err)
	}
	if previousDigest != imageDigest {
		return fmt.Errorf("found image %q with different digests %s vs %s",
			name, previousDigest, imageDigest)
	}
	return nil
}
