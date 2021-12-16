// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"errors"
	"fmt"
	"testing"

	semver3 "github.com/Masterminds/semver/v3"
)

var nextSemVerTests = []struct {
	previous, desired string
	expected          string
}{
	{
		previous: "0.0.0",
		desired:  "",
		expected: "0.0.1",
	},
	{
		previous: "0.0.1",
		desired:  "",
		expected: "0.0.2",
	},
	{
		previous: "0.0.1",
		desired:  "0.1.0",
		expected: "0.1.0",
	},
	{
		previous: "0.9.1",
		desired:  "1.0.0",
		expected: "1.0.0",
	},
	{
		previous: "0.9.9",
		desired:  "",
		expected: "0.9.10",
	},
	{
		previous: "0.9.9-rc",
		desired:  "",
		expected: "0.9.9",
	},
}

func version(v string) *semver3.Version {
	if v == "" {
		return nil
	}
	return semver3.MustParse(v)
}

func TestNextSemVer(t *testing.T) {
	for _, tc := range nextSemVerTests {
		testName := fmt.Sprintf("(%q, %q)->%q", tc.previous, tc.desired, tc.expected)
		t.Run(testName, func(t *testing.T) {
			next, err := nextSemVer(version(tc.previous), version(tc.desired))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			got := next.String()
			want := tc.expected
			if got != want {
				t.Fatalf("want %s got %s", want, got)
			}
		})
	}
}

var mainEIssues = []struct {
	args     []string
	expected error
}{
	{
		args:     []string{"next-semver", "0.0.0"},
		expected: nil,
	},
	{
		args:     []string{"next-semver", "0.0.1", "0.0.0"},
		expected: ErrBadVersionBump,
	},
	{
		args:     []string{"next-semver", "0.0.1", "0.1.0"},
		expected: nil,
	},
	{
		args:     []string{"next-semver", "1.0.1", "0.9.0"},
		expected: ErrBadVersionBump,
	},
	{
		args:     []string{"next-semver", "blah1.0.1", "0.9.0"},
		expected: semver3.ErrInvalidSemVer,
	},
	{
		args:     []string{"next-semver", "1.0.1", "blah0.9.0"},
		expected: semver3.ErrInvalidSemVer,
	},
	{
		args:     []string{"next-semver", "1.0.1", "0.9.0.1"},
		expected: semver3.ErrInvalidSemVer,
	},
	{
		args:     []string{"next-semver"},
		expected: ErrBadInputs,
	},
}

func TestMainE(t *testing.T) {
	for _, tc := range mainEIssues {
		testName := fmt.Sprintf("%v->%v", tc.args, tc.expected)
		t.Run(testName, func(t *testing.T) {
			got := mainE(tc.args...)
			want := tc.expected
			if !errors.Is(got, want) {
				t.Fatalf("want %v got %v", want, got)
			}
		})
	}
}
