#!/bin/bash

set -euo pipefail

desired_version=${1:-}

old_version=$(cat version)
(cd next-semver && go build .)
version=$(./next-semver/next-semver "${old_version}" "${desired_version}")
echo "${version}" > version
git add version
git commit -s -m "Bump version to ${version}"
git push
git tag "v${version}" && git push origin "v${version}"

