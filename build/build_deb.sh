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
    VERSION=${VERSION/master-/}
fi
RELEASE=`date +"%y%m%d%H"`
FULL_VERSION=$VERSION-$RELEASE
BUILDROOT=$OUTPUT_DIR/yunion-$PKG-$FULL_VERSION
function cleanup {
  rm -rf "$BUILDROOT"
  echo "Deleted temp working directory $BUILDROOT"
}
# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

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

if [[ -n "$GOARCH" ]]; then
    case "$GOARCH" in
		"arm64" | "arm" | "aarch64")
	        CURRENT_ARCH="arm64"
            ;;
		"x86" | "x86_64" | "i686" | "i386" | "amd64")
			CURRENT_ARCH="amd64"
            ;;
	esac
fi

echo "Package: yunion-$PKG
Version: $FULL_VERSION
Section: base
Priority: optional
Architecture: $CURRENT_ARCH
Maintainer: wanyaoqi@yunionyun.com
Description: $DESCRIPTION" > $BUILDROOT/DEBIAN/control
if [ ${#REQUIRES[@]} -gt 0 ]; then
    DEPS=$(IFS=, ; echo "${REQUIRES[*]}")
    echo "Depends: $DEPS" >> $BUILDROOT/DEBIAN/control
fi
chmod 0755 $BUILDROOT/DEBIAN/control

cat $BUILDROOT/DEBIAN/control

if [ -n "$SERVICE" ]; then
echo "#!/bin/bash

/usr/bin/systemctl --no-reload disable yunion-${PKG}.service >/dev/null 2>&1 || :
/usr/bin/systemctl stop yunion-${PKG}.service >/dev/null 2>&1 ||:
" > $BUILDROOT/DEBIAN/preinst
chmod 0755 $BUILDROOT/DEBIAN/preinst
echo "#!/bin/bash

/usr/bin/systemctl preset yunion-${PKG}.service >/dev/null 2>&1 ||:
/usr/bin/systemctl daemon-reload >/dev/null 2>&1 ||:
" > $BUILDROOT/DEBIAN/postinst
chmod 0755 $BUILDROOT/DEBIAN/postinst
fi

dpkg-deb --build $BUILDROOT
