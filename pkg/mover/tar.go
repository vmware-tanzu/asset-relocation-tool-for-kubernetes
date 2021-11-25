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

func (tfw *tarFileWriter) Close() error {
	if err := tfw.Writer.Close(); err != nil {
		return err
	}
	return tfw.WriteCloser.Close()
}

func (tfw *tarFileWriter) RawWriter() io.Writer {
	// this flush here allows for another tar writer to continue on the stream
	tfw.Flush()
	return tfw.WriteCloser
}

func (tfw *tarFileWriter) WriteFile(name string, data []byte, permission fs.FileMode) error {
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

func reopenTarFileWriter(tarFile string) (*tarFileWriter, error) {
	f, err := os.OpenFile(tarFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen %s %s", tarFile, err)
	}
	tarBlockSize := int64(512)
	if _, err = f.Seek(-(2 * tarBlockSize), os.SEEK_END); err != nil {
		log.Fatalln(err)
	}
	return &tarFileWriter{Writer: tar.NewWriter(f), WriteCloser: f}, nil
}
