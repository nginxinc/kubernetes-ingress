#! /bin/bash

git_commit=$1
git_tag=$2
docker_tag=edge

commit_tag=$(git tag --contains ${git_commit})

if [[ ${commit_tag} == ${git_tag} ]]; then
    # we're on the tag, use the docker image for the tag
    docker_tag=${git_tag//v/}
    echo ${docker_tag}
    exit 0

else
    # we're on master or a branch, try to download the latest edge and see if sha matches
    docker pull nginx/nginx-ingress:${docker_tag} >/dev/null 2>&1
    DOCKER_SHA=$(docker inspect --format '{{ index .Config.Labels "org.opencontainers.image.revision" }}' nginx/nginx-ingress:${docker_tag})
    if [[ ${DOCKER_SHA} == ${git_commit} ]]; then
        # we're on the same commit as the latest edge
        echo ${docker_tag}
        exit 0
    fi
fi

echo "fail"
