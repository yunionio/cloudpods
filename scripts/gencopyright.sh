#!/bin/bash

set -e

pushd $(dirname $BASH_SOURCE) > /dev/null
ROOT_DIR=$(cd .. && pwd -P)
popd > /dev/null

COPYRIGHT_TXT=$ROOT_DIR/scripts/copyright.txt

LINECNT=$(wc -l $COPYRIGHT_TXT | awk '{print $1}')

function patch() {
    if ! (head -n $LINECNT $1 | diff -q $COPYRIGHT_TXT - > /dev/null); then
        echo "patch $1"
        OUT=$(mktemp) || { echo "Failed to create temp file"; exit 1; }
        cat $COPYRIGHT_TXT > $OUT
        cat $1 >> $OUT
        mv $OUT $1
    fi
}

for top in $@
do
    for f in $(find $top -iname "*.go")
    do
        patch $f
    done
done
