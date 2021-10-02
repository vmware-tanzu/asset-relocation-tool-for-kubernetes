#!/bin/bash
# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
set -euxo pipefail

cat <<EOF > .env
PASSWORD=test-password-$(date +%s)
UID="$(id -u)"
GID="$(id -g)"
EOF

# First prepare the password authentication and the certificate
mkdir -p data/auth data/certs
docker-compose up pwdgen mkcert
# Then run the tests using the generated data
docker-compose up -d test-relok8s

docker-compose logs -f test-relok8s

msg="FAIL"
status=1
if docker-compose ps |grep test-relok8s | grep -s 'Exit 0'; then
  msg="PASS"
  status=0
fi

docker-compose stop
docker-compose down

echo "${msg}"
exit ${status}