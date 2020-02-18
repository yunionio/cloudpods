#!/bin/bash

set -e

pushd $(dirname $BASH_SOURCE) > /dev/null
CUR_DIR=$(pwd -P)
popd > /dev/null


PROVIDER=$1

if [ -z "$PROVIDER" ]; then
    echo "Usage: $0 <cloud-provider-id-or-name>"
    exit 1
fi

PROVIDER_ID=$(climc cloud-provider-show $PROVIDER | grep -w " id " | awk '{print $4}')
PROVIDER_NAME=$(climc cloud-provider-show $PROVIDER | grep -w " name " | awk '{print $4}')

if [ -z "$PROVIDER_ID" ]; then
    echo "Cloud provider $PROVIDER not found"
    exit 1
fi


echo "Ready to clean cloud provider $PROVIDER_NAME($PROVIDER_ID)..."

echo "1. Disable cloud provider $PROVIDER_NAME($PROVIDER_ID) ..."

climc cloud-provider-disable $PROVIDER_ID

echo "2. Clean all hosts ..."

for hostid in $(climc host-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_host.sh $hostid
done

echo "3. Clean all loadbalancer ..."

for lbid in $(climc lb-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_loadbalancer.sh $lbid
done

echo "4. Clean all loadbalancer acl ..."

for lbaclid in $(climc lbacl-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_loadbalancer_acl.sh $lbaclid
done

echo "5. Clean all loadbalancer certificate ..."

for lbcertid in $(climc lbcert-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_loadbalancer_certificate.sh $lbcertid
done

echo "6. Clean all vpc ..."

for vpcid in $(climc vpc-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_vpc.sh $vpcid
done

echo "7. Clean all eips ..."
for eipid in $(climc eip-list --admin --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_eip.sh $eipid
done

echo "8. Clean all snapshots ..."
for snapid in $(climc snapshot-list --admin --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    echo "to clean snapshot $snapid ..."
    $CUR_DIR/clean_snapshot.sh $snapid
done

echo "9. Clean all storagecache ..."

for cacheid in $(climc storage-cache-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_storagecache.sh $cacheid
done

echo "10. Clean all storages ..."

for storeid in $(climc storage-list --manager $PROVIDER_ID --admin --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    $CUR_DIR/clean_storage.sh $storeid
done

echo "Delete cloud provider ..."

climc cloud-provider-delete $PROVIDER_ID
