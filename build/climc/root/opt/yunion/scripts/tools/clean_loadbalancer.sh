#!/bin/bash

set -e

LB=$1

if [ -z "$LB" ]; then
    echo "Usage: $0 <loadbalancer_id_or_name>"
    exit 1
fi

LB_ID=$(climc lb-show $LB | grep -w " id " | awk '{print $4}')
LB_NAME=$(climc lb-show $LB | grep -w " name " | awk '{print $4}')

if [ -z "$LB_ID" ]; then
    echo "Loadbalancer $LB not found"
    exit 1
fi

echo "Ready to remove loadbalancer $LB_NAME($LB_ID) ..."

clean_loadbalancer() {
    local LID=$1
    climc lb-purge $LID
}

clean_loadbalancer $LB_ID
