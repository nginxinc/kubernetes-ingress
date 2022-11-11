#! /bin/bash

git_commit=$1
git_tag=$2
docker_tag=edge

commit_tag=$(git describe --exact-match ${git_commit} 2>/dev/null)

if [[ ${commit_tag} == ${git_tag} ]]; then
    # we're on the exact commit of the tag, use the docker image for the tag
    docker_tag=${git_tag//v/}
    echo ${docker_tag}
    exit 0

else
    # we're on a random commit, pull the 'edge' docker image to compare commits
    # if it's the latest commit from 'main' the SHA will match the 'revision' of the 'edge' docker image
    docker pull nginx/nginx-ingress:${docker_tag} >/dev/null 2>&1
    DOCKER_SHA=$(docker inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' nginx/nginx-ingress:${docker_tag})
    if [[ ${DOCKER_SHA} == ${git_commit} ]]; then
        # we're on the same commit as the latest edge
        echo ${docker_tag}
        exit 0
    fi
fi

echo "fail"
