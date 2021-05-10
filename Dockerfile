FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/ubuntu
LABEL description="relok8s"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY build/relok8s-linux /usr/local/bin/relok8s
ENTRYPOINT ["/usr/local/bin/relok8s"]
