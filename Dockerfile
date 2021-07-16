# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/photon:4.0
LABEL description="Asset Relocation Tool for Kubernetes"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

WORKDIR /

RUN yum -y install diffutils jq

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
COPY build/relok8s-linux /usr/local/bin/relok8s
ENTRYPOINT ["/usr/local/bin/relok8s"]
