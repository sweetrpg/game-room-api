#!/bin/bash

set -e

export DOCKER_BUILDKIT=1
registry=registry.sweetrpg.com
name=sweetrpg-game-room-api

docker push \
    ${registry}/${name}
