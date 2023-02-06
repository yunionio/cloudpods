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

package bingocloud

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SNode struct {
	multicloud.SHostBase
	multicloud.STagBase

	cluster *SCluster

	ClusterId    string
	CpuCores     int
	CpuMax       int
	CpuModel     string
	CpuNode      int
	CpuSockets   int
	CpuUsed      int
	DiskMax      int
	DiskNode     int
	DiskUsed     int
	MemNode      int
	MemoryMax    int
	MemoryUsed   int
	NodeId       string
	NodeName     string
	ScheduleTags string
	Status       string
}

func (self *SRegion) GetNodes(clusterId, nodeId string) ([]SNode, error) {
	params := map[string]string{}
	if len(clusterId) > 0 {
		params["ClusterId"] = clusterId
	}
	if len(clusterId) > 0 {
		params["NodeId"] = nodeId
	}
	resp, err := self.invoke("DescribeNodes", params)
	if err != nil {
		return nil, err
	}
	var ret []SNode
	return ret, resp.Unmarshal(&ret, "nodeSet")
}

func (self *SNode) GetId() string {
	return self.NodeId
}

func (self *SNode) GetGlobalId() string {
	return self.NodeId
}

func (self *SNode) GetName() string {
	return self.NodeName
}

func (self *SNode) GetAccessIp() string {
	info := strings.Split(self.NodeId, "@")
	if len(info) == 2 {
		return info[1]
	}
	return ""
}

func (self *SNode) GetAccessMac() string {
	return ""
}

func (self *SNode) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SNode) GetSN() string {
	return ""
}

func (self *SNode) GetCpuCount() int {
	return self.CpuNode
}

func (self *SNode) GetNodeCount() int8 {
	return int8(self.CpuSockets)
}

func (self *SNode) GetCpuDesc() string {
	return self.CpuModel
}

func (self *SNode) GetCpuMhz() int {
	return 0
}

func (self *SNode) GetCpuCmtbound() float32 {
	if self.CpuMax > self.CpuNode {
		return float32(self.CpuMax) / float32(self.CpuNode)
	}
	return 1
}

func (self *SNode) GetMemSizeMB() int {
	return self.MemNode
}

func (self *SNode) GetMemCmtbound() float32 {
	if self.MemoryMax > self.MemNode {
		return float32(self.MemoryMax) / float32(self.MemNode)
	}
	return 1
}

func (self *SNode) GetReservedMemoryMb() int {
	return 0
}

func (self *SNode) GetStorageSizeMB() int {
	if self.DiskMax > 0 {
		return self.DiskMax * 1024
	}
	return self.DiskNode * 1024
}

func (self *SNode) GetStorageType() string {
	return api.STORAGE_LOCAL_SSD
}

func (self *SNode) GetNodeType() string {
	return api.HOST_TYPE_BINGO_CLOUD
}

func (self *SNode) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.cluster.GetIStorages()
}

func (self *SNode) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.cluster.GetIStorageById(id)
}

func (self *SNode) GetEnabled() bool {
	return true
}

func (self *SNode) GetNodeStatus() string {
	if self.Status == "available" {
		return api.HOST_ONLINE
	}
	return api.HOST_OFFLINE
}

func (self *SNode) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SNode) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	var vms []SInstance
	part, nextToken, err := self.cluster.region.GetInstances("", self.NodeId, MAX_RESULT, "")
	vms = append(vms, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.cluster.region.GetInstances("", self.NodeId, MAX_RESULT, nextToken)
		if err != nil {
			return nil, err
		}
		vms = append(vms, part...)
	}
	var ret []cloudprovider.ICloudVM
	for i := range vms {
		vms[i].node = self
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (self *SNode) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, _, err := self.cluster.region.GetInstances(id, self.NodeId, 1, "")
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].GetGlobalId() == id {
			vms[i].node = self
			return &vms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SNode) GetIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.cluster.region.GetIVpcs()
	if err != nil {
		return nil, err
	}
	var ret []cloudprovider.ICloudWire
	for _, vpc := range vpcs {
		wires, err := vpc.GetIWires()
		if err != nil {
			return nil, err
		}
		ret = append(ret, wires...)
	}
	return ret, nil
}

func (self *SNode) GetSchedtags() ([]string, error) {
	return []string{}, nil
}

func (self *SNode) GetIsMaintenance() bool {
	return self.Status == "maintain"
}

func (self *SNode) GetVersion() string {
	return ""
}

func (self *SNode) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SNode) GetHostStatus() string {
	if self.Status != "available" {
		return api.HOST_OFFLINE
	}
	return api.HOST_ONLINE
}

func (self *SNode) GetHostType() string {
	return api.HOST_TYPE_BINGO_CLOUD
}

func (self *SCluster) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	nodes, err := self.region.GetNodes(self.ClusterId, id)
	if err != nil {
		return nil, err
	}
	for i := range nodes {
		if nodes[i].GetGlobalId() == id {
			nodes[i].cluster = self
			return &nodes[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SCluster) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	nodes, err := self.region.GetNodes(self.ClusterId, "")
	if err != nil {
		return nil, err
	}
	var ret []cloudprovider.ICloudHost
	for i := range nodes {
		nodes[i].cluster = self
		ret = append(ret, &nodes[i])
	}
	return ret, nil
}

func (self *SNode) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}
