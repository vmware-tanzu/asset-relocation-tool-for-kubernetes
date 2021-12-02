// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
)

const (
	originalChart = "original-chart"
	imagesTar     = "images.tar"

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
	tmpTarballFilename, err := tarChartData(bcd, log)
	if err != nil {
		return err
	}
	// TODO(josvaz): check if this may fail across different mounts
	if err := os.Rename(tmpTarballFilename, tarFile); err != nil {
		return fmt.Errorf("failed renaming %s -> %s: %w", tmpTarballFilename, tarFile, err)
	}
	log.Printf("Intermediate bundle complete at %s\n", tarFile)
	return nil
}

func tarChartData(bcd *bundledChartData, log Logger) (string, error) {
	tmpTarball, err := os.CreateTemp("", "intermediate-bundle-tar-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary tar file: %w", err)
	}
	tmpTarballFilename := tmpTarball.Name()
	tfw := wrapAsTarFileWriter(tmpTarball)
	defer tfw.Close()

	// hints file goes first to be extracted quickly on demand
	log.Printf("Writing %s...\n", IntermediateBundleHintsFilename)
	if err := tfw.WriteMemFile(IntermediateBundleHintsFilename, bcd.rawHints, defaultPerm); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", IntermediateBundleHintsFilename, err)
	}

	log.Printf("Writing Helm Chart files at %s/...\n", originalChart)
	if err := tarChart(tfw, bcd.chart); err != nil {
		return "", fmt.Errorf("failed archiving %s/: %w", originalChart, err)
	}

	if err := packImages(tfw, bcd.imageChanges, log); err != nil {
		return "", fmt.Errorf("failed archiving images: %w", err)
	}

	return tmpTarballFilename, nil
}

// tarChart tars all files from the original chart into `original-chart/`
func tarChart(tfw *tarFileWriter, chart *chart.Chart) error {
	for _, file := range chart.Raw {
		if err := tfw.WriteMemFile(filepath.Join(originalChart, file.Name), file.Data, defaultPerm); err != nil {
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
	return tfw.WriteIOFile(imagesTar, info.Size(), f, defaultPerm)
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

	logger.Printf("Writing %d images...\n", len(refToImage))
	if err := tarball.MultiRefWrite(refToImage, imagesFile); err != nil {
		return "", err
	}
	return imagesFile.Name(), nil
}

// IsIntermediateBundle returns tue only if VerifyIntermediateBundle finds no errors
func IsIntermediateBundle(bundlePath string) bool {
	return VerifyIntermediateBundle(bundlePath) == nil
}

type fileValidations struct {
	filename, format string
	validate         func(io.Reader) error
}

// VerifyIntermediateBundle returns true if the path points to an uncompressed
// tarball with:
//  A hints.yaml YAML file
//  A manifest.json for the images
//  A directory container an unpacked chart directory with valid YAMLs Chart.yaml & values.yaml
func VerifyIntermediateBundle(bundlePath string) error {
	validations := []fileValidations{
		{filename: "hints.yaml", format: "YAML", validate: validateYAML},
		{filename: originalChart + "/Chart.yaml", format: "YAML", validate: validateYAML},
		{filename: originalChart + "/values.yaml", format: "YAML", validate: validateYAML},
		{filename: imagesTar, format: "TAR", validate: validateTAR},
	}
	for _, fv := range validations {
		r, err := openFromTar(bundlePath, fv.filename)
		if err != nil {
			return fmt.Errorf("failed to open file %s from tar: %w", fv.filename, err)
		}
		defer r.Close()
		if err := fv.validate(r); err != nil {
			return fmt.Errorf("%w: %s is not valid %s: %v",
				ErrNotIntermediateBundle, fv.filename, fv.format, err)
		}
	}
	return nil
}

type intermediateBundle struct {
	bundlePath string
}

func openBundle(bundlePath string) (*intermediateBundle, error) {
	if err := VerifyIntermediateBundle(bundlePath); err != nil {
		return nil, err
	}
	return &intermediateBundle{bundlePath}, nil
}

func (ib *intermediateBundle) ExtractChartTo(dir string) error {
	err := untar(ib.bundlePath, originalChart, dir)
	if err != nil {
		return fmt.Errorf("failed to untar chart from bundle %s into %s: %w",
			ib.bundlePath, dir, err)
	}
	return nil
}

func (ib *intermediateBundle) LoadHints(log Logger) ([]byte, error) {
	r, err := openFromTar(ib.bundlePath, IntermediateBundleHintsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract %s from bundle at %s: %w",
			IntermediateBundleHintsFilename, ib.bundlePath, err)
	}
	return io.ReadAll(r)
}

func refToTag(imageRef name.Reference) (name.Tag, error) {
	// more often that not an name.Reference is actually backed by a name.Tag
	if tag, ok := (imageRef).(name.Tag); ok {
		return tag, nil
	}
	return name.NewTag(fmt.Sprintf("%s:%s", imageRef.Context().Name(), imageRef.Identifier()))
}

func (ib *intermediateBundle) LoadImage(imageRef name.Reference) (v1.Image, string, error) {
	tag, err := refToTag(imageRef)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make tag from %s: %w", imageRef.Name(), err)
	}
	image, err := tarball.Image(newTarInTarOpener(ib.bundlePath, imagesTar), &tag)
	if err != nil {
		return nil, "", fmt.Errorf("failed to export image %s from tarball %s: %w", tag.Name(), ib.bundlePath, err)
	}
	digest, err := image.Digest()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get image digest for %s: %w", tag.Name(), err)
	}
	return image, digest.String(), nil
}

func validateYAML(r io.Reader) error {
	yamlContents, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var data interface{}
	return yaml.Unmarshal(yamlContents, &data)
}

func validateTAR(r io.Reader) error {
	tr := tar.NewReader(r)
	_, err := tr.Next()
	return err
}
