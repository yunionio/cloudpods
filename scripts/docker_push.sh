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
    GOOS=linux make cmd/$1
}

add_host_specific_libraries() {
    for libraries in 'libsoftokn3.so' 'libsqlite3.so.0' 'libfreeblpriv3.so'; do
        lib_path=""
        for base_lib_path in '/lib64/' '/lib/' '/usr/lib/x86_64-linux-gnu/' '/lib/x86_64-linux-gnu/'; do
            if [ ! -e $base_lib_path ]; then
                continue
            fi
            lib_path="$(find $base_lib_path -name $libraries -print -quit)"
            if [ -z "$lib_path" ]; then
                continue
            fi
            real_lib_path="$(readlink -f $lib_path)"
            cp $real_lib_path $1/lib/$libraries
            break
        done
        if [ -z "$lib_path" ]; then
            echo "failed find $libraries ..."
            exit 1
        fi
    done
}

build_bundle_libraries() {
    for bundle_component in 'host' 'host-deployer' 'baremetal-agent'; do
        if [ $1 == $bundle_component ]; then
            $CUR_DIR/bundle-libraries.sh _output/bin/bundles/$1 _output/bin/$1
            if [ $bundle_component == 'host' ]; then
                add_host_specific_libraries _output/bin/bundles/$1
            fi
            break
        fi
    done
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
for component in $COMPONENTS; do
    build_bin $component
    build_bundle_libraries $component
    img_name="$REGISTRY/$component:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
    push_image "$img_name"
done
