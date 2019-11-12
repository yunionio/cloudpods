#!/bin/bash

pushd $(dirname "$BASH_SOURCE") > /dev/null
CUR_DIR=$(pwd)
popd > /dev/null

PRN=$1
if [ -z "$PRN" ]; then
    echo "$0 <PRN>"
    exit 1
fi

pulldata=$(mktemp)
function cleanup {
  rm -rf "$pulldata"
}
trap cleanup EXIT

hub api repos/{owner}/{repo}/pulls > $pulldata

PR_NUMBERS=()
for l in $(python -m json.tool $pulldata | grep "\"number\":" | cut -d ':' -f 2 | cut -d "," -f 1 | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
do
    PR_NUMBERS+=("$l")
done

PRNS=("$PRN")

for INDEX in $(python -m json.tool $pulldata | grep "\"title\":" | cut -d '"' -f 4 | grep -nr "Automated cherry pick of #${PRN}:" | awk 'BEGIN{FS=":"}{print $2}')
do
    INDEX=$((INDEX-1))
    PRNS+=("${PR_NUMBERS[$INDEX]}")
done

echo "Going to merge the following pull requests ${PRNS[@]}:"

REVIEWER_CHECK=yes
for PRN in "${PRNS[@]}"
do
    $CUR_DIR/approve.sh $PRN $REVIEWER_CHECK
    if [ "$?" -ne "0" ]; then
        echo "Merge failed, exit."
        exit 1
    fi
    if [ -n "$REVIEWER_CHECK" ]; then
        REVIEWER_CHECK=
    fi
done
