#!/usr/bin/env bash

# Updates the files required for a new minor release. Run this script in the master branch.
#
# Usage:
# hack/prepare-minor-release-in-master.sh ic-version helm-chart-version
#
# Example:
# hack/prepare-minor-release-in-master.sh 1.5.5 0.3.5

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

sed -i "" "s/$prev_ic_version/$ic_version/g" README.md

# update repo CHANGELOG
sed -i "" "1r hack/changelog-template.txt" CHANGELOG.md
sed -i "" -e "s/%%TITLE%%/### $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" CHANGELOG.md

# update docs CHANGELOG
sed -i "" "1r hack/changelog-template.txt" docs-web/releases.md 
sed -i "" -e "s/%%TITLE%%/## NGINX Ingress Controller $ic_version/g" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" docs-web/releases.md

# update IC version in the technical-specification doc
sed -i "" "s/$prev_ic_version/$ic_version/g" docs-web/technical-specifications.md 

# update IC version in the building ingress controller doc
sed -i "" "s/$prev_ic_version/$ic_version/g" doc-webs/installation/building-ingress-controller-image.md
