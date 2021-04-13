#!/bin/bash

set -e

if [ -z "$ROOT_DIR" ]; then
    ROOT_DIR=$(dirname $(dirname $(readlink -f "$BASH_SOURCE")))
fi

RPMS_DIR=$ROOT_DIR/_output/rpms
DEBS_DIR=$ROOT_DIR/_output/debs

case $(uname -m) in
    x86_64)
        CURRENT_ARCH=amd64
        ;;
    aarch64)
        CURRENT_ARCH=arm64
        ;;
esac

mkdir -p $DEBS_DIR && cd $DEBS_DIR

find $RPMS_DIR -type f -name *.rpm -exec alien --target=$CURRENT_ARCH {} \;
