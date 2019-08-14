#!/bin/bash

PR=$1
MSG=$2

if [ -z "$PR" ]; then
    echo "Usage: $0 <pr_number>"
    exit 1
fi

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

for msg in /lgtm /approve
do
    echo "Send $msg ..."
    hub api repos/{owner}/{repo}/issues/$PR/comments -f "body=$msg" > /dev/null
    if [ "$?" -ne "0" ]; then
        echo "Send $msg fail!"
        exit 1
    fi
done

echo "Success!"
