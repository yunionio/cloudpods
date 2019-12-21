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

package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type AccessConfig struct {
	Type        string
	Name        string
	NatIP       string
	NetworkTier string
	Kind        string
}

type InstanceDisk struct {
	Type            string
	Mode            string
	Source          string
	DeviceName      string
	Index           int
	Boot            bool
	AutoDelete      bool
	Licenses        []string
	Interface       string
	GuestOsFeatures []GuestOsFeature
	Kind            string
}

type ServiceAccount struct {
	Email  string
	scopes []string
}

type SInstanceTag struct {
	Items       []string
	Fingerprint string
}

type SInstance struct {
	multicloud.SInstanceBase
	host *SHost
	SResourceBase

	Id                 string
	CreationTimestamp  time.Time
	Description        string
	Tags               SInstanceTag
	MachineType        string
	Status             string
	Zone               string
	CanIpForward       bool
	NetworkInterfaces  []SNetworkInterface
	Disks              []InstanceDisk
	Metadata           map[string]string
	ServiceAccounts    []ServiceAccount
	Scheduling         map[string]interface{}
	CpuPlatform        string
	LabelFingerprint   string
	StartRestricted    bool
	DeletionProtection bool
	Kind               string

	guestCpus   int
	memoryMb    int
	machineType string
}

func (region *SRegion) GetInstances(zone string, maxResults int, pageToken string) ([]SInstance, error) {
	instances := []SInstance{}
	params := map[string]string{}
	if len(zone) == 0 {
		return nil, fmt.Errorf("zone params can not be empty")
	}
	resource := fmt.Sprintf("zones/%s/instances", zone)
	return instances, region.List(resource, params, maxResults, pageToken, &instances)
}

func (region *SRegion) GetInstance(id string) (*SInstance, error) {
	instance := &SInstance{}
	return instance, region.Get(id, instance)
}

func (instnace *SInstance) IsEmulated() bool {
	return false
}

func (instance *SInstance) fetchMachineType() error {
	if instance.guestCpus > 0 || instance.memoryMb > 0 || len(instance.machineType) > 0 {
		return nil
	}
	machinetype, err := instance.host.zone.region.GetMachineType(instance.MachineType)
	if err != nil {
		return err
	}
	instance.guestCpus = machinetype.GuestCpus
	instance.memoryMb = machinetype.MemoryMb
	instance.machineType = machinetype.Name
	return nil
}

func (instance *SInstance) Refresh() error {
	_instance, err := instance.host.zone.region.GetInstance(instance.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(instance, _instance)
}

//PROVISIONING, STAGING, RUNNING, STOPPING, STOPPED, SUSPENDING, SUSPENDED, and TERMINATED.
func (instance *SInstance) GetStatus() string {
	switch instance.Status {
	case "PROVISIONING":
		return api.VM_DEPLOYING
	case "STAGING":
		return api.VM_STARTING
	case "RUNNING":
		return api.VM_RUNNING
	case "STOPPING":
		return api.VM_STOPPING
	case "STOPPED":
		return api.VM_READY
	case "SUSPENDING":
		return api.VM_SUSPENDING
	case "SUSPENDED":
		return api.VM_SUSPEND
	case "TERMINATED":
		return api.VM_DELETING
	default:
		return api.VM_UNKNOWN
	}
}

func (instance *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (instance *SInstance) GetCreatedAt() time.Time {
	return instance.CreationTimestamp
}

func (instance *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (instance *SInstance) GetProjectId() string {
	return instance.host.zone.region.GetProjectId()
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetIHostId() string {
	return instance.host.GetGlobalId()
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	idisks := []cloudprovider.ICloudDisk{}
	for _, disk := range instance.Disks {
		disk, err := instance.host.zone.region.GetDisk(disk.Source)
		if err != nil {
			return nil, errors.Wrap(err, "GetDisk")
		}
		storage, err := instance.host.zone.region.GetStorage(disk.Type)
		if err != nil {
			return nil, errors.Wrap(err, "GetStorage")
		}
		storage.zone = instance.host.zone
		disk.storage = storage
		idisks = append(idisks, disk)
	}
	return idisks, nil
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := []cloudprovider.ICloudNic{}
	for i := range instance.NetworkInterfaces {
		instance.NetworkInterfaces[i].instance = instance
		nics = append(nics, &instance.NetworkInterfaces[i])
	}
	return nics, nil
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	for _, networkinterface := range instance.NetworkInterfaces {
		for _, conf := range networkinterface.AccessConfigs {
			if len(conf.NatIP) > 0 {
				eips, err := instance.host.zone.region.GetEips(conf.NatIP, 0, "")
				if err != nil {
					return nil, errors.Wrapf(err, "region.GetEip(%s)", conf.NatIP)
				}
				if len(eips) == 1 {
					return &eips[0], nil
				}
				eip := &SAddress{
					region:  instance.host.zone.region,
					Id:      instance.SelfLink,
					Status:  "IN_USE",
					Address: conf.NatIP,
				}
				eip.SelfLink = instance.SelfLink
				return eip, nil
			}
		}
	}
	return nil, nil
}

func (instance *SInstance) GetVcpuCount() int {
	instance.fetchMachineType()
	return instance.guestCpus
}

func (instance *SInstance) GetVmemSizeMB() int {
	instance.fetchMachineType()
	return instance.memoryMb
}

func (instance *SInstance) GetBootOrder() string {
	return "cdn"
}

func (instance *SInstance) GetVga() string {
	return "std"
}

func (instance *SInstance) GetVdi() string {
	return "vnc"
}

func (instance *SInstance) GetOSType() string {
	for _, disk := range instance.Disks {
		if disk.Index == 0 {
			for _, license := range disk.Licenses {
				if strings.Index(strings.ToLower(license), "windows") < 0 {
					return osprofile.OS_TYPE_LINUX
				} else {
					return osprofile.OS_TYPE_WINDOWS
				}
			}
		}
	}
	return osprofile.OS_TYPE_LINUX
}

func (instance *SInstance) GetOSName() string {
	for _, disk := range instance.Disks {
		if disk.Index == 0 {
			for _, license := range disk.Licenses {
				return imagetools.NormalizeImageInfo(license, "", "", "", "").OsDistro
			}
		}
	}
	return ""
}

func (instance *SInstance) GetBios() string {
	return "BIOS"
}

func (instance *SInstance) GetMachine() string {
	return "pc"
}

func (instance *SInstance) GetInstanceType() string {
	instance.fetchMachineType()
	return instance.machineType
}

func (instance *SInstance) AssignSecurityGroup(id string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroupIds := []string{}
	isecgroups := []cloudprovider.ICloudSecurityGroup{}
	for _, networkinterface := range instance.NetworkInterfaces {
		globalnetwork, err := instance.host.zone.region.client.GetGlobalNetwork(networkinterface.Network)
		if err != nil {
			return nil, errors.Wrap(err, "GetGlobalNetwork")
		}
		vpc := &SVpc{globalnetwork: globalnetwork, region: instance.host.zone.region}
		_isecgroups, err := vpc.GetISecurityGroups()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetISecurityGroups")
		}
		isecgroups = append(isecgroups, _isecgroups...)
		for _, isecgroup := range _isecgroups {
			if len(instance.ServiceAccounts) > 0 && isecgroup.GetName() == instance.ServiceAccounts[0].Email {
				secgroupIds = append(secgroupIds, isecgroup.GetGlobalId())
			}
			if isecgroup.GetName() == globalnetwork.Name {
				secgroupIds = append(secgroupIds, isecgroup.GetGlobalId())
			}
		}
	}
	if len(instance.NetworkInterfaces) == 1 {
		for _, secgroup := range isecgroups {
			if utils.IsInStringArray(secgroup.GetName(), instance.Tags.Items) {
				secgroupIds = append(secgroupIds, secgroup.GetGlobalId())
			}
		}
	}
	return secgroupIds, nil
}

func (instance *SInstance) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_GOOGLE
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetError() error {
	return nil
}
