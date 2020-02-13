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
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
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

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	vms, _, err := self.zone.region.GetInstances(self.zone.ZoneId, nil, len(vms), 50)
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

func (self *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	if len(gid) == 0 {
		log.Errorf("GetIVMById guest id is empty")
		return nil, ErrorNotFound()
	}

	ivms, _, err := self.zone.region.GetInstances(self.zone.ZoneId, []string{gid}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(ivms) == 0 {
		return nil, ErrorNotFound()
	}
	if len(ivms) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	ivms[0].host = self
	return &ivms[0], nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
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
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AWS), "manufacture")
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

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_AWS
}

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = self
	return inst, nil
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	if strings.ToLower(desc.OsType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		// powershell scripts
		data := fmt.Sprintf("<powershell>%s</powershell>", desc.UserData)
		desc.UserData = base64.StdEncoding.EncodeToString([]byte(data))
	}

	vmId, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.InstanceType, desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks, desc.PublicKey, desc.ExternalSecgroupId, desc.UserData)
	if err != nil {
		return nil, err
	}

	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}

	return vm, err
}

func (self *SHost) _createVM(name, imgId string, sysDisk cloudprovider.SDiskInfo, instanceType string,
	networkId, ipAddr, desc, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupId string, userData string) (string, error) {
	// 网络配置及安全组绑定
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

	if len(secgroupId) == 0 {
		secgroups, err := net.wire.vpc.GetISecurityGroups()
		if err != nil {
			return "", fmt.Errorf("get security group error %s", err)
		}

		if len(secgroups) == 0 {
			// aws 默认就已经创建好了一个默认安全组。正常情况下并不需要手动创建
			secId, err := self.zone.region.createDefaultSecurityGroup(net.wire.vpc.VpcId)
			if err != nil {
				return "", fmt.Errorf("no secgroup for vpc and failed to create a default One!!")
			} else {
				secgroupId = secId
			}
		} else {
			secgroupId = secgroups[0].GetId()
		}
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

	// 镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("getiamge %s fail %s", imgId, err)
		return "", err
	}
	if img.Status != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(dataDisks)+1)
	disks[0].Size = img.SizeGB
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > img.SizeGB {
		disks[0].Size = sysDisk.SizeGB
	}
	disks[0].Category = sysDisk.StorageType

	for i, dataDisk := range dataDisks {
		disks[i+1].Size = dataDisk.SizeGB
		disks[i+1].Category = dataDisk.StorageType
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupId, self.zone.ZoneId, desc, disks, ipAddr, keypair, userData)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("Failed to create specification %s.%s", instanceType, err.Error())
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("Failed to create, instance type should not be empty")
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return AWS_API_VERSION
}
