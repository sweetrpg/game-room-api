#!/bin/bash

set -x
set -e

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
version=$1

export DOCKER_BUILDKIT=1
registry=registry.sweetrpg.com
name=sweetrpg-game-room-api
new_version=$(semver -i patch ${version})

ssh-add ~/.ssh/id_rsa
docker build \
    -f ${scriptdir}/../Dockerfile \
    -t ${registry}/${name}:latest \
    -t ${registry}/${name}:${new_version} \
    --ssh default \
    ${scriptdir}/..
