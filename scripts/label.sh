#!/bin/bash

PRN=$1
MSG=$2
LABEL=$3

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
        hub api repos/{owner}/{repo}/issues/$PRN/comments -f "body=$MSG" > /dev/null
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

if [ -z "$LABEL" ]; then
    echo "Usage: $0 <pr_number> <msg> <label>"
    exit 1
fi

label "$PRN" "$MSG" "$LABEL"
