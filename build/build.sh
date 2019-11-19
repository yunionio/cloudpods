#!/bin/sh

set -e

if [ -z "$ROOT_DIR" ]; then
	pushd $(dirname $(readlink -f "$BASH_SOURCE")) > /dev/null
	ROOT_DIR=$(cd .. && pwd)
	popd > /dev/null
fi

SRC_BIN=$ROOT_DIR/_output/bin
SRC_BUILD=$ROOT_DIR/build
OUTPUT_DIR=$ROOT_DIR/_output/rpms

PKG=$1

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

BUILDROOT=$(mktemp -d 2>/dev/null || mktemp -d -t 'yunion')

echo "Build root ${BUILDROOT}"

# BRANCH=$(git rev-parse --abbrev-ref HEAD)
# TAG=$(git describe --exact-match --tags)
if [ -z "$VERSION" ]; then
	TAG=$(git describe --abbrev=0 --tags || echo testing)
	VERSION=${TAG/\//-}
	VERSION=${VERSION/v/}
fi
RELEASE=`date +"%y%m%d%H"`

SPEC_DIR=$BUILDROOT/SPECS
SPEC_FILE=$SPEC_DIR/${PKG}.spec
RPM_DIR=$BUILDROOT/RPMS

if [ -z "$OWNER" ]; then
	OWNER=yunion
fi

if [ -z "$SERVICE" ]; then
    SERVICE="0"
fi

mkdir -p $SPEC_DIR

echo "# Yunion RPM spec

%global owner   $OWNER
%global pkgname yunion-$PKG
%global homedir /var/run/%{owner}
%global use_systemd $SERVICE

Name: %{pkgname}
Version: $VERSION
Release: $RELEASE
Summary: %{pkgname}

Group: Unspecified
License: GPL
URL: https://www.yunion.io/doc
$(for req in "${REQUIRES[@]}"; do echo Requires: "$req"; done)
%if %{use_systemd}
Requires: systemd
BuildRequires: systemd
%endif

%description
$DESCRIPTION

%prep

%build

%install
install -D -m 0755 $BIN \$RPM_BUILD_ROOT/opt/yunion/bin/$PKG
for bin in $EXTRA_BINS; do
  install -D -m 0755 $SRC_BIN/\$bin \$RPM_BUILD_ROOT/opt/yunion/bin/\$bin
done
if [ -d $ROOT/root ]; then
  rsync -a $ROOT/root/ \$RPM_BUILD_ROOT
fi

%pre
%if %{use_systemd}
getent group %{owner} >/dev/null || /usr/sbin/groupadd -r %{owner}
getent passwd %{owner} >/dev/null || /usr/sbin/useradd -r -s /sbin/nologin -d %{homedir} -M -g %{owner} %{owner}
%endif

%post
%if %{use_systemd}
    mkdir -p /var/run/%{owner}
    chown -R %{owner}:%{owner} /var/run/%{owner}
    /usr/bin/systemctl preset %{pkgname}.service >/dev/null 2>&1 ||:
%endif

%preun
%if %{use_systemd}
    /usr/bin/systemctl --no-reload disable %{pkgname}.service >/dev/null 2>&1 || :
    /usr/bin/systemctl stop %{pkgname}.service >/dev/null 2>&1 ||:
%endif

%postun
%if %{use_systemd}
    /usr/bin/systemctl daemon-reload >/dev/null 2>&1 ||:
%endif

%files
%doc
/opt/yunion/bin/$PKG
$(for b in $EXTRA_BINS; do echo /opt/yunion/bin/$b; done)
" > $SPEC_FILE

if [ -d $ROOT/root/ ]; then
    find $ROOT/root/ -type f | sed -e "s:$ROOT/root::g" >> $SPEC_FILE
fi

rpmbuild --define "_topdir $BUILDROOT" -bb $SPEC_FILE

find $RPM_DIR -type f | while read f; do
	d="$(dirname "$f")"
	d="$OUTPUT_DIR/${d#$RPM_DIR}"
	mkdir -p "$d"
	cp $f $d
done

rm -fr $BUILDROOT
