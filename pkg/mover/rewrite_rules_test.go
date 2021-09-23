// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
)

var _ = Describe("RewriteRules", func() {
	Context("no rules", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{}
			err := rules.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("valid registry", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				Registry: "projects.vmware.com",
			}
			err := rules.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("valid repository prefix", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				RepositoryPrefix: "myprojects/subfolder",
			}
			err := rules.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("valid registry and repository prefix", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				Registry:         "projects.vmware.com",
				RepositoryPrefix: "myprojects/subfolder",
			}
			err := rules.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("invalid registry", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				Registry: "a host with spaces",
			}
			err := rules.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("registry rule is not valid"))
		})
	})

	Context("invalid repository prefix", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				RepositoryPrefix: "repositories/cannot/contain+plusses",
			}
			err := rules.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("repository prefix is not valid"))
		})
	})

	Context("invalid registry and repository prefix", func() {
		It("returns no error", func() {
			rules := mover.RewriteRules{
				Registry:         "a.domain.with.an.invalid.port:lolwut",
				RepositoryPrefix: "these#symbols&aren't@allowed",
			}
			err := rules.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("registry rule is not valid"))
		})
	})
})
