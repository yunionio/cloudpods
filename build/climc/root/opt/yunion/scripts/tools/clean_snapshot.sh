#/bin/bash

set -e

SNAPSHOT=$1

if [ -z "$SNAPSHOT" ]; then
    echo "Usage: $0 <snapshot_id_or_name>"
    exit 1
fi

SNAPSHOT_NAME=$(climc snapshot-show $SNAPSHOT | grep -w " name " | awk '{print $4}')
SNAPSHOT_ID=$(climc snapshot-show $SNAPSHOT | grep -w " id " | awk '{print $4}')

if [ -z "$SNAPSHOT_ID" ]; then
    echo "Cannot find snapshot $SNAPSHOT_ID"
    exit 1
fi

echo "To clean snapshot $SNAPSHOT_NAME($SNAPSHOT_ID)..."

clean_snapshot() {
    local SNAP_ID=$1
    climc snapshot-purge $SNAP_ID > /dev/null
}

clean_snapshot $SNAPSHOT_ID
