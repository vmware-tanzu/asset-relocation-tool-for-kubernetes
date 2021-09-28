#!/bin/bash
set -euxo pipefail

cat <<EOF > .env
USERNAME=testuser
PASSWORD=testpassword-$(date +%s)
EOF

docker-compose up -d --remove-orphans

docker-compose logs -f test-relok8s

msg="FAIL"
status=1
if docker-compose ps |grep test-relok8s | grep -s 'Exit 0'; then
  msg="PASS"
  status=0
fi

docker-compose down || true

echo "${msg}"
exit ${status}
