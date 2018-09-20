package azure

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	DISK_RESOURCE     = "disk"
	INSTANCE_RESOURCE = "instance"
	VPC_RESOURCE      = "vpc"
	NIC_RESOURCE      = "nic"
	IMAGE_RESOURCE    = "image"
	STORAGE_RESOURCE  = "storage"
	SECGRP_RESOURCE   = "secgroup"
	EIP_RESOURCE      = "eip"
	SNAPSHOT_RESOURCE = "snapshot"
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
	groupClient := resources.NewGroupsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	groupClient.Authorizer = self.client.authorizer
	if result, err := groupClient.List(context.Background(), "", nil); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&resourceGroups, result.Values()); err != nil {
		return nil, err
	}
	return resourceGroups, nil
}

func (self *SRegion) GetResourceGroupDetail(groupName string) (*SResourceGroup, error) {
	resourceGroup := SResourceGroup{}
	groupClient := resources.NewGroupsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	groupClient.Authorizer = self.client.authorizer
	if result, err := groupClient.Get(context.Background(), groupName); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if jsonutils.Update(&resourceGroup, result); err != nil {
		return nil, err
	}
	return &resourceGroup, nil
}

func (self *SRegion) CreateResourceGroup(groupName string) (*SResourceGroup, error) {
	if group, err := self.GetResourceGroupDetail(groupName); err == nil {
		return group, nil
	} else {
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
}
