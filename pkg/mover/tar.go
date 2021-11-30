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
	tarWriterDisabled bool
}

func (tfw *tarFileWriter) Close() error {
	if err := tfw.Writer.Close(); err != nil {
		return err
	}
	return tfw.WriteCloser.Close()
}

func (tfw *tarFileWriter) ContinueWithRawWriter() io.WriteCloser {
	// this flush here allows for another tar writer to continue on the stream
	tfw.Writer.Flush()
	tfw.tarWriterDisabled = false
	return tfw.WriteCloser
}

func (tfw *tarFileWriter) WriteFile(name string, data []byte, permission fs.FileMode) error {
	if tfw.tarWriterDisabled {
		return fmt.Errorf("No more tar writing operations allowed after ContinueWithRawWriter")
	}
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

func newTarFileWriter(tarFile string) (*tarFileWriter, error) {
	f, err := os.Create(tarFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar file %s: %v", tarFile, err)
	}
	return &tarFileWriter{Writer: tar.NewWriter(f), WriteCloser: f}, nil
}
