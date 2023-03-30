#!/usr/bin/env bash

# Updates the files required for a new release. Run this script in the release branch. Use it from
# hack/prepare-major-release.sh or hack/prepare-minor-release.sh
#
# Usage:
# hack/common-release-prep.sh prev_ic_version ic-version prev_helm_chart_version helm-chart-version
#
# Example:
# hack/prepare-major-release.sh 1.12.1 1.13.0 0.10.1 0.11.0

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
    deployments/helm-chart/values-nsm.yaml
    deployments/helm-chart/values-plus.yaml
    deployments/helm-chart/values.yaml
)

FILE_TO_UPDATE_HELM_CHART_VERSION=(
    deployments/helm-chart/Chart.yaml
    deployments/helm-chart/README.md
)

DOCS_TO_UPDATE_FOLDER=docs/content

prev_ic_version=$1
ic_version=$2
prev_helm_chart_version=$3
helm_chart_version=$4

sed -i "" "s/$prev_ic_version/$ic_version/g" ${FILES_TO_UPDATE_IC_VERSION[*]}
sed -i "" "s/$prev_helm_chart_version/$helm_chart_version/g" ${FILE_TO_UPDATE_HELM_CHART_VERSION[*]}

# copy the helm chart README to the docs
{
    sed -n '1,10p' docs/content/installation/installation-with-helm.md
    sed -n '3,$p' deployments/helm-chart/README.md
} >file2.new && mv file2.new docs/content/installation/installation-with-helm.md

sed -i '' '/^|Parameter | Description | Default |/i\
{{% table %}}\
' docs/content/installation/installation-with-helm.md

line_number=$(grep -n -e "|" docs/content/installation/installation-with-helm.md | tail -n 1 | cut -d : -f 1)

sed -i '' "${line_number}a\\
{{% /table %}}
" docs/content/installation/installation-with-helm.md

# update repo CHANGELOG
sed -i "" "1r hack/repo-changelog-template.txt" CHANGELOG.md
sed -i "" -e "s/%%TITLE%%/### $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" CHANGELOG.md

# update docs
find $DOCS_TO_UPDATE_FOLDER -type f -name "*.md" -exec sed -i "" "s/v$prev_ic_version/v$ic_version/g" {} +
find $DOCS_TO_UPDATE_FOLDER/installation -type f -name "*.md" -exec sed -i "" "s/$prev_ic_version/$ic_version/g" {} +
find $DOCS_TO_UPDATE_FOLDER/app-protect-* -type f -name "*.md" -exec sed -i "" "s/$prev_ic_version/$ic_version/g" {} +

# update IC version in the technical-specification doc
sed -i "" "s/$prev_ic_version/$ic_version/g" $DOCS_TO_UPDATE_FOLDER/technical-specifications.md
