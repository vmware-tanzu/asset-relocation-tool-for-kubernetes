// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
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
