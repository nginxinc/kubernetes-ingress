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

DOCS_TO_UPDATE_FOLDER=docs-web

if [ $# != 2 ];
then
    echo "Invalid number of arguments" 1>&2
    echo "Usage: $0 ic-version helm-chart-version" 1>&2
    exit 1
fi

ic_version=$1
helm_chart_version=$2

prev_ic_version=$(echo $ic_version | awk -F. '{ printf("%s\\.%s\\.%d", $1, $2, $3-1) }')
prev_helm_chart_version=$(echo $helm_chart_version | awk -F. '{ printf("%s\\.%s\\.%d", $1, $2, $3-1) }')

sed -i "" "s/$prev_ic_version/$ic_version/g" ${FILES_TO_UPDATE_IC_VERSION[*]}
sed -i "" "s/$prev_helm_chart_version/$helm_chart_version/g" ${FILE_TO_UPDATE_HELM_CHART_VERSION[*]}

# update repo CHANGELOG
sed -i "" "1r hack/changelog-template.txt" CHANGELOG.md
sed -i "" -e "s/%%TITLE%%/### $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" CHANGELOG.md

# update docs CHANGELOG
sed -i "" "1r hack/changelog-template.txt" $DOCS_TO_UPDATE_FOLDER/releases.md 
sed -i "" -e "s/%%TITLE%%/## NGINX Ingress Controller $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" $DOCS_TO_UPDATE_FOLDER/releases.md

# update docs
find $DOCS_TO_UPDATE_FOLDER -type f -name "*.md" -exec sed -i "" "s/v$prev_ic_version/v$ic_version/g" {} +
find $DOCS_TO_UPDATE_FOLDER -type f -name "*.rst" -exec sed -i "" "s/v$prev_ic_version/v$ic_version/g" {} +

# update IC version in the technical-specification doc
sed -i "" "s/$prev_ic_version/$ic_version/g" $DOCS_TO_UPDATE_FOLDER/technical-specifications.md 

# update IC version in the building ingress controller doc
sed -i "" "s/$prev_ic_version/$ic_version/g" $DOCS_TO_UPDATE_FOLDER/installation/building-ingress-controller-image.md

# update IC version in the helm doc  
sed -i "" "s/$prev_ic_version/$ic_version/g" $DOCS_TO_UPDATE_FOLDER/installation/installation-with-helm.md
