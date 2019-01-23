package cloudprovider

import (
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/util/secrules"
)

type SManagedVMCreateConfig struct {
	Name              string
	ExternalImageId   string
	OsDistribution    string
	OsVersion         string
	InstanceType      string // InstanceType 不为空时，直接采用InstanceType创建机器。
	Cpu               int
	Memory            int
	ExternalNetworkId string
	IpAddr            string
	Description       string
	StorageType       string
	SysDiskSize       int
	DataDisks         []int
	PublicKey         string
	SecGroupId        string
	SecGroupName      string
	SecRules          []secrules.SecurityRule

	BillingCycle billing.SBillingCycle
}
