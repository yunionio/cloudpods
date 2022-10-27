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

package qcloud

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeNodePool struct {
	multicloud.SResourceBase
	QcloudTags

	cluster *SKubeCluster

	NodePoolId            string
	Name                  string
	ClusterInstanceId     string
	LifeState             string
	LaunchConfigurationId string
	AutoscalingGroupId    string
	Labels                []struct {
		Name  string
		Value string
	}
	Taints []struct {
		Kye    string
		Value  string
		Effect string
	}
	NodeCountSummary struct {
		ManuallyAdded struct {
			Joining      int
			Initializing int
			Normal       int
			Total        int
		}
		AutoscalingAdded struct {
			Joining      int
			Initializing int
			Normal       int
			Total        int
		}
	}
	AutoscalingGroupStatus string
	MaxNodesNum            int
	MinNodesNum            int
	DesiredNodesNum        int
	NodePoolOs             string
	OsCustomizeType        string
	ImageId                string
	DesiredPodNum          int
	UserScript             string
}

func (self *SKubeNodePool) GetId() string {
	return self.NodePoolId
}

func (self *SKubeNodePool) GetGlobalId() string {
	return self.NodePoolId
}

func (self *SKubeNodePool) GetName() string {
	return self.Name
}

func (self *SKubeNodePool) GetStatus() string {
	return self.LifeState
}

func (self *SKubeCluster) GetIKubeNodePools() ([]cloudprovider.ICloudKubeNodePool, error) {
	pools, err := self.region.GetKubeNodePools(self.ClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeNodePools")
	}
	ret := []cloudprovider.ICloudKubeNodePool{}
	for i := range pools {
		pools[i].cluster = self
		ret = append(ret, &pools[i])
	}
	return ret, nil
}

func (self *SRegion) GetKubeNodePools(clusterId string) ([]SKubeNodePool, error) {
	params := map[string]string{
		"ClusterId": clusterId,
	}
	resp, err := self.tkeRequest("DescribeClusterNodePools", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeClusterNodePools")
	}
	pools := []SKubeNodePool{}
	err = resp.Unmarshal(&pools, "NodePoolSet")
	if err != nil {
		return nil, errors.Wrapf(err, "NodePoolSet")
	}
	return pools, nil
}
