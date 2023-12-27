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
	"net/url"
	"time"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	Id          string
	Name        string
	Description string
}

type ExtraSpecs struct {
	CpuPolicy   string `json:"hw:cpu_policy,omitempty"`
	MemPageSize string `json:"hw:mem_page_size,omitempty"`
}

type Resource struct {
	Id    string
	Links []Link
}

type VolumesAttached struct {
	Id                  string
	DeleteOnTermination bool
}

type SFault struct {
	Message string
	Code    int
	Details string
}

type SInstance struct {
	multicloud.SInstanceBase
	OpenStackTags
	host *SHypervisor

	imageObj *SImage

	DiskConfig         string    `json:"OS-DCF:diskConfig,omitempty"`
	AvailabilityZone   string    `json:"OS-EXT-AZ:availability_zone,omitempty"`
	Host               string    `json:"OS-EXT-SRV-ATTR:host,omitempty"`
	Hostname           string    `json:"OS-EXT-SRV-ATTR:hostname,omitempty"`
	HypervisorHostname string    `json:"OS-EXT-SRV-ATTR:hypervisor_hostname,omitempty"`
	InstanceName       string    `json:"OS-EXT-SRV-ATTR:instance_name,omitempty"`
	KernelId           string    `json:"OS-EXT-SRV-ATTR:kernel_id,omitempty"`
	LaunchIndex        int       `json:"OS-EXT-SRV-ATTR:launch_index,omitempty"`
	RamdiskId          string    `json:"OS-EXT-SRV-ATTR:ramdisk_id,omitempty"`
	ReservationId      string    `json:"OS-EXT-SRV-ATTR:reservation_id,omitempty"`
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
	HostId                   string
	HostStatus               string
	Id                       string
	Image                    jsonutils.JSONObject `json:"image"` //有可能是字符串
	KeyName                  string
	Links                    []Link
	Locked                   bool
	Name                     string
	VolumesAttached          []VolumesAttached `json:"os-extended-volumes:volumes_attached,omitempty"`
	Progress                 int
	SecurityGroups           []SecurityGroup
	Status                   string
	Tags                     []string
	TenantId                 string
	TrustedImageCertificates []string
	Updated                  time.Time
	UserId                   string
	Fault                    SFault
}

func (region *SRegion) GetSecurityGroupsByInstance(instanceId string) ([]SecurityGroup, error) {
	resource := fmt.Sprintf("/servers/%s/os-security-groups", instanceId)
	resp, err := region.ecsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	secgroups := []SecurityGroup{}
	err = resp.Unmarshal(&secgroups, "security_groups")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return secgroups, nil
}

func (region *SRegion) GetInstances(host string) ([]SInstance, error) {
	instances := []SInstance{}
	resource := "/servers/detail"
	query := url.Values{}
	query.Set("all_tenants", "True")
	for {
		resp, err := region.ecsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "ecsList")
		}
		part := struct {
			Servers      []SInstance
			ServersLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		for i := range part.Servers {
			if len(host) == 0 || part.Servers[i].Host == host || part.Servers[i].HypervisorHostname == host {
				instances = append(instances, part.Servers[i])
			}
		}
		marker := part.ServersLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return instances, nil
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	resource := "/servers/" + instanceId
	resp, err := region.ecsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	instance := &SInstance{}
	err = resp.Unmarshal(instance, "server")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarsha")
	}
	return instance, nil
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroupIds := []string{}
	secgroups, err := instance.host.zone.region.GetSecurityGroupsByInstance(instance.Id)
	if err != nil {
		return nil, err
	}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.Id)
	}
	return secgroupIds, nil
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetId() string {
	return instance.Id
}

func (instance *SInstance) GetName() string {
	return instance.Name
}

func (instance *SInstance) GetHostname() string {
	return instance.Hostname
}

func (instance *SInstance) GetGlobalId() string {
	return instance.Id
}

func (instance *SInstance) IsEmulated() bool {
	return false
}

func (instance *SInstance) fetchFlavor() error {
	if len(instance.Flavor.Id) > 0 && instance.Flavor.Vcpus == 0 {
		flavor, err := instance.host.zone.region.GetFlavor(instance.Flavor.Id)
		if err != nil {
			return err
		}
		instance.Flavor = *flavor
	}
	return nil
}

func (instance *SInstance) GetInstanceType() string {
	err := instance.fetchFlavor()
	if err != nil {
		return ""
	}
	return instance.Flavor.GetName()
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SDisk{}
	hasSysDisk := false
	for i := 0; i < len(instance.VolumesAttached); i++ {
		disk, err := instance.host.zone.region.GetDisk(instance.VolumesAttached[i].Id)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDisk(%s)", instance.VolumesAttached[i].Id)
		}
		disks = append(disks, *disk)
		if disk.GetDiskType() == api.DISK_TYPE_SYS {
			hasSysDisk = true
		}
	}
	idisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		store, err := instance.host.zone.getStorageByCategory(disks[i].VolumeType, disks[i].Host)
		if err != nil {
			return nil, errors.Wrapf(err, "getStorageByCategory(%s.%s)", disks[i].Id, disks[i].VolumeType)
		}
		disks[i].storage = store
		idisks = append(idisks, &disks[i])
	}

	if !hasSysDisk {
		store := &SNovaStorage{zone: instance.host.zone, host: instance.host}
		disk := &SNovaDisk{storage: store, instanceId: instance.Id, region: instance.host.zone.region}
		idisks = append([]cloudprovider.ICloudDisk{disk}, idisks...)
	}

	return idisks, nil
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics, err := instance.host.zone.region.GetInstancePorts(instance.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstancePorts")
	}
	inics := []cloudprovider.ICloudNic{}
	for i := range nics {
		nics[i].region = instance.host.zone.region
		inics = append(inics, &nics[i])
	}
	return inics, nil
}

func (instance *SInstance) GetVcpuCount() int {
	err := instance.fetchFlavor()
	if err != nil {
		return 0
	}
	return instance.Flavor.Vcpus
}

func (instance *SInstance) GetVmemSizeMB() int {
	err := instance.fetchFlavor()
	if err != nil {
		return 0
	}
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

func (instance *SInstance) getImage() *SImage {
	if instance.imageObj == nil && instance.Image != nil {
		imageId, _ := instance.Image.GetString("id")
		if len(imageId) == 0 {
			imageId, _ = instance.Image.GetString()
		}
		if len(imageId) > 0 {
			image, _ := instance.host.zone.region.GetImage(imageId)
			if image != nil {
				instance.imageObj = image
			}
		}
	}
	return instance.imageObj
}

func (instance *SInstance) GetOsType() cloudprovider.TOsType {
	img := instance.getImage()
	if img != nil {
		return img.GetOsType()
	}
	return cloudprovider.OsTypeLinux
}

func (instance *SInstance) GetFullOsName() string {
	img := instance.getImage()
	if img != nil {
		return img.GetFullOsName()
	}
	return ""
}

func (instance *SInstance) GetBios() cloudprovider.TBiosType {
	img := instance.getImage()
	if img != nil {
		return img.GetBios()
	}
	return "BIOS"
}

func (instance *SInstance) GetOsDist() string {
	img := instance.getImage()
	if img != nil {
		return img.GetOsDist()
	}
	return ""
}

func (instance *SInstance) GetOsVersion() string {
	img := instance.getImage()
	if img != nil {
		return img.GetOsVersion()
	}
	return ""
}

func (instance *SInstance) GetOsLang() string {
	img := instance.getImage()
	if img != nil {
		return img.GetOsLang()
	}
	return ""
}

func (instance *SInstance) GetOsArch() string {
	img := instance.getImage()
	if img != nil {
		return img.GetOsArch()
	}
	return ""
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
	_instance, err := instance.host.zone.region.GetInstance(instance.Id)
	if err != nil {
		return err
	}
	instance.Addresses = nil
	instance.VolumesAttached = nil
	instance.SecurityGroups = nil
	instance.Tags = nil
	return jsonutils.Update(instance, _instance)
}

func (instance *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	if instance.Name != input.NAME {
		params := map[string]map[string]string{
			"server": {
				"name": input.NAME,
			},
		}
		resource := "/servers/" + instance.Id
		_, err := instance.host.zone.region.ecsUpdate(resource, params)
		return err
	}
	return nil
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_OPENSTACK
}

func (self *SInstance) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := self.GetTags()
	if err != nil {
		return errors.Wrapf(err, "GetTags")
	}
	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}
	for k := range removed {
		err = self.host.zone.region.DeleteTags(self.Id, k)
		if err != nil {
			return errors.Wrapf(err, "DeleteTags %s", k)
		}
	}
	if len(added) > 0 {
		return self.host.zone.region.CreateTags(self.Id, added)
	}
	return nil
}

func (self *SRegion) DeleteTags(instanceId string, key string) error {
	resource := fmt.Sprintf("/servers/%s/metadata/%s", instanceId, key)
	_, err := self.ecsDelete(resource)
	return err
}

func (self *SRegion) CreateTags(instanceId string, tags map[string]string) error {
	params := map[string]interface{}{
		"metadata": tags,
	}
	resource := fmt.Sprintf("/servers/%s/metadata", instanceId)
	_, err := self.ecsPost(resource, params)
	return err
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	err := instance.host.zone.region.StartVM(instance.Id)
	if err != nil {
		return errors.Wrapf(err, "StartVM(%s)", instance.Id)
	}
	return cloudprovider.WaitStatus(instance, api.VM_RUNNING, 10*time.Second, 8*time.Minute)
}

func (instance *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	err := instance.host.zone.region.StopVM(instance.Id, opts.IsForce)
	if err != nil {
		return errors.Wrapf(err, "StopVM(%s)", instance.Id)
	}
	return cloudprovider.WaitStatus(instance, api.VM_READY, 10*time.Second, 8*time.Minute)
}

func (region *SRegion) GetInstanceVNCUrl(instanceId string, origin bool) (*cloudprovider.ServerVncOutput, error) {
	params := map[string]map[string]string{
		"remote_console": {
			"protocol": "vnc",
			"type":     "novnc",
		},
	}
	resource := fmt.Sprintf("/servers/%s/remote-consoles", instanceId)
	resp, err := region.ecsPost(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsPost")
	}
	ret := &cloudprovider.ServerVncOutput{
		Protocol:   "openstack",
		InstanceId: instanceId,
		Hypervisor: api.HYPERVISOR_OPENSTACK,
	}

	ret.Url, err = resp.GetString("remote_console", "url")
	if err != nil {
		return nil, errors.Wrapf(err, "remote_console")
	}

	if origin {
		return ret, nil
	}

	token := string([]byte(ret.Url)[len(ret.Url)-36:])
	vncUrl, _ := url.Parse(ret.Url)
	ret.Url = fmt.Sprintf("ws://%s?token=%s", vncUrl.Host, token)
	ret.Protocol = "vnc"
	return ret, nil
}

func (region *SRegion) GetInstanceVNC(instanceId string, origin bool) (*cloudprovider.ServerVncOutput, error) {
	params := map[string]map[string]string{
		"os-getVNCConsole": {
			"type": "novnc",
		},
	}
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	resp, err := region.ecsPost(resource, params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsPost")
	}
	ret := &cloudprovider.ServerVncOutput{
		Protocol:   "openstack",
		InstanceId: instanceId,
		Hypervisor: api.HYPERVISOR_OPENSTACK,
	}

	ret.Url, err = resp.GetString("console", "url")
	if err != nil {
		return nil, errors.Wrapf(err, "remote_console")
	}

	if origin {
		return ret, nil
	}

	token := string([]byte(ret.Url)[len(ret.Url)-36:])
	vncUrl, _ := url.Parse(ret.Url)
	ret.Url = fmt.Sprintf("ws://%s?token=%s", vncUrl.Host, token)
	ret.Protocol = "vnc"
	return ret, nil
}

func (instance *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	origin := false
	if input != nil {
		origin = input.Origin
	}
	ret, err := instance.host.zone.region.GetInstanceVNCUrl(instance.Id, origin)
	if err == nil {
		return ret, nil
	}
	return instance.host.zone.region.GetInstanceVNC(instance.Id, origin)
}

func (instance *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return instance.host.zone.region.DeployVM(instance.Id, opts)
}

func (instance *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return instance.Id, instance.host.zone.region.ReplaceSystemDisk(instance.Id, desc.ImageId, desc.Password, desc.PublicKey, desc.SysSizeGB)
}

func (instance *SInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	if (len(config.InstanceType) > 0 && instance.GetInstanceType() != config.InstanceType) || instance.GetVcpuCount() != config.Cpu || instance.GetVmemSizeMB() != config.MemoryMB {
		flavor, err := instance.host.zone.region.syncFlavor(config.InstanceType, config.Cpu, config.MemoryMB, 40)
		if err != nil {
			return errors.Wrapf(err, "syncFlavor(%s)", config.InstanceType)
		}
		// When resizing, instances must change flavor!
		if flavor.Name == instance.Flavor.OriginalName {
			return nil
		}
		return instance.host.zone.region.ChangeConfig(instance, flavor.Id)
	}
	return nil
}

func (region *SRegion) ChangeConfig(instance *SInstance, flavorId string) error {
	params := map[string]map[string]string{
		"resize": {
			"flavorRef": flavorId,
		},
	}
	resource := fmt.Sprintf("/servers/%s/action", instance.Id)
	_, err := region.ecsPost(resource, params)
	if err != nil {
		return errors.Wrap(err, "ecsPost")
	}
	err = cloudprovider.WaitStatus(instance, api.VM_SYNC_CONFIG, time.Second*3, time.Minute*4)
	if err != nil {
		return errors.Wrap(err, "WaitStatsAfterChangeConfig")
	}
	return region.instanceOperation(instance.Id, "confirmResize")
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.AttachDisk(instance.Id, diskId)
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return instance.host.zone.region.DetachDisk(instance.Id, diskId)
}

func (region *SRegion) instanceOperation(instanceId, operate string) error {
	params := map[string]string{operate: ""}
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	_, err := region.ecsPost(resource, params)
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
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetInstance(%s)", instanceId)
	}
	status := instance.GetStatus()
	log.Debugf("Instance status on delete is %s", status)
	if status != api.VM_READY {
		log.Warningf("DeleteVM: vm status is %s expect %s", status, api.VM_READY)
	}
	return region.doDeleteVM(instanceId)
}

func (region *SRegion) DeployVM(instanceId string, opts *cloudprovider.SInstanceDeployOptions) error {
	if len(opts.Password) > 0 {
		params := map[string]map[string]string{
			"changePassword": {
				"adminPass": opts.Password,
			},
		}
		resource := fmt.Sprintf("/servers/%s/action", instanceId)
		_, err := region.ecsPost(resource, params)
		return err
	}
	return nil
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	err := instance.host.zone.region.DeleteVM(instance.Id)
	if err != nil {
		return errors.Wrapf(err, "instance.host.zone.region.DeleteVM(%s)", instance.Id)
	}
	return cloudprovider.WaitDeleted(instance, time.Second*5, time.Minute*10)
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
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	_, err := region.ecsPost(resource, params)
	return err
}

func (region *SRegion) DetachDisk(instanceId string, diskId string) error {
	resource := fmt.Sprintf("/servers/%s/os-volume_attachments/%s", instanceId, diskId)
	_, err := region.ecsDelete(resource)
	if err != nil {
		return errors.Wrap(err, "ecsDelete")
	}
	status := ""
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*10 {
		disk, err := region.GetDisk(diskId)
		if err != nil {
			return errors.Wrapf(err, "GetDisk(%s)", diskId)
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
	resource := fmt.Sprintf("/servers/%s/os-volume_attachments", instanceId)
	_, err := region.ecsPost(resource, params)
	if err != nil {
		return errors.Wrap(err, "ecsPost")
	}
	status := ""
	startTime := time.Now()
	for time.Now().Sub(startTime) < time.Minute*10 {
		disk, err := region.GetDisk(diskId)
		if err != nil {
			return errors.Wrapf(err, "GetDisk(%s)", diskId)
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

func (region *SRegion) MigrateVM(instanceId string, hostName string) error {
	params := jsonutils.NewDict()
	migrate := jsonutils.NewDict()
	migrate.Add(jsonutils.JSONNull, "host")
	if hostName != "" {
		migrate.Add(jsonutils.NewString(hostName), "host")
	}
	params.Add(migrate, "migrate")
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	_, err := region.ecsPost(resource, params)
	if err != nil {
		return errors.Wrapf(err, "On Requst Migrate instance:%s", instanceId)
	}
	return nil
}

func (region *SRegion) LiveMigrateVM(instanceId string, hostName string) error {
	params := jsonutils.NewDict()
	osMigrateLive := jsonutils.NewDict()
	osMigrateLive.Add(jsonutils.NewString("auto"), "block_migration")
	osMigrateLive.Add(jsonutils.JSONNull, "host")
	if hostName != "" {
		osMigrateLive.Add(jsonutils.NewString(hostName), "host")
	}
	params.Add(osMigrateLive, "os-migrateLive")
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	_, err := region.ecsPost(resource, params)
	if err != nil {
		return errors.Wrapf(err, "On Requst LiveMigrate instance:%s", instanceId)
	}
	return nil
}

//仅live-migration
func (region *SRegion) ListServerMigration(instanceId string) error {
	resource := fmt.Sprintf("/servers/%s/migrations", instanceId)
	_, err := region.ecsGet(resource)
	if err != nil {
		return errors.Wrapf(err, "ListServerMigration")
	}
	return nil
}

//仅live-migration
func (region *SRegion) DeleteMigration(instanceId string, migrationId string) error {
	resource := fmt.Sprintf("/servers/%s/migrations/%s", instanceId, migrationId)
	_, err := region.ecsDelete(resource)
	if err != nil {
		return errors.Wrapf(err, "On Requst delete LiveMigrate:%s", migrationId)
	}
	return nil
}

//仅live-migration
func (region *SRegion) ForceCompleteMigration(instanceId string, migrationId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONNull, "force_complete")
	resource := fmt.Sprintf("/servers/%s/migrations/%s/action", instanceId, migrationId)
	_, err := region.ecsPost(resource, params)
	if err != nil {
		return errors.Wrapf(err, "On Requst delete LiveMigrate:%s", migrationId)
	}
	return nil
}

func (region *SRegion) GetMigrations(instanceId string, migrationType string) (jsonutils.JSONObject, error) {
	query := url.Values{}
	query.Set("instance_uuid", instanceId)
	query.Set("migration_type", migrationType)
	resource := "/os-migrations"
	migrations, err := region.ecsList(resource, query)
	if err != nil {
		return nil, errors.Wrapf(err, "On Get instance :%s Migration,migration_type:%s", instanceId, migrationType)
	}
	return migrations, nil
}

func (self *SRegion) AssignSecurityGroup(instanceId, projectId, secgroupId string) error {
	if secgroupId == SECGROUP_NOT_SUPPORT {
		return fmt.Errorf("Security groups are not supported. Security group components are not installed")
	}
	secgroup, err := self.GetSecurityGroup(secgroupId)
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroup(%s)", secgroupId)
	}
	params := map[string]map[string]string{
		"addSecurityGroup": {
			"name": secgroup.Name,
		},
	}
	resource := fmt.Sprintf("/servers/%s/action", instanceId)
	_, err = self.ecsDo(projectId, resource, params)
	return err
}

func (instance *SInstance) RevokeSecurityGroup(secgroupId string) error {
	// 若OpenStack不支持安全组，则忽略解绑安全组
	if secgroupId == SECGROUP_NOT_SUPPORT {
		return nil
	}
	secgroup, err := instance.host.zone.region.GetSecurityGroup(secgroupId)
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroup(%s)", secgroupId)
	}
	params := map[string]map[string]string{
		"removeSecurityGroup": {
			"name": secgroup.Name,
		},
	}
	resource := fmt.Sprintf("/servers/%s/action", instance.Id)
	_, err = instance.host.zone.region.ecsDo(instance.GetProjectId(), resource, params)
	return err
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	secgroups, err := instance.host.zone.region.GetSecurityGroupsByInstance(instance.Id)
	if err != nil {
		return errors.Wrapf(err, "GetSecurityGroupsByInstance(%s)", instance.Id)
	}
	local := set.New(set.ThreadSafe)
	for _, secgroup := range secgroups {
		local.Add(secgroup.Id)
	}
	newG := set.New(set.ThreadSafe)
	for _, secgroupId := range secgroupIds {
		newG.Add(secgroupId)
	}
	for _, del := range set.Difference(local, newG).List() {
		secgroupId := del.(string)
		err := instance.RevokeSecurityGroup(secgroupId)
		if err != nil {
			return errors.Wrapf(err, "RevokeSecurityGroup(%s)", secgroupId)
		}
	}
	for _, add := range set.Difference(newG, local).List() {
		secgroupId := add.(string)
		err := instance.host.zone.region.AssignSecurityGroup(instance.Id, instance.GetProjectId(), secgroupId)
		if err != nil {
			return errors.Wrapf(err, "AssignSecurityGroup(%s)", secgroupId)
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

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) RenewInstances(instanceId []string, bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (instance *SInstance) GetProjectId() string {
	return instance.TenantId
}

func (self *SInstance) GetError() error {
	if self.Status == INSTANCE_STATUS_ERROR && len(self.Fault.Message) > 0 {
		return errors.Error(self.Fault.Message)
	}
	return nil
}

func (instance *SInstance) MigrateVM(hostId string) error {
	hostName := ""
	if hostId != "" {
		iHost, err := instance.host.zone.region.GetIHostById(hostId)
		if err != nil {
			return errors.Wrapf(err, "GetIHostById(%s)", hostId)
		}
		hostName = iHost.GetName()
	}

	previousHostName := instance.Host
	err := instance.host.zone.region.MigrateVM(instance.Id, hostName)
	if err != nil {
		return errors.Wrap(err, "MigrateVm")
	}
	err = cloudprovider.WaitMultiStatus(instance, []string{api.VM_SYNC_CONFIG, api.VM_READY, api.VM_UNKNOWN}, time.Second*10, time.Hour*3)
	if err != nil {
		return errors.Wrap(err, "WaitMultiStatus")
	}
	if instance.GetStatus() == api.VM_UNKNOWN {
		return errors.Wrap(errors.ErrInvalidStatus, "GetStatus")
	}
	if instance.GetStatus() == api.VM_READY {
		if instance.Host == previousHostName {
			return errors.Wrap(fmt.Errorf("instance not migrated"), "Check host after migration")
		}
		return nil
	}
	return instance.host.zone.region.instanceOperation(instance.Id, "confirmResize")
}

func (instance *SInstance) LiveMigrateVM(hostId string) error {
	hostName := ""
	if hostId != "" {
		iHost, err := instance.host.zone.region.GetIHostById(hostId)
		if err != nil {
			return errors.Wrapf(err, "GetIHostById(%s)", hostId)
		}
		hostName = iHost.GetName()
	}
	previousHostName := instance.Host
	err := instance.host.zone.region.LiveMigrateVM(instance.Id, hostName)
	if err != nil {
		return errors.Wrap(err, "LiveMIgrateVm")
	}
	err = cloudprovider.WaitMultiStatus(instance, []string{api.VM_SYNC_CONFIG, api.VM_RUNNING, api.VM_UNKNOWN}, time.Second*10, time.Hour*3)
	if err != nil {
		return errors.Wrap(err, "WaitMultiStatus")
	}
	if instance.GetStatus() == api.VM_UNKNOWN {
		return errors.Wrap(errors.ErrInvalidStatus, "GetStatus")
	}
	if instance.GetStatus() == api.VM_RUNNING {
		if instance.Host == previousHostName {
			return errors.Wrap(fmt.Errorf("instance not migrated"), "Check host after migration")
		}
		return nil
	}
	return instance.host.zone.region.instanceOperation(instance.Id, "confirmResize")
}
func (instance *SInstance) GetIHostId() string {
	err := instance.host.zone.fetchHosts()
	if err != nil {
		return ""
	}
	for _, host := range instance.host.zone.hosts {
		if instance.HypervisorHostname == host.HypervisorHostname {
			return host.GetGlobalId()
		}
	}
	return ""
}

func (region *SRegion) GetInstanceMetadata(instanceId string) (map[string]string, error) {
	resource := fmt.Sprintf("/servers/%s/metadata", instanceId)
	resp, err := region.ecsList(resource, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ecsList")
	}
	result := struct {
		Metadata map[string]string
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return result.Metadata, nil
}

func (zone *SZone) CreateVM(hypervisor string, opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	region := zone.region
	network, err := region.GetNetwork(opts.ExternalNetworkId)
	if err != nil {
		return nil, err
	}

	secgroups := []map[string]string{}
	for _, secgroupId := range opts.ExternalSecgroupIds {
		if secgroupId != SECGROUP_NOT_SUPPORT {
			secgroups = append(secgroups, map[string]string{"name": secgroupId})
		}
	}

	image, err := region.GetImage(opts.ExternalImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s)", opts.ExternalImageId)
	}

	sysDiskSizeGB := image.Size / 1024 / 1024 / 1024
	if opts.SysDisk.SizeGB < sysDiskSizeGB {
		opts.SysDisk.SizeGB = sysDiskSizeGB
	}

	if opts.SysDisk.SizeGB < image.GetMinOsDiskSizeGb() {
		opts.SysDisk.SizeGB = image.GetMinOsDiskSizeGb()
	}

	BlockDeviceMappingV2 := []map[string]interface{}{}

	diskIds := []string{}

	defer func() {
		for _, diskId := range diskIds {
			err = region.DeleteDisk(diskId)
			if err != nil {
				log.Errorf("clean disk %s error: %v", diskId, err)
			}
		}
	}()

	if opts.SysDisk.StorageType != api.STORAGE_OPENSTACK_NOVA { //新建volume
		istorage, err := zone.GetIStorageById(opts.SysDisk.StorageExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIStorageById(%s)", opts.SysDisk.StorageExternalId)
		}

		_sysDisk, err := region.CreateDisk(opts.ExternalImageId, istorage.GetName(), "", opts.SysDisk.SizeGB, opts.SysDisk.Name, opts.ProjectId)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateDisk %s", opts.SysDisk.Name)
		}

		diskIds = append(diskIds, _sysDisk.GetGlobalId())

		BlockDeviceMappingV2 = append(BlockDeviceMappingV2, map[string]interface{}{
			"boot_index":            0,
			"uuid":                  _sysDisk.GetGlobalId(),
			"source_type":           "volume",
			"destination_type":      "volume",
			"delete_on_termination": true,
		})
	} else {
		BlockDeviceMappingV2 = append(BlockDeviceMappingV2, map[string]interface{}{
			"boot_index":            0,
			"uuid":                  image.Id,
			"source_type":           "image",
			"destination_type":      "local",
			"delete_on_termination": true,
		})
	}

	var _disk *SDisk
	for index, disk := range opts.DataDisks {
		istorage, err := zone.GetIStorageById(disk.StorageExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIStorageById(%s)", disk.StorageExternalId)
		}
		_disk, err = region.CreateDisk("", istorage.GetName(), "", disk.SizeGB, disk.Name, opts.ProjectId)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateDisk %s", disk.Name)
		}
		diskIds = append(diskIds, _disk.Id)

		mapping := map[string]interface{}{
			"source_type":           "volume",
			"destination_type":      "volume",
			"delete_on_termination": true,
			"boot_index":            index + 1,
			"uuid":                  _disk.Id,
		}

		BlockDeviceMappingV2 = append(BlockDeviceMappingV2, mapping)
	}

	az := zone.ZoneName
	if len(hypervisor) > 0 {
		az = fmt.Sprintf("%s:%s", zone.ZoneName, hypervisor)
	}

	net := map[string]string{
		"uuid": network.NetworkId,
	}
	if len(opts.IpAddr) > 0 {
		net["fixed_ip"] = opts.IpAddr
	}

	params := map[string]map[string]interface{}{
		"server": {
			"name":                    opts.Name,
			"adminPass":               opts.Password,
			"availability_zone":       az,
			"networks":                []map[string]string{net},
			"security_groups":         secgroups,
			"user_data":               opts.UserData,
			"imageRef":                opts.ExternalImageId,
			"block_device_mapping_v2": BlockDeviceMappingV2,
		},
	}
	if len(opts.IpAddr) > 0 {
		params["server"]["accessIPv4"] = opts.IpAddr
	}

	flavor, err := region.syncFlavor(opts.InstanceType, opts.Cpu, opts.MemoryMB, opts.SysDisk.SizeGB)
	if err != nil {
		return nil, err
	}
	params["server"]["flavorRef"] = flavor.Id

	if len(opts.PublicKey) > 0 {
		keypairName, err := region.syncKeypair(opts.Name, opts.PublicKey)
		if err != nil {
			return nil, err
		}
		params["server"]["key_name"] = keypairName
	}

	resp, err := region.ecsCreate(opts.ProjectId, "/servers", params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsCreate")
	}
	diskIds = []string{}
	instance := &SInstance{}
	err = resp.Unmarshal(instance, "server")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return instance, nil
}
