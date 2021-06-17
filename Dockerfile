FROM harbor-repo.vmware.com/dockerhub-proxy-cache/amidos/dcind:2.1.0 AS dcind
FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/golang:1.16-buster
LABEL description="relok8s"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

WORKDIR /

ENV DOCKER_VERSION=18.09.9 \
    DOCKER_COMPOSE_VERSION=1.29.1

ARG CRYPTOGRAPHY_DONT_BUILD_RUST=1

# Install Docker and Docker Compose
RUN apt-get update && apt-get install -y \
        curl \
        iptables \
        python3-pip && \
    rm -rf /var/lib/apt/lists/* && \
    curl https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz | tar zx && \
    mv /docker/* /bin/ && \
    chmod +x /bin/docker* && \
    pip3 install docker-compose==${DOCKER_COMPOSE_VERSION}

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

# Copy functions to start/stop docker daemon
COPY --from=dcind /docker-lib.sh /docker-lib.sh
COPY --from=dcind /entrypoint.sh /entrypoint.sh

COPY build/relok8s-linux /usr/local/bin/relok8s
ENTRYPOINT ["/usr/local/bin/relok8s"]
