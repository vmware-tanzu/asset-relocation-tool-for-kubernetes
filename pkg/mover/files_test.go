// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing/fstest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testfile struct {
	contents string
	isDir    bool
}

var SampleFolder = map[string]testfile{
	"file1":              {contents: "file 1 contents"},
	"dir1":               {isDir: true},
	"dir1/file2":         {contents: "file 2 contents"},
	"dir1/dir2":          {isDir: true},
	"dir1/dir2/file3":    {contents: "file 3 contents"},
	"dir1/dir2/emptydir": {isDir: true},
}

func mustMkdirTemp(pattern string) string {
	f, err := os.MkdirTemp("", pattern)
	if err != nil {
		log.Fatalf("failed to create temp folder %q: %v", pattern, err)
	}
	return f
}

func mustMakeFolder(folder string) {
	err := os.MkdirAll(folder, 0740)
	if err != nil {
		log.Fatalf("failed to create folder %q: %v", folder, err)
	}
}

func populate(files map[string]testfile) string {
	dst := mustMkdirTemp("original-*")
	for name, file := range files {
		if file.isDir {
			mustMakeFolder(filepath.Join(dst, name))
			continue
		}
		folder := filepath.Join(dst, filepath.Dir(name))
		mustMakeFolder(folder)
		filename := filepath.Join(dst, name)
		err := os.WriteFile(filename, []byte(file.contents), 0640)
		if err != nil {
			log.Fatalf("failed to create test file %q: %v", filename, err)
		}
	}
	return dst
}

func expectedFilesFrom(files map[string]testfile) []string {
	filenames := make([]string, 0, len(files))
	for name := range files {
		filenames = append(filenames, name)
	}
	fmt.Println("filenames:")
	fmt.Println(filenames)
	fmt.Println("size:", len(filenames))
	return filenames
}

func cleanup(folders ...string) {
	for _, folder := range folders {
		os.RemoveAll(folder)
	}
}

var _ = Describe("Files", func() {
	Context("recursive copy of folders and files", func() {
		It("creates an exact duplicate of files and folders", func() {
			src := populate(SampleFolder)
			dst := mustMkdirTemp("dest-*")
			err := copyRecursive(src, dst)
			Expect(err).NotTo(HaveOccurred())
			err = fstest.TestFS(os.DirFS(dst), expectedFilesFrom(SampleFolder)...)
			Expect(err).NotTo(HaveOccurred())
			cleanup(dst, src)
		})
	})
})
