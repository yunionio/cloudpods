#!/bin/bash

set -e

if [ -z "$ROOT_DIR" ]; then
	pushd $(dirname $(readlink -f "$BASH_SOURCE")) > /dev/null
	ROOT_DIR=$(cd .. && pwd)
	popd > /dev/null
fi

SRC_BIN=$ROOT_DIR/_output/bin
SRC_BUILD=$ROOT_DIR/build
OUTPUT_DIR=$ROOT_DIR/_output/debs

PKG=$1
BIN_PATH=${2:-/opt/yunion/bin}

if [ -z "$PKG" ]; then
    echo "Usage: $0 <package>"
    exit 1
fi

BIN="$SRC_BIN/$PKG"
ROOT="$SRC_BUILD/$PKG"

if [ ! -x "$BIN" ]; then
    echo "$BIN not exists"
    exit 1
fi

if [ ! -x "$ROOT" ]; then
    echo "$ROOT not exists"
    exit 1
fi

. $ROOT/vars

if [ -z "$VERSION" ]; then
    TAG=$(git describe --abbrev=0 --tags || echo 000000)
    VERSION=${TAG/\//-}
    VERSION=${VERSION/v/}
fi
RELEASE=`date +"%y%m%d%H"`
FULL_VERSION=$VERSION-$RELEASE
BUILDROOT=$OUTPUT_DIR/yunion-$1-$FULL_VERSION
rm -rf $BUILDROOT
mkdir -p $BUILDROOT/DEBIAN
mkdir -p $BUILDROOT/$BIN_PATH

cp -rf $BIN $BUILDROOT/$BIN_PATH
if [ -d $ROOT/root ]; then
    cp -rf $ROOT/root/* $BUILDROOT/
fi


echo "Build root ${BUILDROOT}"

case $(uname -m) in
    x86_64)
        CURRENT_ARCH=amd64
        ;;
    aarch64)
        CURRENT_ARCH=arm64
        ;;
esac

echo "Package: yunion-$1
Version: $FULL_VERSION
Section: base
Priority: optional
Architecture: $CURRENT_ARCH
Maintainer: wanyaoqi@yunionyun.com
Description: Yunion $1
" > $BUILDROOT/DEBIAN/control
chmod 0755 $BUILDROOT/DEBIAN/control


dpkg-deb --build $BUILDROOT