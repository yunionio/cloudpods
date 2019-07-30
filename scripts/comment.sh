#!/bin/bash

PR=$1
MSG=$2

if [ -z "$PR" ]; then
    echo "Usage: $0 <pr_number> <msg>"
    exit 1
fi

shift
for msg in "$@"
do
    echo Comment: $msg
    hub api repos/{owner}/{repo}/issues/$PR/comments -f "body=$msg" > /dev/null
    if [ "$?" -eq "0" ]; then
        echo "Success!"
    fi
done
