// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// cachedImage wraps a remote image with a local layer cache at dir
type cachedImage struct {
	v1.Image
	dir string
}

// cachedLayer wraps a remote layer with a local cache at dir
type cachedLayer struct {
	v1.Layer
	dir string
}

// teeLayerDump allows to download a layer while saving it in a cached file
// at the same time.
// Once the teeLayerDump is closed and if the stream has been fully read,
// the local cached file can be used instead of the remote download.
type teeLayerDump struct {
	dump       io.Reader
	layer      *cachedLayer
	f          *os.File
	rc         io.ReadCloser
	compressed bool
	done       bool
}

// NewCachedImage wrap a v1.Image, usually a remote one, so that layer downloads
// are cached at the given dir.
// Once the first download happens from an image layer, the following download
// won't happen and instead the layer is copied from that local directory cache.
func NewCachedImage(img v1.Image, dir string) v1.Image {
	return &cachedImage{Image: img, dir: dir}
}

// wrapAsCachedLayer returns a cachedLayer ensuring the wrapping only happens
// once, as it will not wrap a layer that is already a cachedLayer
//
// The reason for this is that a v1.Image interface can return layers from
// different methods and we do not know if one of them consumes layers from
// another so we do not know when the wrap must happen. We need to do it on
// every method but avoid wrapping the layer several times.
func (img *cachedImage) cachedLayer(layer v1.Layer) cachedLayer {
	if ly, ok := (layer).(cachedLayer); ok {
		return ly // Already a cached layer, so just return it
	}
	return newCachedLayer(layer, img.dir)
}

// Layers implements v1.Image's Layers wrapping layers as cachedLayers
func (img *cachedImage) Layers() ([]v1.Layer, error) {
	layers, err := img.Image.Layers()
	for i, layer := range layers {
		layers[i] = img.cachedLayer(layer)
	}
	return layers, err
}

// LayerByDigest implements v1.Image's LayerByDigest wrapping layers as cachedLayers
func (img *cachedImage) LayerByDigest(hash v1.Hash) (v1.Layer, error) {
	layer, err := img.Image.LayerByDigest(hash)
	return img.cachedLayer(layer), err
}

// LayerByDiffID implements v1.Image's LayerByDiffID wrapping layers as cachedLayers
func (img *cachedImage) LayerByDiffID(hash v1.Hash) (v1.Layer, error) {
	layer, err := img.Image.LayerByDiffID(hash)
	return img.cachedLayer(layer), err
}

// newCachedLayer wraps a v1.Layer, usually a remote one, so that its download
// is cached at the given dir.
func newCachedLayer(layer v1.Layer, dir string) cachedLayer {
	return cachedLayer{Layer: layer, dir: dir}
}

// Uncompressed implements v1.Layer's Uncompressed so that the IO can be read
// from a local cached file, is available.
// Otherwise the original stream is open but it is dumped to a local cached file
// while it is being consumed.
func (ly cachedLayer) Uncompressed() (io.ReadCloser, error) {
	return ly.openLayer(false)
}

// Compressed implements v1.Layer's Compressed so that the IO can be read
// from a local cached file, is available.
// Otherwise the original stream is open but it is dumped to a local cached file
// while it is being consumed.
func (ly cachedLayer) Compressed() (io.ReadCloser, error) {
	return ly.openLayer(true)
}

// openLayer tries to open the local cached file for the layer first,
// if not found, then it opens a download dump to consume the layer at the same
// time it is being saved on a local cache file.
func (ly cachedLayer) openLayer(compressed bool) (io.ReadCloser, error) {
	r, err := ly.openCached(compressed)
	if errors.Is(err, fs.ErrNotExist) {
		open := ly.Layer.Uncompressed
		if compressed {
			open = ly.Layer.Compressed
		}
		r, err := open()
		if err != nil {
			return r, err
		}
		return ly.dumpAndCache(r, compressed)
	}
	return r, err
}

// openCached opens a layer to be read from a local cached file
func (ly *cachedLayer) openCached(compressed bool) (io.ReadCloser, error) {
	digest, err := ly.Digest()
	if err != nil {
		return nil, err
	}
	cachedPath := cachedLayerPath(ly.dir, digest, compressed)
	return os.Open(cachedPath)
}

// dumpAndCache uses a teeLayerDump to download a copy of the layer to a local
// file while the uncompressed stream is being consumed
func (ly *cachedLayer) dumpAndCache(rc io.ReadCloser, compressed bool) (io.ReadCloser, error) {
	f, err := os.CreateTemp(ly.dir, fmt.Sprintf("incoming-layer-*%s", layerExtension(compressed)))
	if err != nil {
		return nil, err
	}
	return &teeLayerDump{
		dump:       io.TeeReader(rc, f),
		layer:      ly,
		f:          f,
		rc:         rc,
		compressed: compressed}, nil
}

// Read implements a io.Reader Read, when EOL is hit, the temporary downloaded
// file is turned into a cached layer
func (tld *teeLayerDump) Read(buf []byte) (int, error) {
	n, err := tld.dump.Read(buf)
	if err == io.EOF && !tld.done {
		err = tld.createCachedLayerFile()
	}
	return n, err
}

// createCachedLayerFile turns a temporary download file into a cached layer file
func (tld *teeLayerDump) createCachedLayerFile() error {
	if err := tld.f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary download file: %w", err)
	}
	digest, err := tld.layer.Digest()
	if err != nil {
		return fmt.Errorf("failed to get digest from layer: %w", err)
	}
	tmpName := tld.f.Name()
	cachedPath := cachedLayerPath(tld.layer.dir, digest, tld.compressed)
	if err := os.Rename(tmpName, cachedPath); err != nil {
		return fmt.Errorf("failed to rename cached layer file: %w", err)
	}
	tld.done = true
	return nil
}

// Close implements a io.Closer completing the caching of a layer download
// to a local file. Only if the layer stream was fully downloaded the local
// copy can be used as a cache, otherwise the downloaded copy is discarded.
func (tld *teeLayerDump) Close() error {
	return tld.rc.Close()
}

// cachedLayerPath returns the full path to a local cached layer file given
// the cache directory and the layer digest and whether or not is compressed
func cachedLayerPath(dir string, digest v1.Hash, compressed bool) string {
	return filepath.Join(dir, cachedLayerFilename(digest, compressed))
}

// cachedLayerFilename builds a cached layer filename from the digest and whether
// or not is compressed
func cachedLayerFilename(digest v1.Hash, compressed bool) string {
	return fmt.Sprintf("%s-%s%s", digest.Algorithm, digest.Hex, layerExtension(compressed))
}

// layerExtension depending on whether or not the layer is compressed
func layerExtension(compressed bool) string {
	if compressed {
		return ".gz" // Seems compressed means gzip-ed in practice
	}
	return ""
}
