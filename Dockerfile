FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/ubuntu
LABEL description="Chart mover"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY build/chart-mover-linux /usr/local/bin/chart-mover
ENTRYPOINT ["/usr/local/bin/chart-mover"]

