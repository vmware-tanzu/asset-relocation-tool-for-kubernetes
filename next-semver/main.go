// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"errors"
	"fmt"
	"os"

	semver3 "github.com/Masterminds/semver/v3"
)

var (
	// ErrBadInputs when the number of CLI inputs are wrong
	ErrBadInputs = errors.New("bad inputs")

	// ErrBadVersionBump when the desired version is not after the previous one
	ErrBadVersionBump = errors.New("bad version bump")
)

func nextSemVer(previous, desired *semver3.Version) (*semver3.Version, error) {
	if desired == nil {
		next := previous.IncPatch()
		return &next, nil
	}
	if !desired.GreaterThan(previous) {
		return nil, fmt.Errorf("%w: %v is not after %v", ErrBadVersionBump, desired, previous)
	}
	return desired, nil
}

func mainE(args ...string) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("%w\nusage: %s {semver-version} [{desired_semver}]\n$ %s 1.2.3",
			ErrBadInputs, args[0], args[0])
	}
	previous, err := semver3.NewVersion(args[1])
	if err != nil {
		return fmt.Errorf("failed to parse previous version %q: %w", args[1], err)
	}
	var desired *semver3.Version
	if len(args) > 2 && args[2] != "" {
		desired, err = semver3.NewVersion(args[2])
		if err != nil {
			return fmt.Errorf("failed to parse desired version %q: %w", args[2], err)
		}
	}
	next, err := nextSemVer(previous, desired)
	if err != nil {
		return err
	}
	fmt.Println(next)
	return nil
}

func main() {
	if err := mainE(os.Args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
