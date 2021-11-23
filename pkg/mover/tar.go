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
	"path/filepath"
	"strings"
)

func tarDirectory(rootPath, tarFile string) error {
	f, err := os.Create(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %v", tarFile, err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	defer tw.Close()
	return filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("incoming walk error: %v", err)
		}
		if path == rootPath {
			return nil
		}
		name := strings.TrimPrefix(path, rootPath+"/")
		hdr := &tar.Header{
			Name:    name,
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		if info.IsDir() {
			hdr.Typeflag = tar.TypeDir
			return tw.WriteHeader(hdr)
		}
		hdr.Size = info.Size()
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}
		source, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %v", path, err)
		}
		defer source.Close()
		if _, err := io.Copy(tw, source); err != nil {
			return fmt.Errorf("failed to tar source file %s: %v", path, err)
		}
		return nil
	})
}
