#/bin/bash

set -e

EIP=$1

if [ -z "$EIP" ]; then
    echo "Usage: $0 <eip_id_or_name>"
    exit 1
fi

EIP_NAME=$(climc eip-show $EIP | grep -w " name " | awk '{print $4}')
EIP_ID=$(climc eip-show $EIP | grep -w " id " | awk '{print $4}')

if [ -z "$EIP_ID" ]; then
    echo "Cannot find eip $EIP_ID"
    exit 1
fi

echo "To clean eip $EIP_NAME($EIP_ID)..."

clean_eip() {
    local EID=$1
    climc eip-purge $EID
}

clean_eip $EIP_ID
