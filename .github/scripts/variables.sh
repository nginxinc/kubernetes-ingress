#!/usr/bin/env bash

if [ "$1" = "" ]; then
    echo "ERROR: paramater needed"
    exit 2
fi

INPUT=$1
ROOTDIR=$(git rev-parse --show-toplevel || echo ".")
if [ "$PWD" != "$ROOTDIR" ]; then
    # shellcheck disable=SC2164
    cd "$ROOTDIR";
fi

case $INPUT in
  docker_md5)
    docker_md5=$(find . -type f \( -name "Dockerfile" -o -name version.txt \) -not -path "./tests*" -exec md5sum {} + | LC_ALL=C sort  | md5sum | awk '{ print $1 }')
    echo "docker_md5=${docker_md5:0:8}"
    ;;

  go_code_md5)
    echo "go_code_md5=$(find . -type f \( -name "*.go" -o -name go.mod -o -name go.sum -o -name "*.tmpl" \) -not -path "./docs*"  -exec md5sum {} + | LC_ALL=C sort  | md5sum | awk '{ print $1 }')"
    ;;

  *)
    echo "ERROR: option not found"
    exit 2
    ;;
esac
