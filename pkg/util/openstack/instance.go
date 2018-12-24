package openstack

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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

type SAddresses struct {
	Private []SInstanceNic
	Public  []SInstanceNic
}

type ExtraSpecs struct {
	CpuPolicy   string `json:"hw:cpu_policy,omitempty"`
	MemPageSize int    `json:"hw:mem_page_size,omitempty"`
}

type SFlavor struct {
	Disk         int
	Ephemeral    int
	ExtraSpecs   ExtraSpecs
	OriginalName string
	RAM          int
	Swap         string
	Vcpus        int8
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

type SInstance struct {
	host   cloudprovider.ICloudHost
	hostV2 *SHostV2
	hostV3 *SHostV3

	flavor *SFlavor

	DiskConfig         string `json:"OS-DCF:diskConfig,omitempty"`
	AvailabilityZone   string `json:"OS-EXT-AZ:availability_zone,omitempty"`
	Host               string `json:"OS-EXT-SRV-ATTR:host,omitempty"`
	Hostname           string `json:"OS-EXT-SRV-ATTR:hostname,omitempty"`
	HypervisorHostname string `json:"OS-EXT-SRV-ATTR:hypervisor_hostname,omitempty"`
	InstanceName       string `json:"OS-EXT-SRV-ATTR:instance_name,omitempty"`
	KernelID           string `json:"OS-EXT-SRV-ATTR:kernel_id,omitempty"`
	LaunchIndex        int    `json:"OS-EXT-SRV-ATTR:launch_index,omitempty"`
	RamdiskID          string `json:"OS-EXT-SRV-ATTR:ramdisk_id,omitempty"`
	ReservationID      string `json:"OS-EXT-SRV-ATTR:reservation_id,omitempty"`
	RootDeviceName     string `json:"OS-EXT-SRV-ATTR:root_device_name,omitempty"`
	UserData           string `json:"OS-EXT-SRV-ATTR:user_data,omitempty"`
	PowerState         int    `json:"OS-EXT-STS:power_state,omitempty"`
	TaskState          string `json:"OS-EXT-STS:task_state,omitempty"`
	VmState            string `json:"OS-EXT-STS:vm_state,omitempty"`
	//LaunchedAt         time.Time `json:"OS-SRV-USG:launched_at,omitempty"`
	TerminatedAt string `json:"OS-SRV-USG:terminated_at,omitempty"`

	AccessIPv4               string
	AccessIPv6               string
	Addresses                SAddresses
	ConfigDrive              string
	Created                  time.Time
	Description              string
	Flavor                   Resource
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
}

func (region *SRegion) GetSecurityGroupsByInstance(instanceId string) ([]SecurityGroup, error) {
	_, resp, err := region.Get("compute", fmt.Sprintf("/servers/%s/os-security-groups", instanceId), "", nil)
	if err != nil {
		return nil, err
	}
	secgroups := []SecurityGroup{}
	return secgroups, resp.Unmarshal(&secgroups, "security_groups")
}

func (region *SRegion) GetInstances(zoneName string, hostName string) ([]SInstance, error) {
	_, maxVersion, _ := region.GetVersion("compute")
	_, resp, err := region.Get("compute", "/servers/detail", maxVersion, nil)
	if err != nil {
		return nil, err
	}
	instances, result := []SInstance{}, []SInstance{}
	servers, err := resp.Get("servers")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(servers.String()), &instances); err != nil {
		return nil, err
	}
	for i := 0; i < len(instances); i++ {
		if len(zoneName) == 0 || instances[i].AvailabilityZone == zoneName {
			if len(hostName) == 0 || hostName == instances[i].Host {
				result = append(result, instances[i])
			}
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
	server, err := resp.Get("server")
	if err != nil {
		return nil, err
	}
	instance := &SInstance{}
	return instance, json.Unmarshal([]byte(server.String()), instance)
}

func (instance *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	secgroups, err := instance.getRegion().GetSecurityGroupsByInstance(instance.ID)
	if err == nil {
		secgroupIds := jsonutils.NewArray()
		for _, secgroup := range secgroups {
			secgroupIds.Add(jsonutils.NewString(secgroup.ID))
		}
		data.Add(secgroupIds, "secgroupIds")
	}

	if instance.flavor == nil {
		if err := instance.fetchFlavor(); err != nil {
			log.Errorf("fetch flavor for instance %s failed error: %v", instance.Name, err)
		}
	}
	if instance.flavor != nil {
		priceKey := fmt.Sprintf("%s::%s", instance.getZone().ZoneName, instance.flavor.OriginalName)
		data.Add(jsonutils.NewString(priceKey), "price_key")
	}

	data.Add(jsonutils.NewString(instance.getZone().GetGlobalId()), "zone_ext_id")
	return data
}

func (instance *SInstance) GetCreateTime() time.Time {
	return instance.Created
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	if instance.hostV3 != nil {
		return instance.hostV3
	}
	return instance.hostV2
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
	_, resp, err := instance.getRegion().Get("compute", "/flavors/"+instance.Flavor.ID, "", nil)
	if err != nil {
		log.Errorf("fetch instance %s flavor error: %v", instance.Name, err)
		return err
	}
	instance.flavor = &SFlavor{}
	flavor, err := resp.Get("flavor")
	if err != nil {
		log.Errorf("fetch instance %s flavor error: %v", instance.Name, err)
		return err
	}
	return json.Unmarshal([]byte(flavor.String()), instance.flavor)
}

func (instance *SInstance) GetInstanceType() string {
	if instance.flavor == nil {
		if err := instance.fetchFlavor(); err != nil {
			return ""
		}
	}
	return instance.flavor.OriginalName
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SDisk{}
	for i := 0; i < len(instance.VolumesAttached); i++ {
		disk, err := instance.getRegion().GetDisk(instance.VolumesAttached[i].ID)
		if err != nil {
			return nil, err
		}
		disks = append(disks, *disk)
	}
	iDisks := []cloudprovider.ICloudDisk{}
	for i := 0; i < len(disks); i++ {
		store, err := instance.getZone().getStorageByCategory(disks[i].VolumeType)
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
	for i := 0; i < len(instance.Addresses.Private); i++ {
		instance.Addresses.Private[i].instance = instance
		nics = append(nics, &instance.Addresses.Private[i])
	}
	for i := 0; i < len(instance.Addresses.Public); i++ {
		instance.Addresses.Public[i].instance = instance
		nics = append(nics, &instance.Addresses.Public[i])
	}
	return nics, nil
}

func (instance *SInstance) GetVcpuCount() int8 {
	if instance.flavor == nil {
		if err := instance.fetchFlavor(); err != nil {
			return 0
		}
	}
	return instance.flavor.Vcpus
}

func (instance *SInstance) GetVmemSizeMB() int {
	if instance.flavor == nil {
		if err := instance.fetchFlavor(); err != nil {
			return 0
		}
	}
	return instance.flavor.RAM
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
		return models.VM_RUNNING
	case INSTANCE_STATUS_BUILD, INSTANCE_STATUS_PASSWORD:
		return models.VM_DEPLOYING
	case INSTANCE_STATUS_DELETED:
		return models.VM_DELETING
	case INSTANCE_STATUS_HARD_REBOOT, INSTANCE_STATUS_REBOOT:
		return models.VM_STARTING
	case INSTANCE_STATUS_MIGRATING:
		return models.VM_MIGRATING
	case INSTANCE_STATUS_PAUSED, INSTANCE_STATUS_SUSPENDED:
		return models.VM_SUSPEND
	case INSTANCE_STATUS_RESIZE, INSTANCE_STATUS_VERIFY_RESIZE:
		return models.VM_CHANGE_FLAVOR
	case INSTANCE_STATUS_SHELVED, INSTANCE_STATUS_SHELVED_OFFLOADED, INSTANCE_STATUS_SHUTOFF, INSTANCE_STATUS_SOFT_DELETED:
		return models.VM_READY
	default:
		return models.VM_UNKNOWN
	}
}

func (instance *SInstance) Refresh() error {
	new, err := instance.getRegion().GetInstance(instance.ID)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonutils.Marshal(new).String()), instance)
}

func (instance *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_OPENSTACK
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return cloudprovider.ErrNotImplemented
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
	url, err := instance.getRegion().GetInstanceVNCUrl(instance.ID)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(url), "url")
	ret.Add(jsonutils.NewString("openstack"), "protocol")
	ret.Add(jsonutils.NewString(instance.ID), "instance_id")
	return ret, nil
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (instance *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) getZone() *SZone {
	if instance.hostV3 != nil {
		return instance.hostV3.zone
	}
	return instance.hostV2.zone
}

func (instance *SInstance) getRegion() *SRegion {
	return instance.getZone().region
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return instance.getRegion().AttachDisk(instance.ID, diskId)
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return instance.getRegion().DetachDisk(instance.ID, diskId)
}

func (region *SRegion) CreateInstance(name string, imageId string, instanceType string, securityGroupId string,
	zoneId string, desc string, passwd string, disks []SDisk, networkId string, ipAddr string,
	keypair string, userData string, bc *billing.SBillingCycle) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) doStartVM(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) doStopVM(instanceId string, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) doDeleteVM(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) StartVM(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) StopVM(instanceId string, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteVM(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) ReplaceSystemDisk(instanceId string, imageId string, passwd string, keypairName string, sysDiskSizeGB int) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DetachDisk(instanceId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) AttachDisk(instanceId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) AssignSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetBillingType() string {
	return models.BILLING_TYPE_PREPAID
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
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) RenewInstances(instanceId []string, bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}
