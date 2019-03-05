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

func (r *SResourceGroup) GetName() string {
	return r.Name
}

func (r *SResourceGroup) GetId() string {
	return r.ID
}

func (r *SResourceGroup) GetGlobalId() string {
	return r.ID
}

func (r *SResourceGroup) GetStatus() string {
	return ""
}

func (r *SResourceGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (r *SResourceGroup) IsEmulated() bool {
	return false
}

func (r *SResourceGroup) Refresh() error {
	return nil
}
