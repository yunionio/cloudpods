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

package oracle

import (
	"context"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"
)

type SInstance struct {
	multicloud.SInstanceBase
	SOracleTag
	host *SHost

	osInfo *imagetools.ImageInfo

	AvailabilityDomain string
	CompartmentId      string
	DefinedTags        struct {
		OracleTags struct {
			CreatedBy string
			CreatedOn time.Time
		}
	}
	DisplayName      string
	ExtendedMetadata struct {
	}
	FaultDomain  string
	FreeformTags struct {
	}
	Id            string
	ImageId       string
	LaunchMode    string
	LaunchOptions struct {
		BootVolumeType                  string
		Firmware                        string
		NetworkType                     string
		RemoteDataVolumeType            string
		IsPvEncryptionInTransitEnabled  bool
		IsConsistentVolumeNamingEnabled bool
	}
	InstanceOptions struct {
		AreLegacyImdsEndpointsDisabled bool
	}
	AvailabilityConfig struct {
		IsLiveMigrationPreferred bool
		RecoveryAction           string
	}
	// CREATING_IMAGE, MOVING, PROVISIONING, RUNNING, STARTING, STOPPED, STOPPING, TERMINATED, TERMINATING
	LifecycleState string
	Metadata       struct {
	}
	Region      string
	Shape       string
	ShapeConfig struct {
		Ocpus                     float64
		MemoryInGBs               float64
		ProcessorDescription      string
		NetworkingBandwidthInGbps float64
		MaxVnicAttachments        int
		Gpus                      int
		LocalDisks                int
		Vcpus                     int
	}
	IsCrossNumaNode bool
	SourceDetails   struct {
		SourceType string
		ImageId    string
	}
	SystemTags struct {
	}
	TimeCreated time.Time
	AgentConfig struct {
		IsMonitoringDisabled  bool
		IsManagementDisabled  bool
		AreAllPluginsDisabled bool
		PluginsConfig         []struct {
			Name         string
			DesiredState string
		}
	}
}

func (self *SRegion) GetInstances(zoneId string) ([]SInstance, error) {
	params := map[string]interface{}{}
	if len(zoneId) > 0 {
		params["availabilityDomain"] = zoneId
	}
	resp, err := self.list(SERVICE_IAAS, "instances", params)
	if err != nil {
		return nil, err
	}
	ret := []SInstance{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	resp, err := self.get(SERVICE_IAAS, "instances", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SInstance{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetId() string {
	return self.Id
}

func (self *SInstance) GetName() string {
	return self.DisplayName
}

func (self *SInstance) GetHostname() string {
	return self.DisplayName
}

func (self *SInstance) GetGlobalId() string {
	return self.Id
}

func (self *SInstance) GetInstanceType() string {
	return self.Shape
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	ret := []cloudprovider.ICloudDisk{}
	root, err := self.host.zone.region.GetBootVolumeAttachments(self.AvailabilityDomain, self.Id)
	if err != nil {
		return nil, err
	}
	for _, r := range root {
		disk, err := self.host.zone.region.GetBootDisk(r.BootVolumeId)
		if err != nil {
			return nil, err
		}
		disk.storage = &SStorage{zone: self.host.zone, boot: true}
		ret = append(ret, disk)
	}
	data, err := self.host.zone.region.GetAttachedDisks(self.Id)
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		disk, err := self.host.zone.region.GetDisk(d.VolumeId)
		if err != nil {
			return nil, err
		}
		disk.storage = &SStorage{zone: self.host.zone}
		ret = append(ret, disk)
	}
	return ret, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := self.host.zone.region.GetVnicAttachments(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNic{}
	for _, n := range nics {
		nic, err := self.host.zone.region.GetInstanceNic(n.VnicId)
		if err != nil {
			return nil, err
		}
		ret = append(ret, nic)
	}
	return ret, nil
}

func (self *SInstance) GetVcpuCount() int {
	return int(self.ShapeConfig.Ocpus)
}

func (self *SInstance) GetVmemSizeMB() int {
	return int(self.ShapeConfig.MemoryInGBs) * 1024
}

func (self *SInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if self.osInfo != nil {
		return self.osInfo
	}
	image, err := self.host.zone.region.GetImage(self.ImageId)
	if err != nil {
		return &imagetools.ImageInfo{}
	}
	osInfo := imagetools.NormalizeImageInfo(image.DisplayName, "", image.OperatingSystem, "", image.OperatingSystemVersion)
	self.osInfo = &osInfo
	return self.osInfo
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(ins.getNormalizedOsInfo().OsType)
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(ins.getNormalizedOsInfo().OsBios)
}

func (ins *SInstance) GetFullOsName() string {
	return ""
}

func (ins *SInstance) GetOsLang() string {
	return ins.getNormalizedOsInfo().OsLang
}

func (ins *SInstance) GetOsArch() string {
	return ins.getNormalizedOsInfo().OsArch
}

func (ins *SInstance) GetOsDist() string {
	return ins.getNormalizedOsInfo().OsDistro
}

func (ins *SInstance) GetOsVersion() string {
	return ins.getNormalizedOsInfo().OsVersion
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetStatus() string {
	// CREATING_IMAGE, MOVING, PROVISIONING, RUNNING, STARTING, STOPPED, STOPPING, TERMINATED, TERMINATING
	switch self.LifecycleState {
	case "CREATING_IMAGE", "RUNNING":
		return api.VM_RUNNING
	case "MOVING":
		return api.VM_MIGRATING
	case "PROVISIONING":
		return api.VM_DEPLOYING
	case "STARTING":
		return api.VM_STARTING
	case "STOPPED":
		return api.VM_READY
	case "STOPPING":
		return api.VM_STOPPING
	case "TERMINATED", "TERMINATING":
		return api.VM_DELETING
	default:
		return api.VM_UNKNOWN
	}
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) Refresh() error {
	vm, err := self.host.zone.region.GetInstance(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, vm)
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ORACLE
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SInstance) GetCreatedAt() time.Time {
	return self.TimeCreated
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) SaveImage(opts *cloudprovider.SaveImageOptions) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

type SVolumeAttachment struct {
	VolumeId string `json:"volume-id"`
}

func (self *SRegion) GetAttachedDisks(instanceId string) ([]SVolumeAttachment, error) {
	params := map[string]interface{}{
		"instanceId": instanceId,
	}
	resp, err := self.list(SERVICE_IAAS, "volumeAttachments", params)
	if err != nil {
		return nil, err
	}
	ret := []SVolumeAttachment{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SBootVolumeAttachment struct {
	BootVolumeId string `json:"boot-volume-id"`
}

func (self *SRegion) GetBootVolumeAttachments(zoneId, instanceId string) ([]SBootVolumeAttachment, error) {
	params := map[string]interface{}{
		"instanceId":         instanceId,
		"availabilityDomain": zoneId,
	}
	resp, err := self.list(SERVICE_IAAS, "bootVolumeAttachments", params)
	if err != nil {
		return nil, err
	}
	ret := []SBootVolumeAttachment{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

type SVnicAttachment struct {
	VnicId string `json:"vnic-id"`
}

func (self *SRegion) GetVnicAttachments(instanceId string) ([]SVnicAttachment, error) {
	params := map[string]interface{}{}
	if len(instanceId) > 0 {
		params["instanceId"] = instanceId
	}
	resp, err := self.list(SERVICE_IAAS, "vnicAttachments", params)
	if err != nil {
		return nil, err
	}
	ret := []SVnicAttachment{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
