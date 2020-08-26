#!/bin/bash

set -e

pushd $(dirname $BASH_SOURCE) > /dev/null
CUR_DIR=$(pwd -P)
popd > /dev/null


ACCOUNT=$1

if [ -z "$ACCOUNT" ]; then
    echo "Usage: $0 <cloud-acount-id-or-name>"
    exit 1
fi

ACCOUNT_ID=$(climc cloud-account-show $ACCOUNT | grep -w " id " | awk '{print $4}')
ACCOUNT_NAME=$(climc cloud-account-show $ACCOUNT | grep -w " name " | awk '{print $4}')

if [ -z "$ACCOUNT_ID" ]; then
    echo "Cloud account $ACCOUNT not found"
    exit 1
fi


echo "Ready to clean cloud account $ACCOUNT_NAME($ACCOUNT_ID)..."

echo "1. Disable cloud account $ACCOUNT_NAME($ACCOUNT_ID) ..."

climc cloud-account-disable $ACCOUNT_ID

echo "2. Clean all providers ..."

for provider_id in $(climc cloud-provider-list --account $ACCOUNT_ID --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    echo "### to clean cloud provider $provider_id ..."
    $CUR_DIR/clean_provider.sh $provider_id
done

echo "3. Delete cloud account ..."

climc cloud-account-delete $ACCOUNT_ID
