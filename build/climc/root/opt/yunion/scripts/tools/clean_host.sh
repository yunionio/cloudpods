#!/bin/bash

set -e

HOST=$1

if [ -z "$HOST" ]; then
    echo "Usage: $0 <host_id_or_name>"
    exit 1
fi

error_exit() {
    echo "Error: $1"
    exit 1
}

HOSTID=$(climc host-show $HOST | grep -w " id " | awk '{print $4}')

HOSTNAME=$(climc host-show $HOST | grep -w " name " | awk '{print $4}')

if [ -z "$HOSTNAME" ]; then
    error_exit "Cannot find host $HOST"
fi

if [ -z "$HOSTID" ]; then
    error_exit "Cannot find host $HOST"
fi

echo "Ready to purge host $HOSTNAME($HOSTID)..."

echo "1. Disable $HOSTNAME($HOSTID)"

if ! climc host-disable $HOSTID > /dev/null; then
    error_exit "Fail to disable $HOST"
fi

echo "2. Clean servers on $HOSTNAME($HOSTID)"

for i in $(climc server-list --host $HOSTID --admin --limit 2048 --system | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc server-update --delete enable $i > /dev/null
    climc server-purge $i > /dev/null
done

for i in $(climc server-list --host $HOSTID --admin --limit 2048 --pending-delete --system | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc server-purge $i > /dev/null
done

for i in $(climc server-list --admin --limit 2048 --system  --filter "backup_host_id.equals($HOSTID)" | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc server-delete-backup $i > /dev/null
done

echo "3. Clean disks on $HOSTNAME($HOSTID)"

for storageid in $(climc host-storage-list --host $HOSTID --show-emulated | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $4}')
do
    STORAGENAME=$(climc storage-show $storageid | grep -w " name " | awk '{print $4}')
    STORAGE_TYPE=$(climc storage-show $storageid | grep -w " storage_type " | awk '{print $4}')
    if [ "$STORAGE_TYPE" == "local" ]; then
        for diskid in $(climc disk-list --admin --system --details --search $storageid --limit 2048 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
        do
            climc disk-purge $diskid > /dev/null
        done
        for diskid in $(climc disk-list --admin --system --details --search $storageid --pending-delete --limit 2048 | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
        do
            climc disk-purge $diskid > /dev/null
        done
    fi
done

echo "4. Remove host $HOSTNAME($HOSTID)"

BAREMETAL=$(climc host-show $HOSTID | grep -w " is_baremetal " | awk '{print $4}')

HOSTTYPE=$(climc host-show $HOSTID | grep -w " host_type " | awk '{print $4}')

if [ "$BAREMETAL" == "true" ] && [ "$HOSTTYPE" != "baremetal" ]; then
    climc host-update --host-type baremetal $HOSTID > /dev/null
fi

climc host-delete $HOSTID > /dev/null

echo "DONE."
