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

package aliyun

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeNode struct {
	multicloud.SResourceBase
	AliyunTags

	cluster *SKubeCluster

	HostName           string
	ImageId            string
	InstanceChargeType string
	InstanceId         string
	InstanceName       string
	InstanceRole       string
	InstanceStatus     string
	InstanceType       string
	InstanceTypeFamily string
	IpAddress          []string
	IsAliyunNode       bool
	NodeName           string
	NodeStatus         string
	NodepoolId         string
	Source             string
	State              string
	SpotStrategy       string
}

func (self *SKubeNode) GetGlobalId() string {
	return self.NodeName
}

func (self *SKubeNode) GetId() string {
	return self.NodeName
}

func (self *SKubeNode) GetName() string {
	return self.NodeName
}

func (self *SKubeNode) GetStatus() string {
	return self.NodeStatus
}

func (self *SKubeNode) GetINodePoolId() string {
	return self.NodepoolId
}

func (self *SRegion) GetKubeNodes(clusterId string, pageSize, pageNumber int) ([]SKubeNode, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	params := map[string]string{
		"page_size":   fmt.Sprintf("%d", pageSize),
		"page_number": fmt.Sprintf("%d", pageNumber),
		"PathPattern": fmt.Sprintf("/clusters/%s/nodes", clusterId),
	}
	resp, err := self.k8sRequest("DescribeClusterNodes", params, nil)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeClusterNodes")
	}
	nodes := []SKubeNode{}
	err = resp.Unmarshal(&nodes, "nodes")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCnt, _ := resp.Int("page", "total_count")
	return nodes, int(totalCnt), nil
}

func (self *SKubeCluster) GetIKubeNodes() ([]cloudprovider.ICloudKubeNode, error) {
	nodes := []SKubeNode{}
	for {
		part, total, err := self.region.GetKubeNodes(self.ClusterId, 100, len(nodes)/100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetKubeNodes")
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
