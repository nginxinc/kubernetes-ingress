#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# download controller-gen binary
VERSION=v0.2.5
CONTROLLER_GEN_BASENAME=controller-gen-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
CONTROLLER_GEN=${CONTROLLER_GEN_BASENAME}-${VERSION}
test -x "hack/${CONTROLLER_GEN}" || curl -f -L -o "hack/${CONTROLLER_GEN}" "https://github.com/openshift/kubernetes-sigs-controller-tools/releases/download/${VERSION}/${CONTROLLER_GEN_BASENAME}"
chmod +x "hack/${CONTROLLER_GEN}"

# regenerate the schemas in the CRD manifests
hack/${CONTROLLER_GEN} schemapatch:manifests=./deployments/common/ paths="./pkg/apis/configuration/..." output:dir=./deployments/common