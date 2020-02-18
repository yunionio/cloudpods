#!/bin/bash

set -e

LB_ACL=$1

if [ -z "$LB_ACL" ]; then
    echo "Usage: $0 <loadbalancer_id_or_name>"
    exit 1
fi

LB_ACL_ID=$(climc lbacl-show $LB_ACL | grep -w " id " | awk '{print $4}')
LB_ACL_NAME=$(climc lbacl-show $LB_ACL | grep -w " name " | awk '{print $4}')

if [ -z "$LB_ACL_ID" ]; then
    echo "Loadbalancer ACL $LB_ACL_ID not found"
    exit 1
fi

echo "Ready to remove loadbalancer acl $LB_ACL_NAME($LB_ACL_ID) ..."

clean_loadbalancer_acl() {
    local LID=$1
    climc lbacl-purge $LID
}

clean_loadbalancer_acl $LB_ACL_ID
