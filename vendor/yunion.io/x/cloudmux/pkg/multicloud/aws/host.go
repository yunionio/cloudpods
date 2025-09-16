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
	"fmt"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Name, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId())
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
	vms, err := self.zone.region.GetInstances(self.zone.ZoneName, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}

	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = self
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.zone.ZoneName, "", []string{id})
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	for i := range vms {
		if vms[i].InstanceId == id {
			vms[i].host = self
			return &vms[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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

func (self *SHost) GetStorageSizeMB() int64 {
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
		return nil, errors.Wrap(err, "GetInstance")
	}
	inst.host = self
	return inst, nil
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vm, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.InstanceType,
		desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks,
		desc.PublicKey, desc.ExternalSecgroupIds, desc.UserData, desc.Tags, desc.EnableMonitorAgent)
	if err != nil {
		return nil, errors.Wrap(err, "_createVM")
	}
	vm.host = self
	return vm, err
}

func (self *SHost) _createVM(name, imgId string, sysDisk cloudprovider.SDiskInfo, instanceType string,
	networkId, ipAddr, desc, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupIds []string, userData string,
	tags map[string]string, enableMonitorAgent bool,
) (*SInstance, error) {
	if len(instanceType) == 0 {
		return nil, fmt.Errorf("missing instance type params")
	}
	// 同步keypair
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.SyncKeypair(publicKey)
		if err != nil {
			return nil, err
		}
	}

	// 镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s)", imgId)
	}
	if img.Status != ImageStatusAvailable {
		return nil, fmt.Errorf("image not ready status: %s", img.Status)
	}
	if len(dataDisks) == 0 {
		dataDisks = []cloudprovider.SDiskInfo{}
	}
	if sysDisk.SizeGB < img.GetMinOsDiskSizeGb() {
		sysDisk.SizeGB = img.GetMinOsDiskSizeGb()
	}
	disks := append([]cloudprovider.SDiskInfo{sysDisk}, dataDisks...)

	instance, err := self.zone.region.CreateInstance(name, img, instanceType, networkId, secgroupIds, self.zone.ZoneName, desc, disks, ipAddr, keypair, userData, tags, enableMonitorAgent)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateInstance")
	}
	return instance, nil
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return AWS_API_VERSION
}
