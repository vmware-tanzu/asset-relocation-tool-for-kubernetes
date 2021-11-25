// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

func saveIntermediateBundle(cd *ChartData, tarFile string, log Logger) error {
	tfw, err := newTarFileWriter(tarFile)
	if err != nil {
		return err
	}

	// hints file goes first to be extracted quickly on demand
	log.Printf("Writing %s...\n", HintsFilename)
	if err := tfw.WriteFile(HintsFilename, cd.rawHints, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write %s: %w", HintsFilename, err)
	}

	log.Printf("Writing Helm Chart files at %s/...\n", cd.chart.Metadata.Name)
	if err := tarChart(tfw, cd.chart); err != nil {
		return fmt.Errorf("failed archiving original-chart/: %w", err)
	}
	if err := tfw.Close(); err != nil {
		return fmt.Errorf("failed mid-closing intermediate bundle at %s: %w", tarFile, err)
	}

	tfw, err = reopenTarFileWriter(tarFile)
	if err != nil {
		return err
	}
	if err := packImages(tfw.RawWriter(), cd.imageChanges, log); err != nil {
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

// IsIntermediateBundle returns tue only if VerifyIntermediateBundle finds no errors
func IsIntermediateBundle(bundlePath string) bool {
	return VerifyIntermediateBundle(bundlePath) == nil
}

type fileValidations struct {
	filename, format string
	validate         func(io.Reader) error
}

func baseDir(path string) string {
	separator := string(filepath.Separator)
	dir := filepath.Dir(filepath.Clean(path))
	elements := strings.Split(dir, separator)
	baseDir := elements[0]
	if len(elements) > 1 && baseDir == "" {
		baseDir = separator + elements[1]
	}
	return baseDir
}

// VerifyIntermediateBundle returns true if the path points to an uncompressed
// tarball with:
// - A hints.yaml YAML file
// - A manifest.json for the images
// - A directory container an unpacked chart directory with valid YAMLs Chart.yaml & values.yaml
func VerifyIntermediateBundle(bundlePath string) error {
	chartDir, err := bundleChartDir(bundlePath)
	if err != nil {
		return err
	}
	validations := []fileValidations{
		{filename: "hints.yaml", format: "YAML", validate: validateYAML},
		{filename: "manifest.json", format: "JSON", validate: validateJSON},
		{filename: fmt.Sprintf("%s/Chart.yaml", chartDir), format: "YAML", validate: validateYAML},
		{filename: fmt.Sprintf("%s/values.yaml", chartDir), format: "YAML", validate: validateYAML},
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

func bundleChartDir(bundlePath string) (string, error) {
	chartDir := ""
	err := tarList(bundlePath, func(hdr *tar.Header) error {
		dir := baseDir(hdr.Name)
		if dir != "." && dir != "/" {
			chartDir = dir
			return errEndOfList
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to list tar %s: %w", bundlePath, err)
	}
	if chartDir == "" {
		return "", fmt.Errorf("failed to find the chart folder in tar %s", bundlePath)
	}
	return chartDir, nil
}

func (ib *intermediateBundle) ExtractChartTo(dir string) error {
	chartDir, err := bundleChartDir(ib.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to detect chart directory from bundle %s: %w", ib.bundlePath, err)
	}
	err = untar(ib.bundlePath, chartDir, dir)
	if err != nil {
		return fmt.Errorf("failed to untar chart %s from bundle %s into %s: %w",
			chartDir, ib.bundlePath, dir, err)
	}
	return nil
}

func (ib *intermediateBundle) LoadHints(log Logger) ([]byte, error) {
	r, err := openFromTar(ib.bundlePath, HintsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract %s from bundle at %s: %w",
			HintsFilename, ib.bundlePath, err)
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
	image, err := tarball.ImageFromPath(ib.bundlePath, &tag)
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

func validateJSON(r io.Reader) error {
	jsonContents, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var data interface{}
	return json.Unmarshal(jsonContents, &data)
}
