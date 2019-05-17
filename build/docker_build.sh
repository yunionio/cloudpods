#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SRC_ROOT=$(realpath $(dirname "${BASH_SOURCE[0]}")/..)
BIN_DIR=$SRC_ROOT/_output/bin
WORK_DIR="/go/src/yunion.io/x/onecloud"
CMDS=$@

docker run --rm -it \
    -w $WORK_DIR \
    -v $SRC_ROOT:$WORK_DIR \
    d3lx/golang:yunion \
    make $CMDS

echo "Build finish"
ls -alh $BIN_DIR
