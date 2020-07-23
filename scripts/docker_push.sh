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
	case "$1" in
		climc)
			GOOS=linux make cmd/$1 cmd/*cli
			;;
		ansibleserver|\
		host|\
		region-dns|\
		vpcagent)
			docker run --rm \
				-v $SRC_DIR:/root/go/src/yunion.io/x/onecloud \
				-v $SRC_DIR/_output/alpine-build:/root/go/src/yunion.io/x/onecloud/_output \
				registry.cn-beijing.aliyuncs.com/yunionio/alpine-build:1.0-1 \
				/bin/sh -c "set -ex; cd /root/go/src/yunion.io/x/onecloud; make cmd/$1; chown -R $(id -u):$(id -g) _output"
			;;
		*)
			GOOS=linux make cmd/$1
			;;
	esac
}


build_bundle_libraries() {
    for bundle_component in 'host-deployer' 'baremetal-agent'; do
        if [ $1 == $bundle_component ]; then
            $CUR_DIR/bundle_libraries.sh _output/bin/bundles/$1 _output/bin/$1
            break
        fi
    done
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

cd $SRC_DIR
for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        echo "Please build image for climc"
        continue
    fi
    build_bin $component
    build_bundle_libraries $component
    img_name="$REGISTRY/$component:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
    push_image "$img_name"
done
