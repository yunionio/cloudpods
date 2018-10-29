package azure

import (
	"yunion.io/x/jsonutils"
)

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

func (self *SRegion) GetResourceGroups() ([]SResourceGroup, error) {
	resourceGroups := []SResourceGroup{}
	return resourceGroups, self.client.List("resourcegroups", &resourceGroups)
}

func (self *SRegion) GetResourceGroupDetail(groupName string) (*SResourceGroup, error) {
	resourceGroup := SResourceGroup{}
	return &resourceGroup, self.client.Get("resourcegroups/"+groupName, []string{}, &resourceGroup)
}

func (self *SRegion) CreateResourceGroup(groupName string) (*SResourceGroup, error) {
	resourceGroup := SResourceGroup{Name: groupName, Location: self.Name}
	return &resourceGroup, self.client.Create(jsonutils.Marshal(resourceGroup), &resourceGroup)
}
