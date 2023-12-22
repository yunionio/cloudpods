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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	DiskMax      int64
	DiskNode     int64
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

func (self *SNode) GetStorageSizeMB() int64 {
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

func (node *SNode) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := node.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "getIWires")
	}
	return cloudprovider.GetHostNetifs(node, wires), nil
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

func (self *SNode) getIWires() ([]cloudprovider.ICloudWire, error) {
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
	var err error
	log.Debugf("Try instancetype : %s", desc.InstanceType)

	img, err := self.cluster.region.GetImageById(desc.ExternalImageId)
	if err != nil {
		log.Errorf("get image %s fail %s", desc.ExternalImageId, err)
		return nil, err
	}

	params := map[string]string{}
	params["InstanceName"] = desc.Name
	params["ImageId"] = desc.ExternalImageId
	params["MinCount"] = "1"
	params["MaxCount"] = "1"

	if len(desc.ExternalSecgroupIds) > 0 {
		for i := range desc.ExternalSecgroupIds {
			secGroup, err := self.cluster.region.GetISecurityGroupById(desc.ExternalSecgroupIds[i])
			if err != nil {
				return nil, err
			}
			params[fmt.Sprintf("SecurityGroup.%v", i+1)] = secGroup.GetName()
		}
	}

	disks := make([]SDisk, len(desc.DataDisks)+1)
	disks[0].Size = int(img.GetSizeGB())
	if desc.SysDisk.SizeGB > 0 && desc.SysDisk.SizeGB > int(img.GetSizeGB()) {
		disks[0].Size = desc.SysDisk.SizeGB
	}
	for i, dataDisk := range desc.DataDisks {
		disks[i+1].Size = dataDisk.SizeGB
	}

	for i, disk := range disks {
		var deviceName string
		var err error

		if i == 0 {
			params[fmt.Sprintf("BlockDeviceMapping.%v.Ebs.DeleteOnTermination", i+1)] = "true"
			params[fmt.Sprintf("BlockDeviceMapping.%v.Ebs.VolumeSize", i+1)] = strconv.Itoa(disk.Size)
			if len(img.RootDeviceName) > 0 {
				deviceName = img.RootDeviceName
			} else {
				deviceName = fmt.Sprintf("/dev/vda")
			}
		} else {
			params[fmt.Sprintf("BlockDeviceMapping.%v.Ebs.DeleteOnTermination", i+1)] = "true"
			params[fmt.Sprintf("BlockDeviceMapping.%v.Ebs.VolumeSize", i+1)] = strconv.Itoa(disk.Size)
			deviceName, err = nextDeviceName([]string{deviceName})
			if err != nil {
				return nil, errors.Wrap(err, "nextDeviceName")
			}
		}
		params[fmt.Sprintf("BlockDeviceMapping.%v.DeviceName", i+1)] = deviceName
	}

	params["InstanceType"] = desc.InstanceType
	params["Password"] = desc.Password
	params["NetworkInterface.1.VpcId"] = desc.ExternalVpcId
	params["NetworkInterface.1.SubnetId"] = desc.ExternalNetworkId
	params["AllowNodes.1"] = self.NodeId

	resp, err := self.cluster.region.invoke("RunInstances", params)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrap(err, "CreateVM")
	}

	var tmpInst = struct {
		InstancesSet struct {
			InstanceId string
		}
	}{}

	_ = resp.Unmarshal(&tmpInst)
	insets, _, err := self.cluster.region.GetInstances(tmpInst.InstancesSet.InstanceId, "", MAX_RESULT, "")
	if err != nil {
		log.Errorf("GetInstance %s: %s", "", err)
		return nil, errors.Wrap(err, "CreateVM")
	}
	if len(insets) > 0 {
		insets[0].node = self
		return &insets[0], nil
	}
	return nil, errors.Wrap(cloudprovider.ErrUnknown, "CreateVM")
}
