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

package rockbase

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"

	"yunion.io/x/jsonutils"
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
	vm, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}

	vm.host = self
	return vm, nil
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
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_ROCKBASE), "manufacture")
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
	return api.HOST_TYPE_ROCKBASE
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ROCKBASE_API_VERSION
}

// 不支持user data
// 不支持指定keypair
func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(opts)
	if err != nil {
		return nil, err
	}

	vm, err := self.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, err
	}

	vm.host = self
	return vm, err
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
	GpuType   string
	CPU       int
	MemoryMB  int
	GPU       int
}

// 格式: {机型}.c{cpu}.m{memGB}[.g{gpu}]，机型支持 UCloud GpuType 如 T4S、2080Ti-4C、T4/4
var instanceTypeRe = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9/_-]*)\.c(\d+)\.m(\d+)(?:\.g(\d+))?$`)

func ParseInstanceType(instanceType string) (SInstanceType, error) {
	i := SInstanceType{}
	matches := instanceTypeRe.FindStringSubmatch(instanceType)
	if matches == nil {
		return i, fmt.Errorf("invalid instance type %s", instanceType)
	}

	hostType := matches[1]
	cpu, err := strconv.Atoi(matches[2])
	if err != nil {
		return i, err
	}
	memGB, err := strconv.Atoi(matches[3])
	if err != nil {
		return i, err
	}

	i.CPU = cpu
	i.MemoryMB = memGB * 1024
	if len(matches[4]) > 0 {
		gpu, err := strconv.Atoi(matches[4])
		if err != nil {
			return i, err
		}
		i.GPU = gpu
		i.GpuType = hostType
		i.UHostType = "G"
	} else {
		i.UHostType = hostType
	}
	return i, nil
}

func (self *SHost) _createVM(opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	net, err := self.zone.region.getNetwork(opts.ExternalNetworkId)
	if err != nil {
		return "", errors.Wrapf(err, "getNetwork %s", opts.ExternalNetworkId)
	}

	if len(opts.Password) == 0 {
		return "", fmt.Errorf("CreateVM password should not be emtpty")
	}

	img, err := self.zone.region.GetImage(opts.ExternalImageId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImage %s", opts.ExternalImageId)
	}

	if img.GetStatus() != api.CACHED_IMAGE_STATUS_ACTIVE {
		return "", errors.Wrapf(cloudprovider.ErrInvalidStatus, "image %s status %s, expect %s", opts.ExternalImageId, img.GetStatus(), api.CACHED_IMAGE_STATUS_ACTIVE)
	}

	disks := make([]SDisk, len(opts.DataDisks)+1)
	disks[0].SizeGB = int(img.ImageSizeGB)
	if opts.SysDisk.SizeGB > 0 && opts.SysDisk.SizeGB > int(img.ImageSizeGB) {
		disks[0].SizeGB = opts.SysDisk.SizeGB
	}
	disks[0].DiskType = opts.SysDisk.StorageType

	for i, dataDisk := range opts.DataDisks {
		disks[i+1].SizeGB = dataDisk.SizeGB
		disks[i+1].DiskType = dataDisk.StorageType
	}

	var vmId string
	i, err := ParseInstanceType(opts.InstanceType)
	if err != nil {
		if opts.Cpu <= 0 || opts.MemoryMB <= 0 {
			return "", err
		} else {
			i.UHostType = "O"
			i.CPU = opts.Cpu
			i.MemoryMB = opts.MemoryMB
		}
	}

	vmId, err = self.zone.region.CreateInstance(opts.Name, opts.ExternalImageId, i.UHostType, opts.Password, net.wire.vpc.GetId(), opts.ExternalNetworkId, opts.ExternalSecgroupIds, self.zone.ZoneId, opts.Description, opts.IpAddr, i.CPU, i.MemoryMB, i.GPU, i.GpuType, disks, opts.BillingCycle)
	if err != nil {
		return "", fmt.Errorf("Failed to create: %v", err)
	}

	return vmId, nil
}

// https://docs.ucloud.cn/api/uhost-api/create_uhost_instance
// https://docs.ucloud.cn/api/uhost-api/specification
// 支持8-30位字符, 不能包含[A-Z],[a-z],[0-9]和[()`~!@#$%^&*-+=_|{}[]:;'<>,.?/]之外的非法字符
func (self *SRegion) CreateInstance(name, imageId, hostType, password, vpcId, SubnetId string, securityGroupId []string,
	zoneId, desc, ipAddr string, cpu, memMB, gpu int, gpuType string, disks []SDisk, bc *billing.SBillingCycle) (string, error) {
	params := NewRockbaseParams()
	params.Set("Zone", zoneId)
	params.Set("ImageId", imageId)
	params.Set("Password", base64.StdEncoding.EncodeToString([]byte(password)))
	params.Set("LoginMode", "Password")
	params.Set("Name", name)
	params.Set("MachineType", hostType)
	params.Set("CPU", cpu)
	params.Set("Memory", memMB)
	params.Set("VPCId", vpcId)
	params.Set("SubnetId", SubnetId)
	for _, id := range securityGroupId {
		params.Set("SecurityGroupId", id)
	}
	if gpu > 0 {
		params.Set("GPU", gpu)
		if len(gpuType) > 0 {
			params.Set("GpuType", gpuType)
		}
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
		return "", errors.Wrapf(err, "CreateUHostInstance")
	}

	if len(ret.UHostIds) == 1 {
		return ret.UHostIds[0], nil
	}

	return "", fmt.Errorf("CreateInstance %d instance created. %s", len(ret.UHostIds), ret.UHostIds)
}
