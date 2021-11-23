// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// copyRecursive copies a source to a destination folder recursively.
// The target folder must exist already.
func copyRecursive(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		destPath := filepath.Join(destination, path[len(source):])
		if path == source {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(destPath, defaultTarPermissions)
		}
		return copyFile(path, destPath)
	})
}

// copyFile copies a source file to a destination filename.
// All contents of the file are duplicated, it does not do no hard links.
func copyFile(source, destination string) error {
	sourceFileStat, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", source)
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
