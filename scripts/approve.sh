#!/bin/bash

pushd $(dirname "$BASH_SOURCE") > /dev/null
CUR_DIR=$(pwd)
popd > /dev/null

PR=$1
REVIEWER_CHECK=$2

if [ -z "$PR" ]; then
    echo "Usage: $0 <pr_number>"
    exit 1
fi

function mergeable() {
    local PRN=$1
    while true; do
        local STATE=$(hub api repos/{owner}/{repo}/pulls/$PRN | python -m json.tool | grep '"mergeable":' | cut -d ":" -f 2 | cut -d "," -f 1)
        STATE=$(eval "echo $STATE")
        echo "merge state is $STATE"
        case "$STATE" in
            true)
                return 0
                ;;
            null)
                ;;
            *)
                return 1
                ;;
        esac
    done
}

function pr_state() {
    local PRN=$1
    hub api repos/{owner}/{repo}/pulls/$PRN | python -m json.tool | grep '"state"' | cut -d '"' -f 4
}

function last_commits() {
    local PRN=$1
    hub api repos/{owner}/{repo}/pulls/$PRN/commits | python -m json.tool | grep '"comments_url"' | awk '{ lifo[NR]=$0; lno=NR } END{ for(;lno>0;lno--){ print lifo[lno]; } }' | cut -d "/" -f 8
}

function last_check() {
    local CMT=$1
    hub api repos/{owner}/{repo}/commits/${CMT}/check-runs -H 'Accept: application/vnd.github.antiope-preview+json' | python -m json.tool | grep conclusion | cut -d '"' -f 4
}

function check_label() {
    local PRN=$1
    local LABEL=$2
    hub api repos/{owner}/{repo}/issues/${PRN}/labels | python -m json.tool | grep '"name": "'$LABEL'"'
}

function label() {
    local PRN=$1
    local MSG=$2
    local LABEL=$3

    if check_label $PRN $LABEL > /dev/null; then
        echo "Label $LABEL success!"
        return 0
    fi
    for try in $(seq 3)
    do
        echo "Send $MSG ..."
        hub api repos/{owner}/{repo}/issues/${PRN}/comments -f "body=$MSG" > /dev/null
        if [ "$?" -ne "0" ]; then
            echo "Send $MSG fail!"
            return 1
        fi
        for chk in $(seq 30)
        do
            sleep 1
            if check_label $PRN $LABEL > /dev/null; then
                echo "Label $LABEL success!"
                return 0
            fi
        done
    done
    return 1
}

echo "#1. check status of pull request $PR"

STATE=$(pr_state $PR)
if [ "$STATE" != "open" ]; then
    echo "Pull request $PR state $STATE, exit..."
    exit 0
fi

echo "Pull request state is open, continue..."

echo "#2. check pull request $PR is mergeable"

if ! mergeable $PR; then
    echo "Pull request $PR is not mergeable (DIRTY), exit..."
    exit 1
fi

echo "#3. check all checks of pull request $PR have been passed"

CHECKED=
for COMMIT in $(last_commits $PR)
do
    for RESULT in $(last_check $COMMIT)
    do
        echo "Commit $COMMIT check result: $RESULT"
        if [ "$RESULT" != "success" ]; then
            echo "Cannot approve before all checks success"
            exit 1
        fi
        CHECKED=yes
    done
    if [ -n "$CHECKED" ]; then
        break
    fi
done

if [ -z "$CHECKED" ]; then
    echo "No check passed, give up and try later ..."
    exit 1
fi

echo "All check passed, going to approve and lgtm the Pull Request #$PR..."

if ! label "$PR" "/lgtm" "lgtm"; then
    echo "Label lgtm failed"
    exit 1
fi

if [ -n "$REVIEWER_CHECK" ]; then
    echo "Check all requested reviwers /lgtm the pull request: "
    pullfile=$(mktemp)
    commentfile=$(mktemp)
    function cleanup {
        rm -rf "$pullfile" "$commentfile"
    }
    trap cleanup EXIT
    hub api repos/{owner}/{repo}/pulls/$PR > $pullfile
    hub api repos/{owner}/{repo}/issues/$PR/comments > $commentfile
    if ! $CUR_DIR/advchecks.py $pullfile $commentfile; then
        echo "Not all assigned reviwers comment lgtm, give up..."
        exit 1
    fi
    echo "passed!"
fi

if ! label "$PR" "/approve" "approved"; then
    echo "Label approved failed"
    exit 1
fi

echo "Success!"
