FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/photon:4.0
LABEL description="relok8s"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"

WORKDIR /

RUN yum -y install diffutils jq

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
COPY build/relok8s-linux /usr/local/bin/relok8s
ENTRYPOINT ["/usr/local/bin/relok8s"]
