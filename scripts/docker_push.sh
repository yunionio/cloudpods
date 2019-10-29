#!/bin/bash

set -o errexit
set -o pipefail

readlink_mac() {
  cd `dirname $1`
  TARGET_FILE=`basename $1`

  # Iterate down a (possible) chain of symlinks
  while [ -L "$TARGET_FILE" ]
  do
    TARGET_FILE=`readlink $TARGET_FILE`
    cd `dirname $TARGET_FILE`
    TARGET_FILE=`basename $TARGET_FILE`
  done

  # Compute the canonicalized name by finding the physical path
  # for the directory we're in and appending the target file.
  PHYS_DIR=`pwd -P`
  REAL_PATH=$PHYS_DIR/$TARGET_FILE
}

pushd $(cd "$(dirname "$0")"; pwd) > /dev/null
readlink_mac $(basename "$0")
cd "$(dirname "$REAL_PATH")"
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
    sudo docker build -t "$tag" -f "$2" "$3"
}

push_image() {
    local tag=$1
    sudo docker push "$tag"
}

COMPONENTS=$@

cd $SRC_DIR
for compent in $COMPONENTS; do
    build_bin $compent
    img_name="$REGISTRY/$compent:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile.$compent $SRC_DIR
    push_image "$img_name"
done
