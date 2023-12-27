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

package zstack

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SInstanceCdrome struct {
}

type SInstance struct {
	multicloud.SInstanceBase
	ZStackTags
	host *SHost

	osInfo *imagetools.ImageInfo

	ZStackBasic
	ZoneUUID             string `json:"zoneUuid"`
	ClusterUUID          string `json:"clusterUuid"`
	HostUUID             string `json:"hostUuid"`
	LastHostUUID         string `json:"lastHostUuid"`
	RootVolumeUUID       string `json:"rootVolumeUuid"`
	Platform             string `json:"platform"`
	InstanceOfferingUUID string `json:"instanceOfferingUuid"`

	DefaultL3NetworkUUID string            `json:"defaultL3NetworkUuid"`
	Type                 string            `json:"type"`
	HypervisorType       string            `json:"hypervisorType"`
	MemorySize           int               `json:"memorySize"`
	CPUNum               int               `json:"cpuNum"`
	CPUSpeed             int               `json:"cpuSpeed"`
	State                string            `json:"state"`
	InternalID           string            `json:"internalId"`
	VMNics               []SInstanceNic    `json:"vmNics"`
	AllVolumes           []SDisk           `json:"allVolumes"`
	VMCdRoms             []SInstanceCdrome `json:"vmCdRoms"`
	ZStackTime
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instance := &SInstance{}
	err := region.client.getResource("vm-instances", instanceId, instance)
	if err != nil {
		return nil, err
	}
	if instance.State == "Destroyed" {
		return nil, cloudprovider.ErrNotFound
	}
	return instance, nil
}

func (region *SRegion) GetInstances(hostId string, instanceId string, nicId string) ([]SInstance, error) {
	instance := []SInstance{}
	params := url.Values{}
	params.Add("q", "type=UserVm")
	params.Add("q", "state!=Destroyed")
	if len(hostId) > 0 {
		params.Add("q", "lastHostUuid="+hostId)
	}
	if len(instanceId) > 0 {
		params.Add("q", "uuid="+instanceId)
	}
	if len(nicId) > 0 {
		params.Add("q", "vmNics.uuid="+nicId)
	}
	if SkipEsxi {
		params.Add("q", "hypervisorType!=ESX")
	}
	return instance, region.client.listAll("vm-instances", params, &instance)
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	ids := []string{}
	secgroups, err := instance.host.zone.region.GetSecurityGroups("", instance.UUID, "")
	if err != nil {
		return nil, err
	}
	for _, secgroup := range secgroups {
		ids = append(ids, secgroup.UUID)
	}
	return ids, nil
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetIHostId() string {
	if len(instance.LastHostUUID) > 0 {
		return instance.LastHostUUID
	}
	return instance.HostUUID
}

func (instance *SInstance) GetId() string {
	return instance.UUID
}

func (instance *SInstance) GetName() string {
	return instance.Name
}

func (instance *SInstance) GetHostname() string {
	return instance.Name
}

func (instance *SInstance) GetGlobalId() string {
	return instance.GetId()
}

func (instance *SInstance) IsEmulated() bool {
	return false
}

func (instance *SInstance) GetInstanceType() string {
	if len(instance.InstanceOfferingUUID) > 0 {
		offer, err := instance.host.zone.region.GetInstanceOffering(instance.InstanceOfferingUUID)
		if err == nil {
			return offer.Name
		}
	}
	return instance.Type
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	idisks := []cloudprovider.ICloudDisk{}
	rootDisk, err := instance.host.zone.region.GetDiskWithStorage(instance.RootVolumeUUID)
	if err != nil {
		return nil, err
	}
	idisks = append(idisks, rootDisk)
	for i := 0; i < len(instance.AllVolumes); i++ {
		if instance.AllVolumes[i].UUID != instance.RootVolumeUUID {
			dataDisk, err := instance.host.zone.region.GetDiskWithStorage(instance.AllVolumes[i].UUID)
			if err != nil {
				return nil, err
			}
			idisks = append(idisks, dataDisk)
		}
	}
	return idisks, nil
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	iNics := []cloudprovider.ICloudNic{}
	for i := 0; i < len(instance.VMNics); i++ {
		instance.VMNics[i].instance = instance
		iNics = append(iNics, &instance.VMNics[i])
	}
	return iNics, nil
}

func (instance *SInstance) GetVcpuCount() int {
	return instance.CPUNum
}

func (instance *SInstance) GetVmemSizeMB() int {
	return instance.MemorySize / 1024 / 1024
}

func (instance *SInstance) GetBootOrder() string {
	return instance.host.zone.region.GetBootOrder(instance.UUID)
}

func (region *SRegion) GetBootOrder(instanceId string) string {
	resp, err := region.client.get("vm-instances", instanceId, "boot-orders")
	if err != nil {
		return "dcn"
	}
	orders := []string{}
	err = resp.Unmarshal(&orders, "orders")
	if err != nil {
		return "dcn"
	}
	order := ""
	for _, _order := range orders {
		switch _order {
		case "CdRom":
			order += "c"
		case "HardDisk":
			order += "d"
		default:
			log.Errorf("Unknown BootOrder %s for instance %s", _order, instanceId)
		}
	}
	return order
}

func (instance *SInstance) GetVga() string {
	return "std"
}

func (instance *SInstance) GetVdi() string {
	return "vnc"
}

func (instance *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if instance.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(instance.Platform, "", "", "", "")
		instance.osInfo = &osInfo
	}
	return instance.osInfo
}

func (instance *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(instance.getNormalizedOsInfo().OsType)
}

func (instance *SInstance) GetFullOsName() string {
	return instance.Platform
}

func (instance *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(instance.getNormalizedOsInfo().OsBios)
}

func (instance *SInstance) GetOsDist() string {
	return instance.getNormalizedOsInfo().OsDistro
}

func (instance *SInstance) GetOsVersion() string {
	return instance.getNormalizedOsInfo().OsVersion
}

func (instance *SInstance) GetOsLang() string {
	return instance.getNormalizedOsInfo().OsLang
}

func (instance *SInstance) GetOsArch() string {
	return instance.getNormalizedOsInfo().OsArch
}

func (instance *SInstance) GetMachine() string {
	return "pc"
}

func (instance *SInstance) GetStatus() string {
	switch instance.State {
	case "Stopped":
		return api.VM_READY
	case "Running":
		return api.VM_RUNNING
	case "Destroyed":
		return api.VM_DEALLOCATED
	default:
		log.Errorf("Unknown instance %s status %s", instance.Name, instance.State)
		return api.VM_UNKNOWN
	}
}

func (instance *SInstance) Refresh() error {
	new, err := instance.host.zone.region.GetInstance(instance.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(instance, new)
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ZSTACK
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	err := instance.host.zone.region.StartVM(instance.UUID)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(instance, api.VM_RUNNING, 5*time.Second, 5*time.Minute)
}

func (region *SRegion) StartVM(instanceId string) error {
	params := map[string]interface{}{
		"startVmInstance": jsonutils.NewDict(),
	}
	_, err := region.client.put("vm-instances", instanceId, jsonutils.Marshal(params))
	return err
}

func (instance *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := instance.host.zone.region.StopVM(instance.UUID, opts.IsForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(instance, api.VM_READY, 5*time.Second, 5*time.Minute)
}

func (region *SRegion) StopVM(instanceId string, isForce bool) error {
	option := "grace"
	if isForce {
		option = "cold"
	}
	params := map[string]interface{}{
		"stopVmInstance": map[string]string{
			"type":   option,
			"stopHA": "true",
		},
	}
	_, err := region.client.put("vm-instances", instanceId, jsonutils.Marshal(params))
	return err
}

func (instance *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	info, err := instance.host.zone.region.GetInstanceConsoleInfo(instance.UUID)
	if err != nil {
		return nil, err
	}
	authURL, _ := url.Parse(instance.host.zone.region.client.authURL)
	url := fmt.Sprintf("%s://%s:5000/thirdparty/vnc_auto.html?host=%s&port=%d&token=%s&title=%s", info.Scheme, authURL.Hostname(), info.Hostname, info.Port, info.Token, instance.Name)
	if ver, _ := instance.host.zone.region.client.GetVersion(); ver != nil {
		if version.GE(ver.Version, "4.0.0") {
			url = fmt.Sprintf("%s://%s:5000/novnc/index.html?host=%s&port=%d&token=%s&title=%s&language=zh-CN&lowVersion=false", info.Scheme, authURL.Hostname(), info.Hostname, info.Port, info.Token, instance.Name)
		}
	}
	password, _ := instance.host.zone.region.GetInstanceConsolePassword(instance.UUID)
	if len(password) > 0 {
		url = url + fmt.Sprintf("&password=%s", password)
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        url,
		Protocol:   "zstack",
		InstanceId: instance.UUID,
		Hypervisor: api.HYPERVISOR_ZSTACK,
	}
	return ret, nil
}

func (instance *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	params := map[string]interface{}{
		"updateVmInstance": map[string]string{
			"name": input.NAME,
		},
	}
	return instance.host.zone.region.UpdateVM(instance.UUID, jsonutils.Marshal(params))
}

func (region *SRegion) UpdateVM(instanceId string, params jsonutils.JSONObject) error {
	_, err := region.client.put("vm-instances", instanceId, params)
	return err
}

func (instance *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.Password) > 0 {
		params := map[string]interface{}{
			"changeVmPassword": map[string]string{
				"account":  opts.Username,
				"password": opts.Password,
			},
		}
		err := instance.host.zone.region.UpdateVM(instance.UUID, jsonutils.Marshal(params))
		if err != nil {
			return err
		}
	}
	if len(opts.PublicKey) > 0 {
		params := map[string]interface{}{
			"setVmSshKey": map[string]string{
				"SshKey": opts.PublicKey,
			},
		}
		err := instance.host.zone.region.UpdateVM(instance.UUID, jsonutils.Marshal(params))
		if err != nil {
			return err
		}
	}
	if opts.DeleteKeypair {
		err := instance.host.zone.region.client.delete("vm-instances", fmt.Sprintf("%s/ssh-keys", instance.UUID), "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (instance *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return instance.host.zone.region.RebuildRoot(instance.UUID, desc.ImageId, desc.SysSizeGB)
}

func (region *SRegion) RebuildRoot(instanceId, imageId string, sysSizeGB int) (string, error) {
	params := map[string]interface{}{
		"changeVmImage": map[string]string{
			"imageUuid": imageId,
		},
	}
	instance := &SInstance{}
	resp, err := region.client.put("vm-instances", instanceId, jsonutils.Marshal(params))
	if err != nil {
		return "", err
	}
	err = resp.Unmarshal(instance, "inventory")
	if err != nil {
		return "", err
	}
	disk, err := region.GetDisk(instance.RootVolumeUUID)
	if err != nil {
		return "", err
	}
	if sysSizeGB > disk.GetDiskSizeMB()*1024 {
		return instance.RootVolumeUUID, region.ResizeDisk(disk.UUID, int64(sysSizeGB)*1024)
	}
	return instance.RootVolumeUUID, nil
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	offerings, err := instance.host.zone.region.GetInstanceOfferings("", "", config.Cpu, config.MemoryMB)
	if err != nil {
		return err
	}
	if len(config.InstanceType) > 0 {
		for _, offering := range offerings {
			if offering.Name == config.InstanceType {
				return instance.host.zone.region.ChangeConfig(instance.UUID, offering.UUID)
			}
		}
		offering, err := instance.host.zone.region.CreateInstanceOffering(config.InstanceType, config.Cpu, config.MemoryMB, "UserVm")
		if err != nil {
			return err
		}
		return instance.host.zone.region.ChangeConfig(instance.UUID, offering.UUID)
	}
	for _, offering := range offerings {
		log.Debugf("try instance offering %s(%s) ...", offering.Name, offering.UUID)
		err := instance.host.zone.region.ChangeConfig(instance.UUID, offering.UUID)
		if err != nil {
			log.Errorf("failed to change config for instance %s(%s) error: %v", instance.Name, instance.UUID, err)
		} else {
			return nil
		}
	}
	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (region *SRegion) ChangeConfig(instanceId, offeringId string) error {
	params := map[string]interface{}{
		"changeInstanceOffering": map[string]string{
			"instanceOfferingUuid": offeringId,
		},
	}
	return region.UpdateVM(instanceId, jsonutils.Marshal(params))
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.AttachDisk(instance.UUID, diskId)
}

func (region *SRegion) AttachDisk(instanceId string, diskId string) error {
	_, err := region.client.post(fmt.Sprintf("volumes/%s/vm-instances/%s", diskId, instanceId), jsonutils.NewDict())
	return err
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.DetachDisk(instance.UUID, diskId)
}

func (region *SRegion) DetachDisk(instanceId, diskId string) error {
	url := fmt.Sprintf("volumes/%s/vm-instances?vmUuid=%s", diskId, instanceId)
	err := region.client.delete(url, "", "")
	if err != nil && strings.Contains(err.Error(), "is not attached to any vm") {
		return nil
	}
	return err
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	disks, err := instance.GetIDisks()
	if err != nil {
		return errors.Wrapf(err, "GetIDisks")
	}
	err = instance.host.zone.region.DeleteVM(instance.UUID)
	if err != nil {
		return errors.Wrapf(err, "DeleteVM")
	}
	for i := range disks {
		if disks[i].GetDiskType() != api.DISK_TYPE_SYS && disks[i].GetIsAutoDelete() {
			err = disks[i].Delete(ctx)
			if err != nil {
				log.Warningf("delete disk %s failed %s", disks[i].GetId(), err)
			}
		}
	}
	return nil
}

func (region *SRegion) DeleteVM(instanceId string) error {
	err := region.client.delete("vm-instances", instanceId, "Enforcing")
	if err != nil {
		return err
	}
	params := map[string]interface{}{
		"expungeVmInstance": jsonutils.NewDict(),
	}
	_, err = region.client.put("vm-instances", instanceId, jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrapf(err, "expungeVmInstance")
	}
	return nil
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, err := instance.host.zone.region.GetEips("", instance.UUID)
	if err != nil {
		return nil, err
	}
	if len(eips) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(eips) == 1 {
		return &eips[0], nil
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	currentIds, err := instance.GetSecurityGroupIds()
	if err != nil {
		return err
	}
	for _, id := range currentIds {
		if !utils.IsInStringArray(id, secgroupIds) {
			err := instance.host.zone.region.RevokeSecurityGroup(instance.UUID, id)
			if err != nil {
				return err
			}
		}
	}
	for _, id := range secgroupIds {
		if !utils.IsInStringArray(id, currentIds) {
			err := instance.host.zone.region.AssignSecurityGroup(instance.UUID, id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (region *SRegion) AssignSecurityGroup(instanceId, secgroupId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	secgroup, err := region.GetSecurityGroup(secgroupId)
	if err != nil {
		return err
	}
	if len(instance.VMNics) > 0 {
		if !utils.IsInStringArray(instance.VMNics[0].L3NetworkUUID, secgroup.AttachedL3NetworkUUIDs) {
			resource := fmt.Sprintf("security-groups/%s/l3-networks/%s", secgroupId, instance.VMNics[0].L3NetworkUUID)
			_, err := region.client.post(resource, jsonutils.NewDict())
			if err != nil {
				return err
			}
		}
		params := map[string]interface{}{
			"params": map[string]interface{}{
				"vmNicUuids": []string{instance.VMNics[0].UUID},
			},
		}
		resource := fmt.Sprintf("security-groups/%s/vm-instances/nics", secgroupId)
		_, err = region.client.post(resource, jsonutils.Marshal(params))
		return err
	}
	return nil
}

func (region *SRegion) RevokeSecurityGroup(instanceId, secgroupId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		return err
	}
	for _, nic := range instance.VMNics {
		resource := fmt.Sprintf("security-groups/%s/vm-instances/nics?vmNicUuids=%s", secgroupId, nic.UUID)
		err := region.client.delete(resource, "", "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *SInstance) GetBillingType() string {
	return ""
}

func (instance *SInstance) GetCreatedAt() time.Time {
	return instance.CreateDate
}

func (instance *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (instance *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) GetProjectId() string {
	return ""
}

func (instance *SInstance) GetError() error {
	return nil
}

type SConsoleInfo struct {
	Scheme   string `json:"scheme"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Token    string `json:"token"`
}

func (region *SRegion) GetInstanceConsoleInfo(instnaceId string) (*SConsoleInfo, error) {
	params := map[string]interface{}{
		"params": map[string]string{
			"vmInstanceUuid": instnaceId,
		},
	}
	resp, err := region.client.post("consoles", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	info := &SConsoleInfo{}
	err = resp.Unmarshal(info, "inventory")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (region *SRegion) GetInstanceConsolePassword(instnaceId string) (string, error) {
	resp, err := region.client.get("vm-instances", instnaceId, "console-passwords")
	if err != nil {
		return "", err
	}
	if resp.Contains("consolePassword") {
		return resp.GetString("consolePassword")
	}
	return "", nil
}
