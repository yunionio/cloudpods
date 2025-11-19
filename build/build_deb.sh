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

case $(uname -m) in
    x86_64)
        CURRENT_ARCH=amd64
        ;;
    aarch64)
        CURRENT_ARCH=arm64
        ;;
    riscv64)
        CURRENT_ARCH=riscv64
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
        "riscv64")
            CURRENT_ARCH="riscv64"
            ;;
	esac
fi

if [ -z "$VERSION" ]; then
    TAG=$(git describe --abbrev=0 --tags || echo 000000)
    VERSION=${TAG/\//-}
    VERSION=${VERSION/v/}
    VERSION=${VERSION/master-/}
fi
RELEASE=`date +"%y%m%d%H"`
FULL_VERSION=$VERSION-$RELEASE
BUILDROOT=$OUTPUT_DIR/yunion-${PKG}-${FULL_VERSION}_${CURRENT_ARCH}
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

echo "#!/bin/bash
" > $BUILDROOT/DEBIAN/preinst
if [ -f $ROOT/preinst ]; then
    cat $ROOT/preinst >> $BUILDROOT/DEBIAN/preinst
else
    if [ "$SERVICE" == "yes" ] && [ -n "$OWNER" ]; then
        echo "
getent group ${OWNER} >/dev/null || /usr/sbin/groupadd -r ${OWNER}
getent passwd ${OWNER} >/dev/null || /usr/sbin/useradd -r -s /sbin/nologin -d /home/${OWNER} -M -g ${OWNER} ${OWNER}
" >> $BUILDROOT/DEBIAN/preinst
    fi
fi
chmod 0755 $BUILDROOT/DEBIAN/preinst

echo "#!/bin/bash
" > $BUILDROOT/DEBIAN/postinst
if [ -f $ROOT/postinst ]; then
    cat $ROOT/postinst >> $BUILDROOT/DEBIAN/postinst
else
    if [ "$SERVICE" == "yes" ]; then
        echo "
/usr/bin/systemctl preset yunion-${PKG}.service >/dev/null 2>&1 ||:
" >> $BUILDROOT/DEBIAN/postinst
    fi
fi
chmod 0755 $BUILDROOT/DEBIAN/postinst

echo "#!/bin/bash
" > $BUILDROOT/DEBIAN/prerm
if [ -f $ROOT/prerm ]; then
    cat $ROOT/prerm >> $BUILDROOT/DEBIAN/prerm
else
    if [ "$SERVICE" == "yes" ]; then
        echo "
/usr/bin/systemctl --no-reload disable yunion-${PKG}.service >/dev/null 2>&1 || :
/usr/bin/systemctl stop yunion-${PKG}.service >/dev/null 2>&1 ||:
" >> $BUILDROOT/DEBIAN/prerm
    fi
fi
chmod 0755 $BUILDROOT/DEBIAN/prerm

echo "#!/bin/bash
" > $BUILDROOT/DEBIAN/postrm
if [ -f $ROOT/postrm ]; then
    cat $ROOT/postrm >> $BUILDROOT/DEBIAN/postrm
else
    if [ "$SERVICE" == "yes" ]; then
        echo "
/usr/bin/systemctl daemon-reload >/dev/null 2>&1 ||:
" >> $BUILDROOT/DEBIAN/postrm
    fi
fi
chmod 0755 $BUILDROOT/DEBIAN/postrm

dpkg-deb --build $BUILDROOT

case "$CURRENT_ARCH" in
    "amd64")
        DSTARCH="x86_64"
        ;;
    "arm64")
        DSTARCH="aarch64"
        ;;
    "riscv64")
        DSTARCH="riscv64"
        ;;
esac

mkdir -p ${OUTPUT_DIR}/${DSTARCH}
mv ${BUILDROOT}.deb ${OUTPUT_DIR}/${DSTARCH}/
