#!/bin/bash

max_attempts=$1
port=$2
secure=$3
host="${4:-localhost}"

counter=0
until [ $(curl --write-out %{http_code} -ks --output /dev/null http${secure}://${host}:${port}) -eq 404 ]; do
if [ ${counter} -eq ${max_attempts} ]; then
    exit 1
fi
printf '.'; counter=$(($counter+1)); sleep 5;
done
