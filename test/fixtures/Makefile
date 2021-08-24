# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause

.PHONY: build clean rebuild
build: wordpress/Chart.yaml
rebuild: clean build
clean:
	rm -rf wordpress-11.0.4.tgz

wordpress/Chart.yaml:
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm pull bitnami/wordpress --version 11.0.4
	tar zxvf wordpress-11.0.4.tgz
