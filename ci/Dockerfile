FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/photon:4.0

WORKDIR /

# Install required packages
RUN yum -y install build-essential coreutils gawk gcc git go jq make
ENV PATH="/root/go/bin:${PATH}"

# Install go packages used for building and testing
RUN go get -u golang.org/x/tools/cmd/goimports \
    github.com/maxbrunsfeld/counterfeiter/v6 \
    github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
