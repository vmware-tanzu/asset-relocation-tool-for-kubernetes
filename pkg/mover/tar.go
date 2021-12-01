// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type tarFileWriter struct {
	*tar.Writer
	io.WriteCloser
}

func newTarFileWriter(tarFile string) (*tarFileWriter, error) {
	f, err := os.Create(tarFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar file %s: %v", tarFile, err)
	}
	return wrapAsTarFileWriter(f), nil
}

func wrapAsTarFileWriter(wc io.WriteCloser) *tarFileWriter {
	return &tarFileWriter{Writer: tar.NewWriter(wc), WriteCloser: wc}
}

func (tfw *tarFileWriter) Close() error {
	if err := tfw.Writer.Close(); err != nil {
		return err
	}
	return tfw.WriteCloser.Close()
}

func (tfw *tarFileWriter) WriteMemFile(name string, data []byte, permission fs.FileMode) error {
	hdr := &tar.Header{
		Name: name,
		Mode: int64(permission),
		Size: int64(len(data)),
	}
	if err := tfw.WriteHeader(hdr); err != nil {
		log.Fatal(err)
	}
	if _, err := tfw.Writer.Write(data); err != nil {
		return fmt.Errorf("failed to tar %d bytes of data as file %s: %w", len(data), name, err)
	}
	return nil
}

func (tfw *tarFileWriter) WriteIOFile(name string, size int64, r io.Reader, permission fs.FileMode) error {
	hdr := &tar.Header{
		Name: name,
		Mode: int64(permission),
		Size: int64(size),
	}
	if err := tfw.WriteHeader(hdr); err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(tfw.Writer, r); err != nil {
		return fmt.Errorf("failed to tar stream of %d bytes as file %s: %w", size, name, err)
	}
	return nil
}

func untar(tarFile, src, dst string) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("failed to open tar file %s: %w", tarFile, err)
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to untar %s: %w", tarFile, err)
		}
		if src != "" && !strings.HasPrefix(hdr.Name, src) { // skip if not under src
			continue
		}
		path := filepath.Join(dst, strings.TrimPrefix(hdr.Name, src))
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return fmt.Errorf("failed to extract directory %s: %w", path, err)
			}
			continue
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create extraction subdir %s: %w", dir, err)
		}
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", path, err)
		}
		if _, err := io.Copy(f, tr); err != nil {
			return fmt.Errorf("failed extracting file %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed closing extracted file %s: %w", path, err)
		}
	}
}

// tarredFile represents a single file inside a tar. Closing it closes the tar itself.
type tarredFile struct {
	io.Reader
	io.Closer
}

func openFromTar(tarFile, filePath string) (io.ReadCloser, error) {
	f, err := os.Open(tarFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open tar file %s: %w", tarFile, err)
	}
	close := true
	defer func() {
		if close {
			f.Close()
		}
	}()

	tf := tar.NewReader(f)
	for {
		hdr, err := tf.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == filePath {
			if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
				currentDir := filepath.Dir(filePath)
				return openFromTar(tarFile, path.Join(currentDir, hdr.Linkname))
			}
			close = false
			return tarredFile{
				Reader: tf,
				Closer: f,
			}, nil
		}
	}
	return nil, fmt.Errorf("file %s not found in tar", filePath)
}

func newTarInTarOpener(tarFile, tarInTarFile string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return openFromTar(tarFile, tarInTarFile)
	}
}
