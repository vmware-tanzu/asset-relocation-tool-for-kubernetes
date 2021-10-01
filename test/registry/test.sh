#!/bin/bash
set -euxo pipefail

cat <<EOF > .env
PASSWORD=test-password-$(date +%s)
EOF

mkdir -p data/auth data/certs
docker-compose up pwdgen mkcert
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