#!/bin/sh
set -euxo

echo "Generating self certificates..."
mkdir -p /certs
(cd /certs; /mkcert local-registry)

echo "Wait for password..."
while [ ! -f /auth/htpasswd ]; do sleep 1; done

echo "Launching registry..."
/entrypoint.sh /etc/docker/registry/config.yml
