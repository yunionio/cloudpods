package ovn

import (
	"fmt"
)

func vpcLrName(vpcId string) string {
	return fmt.Sprintf("vpc-r/%s", vpcId)
}

// ext
func vpcExtLrName(vpcId string) string {
	return fmt.Sprintf("vpc-ext-r/%s", vpcId)
}

func vpcExtLsName(vpcId string) string {
	return fmt.Sprintf("vpc-ext/%s", vpcId)
}

func vpcR1extpName(vpcId string) string {
	return fmt.Sprintf("vpc-r1ext/%s", vpcId)
}

func vpcExtr1pName(vpcId string) string {
	return fmt.Sprintf("vpc-extr1/%s", vpcId)
}

func vpcR2extpName(vpcId string) string {
	return fmt.Sprintf("vpc-r2ext/%s", vpcId)
}

func vpcExtr2pName(vpcId string) string {
	return fmt.Sprintf("vpc-extr2/%s", vpcId)
}

// distgw
func vpcHostLsName(vpcId string) string {
	return fmt.Sprintf("vpc-h/%s", vpcId)
}

func vpcRhpName(vpcId string) string {
	return fmt.Sprintf("vpc-rh/%s", vpcId)
}

func vpcHrpName(vpcId string) string {
	return fmt.Sprintf("vpc-hr/%s", vpcId)
}

func vpcHostLspName(vpcId string, hostId string) string {
	return fmt.Sprintf("vpc-h/%s/%s", vpcId, hostId)
}

// eipgw
func vpcEipLsName(vpcId string) string {
	return fmt.Sprintf("vpc-e/%s", vpcId)
}

func vpcRepName(vpcId string) string {
	return fmt.Sprintf("vpc-re/%s", vpcId)
}

func vpcErpName(vpcId string) string {
	return fmt.Sprintf("vpc-er/%s", vpcId)
}

func vpcEipLspName(vpcId string, eipgwId string) string {
	return fmt.Sprintf("vpc-ep/%s/%s", vpcId, eipgwId)
}

func netLsName(netId string) string {
	return fmt.Sprintf("subnet/%s", netId)
}

func netNrpName(netId string) string {
	return fmt.Sprintf("subnet-nr/%s", netId)
}

func netRnpName(netId string) string {
	return fmt.Sprintf("subnet-rn/%s", netId)
}

func netMdpName(netId string) string {
	return fmt.Sprintf("subnet-md/%s", netId)
}

// gnpName returns Logical_Switch_Port name for guestnetwork
//
// The name must match what's going to be set on each chassis
func gnpName(netId string, ifname string) string {
	return fmt.Sprintf("iface-%s-%s", netId, ifname)
}
