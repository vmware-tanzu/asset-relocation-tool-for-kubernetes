#!/bin/bash

set -euxo pipefail

echo "Wait for cert file at ${SSL_CERT_FILE}..."
while [ ! -f "${SSL_CERT_FILE}" ]; do sleep 1; done
mkdir -p /etc/docker/certs.d/local-registry/
cp "${SSL_CERT_FILE}" /etc/docker/certs.d/local-registry/ca.crt

echo "Docker auto-login"
/docker-login.sh local-registry:443 "${USERNAME}" "${PASSWORD}"

echo "Test a relok8s move requiring authentication..."
relok8s chart move chart/wordpress-chart -y --image-patterns chart/image-hints.yaml \
  --registry local-registry:443 --repo-prefix relocated/local-example

echo "Success"
