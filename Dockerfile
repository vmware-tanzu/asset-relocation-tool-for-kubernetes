# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
FROM golang:1.16 as builder

COPY . /asset-relocation-tool-for-kubernetes/
ENV GOPATH=
ENV PATH="${PATH}:/root/go/bin"
WORKDIR /asset-relocation-tool-for-kubernetes/
RUN make test && make build

FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/photon:4.0
LABEL description="Asset Relocation Tool for Kubernetes"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

WORKDIR /

RUN yum -y install diffutils jq

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
COPY --from=builder /asset-relocation-tool-for-kubernetes/build/relok8s /usr/local/bin/relok8s
ENTRYPOINT ["/usr/local/bin/relok8s"]

