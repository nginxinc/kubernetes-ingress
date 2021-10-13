#!/usr/bin/env bash

# Updates the files required for a new minor release. Run this script in the release branch.
#
# Usage:
# hack/prepare-minor-release.sh ic-version helm-chart-version
#
# Example:
# hack/prepare-minor-release.sh 1.5.5 0.3.5

DOCS_TO_UPDATE_FOLDER=docs/content

if [ $# != 2 ]; then
    echo "Invalid number of arguments" 1>&2
    echo "Usage: $0 ic-version helm-chart-version" 1>&2
    exit 1
fi

ic_version=$1
helm_chart_version=$2

todays_date=$(date '+%d %b %Y')

prev_ic_version=$(echo $ic_version | awk -F. '{ printf("%s\\.%s\\.%d", $1, $2, $3-1) }')
prev_helm_chart_version=$(echo $helm_chart_version | awk -F. '{ printf("%s\\.%s\\.%d", $1, $2, $3-1) }')

# update docs
hack/common-release-prep.sh $prev_ic_version $ic_version $prev_helm_chart_version $helm_chart_version

# update docs CHANGELOG for minor release
sed -i "" "8r hack/minor-changelog-template.txt" $DOCS_TO_UPDATE_FOLDER/releases.md
sed -i "" -e "s/%%TITLE%%/## NGINX Ingress Controller $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" -e "s/%%DATE%%/$todays_date/g" $DOCS_TO_UPDATE_FOLDER/releases.md
