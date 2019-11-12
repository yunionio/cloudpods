#!/bin/bash

pushd $(dirname "$BASH_SOURCE") > /dev/null
CUR_DIR=$(pwd)
popd > /dev/null

PRN=$1
if [ -z "$PRN" ]; then
    echo "$0 <PRN>"
    exit 1
fi

$CUR_DIR/label.sh $PRN /lgtm lgtm
