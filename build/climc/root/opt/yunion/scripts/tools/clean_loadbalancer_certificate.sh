#!/bin/bash

set -e

LB_CERT=$1

if [ -z "$LB_CERT" ]; then
    echo "Usage: $0 <loadbalancer_id_or_name>"
    exit 1
fi

LB_CERT_ID=$(climc lbcert-show $LB_CERT | grep -w " id " | awk '{print $4}')
LB_CERT_NAME=$(climc lbcert-show $LB_CERT | grep -w " name " | awk '{print $4}')

if [ -z "$LB_CERT_ID" ]; then
    echo "Loadbalancer certificate $LB_CERT_ID not found"
    exit 1
fi

echo "Ready to remove loadbalancer certificate $LB_CERT_NAME($LB_CERT_ID) ..."

clean_loadbalancer_cert() {
    local LID=$1
    climc lbcert-purge $LID
}

clean_loadbalancer_cert $LB_CERT_ID
