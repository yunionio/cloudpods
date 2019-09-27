#!/bin/bash

set -o errexit
set -o pipefail

pushd $(dirname $(readlink -f "$BASH_SOURCE")) > /dev/null
CUR_DIR=$(pwd)
SRC_DIR=$(cd .. && pwd)
popd > /dev/null

DOCKER_DIR="$SRC_DIR/build/docker"

REGISTRY=${REGISTRY:-docker.io/yunion}
TAG=${TAG:-latest}

build_bin() {
    make cmd/$1
}

build_image() {
    local tag=$1
    local file=$2
    local path=$3
    docker build -t "$tag" -f "$2" "$3"
}

push_image() {
    local tag=$1
    docker push "$tag"
}

COMPONENTS=$@

for compent in $COMPONENTS; do
    build_bin $compent
    img_name="$REGISTRY/$compent:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile.$compent $SRC_DIR
    push_image "$img_name"
done
