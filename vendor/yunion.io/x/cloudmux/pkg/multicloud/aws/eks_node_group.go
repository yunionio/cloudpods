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

package aws

import (
	"fmt"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SNodeGroup struct {
	multicloud.SResourceBase
	region *SRegion
	AwsTags

	ClusterName   string
	NodegroupName string
	DiskSize      int
	Status        string
	InstanceTypes []string
	Subnets       []string
	ScalingConfig struct {
		DesiredSize int
		MaxSize     int
		MinSize     int
	}
}

func (self *SNodeGroup) GetId() string {
	return self.NodegroupName
}

func (self *SNodeGroup) GetName() string {
	return self.NodegroupName
}

func (self *SNodeGroup) GetGlobalId() string {
	return self.NodegroupName
}

func (self *SNodeGroup) Refresh() error {
	ng, err := self.region.GetNodegroup(self.ClusterName, self.NodegroupName)
	if err != nil {
		return err
	}
	self.InstanceTypes = nil
	self.Subnets = nil
	return jsonutils.Update(self, ng)
}

func (self *SNodeGroup) GetStatus() string {
	if len(self.Status) == 0 {
		self.Refresh()
	}
	switch strings.ToLower(self.Status) {
	case "active":
		return api.KUBE_CLUSTER_STATUS_RUNNING
	case "creating":
		return api.KUBE_CLUSTER_STATUS_CREATING
	}
	return strings.ToLower(self.Status)
}

func (self *SNodeGroup) GetMinInstanceCount() int {
	if len(self.Subnets) == 0 {
		self.Refresh()
	}
	return self.ScalingConfig.MinSize
}

func (self *SNodeGroup) GetMaxInstanceCount() int {
	if len(self.Subnets) == 0 {
		self.Refresh()
	}
	return self.ScalingConfig.MaxSize
}

func (self *SNodeGroup) GetDesiredInstanceCount() int {
	if len(self.Subnets) == 0 {
		self.Refresh()
	}
	return self.ScalingConfig.DesiredSize
}

func (self *SNodeGroup) GetRootDiskSizeGb() int {
	if len(self.Subnets) == 0 {
		self.Refresh()
	}
	return self.DiskSize
}

func (self *SNodeGroup) Delete() error {
	return self.region.DeleteNodegroup(self.ClusterName, self.NodegroupName)
}

func (self *SNodeGroup) GetNetworkIds() []string {
	if len(self.Subnets) == 0 {
		self.Refresh()
	}
	return self.Subnets
}

func (self *SNodeGroup) GetInstanceTypes() []string {
	if len(self.InstanceTypes) == 0 {
		self.Refresh()
	}
	return self.InstanceTypes
}

func (self *SRegion) GetNodegroup(cluster, name string) (*SNodeGroup, error) {
	params := map[string]interface{}{
		"name":          cluster,
		"nodegroupName": name,
	}
	ret := struct {
		Nodegroup SNodeGroup
	}{}
	err := self.eksRequest("DescribeNodegroup", "/clusters/{name}/node-groups/{nodegroupName}", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeNodegroup")
	}
	ret.Nodegroup.region = self
	return &ret.Nodegroup, nil
}

func (self *SRegion) GetNodegroups(cluster, nextToken string) ([]SNodeGroup, string, error) {
	params := map[string]interface{}{}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	ret := struct {
		Nodegroups []string
		NextToken  string
	}{}
	resource := fmt.Sprintf("/clusters/%s/node-groups", cluster)
	err := self.eksRequest("ListNodegroups", resource, params, &ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "DescribeCluster")
	}
	result := []SNodeGroup{}
	for i := range ret.Nodegroups {
		result = append(result, SNodeGroup{
			region:        self,
			ClusterName:   cluster,
			NodegroupName: ret.Nodegroups[i],
		})
	}
	return result, ret.NextToken, nil
}

func (self *SRegion) DeleteNodegroup(cluster, name string) error {
	params := map[string]interface{}{
		"name":          cluster,
		"nodegroupName": name,
	}
	ret := struct {
		Nodegroup SNodeGroup
	}{}
	return self.eksRequest("DeleteNodegroup", "/clusters/{name}/node-groups/{nodegroupName}", params, &ret)
}

func (self *SNodeGroup) GetDescription() string {
	return self.AwsTags.GetDescription()
}
