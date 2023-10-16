#!/usr/bin/env bash

# Updates the files required for a new release. Run this script in the release branch.
#
# Usage:
# hack/prepare-release.sh ic-version helm-chart-version
#
# Example:
# hack/prepare-release.sh 3.3.0 1.0.0

DOCS_TO_UPDATE_FOLDER=docs/content

FILES_TO_UPDATE_IC_VERSION=(
    README.md
    deployments/daemon-set/nginx-ingress.yaml
    deployments/daemon-set/nginx-plus-ingress.yaml
    deployments/deployment/nginx-ingress.yaml
    deployments/deployment/nginx-plus-ingress.yaml
    charts/nginx-ingress/Chart.yaml
    charts/nginx-ingress/README.md
    charts/nginx-ingress/values-icp.yaml
    charts/nginx-ingress/values-nsm.yaml
    charts/nginx-ingress/values-plus.yaml
    charts/nginx-ingress/values.yaml
)

FILE_TO_UPDATE_HELM_CHART_VERSION=(
    charts/nginx-ingress/Chart.yaml
    charts/nginx-ingress/README.md
)

if [ $# != 2 ]; then
    echo "Invalid number of arguments" 1>&2
    echo "Usage: $0 ic-version helm-chart-version" 1>&2
    exit 1
fi

ic_version=$1
helm_chart_version=$2

current_ic_version=$(yq '.appVersion' <deployments/helm-chart/Chart.yaml)
current_helm_chart_version=$(yq '.version' <deployments/helm-chart/Chart.yaml)

sed -i "" "s/$current_ic_version/$ic_version/g" ${FILES_TO_UPDATE_IC_VERSION[*]}
sed -i "" "s/$current_helm_chart_version/$helm_chart_version/g" ${FILE_TO_UPDATE_HELM_CHART_VERSION[*]}
find $DOCS_TO_UPDATE_FOLDER -type f -name "*.md" ! -name releases.md ! -name CHANGELOG.md -exec sed -i "" "s/$current_ic_version/$ic_version/g" {} +

# update CHANGELOGs
sed -i "" "8r hack/changelog-template.txt" $DOCS_TO_UPDATE_FOLDER/releases.md
sed -i "" -e "s/%%TITLE%%/## $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" $DOCS_TO_UPDATE_FOLDER/releases.md CHANGELOG.md

# copy the helm chart README to the docs
{
    sed -n '1,10p' docs/content/installation/installation-with-helm.md
    sed -n '3,$p' charts/nginx-ingress/README.md
} >file2.new && mv file2.new docs/content/installation/installation-with-helm.md

sed -i '' '/^|Parameter | Description | Default |/i\
{{% table %}}\
' docs/content/installation/installation-with-helm.md

line_number=$(grep -n -e "|" docs/content/installation/installation-with-helm.md | tail -n 1 | cut -d : -f 1)

sed -i '' "${line_number}a\\
{{% /table %}}
" docs/content/installation/installation-with-helm.md
