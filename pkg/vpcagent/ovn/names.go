package ovn

import (
	"fmt"
)

func vpcLrName(vpcId string) string {
	return fmt.Sprintf("vpc-r-%s", vpcId)
}

func vpcHostLsName(vpcId string) string {
	return fmt.Sprintf("vpc-h-%s", vpcId)
}

func vpcRhpName(vpcId string) string {
	return fmt.Sprintf("vpc-rh-%s", vpcId)
}

func vpcHrpName(vpcId string) string {
	return fmt.Sprintf("vpc-hr-%s", vpcId)
}

func netLsName(netId string) string {
	return fmt.Sprintf("subnet-%s", netId)
}

func netNrpName(netId string) string {
	return fmt.Sprintf("subnet-nr-%s", netId)
}

func netRnpName(netId string) string {
	return fmt.Sprintf("subnet-rn-%s", netId)
}

func netMdpName(netId string) string {
	return fmt.Sprintf("subnet-md-%s", netId)
}

func gnpName(netId string, ifname string) string {
	return fmt.Sprintf("iface-%s-%s", netId, ifname)
}
