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
	"strings"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/billing"

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
		params["clusterId"] = clusterId
	}
	if len(clusterId) > 0 {
		params["nodeId"] = nodeId
	}
	resp, err := self.invoke("DescribeNodes", params)
	if err != nil {
		return nil, err
	}
	ret := []SNode{}
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

func (self *SNode) GetInstanceById(instanceId string) (*SInstance, error) {
	instance, err := self.cluster.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}

	instance.node = self
	return instance, nil
}

// CreateVM 创建虚拟机
//{
//    "Tag.1.Key":"测试",
//    "Action":"RunInstances",
//    "MaxCount":"1",
//    "SubnetId":"subnet-A4798336",
//    "RootDevicePersistent":"true",
//    "InstanceName":"我的 Demo 实例",
//    "VpcId":"vpc-D37DF705",
//    "OwnerId":"zhouqifeng",
//    "Version":"2011-11-01",
//    "Tag.1.Value":"test",
//    "ImageId":"ami-03F1FC93",
//    "InstanceType":"m1.small",
//    "MinCount":"1",
//    "SecurityGroup.1":"default"
//}
func (self *SNode) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(desc.Name, desc.Hostname, desc.ExternalImageId, desc.SysDisk, desc.Cpu, desc.MemoryMB,
		desc.InstanceType, desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password,
		desc.DataDisks, desc.PublicKey, desc.ExternalSecgroupId, desc.UserData, desc.BillingCycle,
		desc.ProjectId, desc.OsType, desc.Tags, desc.SPublicIpInfo)
	if err != nil {
		return nil, err
	}
	vm, err := self.GetInstanceById(strings.Trim(vmId, "\""))
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceById")
	}
	return vm, nil
	//return nil, cloudprovider.ErrNotImplemented
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
	vms := []SInstance{}
	part, nextToken, err := self.cluster.region.GetInstances("", self.NodeId, MAX_RESULT, "")
	vms = append(vms, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.cluster.region.GetInstances("", self.NodeId, MAX_RESULT, nextToken)
		if err != nil {
			return nil, err
		}
		vms = append(vms, part...)
	}
	ret := []cloudprovider.ICloudVM{}
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
	return self.cluster.GetIWires()
	//return nil, cloudprovider.ErrNotImplemented
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
	ret := []cloudprovider.ICloudHost{}
	for i := range nodes {
		nodes[i].cluster = self
		ret = append(ret, &nodes[i])
	}
	return ret, nil
}

func (self *SNode) _createVM(name, hostname string, imgId string,
	sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string,
	networkId string, ipAddr string, desc string, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle, projectId, osType string,
	tags map[string]string, publicIp cloudprovider.SPublicIpInfo,
) (string, error) {
	net := self.cluster.getNetworkById(networkId)
	if net == nil {
		return "", fmt.Errorf("invalid network ID %s", networkId)
	}
	if net.wire == nil {
		log.Errorf("network's wire is empty")
		return "", fmt.Errorf("network's wire is empty")
	}

	if net.wire.vpc == nil {
		log.Errorf("wire's vpc is empty")
		return "", fmt.Errorf("wire's vpc is empty")
	}

	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.cluster.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	img, err := self.cluster.region.GetImage(imgId)
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return "", err
	}
	if img.ImageState != "available" {
		log.Errorf("image %s state %s", imgId, img.ImageState)
		return "", fmt.Errorf("image not ready")
	}
	tmpStorages := make(map[string]cloudprovider.ICloudStorage)
	storages, err := self.GetIStorages()
	if err != nil {
		log.Errorf("GetIStorages err %s", err)
		return "", fmt.Errorf("GetIStorages err")
	}
	for i := range storages {
		tmpStorages[storages[i].GetGlobalId()] = storages[i]
	}
	disks := make([]SDisk, len(dataDisks)+1)
	storage, ok := tmpStorages[sysDisk.StorageExternalId]
	if !ok {
		log.Errorf("GetIStorages %s for sysDisk is nil", sysDisk.StorageExternalId)
		return "", fmt.Errorf("GetIStorages err")
	}

	//系统盘
	disks[0].Size = int(img.GetSizeGB())
	disks[0].StorageId = storage.GetId()
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > int(img.GetSizeGB()) {
		disks[0].Size = sysDisk.SizeGB
	}

	//数据盘
	for i := range dataDisks {
		storage, ok := tmpStorages[dataDisks[i].StorageExternalId]
		if !ok {
			log.Errorf("GetIStorages %s for dataDisk is nil", dataDisks[i].StorageExternalId)
			return "", fmt.Errorf("GetIStorages err")
		}
		disks[i+1].StorageId = storage.GetId()
		disks[i+1].Size = dataDisks[i].SizeGB
	}

	securityGroup, err := self.cluster.region.GetISecurityGroupById(secgroupId)
	if err != nil {
		return "", errors.Wrap(err, "SHost.CreateVM.GetSecurityGroupDetails")
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.cluster.region.CreateInstance(name, self.NodeId, img, instanceType, networkId, securityGroup.GetName(), net.VpcId, self.cluster.GetId(), desc, disks, ipAddr, keypair, publicKey, passwd, userData, bc, projectId, tags)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", err
		} else {
			return vmId, nil
		}
	}
	return "", fmt.Errorf("%s", "Failed to create, instance type should not be empty")
}
