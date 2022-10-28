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
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeNode struct {
	multicloud.SResourceBase
	QcloudTags

	cluster *SKubeCluster

	InstanceId               string
	InstanceRole             string
	FailedReason             string
	InstanceState            string
	DrainStatus              string
	InstanceAdvancedSettings struct {
		MountTarget        string
		DockerGraphPath    string
		UserScript         string
		Unschedulable      int
		DesiredPodNumber   int
		PreStartUserScript string
		Labels             []struct {
			Name  string
			Value string
		}
		DataDisks []struct {
			DiskType           string
			FileSystem         string
			DiskSize           int
			AutoFormatAndMount bool
			MountTarget        string
			DiskPartition      string
		}
		ExtraArgs struct {
			Kubelet []string
		}
	}
	CreatedTime        string
	LanIP              string
	NodePoolId         string
	AutoscalingGroupId string
}

func (self *SKubeNode) GetId() string {
	return self.InstanceId
}

func (self *SKubeNode) GetGlobalId() string {
	return self.InstanceId
}

func (self *SKubeNode) GetName() string {
	return self.InstanceId
}

func (self *SKubeNode) GetStatus() string {
	return self.DrainStatus
}

func (self *SKubeNode) GetINodePoolId() string {
	return self.NodePoolId
}

func (self *SKubeCluster) GetIKubeNodes() ([]cloudprovider.ICloudKubeNode, error) {
	nodes := []SKubeNode{}
	for {
		part, total, err := self.region.GetKubeNodes(self.ClusterId, 100, len(nodes)/100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIKubeNodes")
		}
		nodes = append(nodes, part...)
		if len(nodes) >= total || len(part) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudKubeNode{}
	for i := range nodes {
		nodes[i].cluster = self
		ret = append(ret, &nodes[i])
	}
	return ret, nil
}

func (self *SRegion) GetKubeNodes(clusterId string, limit, offset int) ([]SKubeNode, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Limit":     fmt.Sprintf("%d", limit),
		"Offset":    fmt.Sprintf("%d", offset),
		"ClusterId": clusterId,
	}
	resp, err := self.tkeRequest("DescribeClusterInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeClusterInstances")
	}
	nodes := []SKubeNode{}
	err = resp.Unmarshal(&nodes, "InstanceSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return nodes, int(totalCount), nil
}
