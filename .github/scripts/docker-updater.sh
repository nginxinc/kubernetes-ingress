#!/usr/bin/env bash

set -o pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
DOCKER_FILE=${SCRIPT_ROOT}/build/Dockerfile
exclude_strings=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --exclude)
            exclude_strings="$2"
            shift
            shift
            ;;
        *)
            DOCKER_FILE="$1"
            shift
            ;;
    esac
done

# Check if the file exists
if [ ! -f "$DOCKER_FILE" ]; then
    echo "File $DOCKER_FILE does not exist."
    exit 1
fi

function contains_excluded() {
    local line="$1"
    local exclude="$2"
    local IFS=','
    local excluded=($exclude)
    for word in "${excluded[@]}"; do
        if [[ "$line" == *"$word"* ]]; then
            return 0
        fi
    done
    return 1
}

function check_sha() {
    image_sha="$1"
    image=$(echo "$image_sha" | cut -d '@' -f1)
    tag_sha=$(echo "$image_sha" | cut -d '@' -f2)

    docker pull -q "$image" > /dev/null
    latest_digest=$(docker inspect --format='{{index .RepoDigests 0}}' "$image")
    latest_sha=$(echo "$latest_digest" | cut -d '@' -f2)

    if [ "$tag_sha" = "$latest_sha" ]; then
        echo "The provided SHA256 hash is the latest for $image"
    else
        echo "> A newer version of $image is available:"
        echo "> - $image@$tag_sha"
        echo "> + $image@$latest_sha"
        echo "> updating $DOCKER_FILE"
        sed -i -e "s/$tag_sha/$latest_sha/g" "$DOCKER_FILE"
    fi
}
if [ -n "$exclude_strings" ]; then
    echo "excluding images containing one of: '$exclude_strings'"
fi
while IFS= read -r line; do
    if [[ $line =~ ^FROM\ (.+@.+) ]]; then
        image=$(echo "${BASH_REMATCH[1]}" | awk '{print $1}')
        if [ -n "$exclude_strings" ] && contains_excluded "$line" "$exclude_strings"; then
            echo "Skipping $image"
            continue
        fi
        check_sha "$image"
    fi
done < "$DOCKER_FILE"
