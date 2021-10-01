#!/bin/bash

set -euxo pipefail

echo "Docker clone the certificate"
ca_dir=/etc/docker/certs.d/local-registry.io/
mkdir -p "${ca_dir}"
cp "${SSL_CERT_FILE}" "${ca_dir}/ca.crt"

echo "Docker auto-login"
/docker-login.sh local-registry.io "${USERNAME}" "${PASSWORD}"

echo "Test a relok8s move requiring authentication..."
relok8s chart move chart/wordpress-chart -y --image-patterns chart/image-hints.yaml \
  --registry local-registry.io --repo-prefix relocated/local-example

echo "Success"
