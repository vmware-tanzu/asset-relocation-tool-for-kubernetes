// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"helm.sh/helm/v3/pkg/chart"
)

func (cm *ChartMover) saveOfflineBundle() error {
	log := cm.logger

	bundleWorkDir, err := os.MkdirTemp("", "offline-tarball-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to build tar: %w", err)
	}

	log.Printf("Writing chart at %s/...\n", cm.chart.Metadata.Name)
	if err := writeChart(cm.chart, filepath.Join(bundleWorkDir, cm.chart.Metadata.Name)); err != nil {
		return fmt.Errorf("failed archiving chart %s: %w", cm.chart.Name(), err)
	}

	if err := packImages(bundleWorkDir, cm.imageChanges, cm.logger); err != nil {
		return fmt.Errorf("failed archiving images: %w", err)
	}

	log.Printf("Writing hints file %s...\n", HintsFilename)
	hintsPath := filepath.Join(bundleWorkDir, HintsFilename)
	if err := os.WriteFile(hintsPath, cm.rawHints, defaultTarPermissions); err != nil {
		return fmt.Errorf("failed to write hints file: %w", err)
	}

	log.Printf("Packing all as tarball %s...\n", cm.targetOfflineTarPath)
	if err := tarDirectory(bundleWorkDir, cm.targetOfflineTarPath); err != nil {
		return fmt.Errorf("failed to tar bundle as %s: %w", cm.targetOfflineTarPath, err)
	}
	return os.RemoveAll(bundleWorkDir)
}

func writeChart(chart *chart.Chart, targetDir string) error {
	if err := os.MkdirAll(targetDir, defaultTarPermissions); err != nil {
		return fmt.Errorf("failed to create target chart %s: %w", targetDir, err)
	}
	for _, file := range chart.Raw {
		dir := filepath.Dir(file.Name)
		dirPath := filepath.Join(targetDir, dir)
		if err := os.MkdirAll(dirPath, defaultTarPermissions); err != nil {
			return fmt.Errorf("failed to create path %s: %w", dirPath, err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, file.Name), file.Data, defaultTarPermissions); err != nil {
			return fmt.Errorf("failed to write chart's inner file %s: %v", file.Name, err)
		}
	}
	return nil
}

func packImages(archivePath string, imageChanges []*internal.ImageChange, logger Logger) error {
	imagesTarball := filepath.Join(archivePath, "images.tar")
	tagToImage := map[name.Tag]v1.Image{}
	for _, change := range imageChanges {
		imageName := change.ImageReference.Context().Name()
		version := change.ImageReference.Identifier()
		fullImageName := fmt.Sprintf("%s:%s", imageName, version)
		tag, err := name.NewTag(fullImageName)
		if err != nil {
			return fmt.Errorf("failed to create tag %q: %v", fullImageName, err)
		}
		if previousImage, ok := tagToImage[tag]; ok {
			if err := deduplicateByDigest(fullImageName, change.Image, previousImage); err != nil {
				return err
			}
			continue
		}
		tagToImage[tag] = change.Image
		logger.Printf("Processing image %s\n", fullImageName)
	}
	logger.Printf("Packing all %d images within images.tar...\n", len(tagToImage))
	if err := tarball.MultiWriteToFile(imagesTarball, tagToImage); err != nil {
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
