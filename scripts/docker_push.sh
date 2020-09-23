#!/bin/bash

set -o errexit
set -o pipefail

if [ "$DEBUG" == "true" ]; then
    set -ex ;export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
fi

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
    local BUILD_ARCH=$2
    local BUILD_CC=$3
    local BUILD_CGO=$4
    case "$1" in
        baremetal-agent)
            GOOS=linux make cmd/$1
            ;;
        climc)
            docker run --rm \
                -v $SRC_DIR:/root/go/src/yunion.io/x/onecloud \
                -v $SRC_DIR/_output/alpine-build:/root/go/src/yunion.io/x/onecloud/_output \
                -v $SRC_DIR/_output/alpine-build/_cache:/root/.cache \
                registry.cn-beijing.aliyuncs.com/yunionio/alpine-build:1.0-3 \
                /bin/sh -c "set -ex; cd /root/go/src/yunion.io/x/onecloud; $BUILD_ARCH $BUILD_CC $BUILD_CGO GOOS=linux make cmd/$1 cmd/*cli; chown -R $(id -u):$(id -g) _output"
            ;;
        *)
            docker run --rm \
                -v $SRC_DIR:/root/go/src/yunion.io/x/onecloud \
                -v $SRC_DIR/_output/alpine-build:/root/go/src/yunion.io/x/onecloud/_output \
                -v $SRC_DIR/_output/alpine-build/_cache:/root/.cache \
                registry.cn-beijing.aliyuncs.com/yunionio/alpine-build:1.0-3 \
                /bin/sh -c "set -ex; cd /root/go/src/yunion.io/x/onecloud; $BUILD_ARCH $BUILD_CC $BUILD_CGO GOOS=linux make cmd/$1; chown -R $(id -u):$(id -g) _output"
            ;;
    esac
}


build_bundle_libraries() {
    for bundle_component in 'baremetal-agent'; do
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

buildx_and_push() {
    local tag=$1
    local file=$2
    local path=$3
    local arch=$4
    docker buildx build -t "$tag" --platform "linux/$arch" -f "$2" "$3" --push
    docker pull "$tag"
}

push_image() {
    local tag=$1
    docker push "$tag"
}

build_process() {
    local component=$1
    build_bin $component
    build_bundle_libraries $component
    img_name="$REGISTRY/$component:$TAG"
    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
    push_image "$img_name"
}

build_process_with_buildx() {
    local component=$1
    local arch=$2

    build_env="GOARCH=$arch"
    img_name="$REGISTRY/$component:$TAG"
    if [[ $arch == arm64 ]]; then
        img_name="$img_name-$arch"
        build_env="$build_env CC=aarch64-linux-musl-gcc"
        if [[ $component == host ]]; then
            build_env="$build_env CGO_ENABLED=1"
        fi
    fi

    case "$component" in
        host|esxi-agent)
            buildx_and_push $img_name $DOCKER_DIR/multi-arch/Dockerfile.$component $SRC_DIR $arch
            ;;
        *)
            build_bin $component $build_env
            buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR $arch
            ;;
    esac
}

ALL_COMPONENTS=$(ls cmd | grep -v '.*cli$' | xargs)

if [ "$#" -lt 1 ]; then
    echo "No component is specified~"
    echo "You can specify a component in [$ALL_COMPONENTS]"
    echo "If you want to build all components, specify the component to: all."
    exit
elif [ "$#" -eq 1 ] && [ "$1" == "all" ]; then
    echo "Build all onecloud docker images"
    COMPONENTS=$ALL_COMPONENTS
else
    COMPONENTS=$@
fi

cd $SRC_DIR
for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        echo "Please build image for climc"
        continue
    fi
    echo "Start to build component: $component"
    if [[ $component == baremetal-agent ]]; then
        build_process $component
        continue
    fi

    case "$ARCH" in
        all)
            for arch in "arm64" "amd64"; do
                build_process_with_buildx $component $arch
            done
            ;;
        arm64|amd64)
            build_process_with_buildx $component $ARCH
            ;;
        *)
            build_process $component
            ;;
    esac
done
