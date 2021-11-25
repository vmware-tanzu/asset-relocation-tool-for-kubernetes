// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	// ErrNotIntermediateBundle when a verified path does not have expected intermediate bundle contents
	ErrNotIntermediateBundle = errors.New("not an intermediate chart bundle")

	// ErrIntermediateBundleClosed when accessing a bundle that was already closed
	ErrIntermediateBundleClosed = errors.New("intermediate bundle already closed")
)

// IsIntermediateBundle returns tue only if VerifyIntermediateBundle finds no errors
func IsIntermediateBundle(bundlePath string) bool {
	log.Printf("VerifyIntermediateBundle=%v", VerifyIntermediateBundle(bundlePath))
	return VerifyIntermediateBundle(bundlePath) == nil
}

// VerifyIntermediateBundle returns true if the path points to a tarball with the
// expected contents for an intermediate bundle
// An IntermediateBundle is:
// - A Tar file or a folder (if already extracted from the tar), compression is not supported
// Within the tar or folder there should be:
// - A hints.yaml YAML file
// - A images.tar TAR file
// - A directory container an unpacked chart directory
func VerifyIntermediateBundle(bundlePath string) error {
	ib, err := openBundle(bundlePath)
	if err != nil {
		return err
	}
	defer ib.close()
	return ib.validate()
}

type intermediateBundle struct {
	originalPath string
	dir          string
	temporaryDir bool
}

func openBundle(bundlePath string) (*intermediateBundle, error) {
	bundle := &intermediateBundle{originalPath: bundlePath}
	if isFile(bundlePath) {
		if err := validateTar(bundlePath); err != nil {
			return bundle, fmt.Errorf("%s is not tar file: %w", bundlePath, ErrNotIntermediateBundle)
		}
		tempDir, err := os.MkdirTemp("", "intermediate-bundle-dir-*")
		if err != nil {
			return bundle, err
		}
		if err := untar(bundlePath, tempDir); err != nil {
			return bundle, err
		}
		bundle.dir = tempDir
		bundle.temporaryDir = true
		return bundle, nil
	}
	bundle.dir = bundle.originalPath
	return bundle, nil
}

func (ib *intermediateBundle) close() error {
	if ib.dir == "" {
		return fmt.Errorf("bundle at %s: %w", ib.originalPath, ErrIntermediateBundleClosed)
	}
	if ib.temporaryDir {
		if err := os.RemoveAll(ib.dir); err != nil {
			return err
		}
		ib.dir = ""
		return nil
	}
	return nil
}

func (ib *intermediateBundle) chartDir() (string, error) {
	if ib.dir == "" {
		return "", fmt.Errorf("bundle at %s: %w", ib.originalPath, ErrIntermediateBundleClosed)
	}
	d, err := os.Open(ib.dir)
	if err != nil {
		return "", err
	}
	defer d.Close()
	infos, err := d.Readdir(-1)
	if err != nil {
		return "", err
	}
	dirs := []string{}
	for _, info := range infos {
		if info.IsDir() {
			dirs = append(dirs, info.Name())
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("expected a single chart dir found %v: %w", dirs, ErrNotIntermediateBundle)
	}
	return filepath.Join(ib.dir, dirs[0]), nil
}

type fileValidations struct {
	filename, format string
	validate         func(string) error
}

func (ib *intermediateBundle) validate() error {
	if ib.dir == "" {
		return fmt.Errorf("bundle at %s: %w", ib.originalPath, ErrIntermediateBundleClosed)
	}
	validations := []fileValidations{
		{filename: "hints.yaml", format: "YAML", validate: validateYaml},
		{filename: "images.tar", format: "tarball", validate: validateTar},
	}
	if err := validateDirFiles(ib.dir, validations); err != nil {
		return err
	}
	return ib.validateChartDir()
}

func (ib *intermediateBundle) validateChartDir() error {
	chartDir, err := ib.chartDir()
	if err != nil {
		return err
	}
	validations := []fileValidations{
		{filename: "Chart.yaml", format: "YAML", validate: validateYaml},
		{filename: "values.yaml", format: "YAML", validate: validateYaml},
	}
	return validateDirFiles(chartDir, validations)
}

func validateDirFiles(dir string, validations []fileValidations) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	entries, err := d.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", dir, err)
	}
	validated := []string{}
	for _, entry := range entries {
		fullpath := filepath.Join(dir, entry)
		for _, validation := range validations {
			if entry == validation.filename {
				if err := validation.validate(fullpath); err != nil {
					return fmt.Errorf("%s is not a %s: %v: %w", entry, validation.format, err, ErrNotIntermediateBundle)
				}
				validated = append(validated, entry)
			}
		}
	}
	if len(validated) < len(validations) {
		return fmt.Errorf("expected to have found %d files at %s but just found %v",
			len(validations), dir, len(validated))
	}
	return nil
}

func (ib *intermediateBundle) loadHints(log Logger) ([]byte, error) {
	if ib.dir == "" {
		return nil, fmt.Errorf("bundle at %s: %w", ib.originalPath, ErrIntermediateBundleClosed)
	}
	return os.ReadFile(filepath.Join(ib.dir, HintsFilename))
}

func (ib *intermediateBundle) loadImage(imageRef name.Reference) (v1.Image, string, error) {
	if ib.dir == "" {
		return nil, "", fmt.Errorf("bundle at %s: %w", ib.originalPath, ErrIntermediateBundleClosed)
	}
	imagesTar := filepath.Join(ib.dir, "images.tar")
	tag, err := Tag(imageRef)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make tag from %s: %w", imageRef.Name(), err)
	}
	image, err := tarball.ImageFromPath(imagesTar, &tag)
	if err != nil {
		return nil, "", fmt.Errorf("failed to export image %s from tarball %s: %w", tag.Name(), imagesTar, err)
	}
	digest, err := image.Digest()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get image digest for %s: %w", tag.Name(), err)
	}
	return image, digest.String(), nil
}

func Tag(imageRef name.Reference) (name.Tag, error) {
	return name.NewTag(fmt.Sprintf("%s:%s", imageRef.Context().Name(), imageRef.Identifier()))
}

func saveIntermediateBundle(cd *ChartData, targetPath string, log Logger) error {
	bundleWorkDir, err := os.MkdirTemp("", "intermediate-tarball-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory to build tar: %w", err)
	}

	log.Printf("Writing chart at %s/...\n", cd.chart.Metadata.Name)
	if err := writeChart(cd.chart, filepath.Join(bundleWorkDir, cd.chart.Metadata.Name)); err != nil {
		return fmt.Errorf("failed archiving chart %s: %w", cd.chart.Name(), err)
	}

	if err := packImages(bundleWorkDir, cd.imageChanges, log); err != nil {
		return fmt.Errorf("failed archiving images: %w", err)
	}

	log.Printf("Writing hints file %s...\n", HintsFilename)
	hintsPath := filepath.Join(bundleWorkDir, HintsFilename)
	if err := os.WriteFile(hintsPath, cd.rawHints, defaultTarPermissions); err != nil {
		return fmt.Errorf("failed to write hints file: %w", err)
	}

	log.Printf("Packing all as tarball %s...\n", targetPath)
	if err := tarDirectory(bundleWorkDir, targetPath); err != nil {
		return fmt.Errorf("failed to tar bundle as %s: %w", targetPath, err)
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
	logger.Printf("Packing all %d images within images.tar...\n", len(refToImage))
	if err := tarball.MultiRefWriteToFile(imagesTarball, refToImage); err != nil {
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

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func validateTar(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	_, err = tr.Next()
	return err
}

func validateYaml(path string) error {
	yamlContents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var data interface{}
	return yaml.Unmarshal(yamlContents, &data)
}
