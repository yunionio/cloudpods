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

package ucloud

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	projectId string
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Name, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return self.GetId()
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.GetInstances()
	if err != nil {
		return nil, err
	}

	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = self
		ivms[i] = &vms[i]
	}

	return ivms, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstanceByID(id)
	if err != nil {
		return nil, err
	}

	vm.host = self
	return &vm, nil
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
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_UCLOUD), "manufacture")
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
	return api.HOST_TYPE_UCLOUD
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return UCLOUD_API_VERSION
}

// 不支持user data
// 不支持指定keypair
func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.Cpu, desc.MemoryMB, desc.InstanceType, desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks, desc.ExternalSecgroupIds, desc.BillingCycle)
	if err != nil {
		return nil, err
	}

	vm, err := self.zone.region.GetInstanceByID(vmId)
	if err != nil {
		return nil, err
	}

	vm.host = self
	return &vm, err
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "getIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

type SInstanceType struct {
	UHostType string
	CPU       int
	MemoryMB  int
	GPU       int
}

func ParseInstanceType(instanceType string) (SInstanceType, error) {
	i := SInstanceType{}
	segs := strings.Split(instanceType, ".")
	if len(segs) < 3 {
		return i, fmt.Errorf("invalid instance type %s", instanceType)
	} else if len(segs) >= 4 {
		gpu, err := strconv.Atoi(strings.TrimLeft(segs[3], "g"))
		if err != nil {
			return i, err
		}

		i.GPU = gpu
	}

	cpu, err := strconv.Atoi(strings.TrimLeft(segs[1], "c"))
	if err != nil {
		return i, err
	}

	mem, err := strconv.Atoi(strings.TrimLeft(segs[2], "m"))
	if err != nil {
		return i, err
	}

	i.UHostType = segs[0]
	i.CPU = cpu
	i.MemoryMB = mem * 1024
	return i, nil
}

func (self *SHost) _createVM(name, imgId string, sysDisk cloudprovider.SDiskInfo, cpu, memMB int, instanceType string,
	networkId, ipAddr, desc, passwd string,
	dataDisks []cloudprovider.SDiskInfo, secgroupIds []string, bc *billing.SBillingCycle) (string, error) {
	// 网络配置及安全组绑定
	net, _ := self.zone.region.getNetwork(networkId)
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

	if len(passwd) == 0 {
		return "", fmt.Errorf("CreateVM password should not be emtpty")
	}

	// 镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("GetImage %s fail %s", imgId, err)
		return "", err
	}
	if img.GetStatus() != api.CACHED_IMAGE_STATUS_ACTIVE {
		log.Errorf("image %s status %s, expect %s", imgId, img.GetStatus(), api.CACHED_IMAGE_STATUS_ACTIVE)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(dataDisks)+1)
	disks[0].SizeGB = int(img.ImageSizeGB)
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > int(img.ImageSizeGB) {
		disks[0].SizeGB = sysDisk.SizeGB
	}
	disks[0].DiskType = sysDisk.StorageType

	for i, dataDisk := range dataDisks {
		disks[i+1].SizeGB = dataDisk.SizeGB
		disks[i+1].DiskType = dataDisk.StorageType
	}

	// 创建实例
	// https://docs.ucloud.cn/api/uhost-api/uhost_type
	// https://docs.ucloud.cn/compute/uhost/introduction/uhost/type
	var vmId string
	i, err := ParseInstanceType(instanceType)
	if err != nil {
		if cpu <= 0 || memMB <= 0 {
			return "", err
		} else {
			i.UHostType = "N2"
			i.CPU = cpu
			i.MemoryMB = memMB
		}
	}

	vmId, err = self.zone.region.CreateInstance(name, imgId, i.UHostType, passwd, net.wire.vpc.GetId(), networkId, secgroupIds, self.zone.ZoneId, desc, ipAddr, i.CPU, i.MemoryMB, i.GPU, disks, bc)
	if err != nil {
		return "", fmt.Errorf("Failed to create: %v", err)
	}

	return vmId, nil
}

// https://docs.ucloud.cn/api/uhost-api/create_uhost_instance
// https://docs.ucloud.cn/api/uhost-api/specification
// 支持8-30位字符, 不能包含[A-Z],[a-z],[0-9]和[()`~!@#$%^&*-+=_|{}[]:;'<>,.?/]之外的非法字符
func (self *SRegion) CreateInstance(name, imageId, hostType, password, vpcId, SubnetId string, securityGroupId []string,
	zoneId, desc, ipAddr string, cpu, memMB, gpu int, disks []SDisk, bc *billing.SBillingCycle) (string, error) {
	params := NewUcloudParams()
	params.Set("Zone", zoneId)
	params.Set("ImageId", imageId)
	params.Set("Password", base64.StdEncoding.EncodeToString([]byte(password)))
	params.Set("LoginMode", "Password")
	params.Set("Name", name)
	params.Set("UHostType", hostType)
	params.Set("CPU", cpu)
	params.Set("Memory", memMB)
	params.Set("VPCId", vpcId)
	params.Set("SubnetId", SubnetId)
	for _, id := range securityGroupId {
		params.Set("SecurityGroupId", id)
	}
	if gpu > 0 {
		params.Set("GPU", gpu)
	}

	if bc != nil && bc.GetMonths() >= 1 && bc.GetMonths() < 10 {
		params.Set("ChargeType", "Month")
		params.Set("Quantity", bc.GetMonths())
	} else if bc != nil && bc.GetMonths() >= 10 && bc.GetMonths() < 12 {
		params.Set("ChargeType", "Year")
		params.Set("Quantity", 1)
	} else if bc != nil && bc.GetYears() >= 1 {
		params.Set("ChargeType", "Year")
		params.Set("Quantity", bc.GetYears())
	} else {
		params.Set("ChargeType", "Dynamic")
	}

	// boot disk
	params.Set("Disks.0.IsBoot", "True")
	params.Set("Disks.0.Type", disks[0].DiskType)
	params.Set("Disks.0.Size", disks[0].SizeGB)

	// data disk
	for i, disk := range disks[1:] {
		N := i + 1
		params.Set(fmt.Sprintf("Disks.%d.IsBoot", N), "False")
		params.Set(fmt.Sprintf("Disks.%d.Type", N), disk.DiskType)
		params.Set(fmt.Sprintf("Disks.%d.Size", N), disk.SizeGB)
	}

	type Ret struct {
		UHostIds []string
	}

	ret := Ret{}
	err := self.DoAction("CreateUHostInstance", params, &ret)
	if err != nil {
		return "", err
	}

	if len(ret.UHostIds) == 1 {
		return ret.UHostIds[0], nil
	}

	return "", fmt.Errorf("CreateInstance %d instance created. %s", len(ret.UHostIds), ret.UHostIds)
}
