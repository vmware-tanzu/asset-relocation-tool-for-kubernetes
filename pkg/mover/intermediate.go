// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"helm.sh/helm/v3/pkg/chart"
)

const (
	defaultPerm fs.FileMode = 0644
)

type bundledChartData struct {
	chart        *chart.Chart
	imageChanges []*internal.ImageChange
	rawHints     []byte
}

// saveIntermediateBundle will tar in this order:
// - The original chart
// - The hits file
// - The container images detected as references in the chart
//
// The hints file goes first in the tar, followed by the chart files.
// Finally, images are appended using the go-containerregistry tarball lib
func saveIntermediateBundle(bcd *bundledChartData, tarFile string, log Logger) error {
	tmpTarball, err := os.CreateTemp("", "intermediate-bundle-tar-*")
	if err != nil {
		return err
	}
	tmpTarballFilename := tmpTarball.Name()
	tfw := wrapAsTarFileWriter(tmpTarball)

	// hints file goes first to be extracted quickly on demand
	log.Printf("Writing %s...\n", IntermediateBundleHintsFilename)
	if err := tfw.WriteMemFile(IntermediateBundleHintsFilename, bcd.rawHints, defaultPerm); err != nil {
		return fmt.Errorf("failed to write %s: %w", IntermediateBundleHintsFilename, err)
	}

	log.Printf("Writing Helm Chart files at %s/...\n", bcd.chart.Metadata.Name)
	if err := tarChart(tfw, bcd.chart); err != nil {
		return fmt.Errorf("failed archiving original-chart/: %w", err)
	}

	if err := packImages(tfw, bcd.imageChanges, log); err != nil {
		return fmt.Errorf("failed archiving images: %w", err)
	}

	if err := tfw.Close(); err != nil {
		return fmt.Errorf("failed closing intermediate bundle temp file %s: %w",
			tmpTarballFilename, err)
	}

	// TODO: check if this may fail across different mounts
	if err := os.Rename(tmpTarballFilename, tarFile); err != nil {
		return fmt.Errorf("failed renaming %s -> %s: %w", tmpTarballFilename, tarFile, err)
	}
	log.Printf("Intermediate bundle complete at %s\n", tarFile)
	return nil
}

// tarChart tars all files from the original chart into `original-chart/`
func tarChart(tfw *tarFileWriter, chart *chart.Chart) error {
	for _, file := range chart.Raw {
		if err := tfw.WriteMemFile(filepath.Join("original-chart", file.Name), file.Data, defaultPerm); err != nil {
			return fmt.Errorf("failed to write chart's inner file %s: %v", file.Name, err)
		}
	}
	return nil
}

func packImages(tfw *tarFileWriter, imageChanges []*internal.ImageChange, logger Logger) error {
	imagesTarFilename, err := tarImages(imageChanges, logger)
	if err != nil {
		return fmt.Errorf("failed to pack images: %w", err)
	}
	defer os.Remove(imagesTarFilename)
	f, err := os.Open(imagesTarFilename)
	if err != nil {
		return fmt.Errorf("failed to reopen %s for tarring: %w", imagesTarFilename, err)
	}
	defer f.Close()
	info, err := os.Stat(imagesTarFilename)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", imagesTarFilename, err)
	}
	return tfw.WriteIOFile("images.tar", info.Size(), f, defaultPerm)
}

func tarImages(imageChanges []*internal.ImageChange, logger Logger) (string, error) {
	imagesFile, err := os.CreateTemp("", "image-tar-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary images tar file: %w", err)
	}
	defer imagesFile.Close()

	refToImage := map[name.Reference]v1.Image{}
	for _, change := range imageChanges {
		if _, ok := refToImage[change.ImageReference]; ok {
			continue
		}
		refToImage[change.ImageReference] = change.Image
		logger.Printf("Processing image %s\n", change.ImageReference.Name())
	}

	logger.Printf("Writing all %d images...\n", len(refToImage))
	if err := tarball.MultiRefWrite(refToImage, imagesFile); err != nil {
		return "", err
	}
	return imagesFile.Name(), nil
}
