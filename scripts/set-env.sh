#!/bin/bash

GO111MODULE=on
GOPRIVATE=gitlab.eng.vmware.com

echo REGISTRY_AUTH
HARBOR_CREDS=$(vault read /runway_concourse/tanzu-isv-engineering/private-harbor -format=json | jq -r .data)
REGISTRY_AUTH="harbor-repo.vmware.com="$(echo "${HARBOR_CREDS}" | jq -r .username):$(echo "${HARBOR_CREDS}" | jq -r .token)

export GO111MODULE \
    GOPRIVATE \
    REGISTRY_AUTH
