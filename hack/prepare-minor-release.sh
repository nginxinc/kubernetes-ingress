#!/usr/bin/env bash

# Updates the files required for a new minor release. Run this script in the release branch.
#
# Usage:
# hack/prepare-minor-release.sh ic-version helm-chart-version
#
# Example:
# hack/prepare-minor-release.sh 1.5.5 0.3.5

FILES_TO_UPDATE_IC_VERSION=(
    Makefile
    README.md
    build/README.md
    deployments/daemon-set/nginx-ingress.yaml
    deployments/daemon-set/nginx-plus-ingress.yaml
    deployments/deployment/nginx-ingress.yaml
    deployments/deployment/nginx-plus-ingress.yaml
    deployments/helm-chart/Chart.yaml
    deployments/helm-chart/README.md
    deployments/helm-chart/values-icp.yaml
    deployments/helm-chart/values-plus.yaml
    deployments/helm-chart/values.yaml
)

FILE_TO_UPDATE_HELM_CHART_VERSION=( deployments/helm-chart/Chart.yaml )

if [ $# != 2 ];
then
    echo "Invalid number of arguments" 1>&2
    echo "Usage: $0 ic-version helm-chart-version" 1>&2
    exit 1
fi

ic_version=$1
helm_chart_version=$2

prev_ic_version=$(echo $ic_version | awk -F. '{ printf("%s.%s.%d", $1, $2, $3-1) }')
prev_helm_chart_version=$(echo $helm_chart_version | awk -F. '{ printf("%s.%s.%d", $1, $2, $3-1) }')

sed -i "" "s/$prev_ic_version/$ic_version/g" ${FILES_TO_UPDATE_IC_VERSION[*]}
sed -i "" "s/$prev_helm_chart_version/$helm_chart_version/g" ${FILE_TO_UPDATE_HELM_CHART_VERSION[*]}

sed -i "" "1r hack/changelog-template.txt" CHANGELOG.md
sed -i "" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" CHANGELOG.md
