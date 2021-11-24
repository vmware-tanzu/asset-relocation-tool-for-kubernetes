// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testDirContents = []struct {
	path, data string
	dir        bool
}{
	{path: "file-at-root.txt", data: "somedata in the root"},
	{path: ".hiddenfile", data: "somehiddenfile data"},
	{path: "emptyfile"},
	{path: "dir1/somefile", data: "more data"},
	{path: "dir1/dir2/somedeepfile", data: "deep data"},
	{path: "emptydir/", dir: true},
}

func mkdirAll(path string) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(path, defaultTarPermissions); err != nil {
		log.Fatalf("failed to create test dir %s: %v", path, err)
	}
}

func newDir(label string) string {
	dir, err := os.MkdirTemp("", label+"-*")
	if err != nil {
		log.Fatalf("failed to create test temp dir for %s: %v", label, err)
	}
	return dir
}

func newTestDir() string {
	dir := newDir("testDir")
	for _, entry := range testDirContents {
		fullpath := filepath.Join(dir, entry.path)
		if entry.dir {
			mkdirAll(fullpath)
			continue
		}
		mkdirAll(filepath.Dir(fullpath))
		err := os.WriteFile(fullpath, []byte(entry.data), defaultTarPermissions)
		if err != nil {
			log.Fatalf("failed to create test file %s: %v", fullpath, err)
		}
	}
	return dir
}

func newTarFilePath() string {
	return filepath.Join(newDir("test-tar-dir"), "test.tar")
}

func lsRecursive(dir string) []string {
	paths := []string{}
	err := filepath.Walk(dir, func(path string, _ fs.FileInfo, err error) error {
		if path == dir {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(path, dir+"/"))
		return nil
	})
	if err != nil {
		log.Fatalf("failed to ls recursive on %s: %v", dir, err)
	}
	return paths
}

func cleanup(dirs ...string) {
	for _, dir := range dirs {
		err := os.RemoveAll((dir))
		if err != nil {
			log.Fatalf("failed to cleanup %s: %v", dir, err)
		}
	}
}

var _ = Describe("Tar", func() {
	Context("directory", func() {
		It("tar and untar reproduces original files", func() {
			dir := newTestDir()
			tarFile := newTarFilePath()
			err := tarDirectory(dir, tarFile)
			Expect(err).ToNot(HaveOccurred())
			targetDir := newDir("test-target")
			err = untar(tarFile, targetDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(lsRecursive(targetDir)).To(Equal(lsRecursive(dir)))
			cleanup()
		})
	})
})
