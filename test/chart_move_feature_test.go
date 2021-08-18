// +build feature

package test_test

// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

import (
	"time"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/v2/test"
)

var _ = Describe("relok8s chart move command", func() {
	steps := NewSteps()

	// TODO: Excluding these scenarios until we have a fake docker daemon
	XScenario("directory based helm chart", func() {
		steps.When("running relok8s chart move -y fixtures/wordpress --image-patterns fixtures/wordpress-11.0.4.images.yaml --registry harbor-repo.vmware.com --repo-prefix pwall")
		steps.Then("the original images are pulled")
		steps.And("the command says what images will be pushed")
		steps.And("the command says what changes will be made to the chart")
		steps.And("the new images are pushed")
		steps.And("the changes are made to the chart")
		steps.And("the command exits without error")
	})

	XScenario("tgz based helm chart", func() {
		steps.When("running relok8s chart move -y fixtures/wordpress-11.0.4.tgz --image-patterns fixtures/wordpress-11.0.4.images.yaml --registry harbor-repo.vmware.com --repo-prefix pwall")
		steps.Then("the original images are pulled")
		steps.And("the command says what images will be pushed")
		steps.And("the command says what changes will be made to the chart")
		steps.And("the new images are pushed")
		steps.And("the changes are made to the chart")
		steps.And("the command exits without error")
	})

	XScenario("can abort before changes are made", func() {
		steps.When("running relok8s chart move fixtures/wordpress-11.0.4.tgz --image-patterns fixtures/wordpress-11.0.4.images.yaml --registry harbor-repo.vmware.com --repo-prefix pwall")
		steps.Then("the original images are pulled")
		steps.And("the command says what images will be pushed")
		steps.And("the command says what changes will be made to the chart")
		steps.And("the command prompts for confirmation")
		steps.When("the users says no")
		steps.Then("the command exits without error")
	})

	Scenario("missing helm chart", func() {
		steps.When("running relok8s chart move")
		steps.Then("the command exits with an error")
		steps.And("it says the chart is missing")
		steps.And("it prints the usage")
	})

	Scenario("helm chart does not exist", func() {
		steps.When("running relok8s chart move fixtures/does-not-exist")
		steps.Then("the command exits with an error")
		steps.And("it says the chart does not exist")
		steps.And("it prints the usage")
	})

	Scenario("helm chart is empty directory", func() {
		steps.When("running relok8s chart move fixtures/empty-directory")
		steps.Then("the command exits with an error")
		steps.And("it says the chart is missing a critical file")
		steps.And("it prints the usage")
	})

	Scenario("missing image patterns file", func() {
		steps.When("running relok8s chart move fixtures/wordpress-11.0.4.tgz --repo-prefix jconnor")
		steps.Then("the command exits with an error")
		steps.And("it says the image patterns file is missing")
		steps.And("it prints the usage")
	})

	Scenario("no rules are given", func() {
		steps.When("running relok8s chart move fixtures/wordpress-11.0.4.tgz --image-patterns fixtures/wordpress-11.0.4.images.yaml")
		steps.Then("the command exits with an error")
		steps.And("it says that the rules are missing")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		test.DefineCommonSteps(define)

		define.Then(`^it says the chart is missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: requires a chart argument"))
		})

		define.Then(`^it says the chart does not exist$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: failed to load Helm Chart at \"fixtures/does-not-exist\": stat fixtures/does-not-exist: no such file or directory"))
		})

		define.Then(`^it says the chart is missing a critical file$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: failed to load Helm Chart at \"fixtures/empty-directory\": Chart.yaml file is missing"))
		})

		define.Then(`^it says the image patterns file is missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: image patterns file is required. Please try again with '--image-patterns <image patterns file>'"))
		})

		define.Then(`^it says that the rules are missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix"))
		})

		define.Then(`^the original images are pulled$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/wordpress:5.7.2-debian-10-r0... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/apache-exporter:0.8.0-debian-10-r378... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/bitnami-shell:10... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/mariadb:10.5.10-debian-10-r0... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/mysqld-exporter:0.12.1-debian-10-r430... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/memcached:1.6.9-debian-10-r140... Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pulling docker.io/bitnami/memcached-exporter:0.9.0-debian-10-r26... Done"))
		})

		define.Then(`^the command says what images will be pushed$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Images to be pushed:"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/wordpress:5.7.2-debian-10-r0 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/apache-exporter:0.8.0-debian-10-r378 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/bitnami-shell:10 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/mariadb:10.5.10-debian-10-r0 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/mysqld-exporter:0.12.1-debian-10-r430 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/memcached:1.6.9-debian-10-r140 \\(sha256:[a-z0-9]*\\)"))
			Expect(test.CommandSession.Out).To(Say("  harbor-repo.vmware.com/pwall/memcached-exporter:0.9.0-debian-10-r26 \\(sha256:[a-z0-9]*\\)"))
		})

		define.Then(`^the command says what changes will be made to the chart$`, func() {
			Expect(test.CommandSession.Out).To(Say("Changes written to wordpress/values.yaml:"))
			Expect(test.CommandSession.Out).To(Say(".image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".image.repository: pwall/wordpress"))
			Expect(test.CommandSession.Out).To(Say(".metrics.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".metrics.image.repository: pwall/apache-exporter"))
			Expect(test.CommandSession.Out).To(Say(".volumePermissions.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".volumePermissions.image.repository: pwall/bitnami-shell"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.image.repository: pwall/mariadb"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.metrics.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.metrics.image.repository: pwall/mysqld-exporter"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.volumePermissions.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".mariadb.volumePermissions.image.repository: pwall/bitnami-shell"))
			Expect(test.CommandSession.Out).To(Say(".memcached.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".memcached.image.repository: pwall/memcached"))
			Expect(test.CommandSession.Out).To(Say(".memcached.metrics.image.registry: harbor-repo.vmware.com"))
			Expect(test.CommandSession.Out).To(Say(".memcached.metrics.image.repository: pwall/memcached-exporter"))
		})

		define.Then(`^the new images are pushed$`, func() {
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/wordpress:5.7.2-debian-10-r0...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/apache-exporter:0.8.0-debian-10-r378...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/bitnami-shell:10...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/mariadb:10.5.10-debian-10-r0...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/mysqld-exporter:0.12.1-debian-10-r430...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/memcached:1.6.9-debian-10-r140...Done"))
			Eventually(test.CommandSession.Out, time.Minute).Should(Say("Pushing harbor-repo.vmware.com/pwall/memcached-exporter:0.9.0-debian-10-r26...Done"))
		})

		define.Then(`^the changes are made to the chart$`, func() {
			// TODO: Not yet written
			Expect(1).To(Equal(1))
		})
	})
})
