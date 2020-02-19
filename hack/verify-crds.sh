#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..

DIFFROOT="${SCRIPT_ROOT}/deployments/common/"
TMP_DIFFROOT="${SCRIPT_ROOT}/_tmp/deployments/common/"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

VERSION=v0.2.5
CONTROLLER_GEN_BASENAME=controller-gen-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
CONTROLLER_GEN=${CONTROLLER_GEN_BASENAME}-${VERSION}
test -x "hack/${CONTROLLER_GEN}" || curl -f -L -o "hack/${CONTROLLER_GEN}" "https://github.com/openshift/kubernetes-sigs-controller-tools/releases/download/${VERSION}/${CONTROLLER_GEN_BASENAME}"
chmod +x "hack/${CONTROLLER_GEN}"

hack/${CONTROLLER_GEN} schemapatch:manifests=./deployments/common/ paths="./pkg/apis/configuration/..." output:dir=${TMP_DIFFROOT}
echo "diffing ${DIFFROOT} against potentially updated crds"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
if [[ $ret -eq 0 ]]
then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please regenerate crds"
  exit 1
fi
