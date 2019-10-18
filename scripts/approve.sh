#!/bin/bash

PR=$1
MSG=$2

if [ -z "$PR" ]; then
    echo "Usage: $0 <pr_number>"
    exit 1
fi

function mergeable() {
    local PRN=$1
    hub api repos/{owner}/{repo}/pulls/$PRN | python -m json.tool | grep '"mergeable": true'
}

function pr_state() {
    local PRN=$1
    hub api repos/{owner}/{repo}/pulls/$PRN | python -m json.tool | grep '"state"' | cut -d '"' -f 4
}

function last_commit() {
    local PRN=$1
    hub api repos/{owner}/{repo}/pulls/$PRN/commits | python -m json.tool | grep '"comments_url"' | tail -1 | cut -d "/" -f 8
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

if ! mergeable $PR > /dev/null; then
    echo "Pull request $PR is not mergeable (DIRTY), exit..."
    exit 1
fi

STATE=$(pr_state $PR)
if [ "$STATE" != "open" ]; then
    echo "Pull request $PR state != open, exit..."
    exit 1
fi

CHECKED=
COMMIT=$(last_commit $PR)
for RESULT in $(last_check $COMMIT)
do
    CHECKED=yes
    echo "Last commit $COMMIT check result: $RESULT"
    if [ "$RESULT" != "success" ]; then
        echo "Cannot approve before all checks success"
        exit 1
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

if ! label "$PR" "/approve" "approved"; then
    echo "Label approved failed"
    exit 1
fi

echo "Success!"
