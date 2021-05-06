FROM harbor-repo.vmware.com/dockerhub-proxy-cache/amidos/dcind:2.1.0

RUN apk add go git && \
    go get github.com/onsi/ginkgo/ginkgo github.com/onsi/gomega
ENV PATH="/root/go/bin:${PATH}"
