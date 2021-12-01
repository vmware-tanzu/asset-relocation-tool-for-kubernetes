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
	return &tarFileWriter{Writer: tar.NewWriter(f), WriteCloser: f}, nil
}

func (tfw *tarFileWriter) Close() error {
	if err := tfw.Writer.Close(); err != nil {
		return err
	}
	return tfw.WriteCloser.Close()
}

func (tfw *tarFileWriter) WriteMemoryFile(name string, data []byte, permission fs.FileMode) error {
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

func (tfw *tarFileWriter) WriteFSFile(fsys fs.FS, name string, permission fs.FileMode) error {
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", name, err)
	}
	mode := int64(permission)
	if permission == 0 {
		mode = int64(info.Mode())
	}
	hdr := &tar.Header{
		Name: name,
		Mode: mode,
		Size: info.Size(),
	}
	if err := tfw.WriteHeader(hdr); err != nil {
		log.Fatal(err)
	}
	f, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", name, err)
	}
	defer f.Close()
	if _, err := io.Copy(tfw.Writer, f); err != nil {
		return fmt.Errorf("failed to tar stream of %d bytes as file %s: %w", info.Size(), name, err)
	}
	return nil
}
