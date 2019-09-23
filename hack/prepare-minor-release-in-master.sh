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

prev_ic_version=$(echo $ic_version | awk -F. '{ printf("%s.%s.%d", $1, $2, $3-1) }')
prev_helm_chart_version=$(echo $helm_chart_version | awk -F. '{ printf("%s.%s.%d", $1, $2, $3-1) }')

sed -i "" "s/$prev_ic_version/$ic_version/g" README.md

sed -i "" "1r hack/changelog-template.txt" CHANGELOG.md
sed -i "" -e "s/%%IC_VERSION%%/$ic_version/g" -e "s/%%HELM_CHART_VERSION%%/$helm_chart_version/g" CHANGELOG.md
