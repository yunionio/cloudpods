#!/bin/bash
#
# vi: expandtab tabstop=4 shiftwidth=0

set -o errexit
set -o pipefail

if [ "$DEBUG" == "true" ]; then
    set -ex
    export PS4='+(${BASH_SOURCE}:${LINENO}): ${FUNCNAME[0]:+${FUNCNAME[0]}(): }'
fi

readlink_mac() {
    cd $(dirname $1)
    TARGET_FILE=$(basename $1)

    # Iterate down a (possible) chain of symlinks
    while [ -L "$TARGET_FILE" ]; do
        TARGET_FILE=$(readlink $TARGET_FILE)
        cd $(dirname $TARGET_FILE)
        TARGET_FILE=$(basename $TARGET_FILE)
    done

    # Compute the canonicalized name by finding the physical path
    # for the directory we're in and appending the target file.
    PHYS_DIR=$(pwd -P)
    REAL_PATH=$PHYS_DIR/$TARGET_FILE
}

get_current_arch() {
    local current_arch
    case $(uname -m) in
    x86_64)
        current_arch=amd64
        ;;
    aarch64)
        current_arch=arm64
        ;;
    esac
    echo $current_arch
}

pushd $(
    cd "$(dirname "$0")"
    pwd
) >/dev/null
readlink_mac $(basename "$0")
cd "$(dirname "$REAL_PATH")"
CUR_DIR=$(pwd)
SRC_DIR=$(cd .. && pwd)
popd >/dev/null

DOCKER_DIR="$SRC_DIR/build/docker"

# https://docs.docker.com/develop/develop-images/build_enhancements/
export DOCKER_BUILDKIT=1
# https://github.com/docker/buildx#with-buildx-or-docker-1903
export DOCKER_CLI_EXPERIMENTAL=enabled

REGISTRY=${REGISTRY:-docker.io/yunion}
TAG=${TAG:-latest}
CURRENT_ARCH=$(get_current_arch)
ARCH=${ARCH:-$CURRENT_ARCH}

build_bin() {
    local BUILD_ARCH=$2
    local BUILD_CGO=$3
    case "$1" in
    host-image)
        rm -vf _output/bin/$1
        rm -rvf _output/bin/bundles/$1
        if [ -z "$LIBQEMUIO_PATH" ]; then
            echo "Need set \$LIBQEMUIO_PATH env to build host-image"
            exit 1
        fi
        GOOS=linux make cmd/$1
        ;;
    climc)
        rm -vf _output/bin/*cli
        env $BUILD_ARCH $BUILD_CGO make -C "$SRC_DIR" docker-alpine-build F="cmd/$1 cmd/*cli"
        ;;
    host-deployer | telegraf-raid-plugin)
        env $BUILD_ARCH $BUILD_CGO make -C "$SRC_DIR" docker-centos-build F="cmd/$1"
        ;;
    *)
        env $BUILD_ARCH $BUILD_CGO make -C "$SRC_DIR" docker-alpine-build F="cmd/$1"
        ;;
    esac
}

build_bundle_libraries() {
    for bundle_component in 'host-image'; do
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
    local arch
    if [[ "$tag" == *:5000/* ]]; then
        arch=$(arch)
        case $arch in
        x86_64)
            docker buildx build -t "$tag" -f "$2" "$3" --output type=docker --platform linux/amd64
            ;;
        aarch64)
            docker buildx build -t "$tag" -f "$2" "$3" --output type=docker --platform linux/arm64
            ;;
        *)
            echo wrong arch
            exit 1
            ;;
        esac
    else
        if [[ "$tag" == *"amd64" || "$ARCH" == "" || "$ARCH" == "amd64" || "$ARCH" == "x86_64" || "$ARCH" == "x86" ]]; then
            docker buildx build -t "$tag" -f "$file" "$path" --push --platform linux/amd64
        elif [[ "$tag" == *"arm64" || "$ARCH" == "arm64" ]]; then
            docker buildx build -t "$tag" -f "$file" "$path" --push --platform linux/arm64
        else
            docker buildx build -t "$tag" -f "$file" "$path" --push
        fi
        docker pull "$tag"
    fi
}

buildx_and_push() {
    local tag=$1
    local file=$2
    local path=$3
    local arch=$4
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi
    docker buildx build -t "$tag" --platform "linux/$arch" -f "$2" "$3" --push
    docker pull --platform "linux/$arch" "$tag"
}

get_image_name() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name="$REGISTRY/$component:$TAG"
    if [[ "$is_all_arch" == "true" || "$arch" == arm64 || "$component" == host-image ]]; then
        img_name="${img_name}-$arch"
    fi
    echo $img_name
}

build_process() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)
    local build_env=""

    case "$component" in
    host | host-image)
        build_env="$build_env CGO_ENABLED=1"
        ;;
    *)
        build_env="$build_env CGO_ENABLED=0"
        ;;
    esac

    build_bin $component
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi
    build_bundle_libraries $component

    build_image $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR
}

build_process_with_buildx() {
    local component=$1
    local arch=$2
    local is_all_arch=$3
    local img_name=$(get_image_name $component $arch $is_all_arch)

    build_env="GOARCH=$arch"
    case "$component" in
    host | host-image)
        build_env="$build_env CGO_ENABLED=1"
        ;;
    *)
        build_env="$build_env CGO_ENABLED=0"
        ;;
    esac

    case "$component" in
    host | torrent)
        buildx_and_push $img_name $DOCKER_DIR/multi-arch/Dockerfile.$component $SRC_DIR $arch
        ;;
    *)
        build_bin $component $build_env
        buildx_and_push $img_name $DOCKER_DIR/Dockerfile.$component $SRC_DIR $arch
        ;;
    esac
}

general_build() {
    local component=$1
    # 如果未指定，则默认使用当前架构
    local arch=${2:-$CURRENT_ARCH}
    local is_all_arch=$3

    if [[ "$CURRENT_ARCH" == "$arch" ]]; then
        build_process $component $arch $is_all_arch
    else
        build_process_with_buildx $component $arch $is_all_arch
    fi
}

make_manifest_image() {
    local component=$1
    local img_name=$(get_image_name $component "" "false")
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[$(readlink -f ${BASH_SOURCE}):${LINENO} ${FUNCNAME[0]}] return for DRY_RUN"
        return
    fi

    if [[ "$img_name" == *:5000/* ]]; then
        docker push $img_name-amd64
        docker push $img_name-arm64
    fi

    docker buildx imagetools create -t $img_name \
        $img_name-amd64 \
        $img_name-arm64
    docker manifest inspect ${img_name} | grep -wq amd64
    docker manifest inspect ${img_name} | grep -wq arm64
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
mkdir -p $SRC_DIR/_output

show_update_cmd() {
    local component=$1
    local arch=$2
    local spec=$1
    local name=$1
    local tag=${TAG}
    if [[ "$arch" == arm64 || "$component" == host-image ]]; then
        tag="${tag}-$arch"
    fi

    case "$component" in
    'apigateway')
        spec='apiGateway'
        ;;
    'apimap')
        spec='apiMap'
        ;;
    'baremetal-agent')
        spec='baremetalagent'
        name='baremetal-agent'
        ;;
    'host')
        spec='hostagent'
        ;;
    'host-deployer')
        spec='hostdeployer'
        ;;
    'region')
        spec='regionServer'
        ;;
    'region-dns')
        spec='regionDNS'
        ;;
    'vpcagent')
        spec='vpcAgent'
        ;;
    'esxi-agent')
        spec='esxiagent'
        ;;
    esac

    echo "kubectl patch oc -n onecloud default --type='json' -p='[{op: replace, path: /spec/${spec}/imageName, value: ${name}},{"op": "replace", "path": "/spec/${spec}/repository", "value": "${REGISTRY}"},{"op": "add", "path": "/spec/${spec}/tag", "value": "${tag}"}]'"
}

for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        echo "Please build image for climc"
        continue
    fi

    echo "Start to build component: $component"
    if [[ $component == host-image ]]; then
        build_process $component $ARCH "false"
        continue
    fi

    case "$ARCH" in
    all)
        for arch in "arm64" "amd64"; do
            general_build $component $arch "true"
        done
        make_manifest_image $component
        #show_update_cmd $component $ARCH
        ;;
    *)
        if [ -e "$DOCKER_DIR/Dockerfile.$component" ]; then
            general_build $component $ARCH "false"
            #show_update_cmd $component $ARCH
        fi
        ;;
    esac
done

echo ""

for component in $COMPONENTS; do
    if [[ $component == *cli ]]; then
        continue
    fi
    if [[ $component == host-image ]]; then
        continue
    fi
    show_update_cmd $component
done
