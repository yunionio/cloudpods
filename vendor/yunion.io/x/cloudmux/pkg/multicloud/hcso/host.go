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

package hcso

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
	vms  []SInstance

	// 华为私有云没有直接列出host的接口，所有账号下的host都是通过VM反向解析出来的
	// 当账号下没有虚拟机时，如果没有host，会导致调度找不到可用的HOST。
	// 因此，为了避免上述情况始终会在每个zone下返回一台虚拟的host
	IsFake    bool
	projectId string
	Id        string
	Name      string
}

func (self *SHost) GetId() string {
	return self.Id
}

func (self *SHost) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}

	return self.Id
}

func (self *SHost) GetGlobalId() string {
	return self.Id
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	_, err := self.getVMs()
	return errors.Wrap(err, "getVMs")
}

func (self *SHost) IsEmulated() bool {
	return self.IsFake
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	var vms []SInstance
	var err error
	if self.vms != nil {
		vms = self.vms
	} else {
		vms, err = self.getVMs()
		if err != nil {
			return nil, err
		}
	}

	ret := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		vm := vms[i]
		vm.host = self
		ret[i] = &vm
	}

	return ret, nil
}

func (self *SHost) getVMs() ([]SInstance, error) {
	vms, err := self.zone.region.GetInstances()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}

	ret := []SInstance{}
	for i := range vms {
		vm := vms[i]
		if vm.OSEXTAZAvailabilityZone == self.GetId() && vm.HostID == self.GetId() {
			vm.host = self
			ret = append(ret, vm)
		}
	}

	self.vms = ret
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstanceByID(id)
	if vm.HostID != self.GetId() {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetInstanceByID")
	}

	vm.host = self
	return &vm, err
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_HUAWEI), "manufacture")
	info.Add(jsonutils.NewString(self.GetId()), "id")
	info.Add(jsonutils.NewString(self.GetName()), "name")
	return info
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetCpuCount() int {
	return 0
}

func (self *SHost) GetNodeCount() int8 {
	return 0
}

func (self *SHost) GetCpuDesc() string {
	return ""
}

func (self *SHost) GetCpuMhz() int {
	return 0
}

func (self *SHost) GetMemSizeMB() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_HCSO
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	instance, err := self.zone.region.GetInstanceByID(instanceId)
	if err != nil {
		return nil, err
	}

	if instance.HostID != self.GetId() {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetInstanceByID")
	}

	instance.host = self
	return &instance, nil
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(
		desc.Name, desc.ExternalImageId, desc.SysDisk,
		desc.Cpu, desc.MemoryMB, desc.InstanceType,
		desc.ExternalNetworkId, desc.IpAddr,
		desc.Description, desc.Account,
		desc.Password, desc.DataDisks,
		desc.PublicKey, desc.ExternalSecgroupIds,
		desc.UserData, desc.BillingCycle, desc.ProjectId, desc.Tags)
	if err != nil {
		return nil, err
	}

	// VM实际调度到的host， 可能不是当前host.因此需要改写host信息
	vm, err := self.zone.region.GetIVMById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) _createVM(name string, imgId string, sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string,
	networkId string, ipAddr string, desc string, account string, passwd string,
	diskSizes []cloudprovider.SDiskInfo, publicKey string, secgroupIds []string,
	userData string, bc *billing.SBillingCycle, projectId string, tags map[string]string) (string, error) {
	net := self.zone.getNetworkById(networkId)
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

	// 同步keypair
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	//  镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("getiamge %s fail %s", imgId, err)
		return "", err
	}
	if img.Status != ImageStatusActive {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}
	// passwd, windows机型直接使用密码比较方便
	if strings.ToLower(img.Platform) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) && len(passwd) > 0 {
		keypair = ""
	}

	if strings.ToLower(img.Platform) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		if u, err := updateWindowsUserData(userData, img.OSVersion, account, passwd); err == nil {
			userData = u
		} else {
			return "", errors.Wrap(err, "SHost.CreateVM.updateWindowsUserData")
		}
	}

	disks := make([]SDisk, len(diskSizes)+1)
	disks[0].SizeGB = img.SizeGB
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > img.SizeGB {
		disks[0].SizeGB = sysDisk.SizeGB
	}
	disks[0].VolumeType = sysDisk.StorageType

	for i, dataDisk := range diskSizes {
		disks[i+1].SizeGB = dataDisk.SizeGB
		disks[i+1].VolumeType = dataDisk.StorageType
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupIds, net.VpcID, self.zone.GetId(), desc, disks, ipAddr, keypair, publicKey, passwd, userData, bc, projectId, tags)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("create %s failed:%s", instanceType, ErrMessage(err))
		} else {
			return vmId, nil
		}
	}

	// 匹配实例类型
	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, self.zone.GetId())
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	var vmId string
	for _, instType := range instanceTypes {
		instanceTypeId := instType.Name
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vmId, err = self.zone.region.CreateInstance(name, imgId, instanceTypeId, networkId, secgroupIds, net.VpcID, self.zone.GetId(), desc, disks, ipAddr, keypair, publicKey, passwd, userData, bc, projectId, tags)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("create failed: %s", ErrMessage(err))
}
