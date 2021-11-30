// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
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

// saveIntermediateBundle will tar in this order:
// - The original chart
// - The hits file
// - The container images detected as references in the chart
//
// The hints file goes first in the tar, followed by the chart files.
// Finally, images are appended using the go-containerregistry tarball lib
func saveIntermediateBundle(cd *ChartData, tarFile string, log Logger) error {
	tmpTarfile := fmt.Sprintf("%s.tmp", tarFile)
	tfw, err := newTarFileWriter(tmpTarfile)
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
		return fmt.Errorf("failed closing intermediate bundle %s: %w", tmpTarfile, err)
	}
	if err := os.Rename(tmpTarfile, tarFile); err != nil {
		return fmt.Errorf("failed renaming %s -> %s: %w", tmpTarfile, tarFile, err)
	}
	log.Printf("Intermediate bundle complete at %s\n", tarFile)
	return nil
}

// tarChart tars all files from the original chart into `original-chart/`
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
		if _, ok := refToImage[change.ImageReference]; ok {
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
