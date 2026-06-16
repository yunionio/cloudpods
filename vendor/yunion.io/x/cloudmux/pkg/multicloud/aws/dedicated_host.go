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
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	DedicatedHostStateAvailable        = "available"
	DedicatedHostStatePending          = "pending"
	DedicatedHostStateUnderAssessment  = "under-assessment"
	DedicatedHostStatePermanentFailure = "permanent-failure"
	DedicatedHostStateReleased         = "released"
)

type SDedicatedHost struct {
	multicloud.SHostBase
	AwsTags

	zone *SZone

	AllocationTime   time.Time `xml:"allocationTime"`
	AvailabilityZone string    `xml:"availabilityZone"`
	AvailableCapacity struct {
		AvailableInstanceCapacity []struct {
			AvailableCapacity int    `xml:"availableCapacity"`
			InstanceType      string `xml:"instanceType"`
			TotalCapacity     int    `xml:"totalCapacity"`
		} `xml:"availableInstanceCapacity>item"`
		AvailableVCpus int `xml:"availableVCpus"`
	} `xml:"availableCapacity"`
	ClientToken string `xml:"clientToken"`
	HostId      string `xml:"hostId"`
	HostProperties struct {
		Cores          int    `xml:"cores"`
		InstanceFamily string `xml:"instanceFamily"`
		InstanceType   string `xml:"instanceType"`
		Sockets        int    `xml:"sockets"`
		TotalVCpus     int64  `xml:"totalVCpus"`
	} `xml:"hostProperties"`
	Instances []struct {
		InstanceId   string `xml:"instanceId"`
		InstanceType string `xml:"instanceType"`
	} `xml:"instances>item"`
	ReleaseTime time.Time `xml:"releaseTime"`
	State       string    `xml:"state"`
}

func (self *SRegion) GetDedicatedHosts(zoneName string, hostIds ...string) ([]SDedicatedHost, error) {
	params := map[string]string{}
	if len(hostIds) > 0 {
		for i, id := range hostIds {
			params[fmt.Sprintf("HostId.%d", i+1)] = id
		}
	} else {
		idx := 1
		if len(zoneName) > 0 {
			params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
			params[fmt.Sprintf("Filter.%d.Value.1", idx)] = zoneName
			idx++
		}
		// skip released hosts
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "state"
		for i, state := range []string{
			DedicatedHostStateAvailable,
			DedicatedHostStatePending,
			DedicatedHostStateUnderAssessment,
			DedicatedHostStatePermanentFailure,
		} {
			params[fmt.Sprintf("Filter.%d.Value.%d", idx, i+1)] = state
		}
	}

	ret := []SDedicatedHost{}
	for {
		part := struct {
			HostSet   []SDedicatedHost `xml:"hostSet>item"`
			NextToken string           `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeHosts", params, &part)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeHosts")
		}
		ret = append(ret, part.HostSet...)
		if len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetDedicatedHost(hostId string) (*SDedicatedHost, error) {
	hosts, err := self.GetDedicatedHosts("", hostId)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", hostId)
	}
	return &hosts[0], nil
}

func (self *SRegion) AllocateDedicatedHost(zoneName, instanceType, name string, quantity int, autoPlacement bool) ([]SDedicatedHost, error) {
	if len(zoneName) == 0 {
		return nil, fmt.Errorf("missing availability zone")
	}
	if len(instanceType) == 0 {
		return nil, fmt.Errorf("missing instance type")
	}
	if quantity <= 0 {
		quantity = 1
	}
	params := map[string]string{
		"AvailabilityZone": zoneName,
		"InstanceType":     instanceType,
		"Quantity":         fmt.Sprintf("%d", quantity),
	}
	if autoPlacement {
		params["AutoPlacement"] = "on"
	} else {
		params["AutoPlacement"] = "off"
	}
	if len(name) > 0 {
		params["TagSpecification.1.ResourceType"] = "dedicated-host"
		params["TagSpecification.1.Tag.1.Key"] = "Name"
		params["TagSpecification.1.Tag.1.Value"] = name
	}
	ret := struct {
		HostSet []SDedicatedHost `xml:"hostSet>item"`
	}{}
	err := self.ec2Request("AllocateHosts", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "AllocateHosts")
	}
	return ret.HostSet, nil
}

func (self *SRegion) ReleaseDedicatedHost(hostId string) error {
	params := map[string]string{
		"HostId.1": hostId,
	}
	return self.ec2Request("ReleaseHosts", params, nil)
}

func (self *SDedicatedHost) GetId() string {
	return self.HostId
}

func (self *SDedicatedHost) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.HostId
}

func (self *SDedicatedHost) GetGlobalId() string {
	return self.HostId
}

func (self *SDedicatedHost) GetStatus() string {
	switch self.State {
	case DedicatedHostStateAvailable:
		return api.HOST_STATUS_RUNNING
	case DedicatedHostStatePending:
		return api.HOST_STATUS_READY
	case DedicatedHostStateReleased:
		return api.HOST_STATUS_UNKNOWN
	default:
		return api.HOST_STATUS_UNKNOWN
	}
}

func (self *SDedicatedHost) Refresh() error {
	hosts, err := self.zone.region.GetDedicatedHosts(self.zone.ZoneName)
	if err != nil {
		return errors.Wrap(err, "GetDedicatedHosts")
	}
	for i := range hosts {
		if hosts[i].HostId == self.HostId {
			return jsonutils.Update(self, &hosts[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, "%s", self.HostId)
}

func (self *SDedicatedHost) IsEmulated() bool {
	return false
}

func (self *SDedicatedHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.zone.ZoneName, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	ivms := make([]cloudprovider.ICloudVM, 0, len(vms))
	for i := range vms {
		if vms[i].Placement.HostId != self.HostId {
			continue
		}
		bindInstanceHost(&vms[i], self.zone, self)
		ivms = append(ivms, &vms[i])
	}
	return ivms, nil
}

func (self *SDedicatedHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.zone.ZoneName, "", []string{id})
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	for i := range vms {
		if vms[i].InstanceId == id && vms[i].Placement.HostId == self.HostId {
			bindInstanceHost(&vms[i], self.zone, self)
			return &vms[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (self *SDedicatedHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SDedicatedHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SDedicatedHost) GetEnabled() bool {
	return self.State == DedicatedHostStateAvailable
}

func (self *SDedicatedHost) GetHostStatus() string {
	if self.GetEnabled() {
		return api.HOST_ONLINE
	}
	return api.HOST_OFFLINE
}

func (self *SDedicatedHost) GetAccessIp() string {
	return ""
}

func (self *SDedicatedHost) GetAccessMac() string {
	return ""
}

func (self *SDedicatedHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AWS), "manufacture")
	info.Add(jsonutils.NewString(self.HostProperties.InstanceFamily), "instance_family")
	info.Add(jsonutils.NewString(self.HostProperties.InstanceType), "instance_type")
	info.Add(jsonutils.NewInt(int64(self.AvailableCapacity.AvailableVCpus)), "available_vcpus")
	return info
}

func (self *SDedicatedHost) GetSN() string {
	return self.HostId
}

func (self *SDedicatedHost) GetCpuCount() int {
	return int(self.HostProperties.TotalVCpus)
}

func (self *SDedicatedHost) GetNodeCount() int8 {
	return int8(self.HostProperties.Sockets)
}

func (self *SDedicatedHost) GetCpuDesc() string {
	return self.HostProperties.InstanceFamily
}

func (self *SDedicatedHost) GetCpuMhz() int {
	return 0
}

func (self *SDedicatedHost) GetCpuCmtbound() float32 {
	return 1.0
}

func (self *SDedicatedHost) GetMemCmtbound() float32 {
	return 1.0
}

func (self *SDedicatedHost) GetMemSizeMB() int {
	itype := self.HostProperties.InstanceType
	if len(itype) == 0 {
		return 0
	}
	it, err := self.zone.region.GetInstanceType(itype)
	if err != nil || it == nil {
		return 0
	}
	vcpuPerInst := it.VCpuInfo.DefaultVCpus
	if vcpuPerInst <= 0 {
		vcpuPerInst = 1
	}
	totalVCpus := int(self.HostProperties.TotalVCpus)
	if totalVCpus <= 0 {
		return it.MemoryInfo.SizeInMiB
	}
	instances := totalVCpus / vcpuPerInst
	if instances <= 0 {
		instances = 1
	}
	return it.MemoryInfo.SizeInMiB * instances
}

func (self *SDedicatedHost) GetStorageSizeMB() int64 {
	return 0
}

func (self *SDedicatedHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SDedicatedHost) GetHostType() string {
	return api.HOST_TYPE_DEDICATED
}

func (self *SDedicatedHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstance")
	}
	if inst.Placement.HostId != self.HostId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", instanceId)
	}
	bindInstanceHost(inst, self.zone, self)
	return inst, nil
}

func (self *SDedicatedHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vm, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.InstanceType,
		desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks,
		desc.PublicKey, desc.ExternalSecgroupIds, desc.UserData, desc.Tags, desc.EnableMonitorAgent)
	if err != nil {
		return nil, errors.Wrap(err, "_createVM")
	}
	bindInstanceHost(vm, self.zone, self)
	return vm, err
}

func (self *SDedicatedHost) _createVM(name, imgId string, sysDisk cloudprovider.SDiskInfo, instanceType string,
	networkId, ipAddr, desc, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupIds []string, userData string,
	tags map[string]string, enableMonitorAgent bool,
) (*SInstance, error) {
	if len(instanceType) == 0 {
		return nil, fmt.Errorf("missing instance type params")
	}
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.SyncKeypair(publicKey)
		if err != nil {
			return nil, err
		}
	}
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
	instance, err := self.zone.region.CreateInstance(name, img, instanceType, networkId, secgroupIds, self.zone.ZoneName, desc, disks, ipAddr, keypair, userData, tags, enableMonitorAgent, self.HostId)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateInstance")
	}
	return instance, nil
}

func (host *SDedicatedHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SDedicatedHost) GetIsMaintenance() bool {
	return self.State == DedicatedHostStateUnderAssessment || self.State == DedicatedHostStatePermanentFailure
}

func (self *SDedicatedHost) GetVersion() string {
	return AWS_API_VERSION
}

func bindInstanceHost(inst *SInstance, zone *SZone, ihost cloudprovider.ICloudHost) {
	inst.host = &SHost{zone: zone}
	if dh, ok := ihost.(*SDedicatedHost); ok {
		inst.dedicatedHost = dh
	} else if h, ok := ihost.(*SHost); ok {
		inst.host = h
		inst.dedicatedHost = nil
	}
}
