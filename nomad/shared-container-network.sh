#!/usr/bin/env bash
set -Eeuxo pipefail

cd "$(dirname ${BASH_SOURCE[0]})"

RUNTIME="--runtime=runsc"
# RUNTIME=""


CONT=$(docker run $RUNTIME -it -d -p 8084:8084 maxmcd/host-simple-server:latest)

echo  $CONT

docker run -it $RUNTIME --net=container:$CONT nixery.dev/shell/curl bash

docker run -it $RUNTIME --net=container:$CONT nixery.dev/shell/curl curl localhost:8084

docker kill $CONT
