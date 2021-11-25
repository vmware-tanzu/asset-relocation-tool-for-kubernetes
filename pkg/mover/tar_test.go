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
	"sort"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testFile struct {
	path, data string
}

var testDirContents = []testFile{
	{path: "file-at-root.txt", data: "somedata in the root"},
	{path: ".hiddenfile", data: "somehiddenfile data"},
	{path: "emptyfile"},
	{path: "dir1/somefile", data: "more data"},
	{path: "dir1/dir2/somedeepfile", data: "deep data"},
}

func newDir(label string) string {
	dir, err := os.MkdirTemp("", label+"-*")
	if err != nil {
		log.Fatalf("failed to create test temp dir for %s: %v", label, err)
	}
	return dir
}

func tarTestFiles(tarFile string, testFiles []testFile) error {
	tfw, err := newTarFileWriter(tarFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file %s: %w", tarFile, err)
	}
	defer tfw.Close()
	for _, testFile := range testFiles {
		if err := tfw.WriteFile(testFile.path, []byte(testFile.data), os.ModePerm); err != nil {
			log.Fatalf("failed to create test file %s: %v", testFile.path, err)
		}
	}
	return nil
}

func newTarFilePath() string {
	return filepath.Join(newDir("test-tar-dir"), "test.tar")
}

func lsRecursive(dir string) []string {
	paths := []string{}
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(path, dir+"/"))
		return nil
	})
	if err != nil {
		log.Fatalf("failed to list recursively %s: %v", dir, err)
	}
	sort.Strings(paths)
	return paths
}

func lsTar(tarFile string) []string {
	paths := []string{}
	f, err := os.Open(tarFile)
	if err != nil {
		log.Fatalf("failed to dumpTar %s: %v", tarFile, err)
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF || hdr == nil {
			break
		}
		paths = append(paths, hdr.Name)
	}
	sort.Strings(paths)
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
			tarFile := newTarFilePath()
			err := tarTestFiles(tarFile, testDirContents)
			Expect(err).ToNot(HaveOccurred())
			targetDir := newDir("test-target")
			err = untar(tarFile, "", targetDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(lsRecursive(targetDir)).To(Equal(lsTar(tarFile)))
			cleanup(targetDir, filepath.Dir(tarFile))
		})
	})
})
