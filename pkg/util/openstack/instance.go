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

package openstack

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/onecloud/pkg/multicloud"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
)

const (
	INSTANCE_STATUS_ACTIVE            = "ACTIVE"            //The server is active.
	INSTANCE_STATUS_BUILD             = "BUILD"             //The server has not finished the original build process.
	INSTANCE_STATUS_DELETED           = "DELETED"           //The server is permanently deleted.
	INSTANCE_STATUS_ERROR             = "ERROR"             //The server is in error.
	INSTANCE_STATUS_HARD_REBOOT       = "HARD_REBOOT"       //The server is hard rebooting. This is equivalent to pulling the power plug on a physical server, plugging it back in, and rebooting it.
	INSTANCE_STATUS_MIGRATING         = "MIGRATING"         //The server is being migrated to a new host.
	INSTANCE_STATUS_PASSWORD          = "PASSWORD"          //The password is being reset on the server.
	INSTANCE_STATUS_PAUSED            = "PAUSED"            //In a paused state, the state of the server is stored in RAM.A paused server continues to run in frozen state.
	INSTANCE_STATUS_REBOOT            = "REBOOT"            //The server is in a soft reboot state. A reboot command was passed to the operating system.
	INSTANCE_STATUS_REBUILD           = "REBUILD"           //The server is currently being rebuilt from an image.
	INSTANCE_STATUS_RESCUE            = "RESCUE"            //The server is in rescue mode. A rescue image is running with the original server image attached.
	INSTANCE_STATUS_RESIZE            = "RESIZE"            //Server is performing the differential copy of data that changed during its initial copy. Server is down for this stage.
	INSTANCE_STATUS_REVERT_RESIZE     = "REVERT_RESIZE"     //The resize or migration of a server failed for some reason. The destination server is being cleaned up and the original source server is restarting.
	INSTANCE_STATUS_SHELVED           = "SHELVED"           // The server is in shelved state. Depending on the shelve offload time, the server will be automatically shelved offloaded.
	INSTANCE_STATUS_SHELVED_OFFLOADED = "SHELVED_OFFLOADED" // The shelved server is offloaded (removed from the compute host) and it needs unshelved action to be used again.
	INSTANCE_STATUS_SHUTOFF           = "SHUTOFF"           //The server is powered off and the disk image still persists.
	INSTANCE_STATUS_SOFT_DELETED      = "SOFT_DELETED"      //The server is marked as deleted but the disk images are still available to restore.
	INSTANCE_STATUS_SUSPENDED         = "SUSPENDED"         //The server is suspended, either by request or necessity. This status appears for only the XenServer/XCP, KVM, and ESXi hypervisors. Administrative users can suspend an instance if it is infrequently used or to perform system maintenance. When you suspend an instance, its VM state is stored on disk, all memory is written to disk, and the virtual machine is stopped. Suspending an instance is similar to placing a device in hibernation; memory and vCPUs become available to create other instances.
	INSTANCE_STATUS_UNKNOWN           = "UNKNOWN"           //The state of the server is unknown. Contact your cloud provider.
	INSTANCE_STATUS_VERIFY_RESIZE     = "VERIFY_RESIZE"     //System is awaiting confirmation that the server is operational after a move or resize.
)

type SPrivate struct {
	MacAddr string `json:"OS-EXT-IPS-MAC:mac_addr,omitempty"`
	Addr    string `json:"OS-EXT-IPS:type,omitempty"`
	Version int
}

type SecurityGroup struct {
	ID          string
	Name        string
	Description string
}

type ExtraSpecs struct {
	CpuPolicy   string `json:"hw:cpu_policy,omitempty"`
	MemPageSize int    `json:"hw:mem_page_size,omitempty"`
}

type Resource struct {
	ID    string
	Links []Link
}

type Image struct {
	ID    string
	Links []Link
}

type VolumesAttached struct {
	ID                  string
	DeleteOnTermination bool
}

type SFault struct {
	Message string
	Code    int
	Details string
}

type SInstance struct {
	multicloud.SInstanceBase

	host *SHost

	DiskConfig         string    `json:"OS-DCF:diskConfig,omitempty"`
	AvailabilityZone   string    `json:"OS-EXT-AZ:availability_zone,omitempty"`
	Host               string    `json:"OS-EXT-SRV-ATTR:host,omitempty"`
	Hostname           string    `json:"OS-EXT-SRV-ATTR:hostname,omitempty"`
	HypervisorHostname string    `json:"OS-EXT-SRV-ATTR:hypervisor_hostname,omitempty"`
	InstanceName       string    `json:"OS-EXT-SRV-ATTR:instance_name,omitempty"`
	KernelID           string    `json:"OS-EXT-SRV-ATTR:kernel_id,omitempty"`
	LaunchIndex        int       `json:"OS-EXT-SRV-ATTR:launch_index,omitempty"`
	RamdiskID          string    `json:"OS-EXT-SRV-ATTR:ramdisk_id,omitempty"`
	ReservationID      string    `json:"OS-EXT-SRV-ATTR:reservation_id,omitempty"`
	RootDeviceName     string    `json:"OS-EXT-SRV-ATTR:root_device_name,omitempty"`
	UserData           string    `json:"OS-EXT-SRV-ATTR:user_data,omitempty"`
	PowerState         int       `json:"OS-EXT-STS:power_state,omitempty"`
	TaskState          string    `json:"OS-EXT-STS:task_state,omitempty"`
	VmState            string    `json:"OS-EXT-STS:vm_state,omitempty"`
	LaunchedAt         time.Time `json:"OS-SRV-USG:launched_at,omitempty"`
	TerminatedAt       string    `json:"OS-SRV-USG:terminated_at,omitempty"`

	AccessIPv4               string
	AccessIPv6               string
	Addresses                map[string][]SInstanceNic
	ConfigDrive              string
	Created                  time.Time
	Description              string
	Flavor                   SFlavor
	HostID                   string
	HostStatus               string
	ID                       string
	image                    Image //有可能是字符串
	KeyName                  string
	Links                    []Link
	Locked                   bool
	Metadata                 Metadata
	Name                     string
	VolumesAttached          []VolumesAttached `json:"os-extended-volumes:volumes_attached,omitempty"`
	Progress                 int
	SecurityGroups           []SecurityGroup
	Status                   string
	Tags                     []string
	TenantID                 string
	TrustedImageCertificates []string
	Updated                  time.Time
	UserID                   string
	Fault                    SFault
}

func (region *SRegion) GetSecurityGroupsByInstance(instanceId string) ([]SecurityGroup, error) {
	_, resp, err := region.Get("compute", fmt.Sprintf("/servers/%s/os-security-groups", instanceId), "", nil)
	if err != nil {
		return nil, err
	}
	secgroups := []SecurityGroup{}
	return secgroups, resp.Unmarshal(&secgroups, "security_groups")
}

func (region *SRegion) GetInstances(hostName string) ([]SInstance, error) {
	_, maxVersion, _ := region.GetVersion("compute")
	url := "/servers/detail?all_tenants=True"
	instances := []SInstance{}
	for len(url) > 0 {
		_, resp, err := region.List("compute", url, maxVersion, nil)
		if err != nil {
			return nil, err
		}
		_instances := []SInstance{}
		err = resp.Unmarshal(&_instances, "servers")
		if err != nil {
			return nil, errors.Wrap(err, `resp.Unmarshal(&_instances, "servers")`)
		}
		instances = append(instances, _instances...)
		url = ""
		if resp.Contains("servers_links") {
			nextLink := []SNextLink{}
			err = resp.Unmarshal(&nextLink, "servers_links")
			if err != nil {
				return nil, errors.Wrap(err, `resp.Unmarshal(&nextLink, "servers")`)
			}
			for _, next := range nextLink {
				if next.Rel == "next" {
					url = next.Href
					break
				}
			}
		}
	}
	result := []SInstance{}
	for i := 0; i < len(instances); i++ {
		if len(hostName) == 0 || hostName == instances[i].Host {
			result = append(result, instances[i])
		}
	}
	return result, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	_, maxVersion, _ := region.GetVersion("compute")
	_, resp, err := region.Get("compute", "/servers/"+instanceId, maxVersion, nil)
	if err != nil {
		return nil, err
	}
	instance := &SInstance{}
	return instance, resp.Unmarshal(instance, "server")
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroupIds := []string{}
	secgroups, err := instance.host.zone.region.GetSecurityGroupsByInstance(instance.ID)
	if err != nil {
		return nil, err
	}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.ID)
	}
	return secgroupIds, nil
}

func (instance *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	instance.fetchFlavor()

	priceKey := fmt.Sprintf("%s::%s", instance.host.zone.ZoneName, instance.Flavor.OriginalName)
	data.Add(jsonutils.NewString(priceKey), "price_key")

	data.Add(jsonutils.NewString(instance.host.zone.GetGlobalId()), "zone_ext_id")
	return data
}

func (instance *SInstance) GetCreateTime() time.Time {
	return instance.Created
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetId() string {
	return instance.ID
}

func (instance *SInstance) GetName() string {
	return instance.Name
}

func (instance *SInstance) GetGlobalId() string {
	return instance.ID
}

func (instance *SInstance) IsEmulated() bool {
	return false
}

func (instance *SInstance) fetchFlavor() error {
	if len(instance.Flavor.ID) > 0 && instance.Flavor.Vcpus == 0 {
		flavor, err := instance.host.zone.region.GetFlavor(instance.Flavor.ID)
		if err != nil {
			return err
		}
		instance.Flavor = *flavor
	}
	return nil
}

func (instance *SInstance) GetInstanceType() string {
	instance.fetchFlavor()
	return instance.Flavor.GetName()
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SDisk{}
	for i := 0; i < len(instance.VolumesAttached); i++ {
		disk, err := instance.host.zone.region.GetDisk(instance.VolumesAttached[i].ID)
		if err != nil {
			return nil, err
		}
		disks = append(disks, *disk)
	}
	iDisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		store, err := instance.host.zone.getStorageByCategory(disks[i].VolumeType)
		if err != nil {
			return nil, err
		}
		disks[i].storage = store
		iDisks = append(iDisks, &disks[i])
	}
	return iDisks, nil
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := []cloudprovider.ICloudNic{}
	for networkName, address := range instance.Addresses {
		for i := 0; i < len(address); i++ {
			if instance.Addresses[networkName][i].Type == "fixed" {
				instance.Addresses[networkName][i].instance = instance
				nics = append(nics, &instance.Addresses[networkName][i])
			}
		}
	}
	return nics, nil
}

func (instance *SInstance) GetVcpuCount() int {
	instance.fetchFlavor()
	return instance.Flavor.Vcpus
}

func (instance *SInstance) GetVmemSizeMB() int {
	instance.fetchFlavor()
	return instance.Flavor.RAM
}

func (instance *SInstance) GetBootOrder() string {
	return "dcn"
}

func (instance *SInstance) GetVga() string {
	return "std"
}

func (instance *SInstance) GetVdi() string {
	return "vnc"
}

func (instance *SInstance) GetOSType() string {
	return "Linux"
}

func (instance *SInstance) GetOSName() string {
	return "Linux"
}

func (instance *SInstance) GetBios() string {
	return "BIOS"
}

func (instance *SInstance) GetMachine() string {
	return "pc"
}

func (instance *SInstance) GetStatus() string {
	switch instance.Status {
	case INSTANCE_STATUS_ACTIVE, INSTANCE_STATUS_RESCUE:
		return api.VM_RUNNING
	case INSTANCE_STATUS_BUILD, INSTANCE_STATUS_PASSWORD:
		return api.VM_DEPLOYING
	case INSTANCE_STATUS_DELETED:
		return api.VM_DELETING
	case INSTANCE_STATUS_HARD_REBOOT, INSTANCE_STATUS_REBOOT:
		return api.VM_STARTING
	case INSTANCE_STATUS_MIGRATING:
		return api.VM_MIGRATING
	case INSTANCE_STATUS_PAUSED, INSTANCE_STATUS_SUSPENDED:
		return api.VM_SUSPEND
	case INSTANCE_STATUS_RESIZE:
		return api.VM_CHANGE_FLAVOR
	case INSTANCE_STATUS_VERIFY_RESIZE:
		// API请求更改配置后，状态先回变更到 INSTANCE_STATUS_RESIZE 等待一会变成此状态
		// 到达此状态后需要再次发送确认请求，变更才会生效
		// 此状态不能和INSTANCE_STATUS_RESIZE返回一样，避免在INSTANCE_STATUS_RESIZE状态下发送确认请求，导致更改配置失败
		return api.VM_SYNC_CONFIG
	case INSTANCE_STATUS_SHELVED, INSTANCE_STATUS_SHELVED_OFFLOADED, INSTANCE_STATUS_SHUTOFF, INSTANCE_STATUS_SOFT_DELETED:
		return api.VM_READY
	default:
		return api.VM_UNKNOWN
	}
}

func (instance *SInstance) Refresh() error {
	new, err := instance.host.zone.region.GetInstance(instance.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(instance, new)
}

func (instance *SInstance) UpdateVM(ctx context.Context, name string) error {
	if instance.Name != name {
		params := map[string]map[string]string{
			"server": {
				"name": name,
			},
		}
		_, _, err := instance.host.zone.region.Update("compute", "/servers/"+instance.ID, "", jsonutils.Marshal(params))
		return err
	}
	return nil
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_OPENSTACK
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	if err := instance.host.zone.region.StartVM(instance.ID); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(instance, api.VM_RUNNING, 10*time.Second, 8*time.Minute)
}

func (instance *SInstance) StopVM(ctx context.Context, isForce bool) error {
	if err := instance.host.zone.region.StopVM(instance.ID, isForce); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(instance, api.VM_READY, 10*time.Second, 8*time.Minute)
}

func (region *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	_, maxVersion, _ := region.GetVersion("compute")
	params := map[string]map[string]string{
		"remote_console": {
			"protocol": "vnc",
			"type":     "novnc",
		},
	}
	_, resp, err := region.Post("compute", fmt.Sprintf("/servers/%s/remote-consoles", instanceId), maxVersion, jsonutils.Marshal(params))
	if err != nil {
		return "", err
	}
	return resp.GetString("remote_console", "url")
}

func (instance *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	url, err := instance.host.zone.region.GetInstanceVNCUrl(instance.ID)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(url), "url")
	ret.Add(jsonutils.NewString("openstack"), "protocol")
	ret.Add(jsonutils.NewString(instance.ID), "instance_id")
	return ret, nil
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return instance.host.zone.region.DeployVM(instance.ID, name, password, publicKey, deleteKeypair, description)
}

func (instance *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	sysDiskId := ""
	if len(instance.VolumesAttached) > 0 {
		sysDiskId = instance.VolumesAttached[0].ID
	}
	return sysDiskId, instance.host.zone.region.ReplaceSystemDisk(instance.ID, imageId, passwd, publicKey, sysSizeGB)
}

func (instance *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	if instance.GetVcpuCount() != ncpu || instance.GetVmemSizeMB() != vmem {
		flavorId, err := instance.host.zone.region.syncFlavor("", ncpu, vmem, 40)
		if err != nil {
			return err
		}
		return instance.host.zone.region.ChangeConfig(instance, flavorId)
	}
	return nil
}

func (instance *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	if instance.GetInstanceType() != instanceType {
		flavorId, err := instance.host.zone.region.syncFlavor(instanceType, 0, 0, 0)
		if err != nil {
			return err
		}
		return instance.host.zone.region.ChangeConfig(instance, flavorId)
	}
	return nil
}

func (region *SRegion) ChangeConfig(instance *SInstance, flavorId string) error {
	params := map[string]map[string]string{
		"resize": {
			"flavorRef": flavorId,
		},
	}
	_, maxVersion, _ := region.GetVersion("compute")
	_, _, err := region.Post("compute", fmt.Sprintf("/servers/%s/action", instance.ID), maxVersion, jsonutils.Marshal(params))
	if err != nil {
		return err
	}
	if err := cloudprovider.WaitStatus(instance, api.VM_SYNC_CONFIG, time.Second*3, time.Minute*4); err != nil {
		return err
	}
	return region.instanceOperation(instance.ID, "confirmResize")
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.AttachDisk(instance.ID, diskId)
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.DetachDisk(instance.ID, diskId)
}

func (region *SRegion) CreateInstance(name string, imageId string, instanceType string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, networkId string, ipAddr string,
	keypair string, userData string, bc *billing.SBillingCycle) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) instanceOperation(instanceId, operate string) error {
	params := jsonutils.Marshal(map[string]string{operate: ""})
	_, maxVersion, _ := region.GetVersion("compute")
	_, _, err := region.Post("compute", fmt.Sprintf("/servers/%s/action", instanceId), maxVersion, params)
	return err
}

func (region *SRegion) doStopVM(instanceId string, isForce bool) error {
	return region.instanceOperation(instanceId, "os-stop")
}

func (region *SRegion) doDeleteVM(instanceId string) error {
	return region.instanceOperation(instanceId, "forceDelete")
}

func (region *SRegion) StartVM(instanceId string) error {
	return region.instanceOperation(instanceId, "os-start")
}

func (region *SRegion) StopVM(instanceId string, isForce bool) error {
	return region.doStopVM(instanceId, isForce)
}

func (region *SRegion) DeleteVM(instanceId string) error {
	instance, err := region.GetInstance(instanceId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}
		log.Errorf("failed to get instance %s %v", instanceId, err)
		return err
	}
	status := instance.GetStatus()
	log.Debugf("Instance status on delete is %s", status)
	if status != api.VM_READY {
		log.Warningf("DeleteVM: vm status is %s expect %s", status, api.VM_READY)
	}
	return region.doDeleteVM(instanceId)
}

func (region *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	if len(password) > 0 {
		params := map[string]map[string]string{
			"changePassword": {
				"adminPass": password,
			},
		}
		_, maxVersion, _ := region.GetVersion("compute")
		_, _, err := region.Post("compute", fmt.Sprintf("/servers/%s/action", instanceId), maxVersion, jsonutils.Marshal(params))
		return err
	}
	return nil
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return instance.host.zone.region.DeleteVM(instance.ID)
}

func (region *SRegion) ReplaceSystemDisk(instanceId string, imageId string, passwd string, publicKey string, sysDiskSizeGB int) error {
	params := map[string]map[string]string{
		"rebuild": {
			"imageRef": imageId,
		},
	}

	if len(publicKey) > 0 {
		keypairName, err := region.syncKeypair(instanceId, publicKey)
		if err != nil {
			return err
		}
		params["rebuild"]["key_name"] = keypairName
	}

	if len(passwd) > 0 {
		params["rebuild"]["adminPass"] = passwd
	}

	_, maxVersion, _ := region.GetVersion("compute")
	_, _, err := region.Post("compute", fmt.Sprintf("/servers/%s/action", instanceId), maxVersion, jsonutils.Marshal(params))
	return err
}

func (region *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DetachDisk(instanceId string, diskId string) error {
	_, err := region.Delete("compute", fmt.Sprintf("/servers/%s/os-volume_attachments/%s", instanceId, diskId), "")
	if err != nil {
		return err
	}
	status := ""
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*10 {
		disk, err := region.GetDisk(diskId)
		if err != nil {
			return err
		}
		status = disk.Status
		log.Debugf("status %s expect %s", status, DISK_STATUS_AVAILABLE)
		if status == DISK_STATUS_AVAILABLE {
			return nil
		}
		time.Sleep(time.Second * 15)
	}
	return fmt.Errorf("timeout for waitting detach disk, current status: %s", status)
}

func (region *SRegion) AttachDisk(instanceId string, diskId string) error {
	params := map[string]map[string]string{
		"volumeAttachment": {
			"volumeId": diskId,
		},
	}
	_, _, err := region.Post("compute", fmt.Sprintf("/servers/%s/os-volume_attachments", instanceId), "", jsonutils.Marshal(params))
	if err != nil {
		return err
	}
	status := ""
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*10 {
		disk, err := region.GetDisk(diskId)
		if err != nil {
			return err
		}
		status = disk.Status
		log.Debugf("status %s expect %s", status, DISK_STATUS_IN_USE)
		if status == DISK_STATUS_IN_USE {
			return nil
		}
		time.Sleep(time.Second * 15)
	}
	return fmt.Errorf("timeout for waitting attach disk, current status: %s", status)
}

func (instance *SInstance) AssignSecurityGroup(secgroupId string) error {
	if secgroupId == SECGROUP_NOT_SUPPORT {
		return fmt.Errorf("Security groups are not supported. Security group components are not installed")
	}
	secgroup, err := instance.host.zone.region.GetSecurityGroup(secgroupId)
	if err != nil {
		return err
	}
	params := map[string]map[string]string{
		"addSecurityGroup": {
			"name": secgroup.Name,
		},
	}
	_, _, err = instance.host.zone.region.Post("compute", fmt.Sprintf("/servers/%s/action", instance.ID), "", jsonutils.Marshal(params))
	return err
}

func (instance *SInstance) RevokeSecurityGroup(secgroupId string) error {
	// 若OpenStack不支持安全组，则忽略解绑安全组
	if secgroupId == SECGROUP_NOT_SUPPORT {
		return nil
	}
	secgroup, err := instance.host.zone.region.GetSecurityGroup(secgroupId)
	if err != nil {
		return err
	}
	params := map[string]map[string]string{
		"removeSecurityGroup": {
			"name": secgroup.Name,
		},
	}
	_, _, err = instance.host.zone.region.Post("compute", fmt.Sprintf("/servers/%s/action", instance.ID), "", jsonutils.Marshal(params))
	return err
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	secgroups, err := instance.host.zone.region.GetSecurityGroupsByInstance(instance.ID)
	if err != nil {
		return err
	}
	originIds := []string{}
	for _, secgroup := range secgroups {
		if !utils.IsInStringArray(secgroup.ID, secgroupIds) {
			if err := instance.RevokeSecurityGroup(secgroup.ID); err != nil {
				return err
			}
		}
		originIds = append(originIds, secgroup.ID)
	}
	for _, secgroupId := range secgroupIds {
		if !utils.IsInStringArray(secgroupId, originIds) {
			if err := instance.AssignSecurityGroup(secgroupId); err != nil {
				return err
			}
		}
	}
	return nil
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	for networkName, address := range instance.Addresses {
		for i := 0; i < len(address); i++ {
			if instance.Addresses[networkName][i].Type == "floating" {
				return instance.host.zone.region.GetEipByIp(instance.Addresses[networkName][i].Addr)
			}
		}
	}
	return nil, nil
}

func (instance *SInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (instance *SInstance) GetCreatedAt() time.Time {
	return instance.Created
}

func (instance *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (instance *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) RenewInstances(instanceId []string, bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) GetProjectId() string {
	return instance.TenantID
}

func (self *SInstance) GetError() error {
	if self.Status == INSTANCE_STATUS_ERROR && len(self.Fault.Message) > 0 {
		return fmt.Errorf(self.Fault.Message)
	}
	return nil
}
