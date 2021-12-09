// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
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
	if err := os.MkdirAll(cachedir(), defaultPerm); err != nil {
		return fmt.Errorf("failed to create save cache: %w", err)
	}
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
		refToImage[change.ImageReference] = newCachedImage(change.Image)
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
	return verifyIntermediateBundle(bundlePath) == nil
}

// VerifyIntermediateBundle returns true if the path points to an uncompressed
// tarball with:
//  A hints.yaml YAML file
//  A manifest.json for the images
//  A directory container an unpacked chart directory with valid YAMLs Chart.yaml & values.yaml
func verifyIntermediateBundle(bundlePath string) error {
	expectedFiles := []string{
		"hints.yaml",
		originalChart + "/Chart.yaml",
		originalChart + "/values.yaml",
		imagesTar,
	}
	for _, filename := range expectedFiles {
		r, err := openFromTar(bundlePath, filename)
		if err != nil {
			return fmt.Errorf("failed to open file %s from tar: %w", filename, err)
		}
		defer r.Close()
	}
	return nil
}

type intermediateBundle struct {
	bundlePath string
}

func newBundle(bundlePath string) *intermediateBundle {
	return &intermediateBundle{bundlePath}
}

func (ib *intermediateBundle) extractChartTo(dir string) error {
	err := untar(ib.bundlePath, originalChart, dir)
	if err != nil {
		return fmt.Errorf("failed to untar chart from bundle %s into %s: %w",
			ib.bundlePath, dir, err)
	}
	return nil
}

func (ib *intermediateBundle) loadImageHints(log Logger) ([]byte, error) {
	r, err := openFromTar(ib.bundlePath, IntermediateBundleHintsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract %s from bundle at %s: %w",
			IntermediateBundleHintsFilename, ib.bundlePath, err)
	}
	return io.ReadAll(r)
}

// tag gets us a name.Tag from a name.Reference interface
// for some reason we do have name.Reference, which is accepted as is by the
// saving code using tarball.MultiRefWrite, but for loading the tarball.Image
// requires a name.Tag
func tag(imageRef name.Reference) (name.Tag, error) {
	if tag, ok := (imageRef).(name.Tag); ok {
		return tag, nil
	}
	return name.Tag{}, fmt.Errorf("not sure how to convert imageRef %+#v to a tag", imageRef)
}

func (ib *intermediateBundle) loadImage(imageRef name.Reference) (v1.Image, string, error) {
	tag, err := tag(imageRef)
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

type cachedImage struct {
	v1.Image
}

func newCachedImage(img v1.Image) v1.Image {
	return &cachedImage{Image: img}
}

type cachedLayer struct {
	v1.Layer
}

func newCachedLayer(layer v1.Layer) v1.Layer {
	return cachedLayer{Layer: layer}
}

func wrapAsCachedLayer(layer v1.Layer) v1.Layer {
	if _, ok := (layer).(cachedLayer); ok {
		return layer // Already a cached layer, so just return it
	}
	return &cachedLayer{Layer: layer}
}

func (img *cachedImage) Layers() ([]v1.Layer, error) {
	layers, err := img.Image.Layers()
	for i, layer := range layers {
		layers[i] = wrapAsCachedLayer(layer)
	}
	return layers, err
}

func (img *cachedImage) LayerByDigest(hash v1.Hash) (v1.Layer, error) {
	layer, err := img.LayerByDigest(hash)
	return wrapAsCachedLayer(layer), err
}

func (img *cachedImage) LayerByDiffID(hash v1.Hash) (v1.Layer, error) {
	layer, err := img.LayerByDiffID(hash)
	return wrapAsCachedLayer(layer), err
}

func (ly cachedLayer) Uncompressed() (io.ReadCloser, error) {
	r, err := ly.openCached()
	if err == fs.ErrNotExist {
		r, err = ly.dumpAndCache()
	}
	return r, err
}

func (ly *cachedLayer) openCached() (io.ReadCloser, error) {
	digest, err := ly.Digest()
	if err != nil {
		return nil, err
	}
	return os.Open(cachedDigestFilename(digest))
}

type teeLayerDump struct {
	io.Reader
	digest v1.Hash
	f      *os.File
	r      io.ReadCloser
}

func newTeeLayerDump(digest v1.Hash, f *os.File, r io.ReadCloser) io.ReadCloser {
	return &teeLayerDump{Reader: io.TeeReader(r, f), f: f, r: r}
}

func (tld *teeLayerDump) Close() error {
	if err := tld.r.Close(); err != nil {
		return err
	}
	tmpName := tld.f.Name()
	if err := tld.f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, cachedDigestFilename(tld.digest))
}

func (ly *cachedLayer) dumpAndCache() (io.ReadCloser, error) {
	f, err := os.CreateTemp(cachedir(), "incoming-layer-*")
	if err != nil {
		return nil, err
	}
	in, err := ly.Uncompressed()
	if err != nil {
		return in, err
	}
	digest, err := ly.Digest()
	if err != nil {
		return nil, err
	}
	return newTeeLayerDump(digest, f, in), nil
}

func cachedDigestFilename(digest v1.Hash) string {
	return filepath.Join(cachedir(), fmt.Sprintf("%s-%s", digest.Algorithm, digest.Hex))
}

func cachedir() string {
	return filepath.Join(os.TempDir(), "relok8s-save-cache")
}
