#/bin/bash

set -e

VPC=$1

if [ -z "$VPC" ]; then
    echo "Usage: $0 <vpc_id_or_name>"
    exit 1
fi

VPC_NAME=$(climc vpc-show $VPC | grep -w " name " | awk '{print $4}')
VPC_ID=$(climc vpc-show $VPC | grep -w " id " | awk '{print $4}')

if [ -z "$VPC_ID" ]; then
    echo "Cannot find vpc $VPC"
    exit 1
fi

echo "To clean vpc $VPC_NAME($VPC_ID)..."

clean_network() {
    climc network-purge $1 > /dev/null
}


clean_wire() {
    local WIRE_ID=$1
    for net_id in $(climc network-list --admin --wire $WIRE_ID --show-emulated | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
    do
        clean_network $net_id
    done
    climc wire-delete $wire_id > /dev/null
}

clean_vpc() {
    local VPC_ID=$1
    climc vpc-purge $VPC_ID > /dev/null
}

echo "1. Clean wires ..."

for wire_id in $(climc wire-list --vpc $VPC_ID --show-emulated | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    clean_wire $wire_id
done

echo "2. Clean route tables ..."
for routetable_id in $(climc routetable-list --vpc $VPC_ID --show-emulated | egrep '[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}' | awk '{print $2}')
do
    climc routetable-purge $routetable_id >/dev/null
done

echo "3. clean vpc ..."
clean_vpc $VPC_ID
