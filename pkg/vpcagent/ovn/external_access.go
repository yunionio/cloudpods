package ovn

import (
	apis "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

func vpcHasDistgw(vpc *agentmodels.Vpc) bool {
	mode := vpc.ExternalAccessMode
	switch mode {
	case
		apis.VPC_EXTERNAL_ACCESS_MODE_DISTGW,
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW:
		return true
	default:
		return false
	}
}

func vpcHasEipgw(vpc *agentmodels.Vpc) bool {
	mode := vpc.ExternalAccessMode
	switch mode {
	case
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP,
		apis.VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW:
		return true
	default:
		return false
	}
}
