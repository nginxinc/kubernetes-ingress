#!/usr/bin/env bash

set -o pipefail

ROOTDIR=$(git rev-parse --show-toplevel || echo ".")
TMPDIR=/tmp
HELM_CHART_PATH="${ROOTDIR}/charts/nginx-ingress"
DEPLOYMENT_PATH="${ROOTDIR}/deployments"
DEBUG=${DEBUG:-"false"}

DOCS_TO_UPDATE_FOLDER=${ROOTDIR}/docs/content
FILES_TO_UPDATE_IC_VERSION=(
    "${ROOTDIR}/README.md"
    "${DEPLOYMENT_PATH}/daemon-set/nginx-ingress.yaml"
    "${DEPLOYMENT_PATH}/daemon-set/nginx-plus-ingress.yaml"
    "${DEPLOYMENT_PATH}/deployment/nginx-ingress.yaml"
    "${DEPLOYMENT_PATH}/deployment/nginx-plus-ingress.yaml"
    "${HELM_CHART_PATH}/Chart.yaml"
    "${HELM_CHART_PATH}/README.md"
    "${HELM_CHART_PATH}/values-icp.yaml"
    "${HELM_CHART_PATH}/values-nsm.yaml"
    "${HELM_CHART_PATH}/values-plus.yaml"
    "${HELM_CHART_PATH}/values.yaml"
)
FILE_TO_UPDATE_HELM_CHART_VERSION=(
    "${HELM_CHART_PATH}/Chart.yaml"
    "${HELM_CHART_PATH}/README.md"
)

 usage() {
    echo "Usage: $0 <current_ic_version> <current_helm_chart_version> <new_ic_version> <new_helm_chart_version>"
    exit 1
 }

current_ic_version=$1
current_helm_chart_version=$2
new_ic_version=$3
new_helm_chart_version=$4

if [ -z "${current_ic_version}" ]; then
    usage
fi

if [ -z "${current_helm_chart_version}" ]; then
    usage
fi

if [ -z "${new_ic_version}" ]; then
    usage
fi

if [ -z "${new_helm_chart_version}" ]; then
    usage
fi

escaped_current_ic_version=$(printf '%s' "$current_ic_version" | sed -e 's/\./\\./g');
escaped_current_helm_chart_version=$(printf '%s' "$current_helm_chart_version" | sed -e 's/\./\\./g');

echo "Updating versions: "
echo "ic_version: ${current_ic_version} -> ${new_ic_version}"
echo "helm_chart_version: ${current_helm_chart_version} -> ${new_helm_chart_version}"

regex_ic="s#$escaped_current_ic_version#$new_ic_version#g"
regex_helm="s#$escaped_current_helm_chart_version#$new_helm_chart_version#g"

mv "${HELM_CHART_PATH}/values.schema.json" "${TMPDIR}/"
jq --arg version "${new_ic_version}" \
    '.properties.controller.properties.image.properties.tag.default = $version | .properties.controller.properties.image.properties.tag.examples[0] = $version | .properties.controller.examples[0].image.tag = $version | .properties.controller.properties.image.examples[0].tag = $version | .examples[0].controller.image.tag = $version' \
    ${TMPDIR}/values.schema.json \
    > "${HELM_CHART_PATH}/values.schema.json"
rc=$?
if [ $rc -ne 0 ]; then
    echo "ERROR: failed updating ic_version in values.schema.json"
    mv "${TMPDIR}/values.schema.json" "${HELM_CHART_PATH}/values.schema.json"
    exit 2
fi

# update helm chart & deployment files with IC version
for i in "${FILES_TO_UPDATE_IC_VERSION[@]}"; do
    if [ "${DEBUG}" != "false" ]; then
        echo "Processing ${i}"
    fi
    file_name=$(basename "${i}")
    mv "${i}" "${TMPDIR}/${file_name}"
    cat "${TMPDIR}/${file_name}" | sed -e "$regex_ic" > "${i}"
    if [ $? -ne 0 ]; then
        echo "ERROR: failed processing ${i}"
        mv "${TMPDIR}/${file_name}" "${i}"
        exit 2
    fi
done

# update helm chart files with helm chart version
for i in "${FILE_TO_UPDATE_HELM_CHART_VERSION[@]}"; do
    if [ "${DEBUG}" != "false" ]; then
        echo "Processing ${i}"
    fi
    file_name=$(basename "${i}")
    mv "${i}" "${TMPDIR}/${file_name}"
    cat "${TMPDIR}/${file_name}" | sed -e "$regex_helm" > "${i}"
    if [ $? -ne 0 ]; then
        echo "ERROR: failed processing ${i}"
        mv "${TMPDIR}/${file_name}" "${i}"
        exit 2
    fi
done

# update docs with new versions
docs_files=$(find "${DOCS_TO_UPDATE_FOLDER}" -type f -name "*.md" ! -name releases.md ! -name CHANGELOG.md)
for i in ${docs_files}; do
    if [ "${DEBUG}" != "false" ]; then
        echo "Processing ${i}"
    fi
    file_name=$(basename "${i}")
    mv "${i}" "${TMPDIR}/${file_name}"
    cat "${TMPDIR}/${file_name}" | sed -e "$regex_ic" | sed -e "$regex_helm" > "${i}"
    if [ $? -ne 0 ]; then
        echo "ERROR: failed processing ${i}"
        mv "${TMPDIR}/${file_name}" "${i}"
        exit 2
    fi
done
