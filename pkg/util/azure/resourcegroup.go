package azure

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"yunion.io/x/log"
)

const (
	DISK_RESOURCE     = "Microsoft.Compute/disks"
	INSTANCE_RESOURCE = "Microsoft.Compute/virtualMachines"
	VPC_RESOURCE      = "Microsoft.Network/virtualNetworks"
	NIC_RESOURCE      = "Microsoft.Network/networkInterfaces"
	IMAGE_RESOURCE    = "Microsoft.Compute/images"
	STORAGE_RESOURCE  = "Microsoft.Storage/storageAccounts"
	SECGRP_RESOURCE   = "Microsoft.Network/networkSecurityGroups"
	EIP_RESOURCE      = "Microsoft.Network/publicIPAddresses"
	SNAPSHOT_RESOURCE = "Microsoft.Compute/snapshots"
)

var defaultResourceGroups = map[string]string{
	DISK_RESOURCE:     "YunionDiskResource",
	INSTANCE_RESOURCE: "YunionInstanceResource",
	VPC_RESOURCE:      "YunionVpcResource",
	NIC_RESOURCE:      "YunionNicInterface",
	IMAGE_RESOURCE:    "YunionImageResource",
	STORAGE_RESOURCE:  "YunionStorageResource",
	SECGRP_RESOURCE:   "YunionSecgrpResource",
	EIP_RESOURCE:      "YunionEipResource",
	SNAPSHOT_RESOURCE: "YunionSnapshotResource",
}

type GroupProperties struct {
	ProvisioningState string
}

type SResourceGroup struct {
	ID         string
	Name       string
	Location   string
	Properties GroupProperties
	ManagedBy  string
}

func pareResourceGroupWithName(s string, resourceType string) (string, string, string) {
	valid := regexp.MustCompile("resourceGroups/(.+)/providers/.+/(.+)$")
	if resourceGroups := valid.FindStringSubmatch(s); len(resourceGroups) == 3 {
		globalId := fmt.Sprintf("resourceGroups/%s/providers/%s/%s", resourceGroups[1], resourceType, resourceGroups[2])
		return globalId, resourceGroups[1], resourceGroups[2]
	}
	if len(s) == 0 {
		log.Errorf("pareResourceGroupWithName[%s] error", resourceType)
	}
	globalId := fmt.Sprintf("resourceGroups/%s/providers/%s/%s", defaultResourceGroups[resourceType], resourceType, s)
	return globalId, defaultResourceGroups[resourceType], s
}

func (self *SRegion) GetResourceGroups() ([]SResourceGroup, error) {
	resourceGroups := []SResourceGroup{}
	return resourceGroups, self.client.List("resourcegroups", &resourceGroups)
}

func (self *SRegion) GetResourceGroupDetail(groupName string) (*SResourceGroup, error) {
	resourceGroup := SResourceGroup{}
	return &resourceGroup, self.client.Get("resourcegroups/"+groupName, &resourceGroup)
}

func (self *SRegion) CreateResourceGroup(groupName string) (*SResourceGroup, error) {
	group, err := self.GetResourceGroupDetail(groupName)
	if err == nil {
		return group, nil
	}

	groupClient := resources.NewGroupsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	groupClient.Authorizer = self.client.authorizer
	params := resources.Group{
		Name:     &groupName,
		Location: &self.Name,
	}
	if _, err := groupClient.CreateOrUpdate(context.Background(), groupName, params); err != nil {
		return nil, err
	}
	return self.GetResourceGroupDetail(groupName)
}
