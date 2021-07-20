#!/usr/bin/env bash

# Updates the files required for a new major release. Run this script in the release branch.
#
# Usage:
# hack/prepare-major-release.sh prev_ic_version ic-version prev_helm_chart_version helm-chart-version
#
# Example:
# hack/prepare-major-release.sh 1.12.1 1.13.0 0.10.1 0.11.0

DOCS_TO_UPDATE_FOLDER=docs/content

if [ $# != 4 ];
then
    echo "Invalid number of arguments" 1>&2
    echo "Usage: $0 prev_ic_version ic-version prev_helm_chart_version helm-chart-version" 1>&2
    exit 1
fi

prev_ic_version=$1
ic_version=$2
prev_helm_chart_version=$3
helm_chart_version=$4

hack/common-release-prep.sh $prev_ic_version $ic_version $prev_helm_chart_version $helm_chart_version

# update operator installation docs with note
sed -i "" "9r hack/operator-note.txt" $DOCS_TO_UPDATE_FOLDER/installation/installation-with-operator.md
sed -i "" -e "s/%%IC_VERSION%%/$ic_version/g" $DOCS_TO_UPDATE_FOLDER/installation/installation-with-operator.md
