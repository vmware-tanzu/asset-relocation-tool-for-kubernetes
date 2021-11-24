// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func uname(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		u, err := user.LookupId(strconv.Itoa(int(stat.Uid)))
		if err != nil {
			log.Fatalf("Failed to get username for uid %v: %v", stat.Uid, err)
		}
		return u.Username
	}
	return ""
}

func gname(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		g, err := user.LookupGroupId(strconv.Itoa(int(stat.Gid)))
		if err != nil {
			log.Fatalf("Failed to get username for uid %v: %v", stat.Uid, err)
		}
		return g.Name
	}
	return ""
}

func tarDirectory(rootPath, tarFile string) error {
	f, err := os.Create(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %v", tarFile, err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	defer tw.Close()
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
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
			Uname:   uname(info),
			Gname:   gname(info),
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
			return fmt.Errorf("failed to open source file %s: %w", path, err)
		}
		defer source.Close()
		if _, err := io.Copy(tw, source); err != nil {
			return fmt.Errorf("failed to tar source file %s: %w", path, err)
		}
		return nil
	})
}

func untar(tarFile, targetDir string) error {
	f, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %w", tarFile, err)
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
		path := filepath.Join(targetDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(path, defaultTarPermissions); err != nil {
				return fmt.Errorf("failed to extract directory %s: %w", path, err)
			}
			continue
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
