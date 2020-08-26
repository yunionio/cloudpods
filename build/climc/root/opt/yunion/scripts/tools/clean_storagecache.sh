#!/bin/bash

set -e

CACHE=$1

if [ -z "$CACHE" ]; then
    echo "Usage: $0 <storage_cache_id_or_name>"
    exit 1
fi

CACHE_ID=$(climc storage-cache-show $CACHE | grep -w " id " | awk '{print $4}')
CACHE_NAME=$(climc storage-cache-show $CACHE | grep -w " name " | awk '{print $4}')

if [ -z "$CACHE_ID" ]; then
    echo "Storage cache $CACHE not found"
    exit 1
fi

echo "Ready to remove storagecache $CACHE_NAME($CACHE_ID) ..."

for imgid in $(climc storage-cached-image-list --storagecache $CACHE_ID --show-emulated --limit 0 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $4}')
do
    climc storagecache-uncache-image $CACHE_ID $imgid --force # > /dev/null
done

climc storage-cache-delete $CACHE_ID > /dev/null

