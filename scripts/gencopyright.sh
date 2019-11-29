#!/bin/bash

set -e

pushd $(dirname $BASH_SOURCE) > /dev/null
ROOT_DIR=$(cd .. && pwd -P)
popd > /dev/null

COPYRIGHT_TXT=$ROOT_DIR/scripts/copyright.txt
CONTRIBUTOR_TXT=$ROOT_DIR/scripts/contributor.txt

LINECNT=$(wc -l $COPYRIGHT_TXT | awk '{print $1}')
FULLLINECNT=$((LINECNT+2))

function patch() {
    if ! (head -n $FULLLINECNT $1 | tail -n $LINECNT | diff -b -q $COPYRIGHT_TXT - > /dev/null); then
        echo "patch copyright $1"
        OUT=$(mktemp) || { echo "Failed to create temp file"; exit 1; }
        cat $CONTRIBUTOR_TXT > $OUT
        cat $COPYRIGHT_TXT >> $OUT
        echo "" >> $OUT
        cat $1 >> $OUT
        mv $OUT $1
    elif ! (head -n 2 $1 | diff -b -q $CONTRIBUTOR_TXT - > /dev/null); then
        echo "patch contributor $1"
        OUT=$(mktemp) || { echo "Failed to create temp file"; exit 1; }
        head -n 1 $CONTRIBUTOR_TXT > $OUT
        cat $1 >> $OUT
        mv $OUT $1
    fi
}

for top in $@
do
    for f in $(find $top ! -name "*zz_generated*.go" -iname "*.go")
    do
        patch $f
    done
done
