package huawei

import (
	"context"
	"strings"
	"time"

	"strconv"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
)

type IpAddress struct {
	Version            string `json:"version"`
	Addr               string `json:"addr"`
	OSEXTIPSMACMACAddr string `json:"OS-EXT-IPS-MAC.mac_addr"`
	OSEXTIPSPortID     string `json:"OS-EXT-IPS.port_id"`
	OSEXTIPSType       string `json:"OS-EXT-IPS.type"`
}

type Flavor struct {
	Disk  string `json:"disk"`
	Vcpus string `json:"vcpus"`
	RAM   string `json:"ram"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

type Image struct {
	ID string `json:"id"`
}

type VMMetadata struct {
	MeteringImageID           string `json:"metering.image_id"`
	MeteringImagetype         string `json:"metering.imagetype"`
	MeteringResourcespeccode  string `json:"metering.resourcespeccode"`
	ImageName                 string `json:"image_name"`
	OSBit                     string `json:"os_bit"`
	VpcID                     string `json:"vpc_id"`
	MeteringResourcetype      string `json:"metering.resourcetype"`
	CascadedInstanceExtrainfo string `json:"cascaded.instance_extrainfo"`
	OSType                    string `json:"os_type"`
	ChargingMode              string `json:"charging_mode"`
}

type OSExtendedVolumesVolumesAttached struct {
	Device              string `json:"device"`
	BootIndex           string `json:"bootIndex"`
	ID                  string `json:"id"`
	DeleteOnTermination string `json:"delete_on_termination"`
}

type OSSchedulerHints struct {
}

type SecurityGroup struct {
	Name string `json:"name"`
}

type SysTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148849.html
type SInstance struct {
	host *SHost

	Fault                            interface{}                        `json:"fault"`
	ID                               string                             `json:"id"`
	Name                             string                             `json:"name"`
	Addresses                        map[string][]IpAddress             `json:"addresses"`
	Flavor                           Flavor                             `json:"flavor"`
	AccessIPv4                       string                             `json:"accessIPv4"`
	AccessIPv6                       string                             `json:"accessIPv6"`
	Status                           string                             `json:"status"`
	Progress                         string                             `json:"progress"`
	HostID                           string                             `json:"hostId"`
	Updated                          string                             `json:"updated"`
	Created                          string                             `json:"created"`
	Metadata                         VMMetadata                         `json:"metadata"`
	Tags                             []string                           `json:"tags"`
	Description                      string                             `json:"description"`
	Locked                           bool                               `json:"locked"`
	Image                            Image                              `json:"image"`
	ConfigDrive                      string                             `json:"config_drive"`
	TenantID                         string                             `json:"tenant_id"`
	UserID                           string                             `json:"user_id"`
	KeyName                          string                             `json:"key_name"`
	OSExtendedVolumesVolumesAttached []OSExtendedVolumesVolumesAttached `json:"os-extended-volumes.volumes_attached"`
	OSEXTSTSTaskState                string                             `json:"OS-EXT-STS.task_state"`
	OSEXTSTSPowerState               int64                              `json:"OS-EXT-STS.power_state"`
	OSEXTSTSVMState                  string                             `json:"OS-EXT-STS.vm_state"`
	OSEXTSRVATTRHost                 string                             `json:"OS-EXT-SRV-ATTR.host"`
	OSEXTSRVATTRInstanceName         string                             `json:"OS-EXT-SRV-ATTR.instance_name"`
	OSEXTSRVATTRHypervisorHostname   string                             `json:"OS-EXT-SRV-ATTR.hypervisor_hostname"`
	OSDCFDiskConfig                  string                             `json:"OS-DCF.diskConfig"`
	OSEXTAZAvailabilityZone          string                             `json:"OS-EXT-AZ.availability_zone"`
	OSSchedulerHints                 OSSchedulerHints                   `json:"os.scheduler_hints"`
	OSEXTSRVATTRRootDeviceName       string                             `json:"OS-EXT-SRV-ATTR.root_device_name"`
	OSEXTSRVATTRRamdiskID            string                             `json:"OS-EXT-SRV-ATTR.ramdisk_id"`
	EnterpriseProjectID              string                             `json:"enterprise_project_id"`
	OSEXTSRVATTRUserData             string                             `json:"OS-EXT-SRV-ATTR.user_data"`
	OSSRVUSGLaunchedAt               string                             `json:"OS-SRV-USG.launched_at"`
	OSEXTSRVATTRKernelID             string                             `json:"OS-EXT-SRV-ATTR.kernel_id"`
	OSEXTSRVATTRLaunchIndex          int64                              `json:"OS-EXT-SRV-ATTR.launch_index"`
	HostStatus                       string                             `json:"host_status"`
	OSEXTSRVATTRReservationID        string                             `json:"OS-EXT-SRV-ATTR.reservation_id"`
	OSEXTSRVATTRHostname             string                             `json:"OS-EXT-SRV-ATTR.hostname"`
	OSSRVUSGTerminatedAt             string                             `json:"OS-SRV-USG.terminated_at"`
	SysTags                          []SysTag                           `json:"sys_tags"`
	SecurityGroups                   []SecurityGroup                    `json:"security_groups"`
}

func (self *SInstance) GetId() string {
	return self.ID
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.ID
}

func (self *SInstance) GetStatus() string {
	switch self.Status {
	case "ACTIVE":
		return models.VM_RUNNING
	case "MIGRATING", "REBUILD", "BUILD", "RESIZE", "VERIFY_RESIZE": // todo: pending ?
		return models.VM_STARTING
	case "REBOOT", "HARD_REBOOT":
		return models.VM_STOPPING
	case "SHUTOFF":
		return models.VM_READY
	default:
		return models.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	new, err := self.host.zone.region.GetInstanceByID(self.GetId())
	new.host = self.host
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetInstanceType() string {
	return self.Flavor.ID
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	// todo: add price_key here
	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	if len(self.Image.ID) > 0 {
		if image, err := self.host.zone.region.GetImage(self.Image.ID); err != nil {
			log.Errorf("Failed to find image %s for instance %s zone %s", self.Image.ID, self.GetId(), self.OSEXTAZAvailabilityZone)
		} else if meta := image.GetMetadata(); meta != nil {
			data.Update(meta)
		}
	}
	secgroupIds := jsonutils.NewArray()
	for _, secgroup := range self.SecurityGroups {
		secgroupIds.Add(jsonutils.NewString(secgroup.Name))
	}
	data.Add(secgroupIds, "secgroupIds")
	return data
}

func (self *SInstance) GetBillingType() string {
	// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148849.html
	// charging_mode “0”：按需计费    “1”：按包年包月计费
	if self.Metadata.ChargingMode == "1" {
		return models.BILLING_TYPE_PREPAID
	} else {
		return models.BILLING_TYPE_POSTPAID
	}
}

func (self *SInstance) GetExpiredAt() time.Time {
	t, _ := time.Parse(DATETIME_FORMAT, self.OSSRVUSGTerminatedAt)
	return t
}

func (self *SInstance) GetCreateTime() time.Time {
	t, _ := time.Parse(DATETIME_FORMAT, self.Created)
	return t
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	attached := self.OSExtendedVolumesVolumesAttached
	disks := make([]SDisk, 0)
	for _, vol := range attached {
		disk, err := self.host.zone.region.GetDisk(vol.ID)
		if err != nil {
			return nil, err
		}

		disks = append(disks, *disk)
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		storage, err := self.host.zone.getStorageByCategory(disks[i].VolumeType)
		if err != nil {
			return nil, err
		}
		disks[i].storage = storage
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)

	// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0094148849.html
	// OS-EXT-IPS.type
	// todo: 这里没有区分是IPv4 还是 IPv6。统一当IPv4处理了.可能会引发错误
	for _, ipAddresses := range self.Addresses {
		for _, ipAddress := range ipAddresses {
			if ipAddress.OSEXTIPSType == "fixed" {
				nic := SInstanceNic{instance: self, ipAddr: ipAddress.Addr}
				nics = append(nics, &nic)
			}
		}
	}
	return nics, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	ips := make([]string, 0)
	for _, addresses := range self.Addresses {
		for _, address := range addresses {
			if address.OSEXTIPSType != "fixed" && !strings.HasPrefix(address.Addr, "100.") {
				ips = append(ips, address.Addr)
			}
		}
	}

	if len(ips) == 0 {
		return nil, nil
	}

	eips, _, err := self.host.zone.region.GetEips("", 65535)
	if err != nil {
		return nil, err
	}

	for _, eip := range eips {
		if eip.PublicIPAddress == ips[0] {
			return &eip, nil
		}
	}

	return nil, nil
}

func (self *SInstance) GetVcpuCount() int8 {
	cpu, _ := strconv.Atoi(self.Flavor.Vcpus)
	return int8(cpu)
}

func (self *SInstance) GetVmemSizeMB() int {
	mem, _ := strconv.Atoi(self.Flavor.RAM)
	return int(mem)
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

func (self *SInstance) GetOSType() string {
	return osprofile.NormalizeOSType(self.Metadata.OSType)
}

func (self *SInstance) GetOSName() string {
	return self.Metadata.ImageName
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	panic("implement me")
}

func (self *SInstance) AssignSecurityGroups(secgroupIds []string) error {
	panic("implement me")
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_HUAWEI
}

func (self *SInstance) StartVM(ctx context.Context) error {
	panic("implement me")
}

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	panic("implement me")
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	panic("implement me")
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	panic("implement me")
}

func (self *SInstance) UpdateUserData(userData string) error {
	panic("implement me")
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	panic("implement me")
}

func (self *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	panic("implement me")
}

func (self *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	panic("implement me")
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	panic("implement me")
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	panic("implement me")
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	panic("implement me")
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	panic("implement me")
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	panic("implement me")
}

func (self *SRegion) GetInstances(offset int, limit int) ([]SInstance, int, error) {
	querys := map[string]string{}
	querys["offset"] = strconv.Itoa(offset)
	querys["limit"] = strconv.Itoa(limit)

	if len(self.client.projectId) > 0 {
		querys["project_id"] = self.client.projectId
	}

	instances := make([]SInstance, 0)
	err := DoList(self.ecsClient.Servers.List, querys, &instances)
	return instances, len(instances), err
}

func (self *SRegion) GetInstanceByID(instanceId string) (SInstance, error) {
	instance := SInstance{}
	err := DoGet(self.ecsClient.Servers.Get, instanceId, nil, &instance)
	return instance, err
}

func (self *SRegion) GetInstanceByIds(ids []string) ([]SInstance, int, error) {
	instances := make([]SInstance, 0)
	for _, instanceId := range ids {
		instance, err := self.GetInstanceByID(instanceId)
		if err != nil {
			return nil, 0, err
		}
		instances = append(instances, instance)
	}

	return instances, len(instances), nil
}
