#!/bin/bash

set -e

STORAGE=$1

if [ -z "$STORAGE" ]; then
    echo "Usage: $0 <storage_id_or_name>"
    exit 1
fi

STORAGE_ID=$(climc storage-show $STORAGE | grep -w " id " | awk '{print $4}')
STORAGE_NAME=$(climc storage-show $STORAGE | grep -w " name " | awk '{print $4}')

if [ -z "$STORAGE_ID" ]; then
    echo "Storage $STORAGE not found"
    exit 1
fi

echo "Clean disks on storage $STORAGE_NAME($STORAGE_ID) ..."

for disk_id in $(climc disk-list --admin --storage $STORAGE_ID --show-emulated --limit 2048 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc disk-purge $disk_id > /dev/null
done

for disk_id in $(climc disk-list --admin --storage $STORAGE_ID --show-emulated --pending-delete --limit 2048 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc disk-purge $disk_id > /dev/null
done

for snapshot_id in $(climc snapshot-list --limit 2048 --scope system  --admin  --filter "storage_id.equals($STORAGE_ID)" | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc snapshot-delete $snapshot_id > /dev/null
done

# CACHE_ID=$(climc storage-show $STORAGE_ID | grep -w " storagecache_id " | awk '{print $4}')

climc storage-delete $STORAGE_ID > /dev/null
