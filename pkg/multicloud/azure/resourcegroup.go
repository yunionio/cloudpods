// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"fmt"

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
	idStr := fmt.Sprintf("subscriptions/%s/resourcegroups/%s", self.SubscriptionID, groupName)
	return &resourceGroup, self.client.Get(idStr, []string{}, &resourceGroup)
}

// not support update, resource group name is immutable???
func (self *SRegion) UpdateResourceGroup(groupName string, newName string) error {
	resourceGroup := SResourceGroup{Name: newName}
	idStr := fmt.Sprintf("subscriptions/%s/resourcegroups/%s", self.SubscriptionID, groupName)
	return self.client.Patch(idStr, jsonutils.Marshal(&resourceGroup))
}

func (self *SRegion) CreateResourceGroup(groupName string) error {
	resourceGroup := SResourceGroup{Location: self.Name}
	idStr := fmt.Sprintf("subscriptions/%s/resourcegroups/%s", self.SubscriptionID, groupName)
	return self.client.Put(idStr, jsonutils.Marshal(resourceGroup))
}

func (self *SRegion) DeleteResourceGroup(groupName string) error {
	idStr := fmt.Sprintf("subscriptions/%s/resourcegroups/%s", self.SubscriptionID, groupName)
	return self.client.Delete(idStr)
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
	return r.Properties.ProvisioningState
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
