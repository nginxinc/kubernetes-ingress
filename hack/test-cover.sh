#!/usr/bin/env bash

set -e
echo "" >coverage.txt

for d in $(go list ./... | grep -v vendor); do
    echo "testing ${d}"
    go test -tags=aws -shuffle=on -race -coverprofile="profile.out" -covermode=atomic "${d}"
    if [ -f "profile.out" ]; then
        cat "profile.out" >>coverage.txt
        rm profile.out
    fi
done
cat coverage.txt
