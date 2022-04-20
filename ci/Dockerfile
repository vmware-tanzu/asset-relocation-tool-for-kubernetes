# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/ubuntu

WORKDIR /

ARG GO_VERSION=1.18.1

# Install required packages
RUN apt-get update && apt-get install -y curl gcc git jq make wget
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzvf go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH="/root/go/bin:${PATH}:/usr/local/go/bin"

# Install go packages used for building and testing
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@latest && \
    go install github.com/onsi/ginkgo/ginkgo@latest

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
