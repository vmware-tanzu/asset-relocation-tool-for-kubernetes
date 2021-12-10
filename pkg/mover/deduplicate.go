// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// deduplicatedFile holds how many occurrences of a file we have and its latest contents
type deduplicatedFile struct {
	times int
	data  []byte
}

// deduplicatedFiles tracks how many times a file appears, the order of first
// appearance and whether or not any duplicates were seen
type deduplicatedFiles struct {
	names        []string // names allows us to respect original order
	files        map[string]deduplicatedFile
	duplications bool
}

// deduplicateChartFiles finds and fixes duplicate files at inchart.
// If duplicates are found an error is returned, the caller might want to
// report and proceed anyway as the output chat is clean of duplicates.
func deduplicateChartFiles(inchart *chart.Chart) (*chart.Chart, error) {
	dedups := newDeduplicateFiles()
	for _, file := range inchart.Raw {
		dedups.add(file.Name, file.Data)
	}
	if dedups.duplications {
		outchart, err := loader.LoadFiles(dedups.bufferedFiles())
		if err != nil {
			return outchart, err
		}
		return outchart, fmt.Errorf("%w:\n%s", ErrDuplicateChartFiles, dedups)
	}
	return inchart, nil
}

func newDeduplicateFiles() *deduplicatedFiles {
	return &deduplicatedFiles{names: []string{}, files: map[string]deduplicatedFile{}}
}

// Add a file name and data and deduplicate it, but respecting list ordering
// and record occurrence times
func (dedups *deduplicatedFiles) add(name string, data []byte) {
	f := dedups.files[name]
	f.times++
	f.data = data
	dedups.files[name] = f
	if f.times > 1 {
		dedups.duplications = true
		return
	}
	dedups.names = append(dedups.names, name)
}

// String dumps a line per duplicate file with more than one occurrence
func (dedups *deduplicatedFiles) String() string {
	sb := &strings.Builder{}
	for _, name := range dedups.names {
		f := dedups.files[name]
		if f.times > 1 {
			fmt.Fprintf(sb, "%s appears %d times", name, f.times)
		}
	}
	return sb.String()
}

// bufferedFiles returns the list of *chartBuffered.File to reconstruct a Chart
// after duplications where found
func (dedups *deduplicatedFiles) bufferedFiles() []*loader.BufferedFile {
	bufFiles := []*loader.BufferedFile{}
	for _, name := range dedups.names {
		f := dedups.files[name]
		bufFiles = append(bufFiles, &loader.BufferedFile{Name: name, Data: f.data})
	}
	return bufFiles
}
