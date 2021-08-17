// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chart", func() {
	Describe("ParseOutputFlag", func() {
		It("works with default out flag", func() {
			got, err := parseOutputFlag(output)
			want := "./%s-%s.relocated.tgz"
			Expect(got).To(Equal(want))
			Expect(err).To(BeNil())
		})
		It("rejects out flag without wildcard *", func() {
			_, err := parseOutputFlag("nowildcardhere.tgz")
			Expect(err).Should(MatchError(errMissingOutPlaceHolder))
		})
		It("rejects out flag without proper extension", func() {
			_, err := parseOutputFlag("*-wildcardhere")
			Expect(err).Should(MatchError(errBadExtension))
		})
		It("accepts out flag with wildcard", func() {
			got, err := parseOutputFlag("*-wildcardhere.tgz")
			Expect(got).To(Equal("%s-%s-wildcardhere.tgz"))
			Expect(err).To(BeNil())
		})
	})

	Describe("TargetOutput", func() {
		It("works with default out flag", func() {
			outFmt := "./%s-%s.relocated.tgz"
			target := targetOutput("path", outFmt, "my-chart", "0.1")
			Expect(target).To(Equal("path/my-chart-0.1.relocated.tgz"))
		})
		It("builds custom out input as expected", func() {
			target := targetOutput("path", "%s-%s-wildcardhere.tgz", "my-chart", "0.1")
			Expect(target).To(Equal("path/my-chart-0.1-wildcardhere.tgz"))
		})
	})
})
