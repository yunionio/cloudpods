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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/cloudinit"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

const (
	METADATA_SSH_KEYS                   = "ssh-keys"
	METADATA_STARTUP_SCRIPT             = "startup-script"
	METADATA_POWER_SHELL                = "sysprep-specialize-script-ps1"
	METADATA_STARTUP_SCRIPT_POWER_SHELL = "windows-startup-script-ps1"
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

type SMetadataItem struct {
	Key   string
	Value string
}

type SMetadata struct {
	Fingerprint string
	Items       []SMetadataItem
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
	Metadata           SMetadata
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
		return api.VM_READY
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
		_disk, err := instance.host.zone.region.GetDisk(disk.Source)
		if err != nil {
			return nil, errors.Wrap(err, "GetDisk")
		}
		storage, err := instance.host.zone.region.GetStorage(_disk.Type)
		if err != nil {
			return nil, errors.Wrap(err, "GetStorage")
		}
		storage.zone = instance.host.zone
		_disk.storage = storage
		_disk.autoDelete = disk.AutoDelete
		_disk.boot = disk.Boot
		_disk.index = disk.Index
		idisks = append(idisks, _disk)
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
					eips[0].region = instance.host.zone.region
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
	for _, secgrpType := range []string{SECGROUP_TYPE_TAG, SECGROUP_TYPE_SERVICE_ACCOUNT} {
		if strings.Contains(id, fmt.Sprintf("/%s/", secgrpType)) {
			idx := strings.LastIndex(id, "/") + 1
			if idx <= 0 {
				return fmt.Errorf("invalid secgroup %s with id %s", secgrpType, id)
			}
			secgroup := id[idx:]
			switch secgrpType {
			case SECGROUP_TYPE_TAG:
				tag := strings.ToLower(secgroup)
				if !utils.IsInStringArray(tag, instance.Tags.Items) {
					instance.Tags.Items = append(instance.Tags.Items, tag)
					return instance.host.zone.region.SetTags(instance.SelfLink, instance.Tags)
				}
			case SECGROUP_TYPE_SERVICE_ACCOUNT:
				if len(instance.ServiceAccounts) > 0 {
					return fmt.Errorf("instance %s has already set serviceAccount %s", instance.Name, instance.ServiceAccounts[0].Email)
				}
				return instance.host.zone.region.SetServiceAccount(instance.SelfLink, secgroup)
			}
		}
	}
	return fmt.Errorf("unknown secgroup type %s", id)
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
			if isecgroup.GetName() == globalnetwork.Name && !strings.Contains(isecgroup.GetGlobalId(), fmt.Sprintf("/%s/", SECGROUP_TYPE_TAG)) {
				secgroupIds = append(secgroupIds, isecgroup.GetGlobalId())
			}
		}
	}
	if len(instance.NetworkInterfaces) == 1 {
		for _, secgroup := range isecgroups {
			if utils.IsInStringArray(secgroup.GetName(), instance.Tags.Items) && strings.Contains(secgroup.GetGlobalId(), fmt.Sprintf("/%s/", SECGROUP_TYPE_TAG)) {
				secgroupIds = append(secgroupIds, secgroup.GetGlobalId())
			}
		}
	}
	return secgroupIds, nil
}

func (instance *SInstance) SetSecurityGroups(ids []string) error {
	secgroups := map[string][]string{}
	for _, id := range ids {
		for _, secgrpType := range []string{SECGROUP_TYPE_TAG, SECGROUP_TYPE_SERVICE_ACCOUNT} {
			if strings.Contains(id, fmt.Sprintf("/%s/", secgrpType)) {
				idx := strings.LastIndex(id, "/") + 1
				if idx <= 0 {
					return fmt.Errorf("invalid secgroup %s with id %s", secgrpType, id)
				}
				secgroup := id[idx:]
				if len(secgroup) == 0 {
					return fmt.Errorf("invalid secgroup %s with id %s", secgrpType, id)
				}
				if _, ok := secgroups[secgrpType]; !ok {
					secgroups[secgrpType] = []string{}
				}
				if !utils.IsInStringArray(secgroup, secgroups[secgrpType]) {
					secgroups[secgrpType] = append(secgroups[secgrpType], secgroup)
				}
			}
		}
	}
	if tags, ok := secgroups[SECGROUP_TYPE_TAG]; ok && len(tags) > 0 {
		for _, tag := range tags {
			tag = strings.ToLower(tag)
			if !utils.IsInStringArray(tag, instance.Tags.Items) {
				instance.Tags.Items = append(instance.Tags.Items, tag)
			}
		}
		err := instance.host.zone.region.SetTags(instance.SelfLink, instance.Tags)
		if err != nil {
			return errors.Wrap(err, "SetTags")
		}
	}
	if serviceAccounts, ok := secgroups[SECGROUP_TYPE_SERVICE_ACCOUNT]; ok && len(serviceAccounts) > 0 {
		if len(serviceAccounts) > 1 {
			return fmt.Errorf("can not set multi service account for google instance")
		}
		return instance.host.zone.region.SetServiceAccount(instance.SelfLink, serviceAccounts[0])
	}
	return nil
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_GOOGLE
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	return instance.host.zone.region.StartInstance(instance.SelfLink)
}

func (instance *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return instance.host.zone.region.StopInstance(instance.SelfLink)
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return instance.host.zone.region.Delete(instance.SelfLink)
}

func (instance *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) UpdateUserData(userData string) error {
	items := []SMetadataItem{}
	for _, item := range instance.Metadata.Items {
		if item.Key != METADATA_STARTUP_SCRIPT && item.Key != METADATA_POWER_SHELL && item.Key != METADATA_STARTUP_SCRIPT_POWER_SHELL {
			items = append(items, item)
		}
	}
	if len(userData) > 0 {
		items = append(items, SMetadataItem{Key: METADATA_STARTUP_SCRIPT, Value: userData})
		items = append(items, SMetadataItem{Key: METADATA_STARTUP_SCRIPT_POWER_SHELL, Value: userData})
		items = append(items, SMetadataItem{Key: METADATA_POWER_SHELL, Value: userData})
	}
	instance.Metadata.Items = items
	return instance.host.zone.region.SetMetadata(instance.SelfLink, instance.Metadata)
}

func (instance *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	diskId, err := instance.host.zone.region.RebuildRoot(instance.SelfLink, desc.ImageId, desc.SysSizeGB)
	if err != nil {
		return "", errors.Wrap(err, "region.RebuildRoot")
	}
	return diskId, instance.DeployVM(ctx, "", desc.Account, desc.Password, desc.PublicKey, false, "")
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	conf := cloudinit.SCloudConfig{}
	user := cloudinit.NewUser(username)
	if len(password) > 0 {
		user.Password(password)
	}
	if len(publicKey) > 0 {
		user.SshKey(publicKey)
	}
	if len(password) > 0 || len(publicKey) > 0 {
		conf.MergeUser(user)
		items := []SMetadataItem{}
		instance.Refresh()
		for _, item := range instance.Metadata.Items {
			if item.Key != METADATA_STARTUP_SCRIPT_POWER_SHELL && item.Key != METADATA_STARTUP_SCRIPT {
				items = append(items, item)
			}
		}
		items = append(items, SMetadataItem{Key: METADATA_STARTUP_SCRIPT_POWER_SHELL, Value: conf.UserDataPowerShell()})
		items = append(items, SMetadataItem{Key: METADATA_STARTUP_SCRIPT, Value: conf.UserDataScript()})
		instance.Metadata.Items = items
		return instance.host.zone.region.SetMetadata(instance.SelfLink, instance.Metadata)
	}
	return nil
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return instance.host.zone.region.ChangeInstanceConfig(instance.SelfLink, instance.host.zone.Name, config.InstanceType, config.Cpu, config.MemoryMB)
}

func (instance *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.AttachDisk(instance.SelfLink, diskId, false)
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	for _, disk := range instance.Disks {
		if strings.HasSuffix(disk.Source, diskId) {
			return instance.host.zone.region.DetachDisk(instance.SelfLink, disk.DeviceName)
		}
	}
	return nil
}

func (instance *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) GetError() error {
	return nil
}

func getDiskInfo(disk string) (cloudprovider.SDiskInfo, error) {
	result := cloudprovider.SDiskInfo{}
	diskInfo := strings.Split(disk, ":")
	for _, d := range diskInfo {
		if utils.IsInStringArray(d, []string{api.STORAGE_GOOGLE_PD_STANDARD, api.STORAGE_GOOGLE_PD_SSD, api.STORAGE_GOOGLE_LOCAL_SSD}) {
			result.StorageType = d
		} else if memSize, err := fileutils.GetSizeMb(d, 'M', 1024); err == nil {
			result.SizeGB = memSize >> 10
		} else {
			result.Name = d
		}
	}
	if len(result.StorageType) == 0 {
		result.StorageType = api.STORAGE_GOOGLE_PD_STANDARD
	}

	if result.SizeGB == 0 {
		return result, fmt.Errorf("Missing disk size")
	}
	return result, nil
}

func (region *SRegion) CreateInstance(zone, name, desc, instanceType string, cpu, memoryMb int, networkId string, ipAddr, imageId string, disks []string) (*SInstance, error) {
	if len(instanceType) == 0 && (cpu == 0 || memoryMb == 0) {
		return nil, fmt.Errorf("Missing instanceType or cpu &memory info")
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("Missing disk info")
	}
	sysDisk, err := getDiskInfo(disks[0])
	if err != nil {
		return nil, errors.Wrap(err, "getDiskInfo.sys")
	}
	dataDisks := []cloudprovider.SDiskInfo{}
	for _, d := range disks[1:] {
		dataDisk, err := getDiskInfo(d)
		if err != nil {
			return nil, errors.Wrapf(err, "getDiskInfo(%s)", d)
		}
		dataDisks = append(dataDisks, dataDisk)
	}
	conf := &cloudprovider.SManagedVMCreateConfig{
		Name:              name,
		Description:       desc,
		ExternalImageId:   imageId,
		Cpu:               cpu,
		MemoryMB:          memoryMb,
		ExternalNetworkId: networkId,
		IpAddr:            ipAddr,
		SysDisk:           sysDisk,
		DataDisks:         dataDisks,
	}
	return region._createVM(zone, conf)
}

func (region *SRegion) getSecgroupByIds(ids []string) (map[string][]string, error) {
	secgroups := map[string][]string{}
	for _, id := range ids {
		for _, secgrpType := range []string{SECGROUP_TYPE_TAG, SECGROUP_TYPE_SERVICE_ACCOUNT} {
			if strings.Contains(id, fmt.Sprintf("/%s/", secgrpType)) {
				idx := strings.LastIndex(id, "/") + 1
				if idx <= 0 {
					return nil, fmt.Errorf("invalid secgroup %s for %s", id, secgrpType)
				}
				secgroup := id[idx:]
				if len(secgroup) == 0 {
					return nil, fmt.Errorf("invalid secgroup %s for %s", id, secgrpType)
				}
				if _, ok := secgroups[secgrpType]; !ok {
					secgroups[secgrpType] = []string{}
				}
				if !utils.IsInStringArray(secgroup, secgroups[secgrpType]) {
					secgroups[secgrpType] = append(secgroups[secgrpType], secgroup)
				}
			}
		}
	}
	return secgroups, nil
}

func (region *SRegion) _createVM(zone string, desc *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	network, err := region.GetNetwork(desc.ExternalNetworkId)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetNetwork")
	}
	secgroups, err := region.getSecgroupByIds(desc.ExternalSecgroupIds)
	if err != nil {
		return nil, errors.Wrap(err, "getSecgroupByIds")
	}
	serviceAccounts, ok := secgroups[SECGROUP_TYPE_SERVICE_ACCOUNT]
	if ok && len(serviceAccounts) > 1 {
		return nil, fmt.Errorf("Security groups are distributed across multiple service accounts")
	}
	if len(desc.InstanceType) == 0 {
		desc.InstanceType = fmt.Sprintf("custom-%d-%d", desc.Cpu, desc.MemoryMB)
	}
	disks := []map[string]interface{}{}
	if len(desc.SysDisk.Name) == 0 {
		desc.SysDisk.Name = fmt.Sprintf("vdisk-%s-%d", desc.Name, time.Now().UnixNano())
	}
	disks = append(disks, map[string]interface{}{
		"boot": true,
		"initializeParams": map[string]interface{}{
			"diskName":    strings.Replace(desc.SysDisk.Name, "_", "-", -1),
			"sourceImage": desc.ExternalImageId,
			"diskSizeGb":  desc.SysDisk.SizeGB,
			"diskType":    fmt.Sprintf("zones/%s/diskTypes/%s", zone, desc.SysDisk.StorageType),
		},
		"autoDelete": true,
	})
	for _, disk := range desc.DataDisks {
		if len(disk.Name) == 0 {
			disk.Name = fmt.Sprintf("vdisk-%s-%d", desc.Name, time.Now().UnixNano())
		}
		disks = append(disks, map[string]interface{}{
			"boot": false,
			"initializeParams": map[string]interface{}{
				"diskName":   strings.Replace(disk.Name, "_", "-", -1),
				"diskSizeGb": disk.SizeGB,
				"diskType":   fmt.Sprintf("zones/%s/diskTypes/%s", zone, disk.StorageType),
			},
			"autoDelete": true,
		})
	}
	networkInterface := map[string]string{
		"network":    network.Network,
		"subnetwork": network.SelfLink,
	}
	if len(desc.IpAddr) > 0 {
		networkInterface["networkIp"] = desc.IpAddr
	}
	params := map[string]interface{}{
		"name":        desc.Name,
		"description": desc.Description,
		"machineType": fmt.Sprintf("zones/%s/machineTypes/%s", zone, desc.InstanceType),
		"networkInterfaces": []map[string]string{
			networkInterface,
		},
		"disks": disks,
	}
	if tags, ok := secgroups[SECGROUP_TYPE_TAG]; ok && len(tags) > 0 {
		for i := range tags {
			tags[i] = strings.ToLower(tags[i])
		}
		params["tags"] = map[string][]string{
			"items": tags,
		}
	}
	if len(desc.UserData) > 0 {
		params["metadata"] = map[string]interface{}{
			"items": []struct {
				Key   string
				Value string
			}{
				{
					Key:   METADATA_STARTUP_SCRIPT,
					Value: desc.UserData,
				},
				{
					Key:   METADATA_POWER_SHELL,
					Value: desc.UserData,
				},
			},
		}
	}
	if len(serviceAccounts) > 0 {
		params["serviceAccounts"] = []struct {
			Email  string
			Scopes []string
		}{
			{
				Email: serviceAccounts[0],
				Scopes: []string{
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring.write",
					"https://www.googleapis.com/auth/servicecontrol",
					"https://www.googleapis.com/auth/service.management.readonly",
					"https://www.googleapis.com/auth/trace.append",
				},
			},
		}
	}
	log.Debugf("create google instance params: %s", jsonutils.Marshal(params).String())
	instance := &SInstance{}
	resource := fmt.Sprintf("zones/%s/instances", zone)
	err = region.Insert(resource, jsonutils.Marshal(params), instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (region *SRegion) StartInstance(id string) error {
	params := map[string]string{}
	return region.Do(id, "start", nil, jsonutils.Marshal(params))
}

func (region *SRegion) StopInstance(id string) error {
	params := map[string]string{}
	return region.Do(id, "stop", nil, jsonutils.Marshal(params))
}

func (region *SRegion) ResetInstance(id string) error {
	params := map[string]string{}
	return region.Do(id, "reset", nil, jsonutils.Marshal(params))
}

func (region *SRegion) DetachDisk(instanceId, deviceName string) error {
	body := map[string]string{}
	params := map[string]string{"deviceName": deviceName}
	return region.Do(instanceId, "detachDisk", params, jsonutils.Marshal(body))
}

func (instance *SInstance) GetSerialOutput(port int) (string, error) {
	return instance.host.zone.region.GetSerialPortOutput(instance.SelfLink, port)
}

func (region *SRegion) GetSerialPortOutput(id string, port int) (string, error) {
	_content, content, next := "", "", 0
	var err error = nil
	for {
		_content, next, err = region.getSerialPortOutput(id, port, next)
		if err != nil {
			return content, err
		}
		content += _content
		if len(_content) == 0 {
			break
		}
	}
	return content, nil
}

func (region *SRegion) getSerialPortOutput(id string, port int, start int) (string, int, error) {
	resource := fmt.Sprintf("%s/serialPort?port=%d&start=%d", id, port, start)
	result := struct {
		Contents string
		Start    int
		Next     int
	}{}
	err := region.Get(resource, &result)
	if err != nil {
		return "", result.Next, errors.Wrap(err, "")
	}
	return result.Contents, result.Next, nil
}

func (region *SRegion) AttachDisk(instanceId, diskId string, boot bool) error {
	diskId = strings.TrimPrefix(diskId, fmt.Sprintf("%s/%s/", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION))
	diskId = fmt.Sprintf("%s/%s/%s", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, diskId)
	body := map[string]interface{}{
		"source": diskId,
		"boot":   boot,
	}
	if boot {
		body["autoDelete"] = true
	}
	params := map[string]string{"forceAttach": "true"}
	return region.Do(instanceId, "attachDisk", params, jsonutils.Marshal(body))
}

func (region *SRegion) ChangeInstanceConfig(id string, zone string, instanceType string, cpu int, memoryMb int) error {
	if len(instanceType) == 0 {
		instanceType = fmt.Sprintf("custom-%d-%d", cpu, memoryMb)
	}
	params := map[string]string{
		"machineType": fmt.Sprintf("zones/%s/machineTypes/%s", zone, instanceType),
	}
	return region.Do(id, "setMachineType", nil, jsonutils.Marshal(params))
}

func (region *SRegion) SetMetadata(id string, metadata SMetadata) error {
	return region.Do(id, "setMetadata", nil, jsonutils.Marshal(metadata))
}

func (region *SRegion) SetTags(id string, tags SInstanceTag) error {
	return region.Do(id, "setTags", nil, jsonutils.Marshal(tags))
}

func (region *SRegion) SetServiceAccount(id string, email string) error {
	body := map[string]interface{}{
		"email": email,
		"scopes": []string{
			"https://www.googleapis.com/auth/devstorage.read_only",
			"https://www.googleapis.com/auth/logging.write",
			"https://www.googleapis.com/auth/monitoring.write",
			"https://www.googleapis.com/auth/servicecontrol",
			"https://www.googleapis.com/auth/service.management.readonly",
			"https://www.googleapis.com/auth/trace.append",
		},
	}
	return region.Do(id, "setsetServiceAccount", nil, jsonutils.Marshal(body))
}

func (region *SRegion) RebuildRoot(instanceId string, imageId string, sysDiskSizeGb int) (string, error) {
	oldDisk, diskType, deviceName := "", api.STORAGE_GOOGLE_PD_STANDARD, ""
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return "", errors.Wrap(err, "region.GetInstance")
	}
	for _, disk := range instance.Disks {
		if disk.Boot {
			oldDisk = disk.Source
			deviceName = disk.DeviceName
			break
		}
	}

	if len(oldDisk) > 0 {
		disk, err := region.GetDisk(oldDisk)
		if err != nil {
			return "", errors.Wrap(err, "region.GetDisk")
		}
		diskType = disk.Type
		if sysDiskSizeGb == 0 {
			sysDiskSizeGb = disk.SizeGB
		}
	}

	zone, err := region.GetZone(instance.Zone)
	if err != nil {
		return "", errors.Wrap(err, "region.GetZone")
	}

	diskName := fmt.Sprintf("vdisk-%s-%d", instance.Name, time.Now().UnixNano())
	disk, err := region.CreateDisk(diskName, sysDiskSizeGb, zone.Name, diskType, imageId, "create for replace instance system disk")
	if err != nil {
		return "", errors.Wrap(err, "region.CreateDisk.systemDisk")
	}

	if len(deviceName) > 0 {
		err = region.DetachDisk(instanceId, deviceName)
		if err != nil {
			defer region.Delete(disk.SelfLink)
			return "", errors.Wrap(err, "region.DetachDisk")
		}
	}

	err = region.AttachDisk(instanceId, disk.SelfLink, true)
	if err != nil {
		if len(oldDisk) > 0 {
			defer region.AttachDisk(instanceId, oldDisk, true)
		}
		defer region.Delete(disk.SelfLink)
		return "", errors.Wrap(err, "region.AttachDisk.newSystemDisk")
	}

	if len(oldDisk) > 0 {
		defer region.Delete(oldDisk)
	}
	return disk.GetGlobalId(), nil
}
