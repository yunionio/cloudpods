package huawei

import (
	"context"
	"fmt"
	"strings"
	"time"

	"strconv"

	"sort"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/huawei/client/modules"
	"yunion.io/x/pkg/util/osprofile"
)

const (
	InstanceStatusRunning    = "ACTIVE"
	InstanceStatusTerminated = "DELETED"
	InstanceStatusStopped    = "SHUTOFF"
)

type IpAddress struct {
	Version            string `json:"version"`
	Addr               string `json:"addr"`
	OSEXTIPSMACMACAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	OSEXTIPSPortID     string `json:"OS-EXT-IPS:port_id"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
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
// https://support.huaweicloud.com/api-bpconsole/zh-cn_topic_0100166287.html v1.1 支持创建包年/包月的弹性云服务器
type SInstance struct {
	host *SHost

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
	OSExtendedVolumesVolumesAttached []OSExtendedVolumesVolumesAttached `json:"os-extended-volumes:volumes_attached"`
	OSEXTSTSTaskState                string                             `json:"OS-EXT-STS:task_state"`
	OSEXTSTSPowerState               int64                              `json:"OS-EXT-STS:power_state"`
	OSEXTSTSVMState                  string                             `json:"OS-EXT-STS:vm_state"`
	OSEXTSRVATTRHost                 string                             `json:"OS-EXT-SRV-ATTR:host"`
	OSEXTSRVATTRInstanceName         string                             `json:"OS-EXT-SRV-ATTR:instance_name"`
	OSEXTSRVATTRHypervisorHostname   string                             `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	OSDCFDiskConfig                  string                             `json:"OS-DCF:diskConfig"`
	OSEXTAZAvailabilityZone          string                             `json:"OS-EXT-AZ:availability_zone"`
	OSSchedulerHints                 OSSchedulerHints                   `json:"os:scheduler_hints"`
	OSEXTSRVATTRRootDeviceName       string                             `json:"OS-EXT-SRV-ATTR:root_device_name"`
	OSEXTSRVATTRRamdiskID            string                             `json:"OS-EXT-SRV-ATTR:ramdisk_id"`
	EnterpriseProjectID              string                             `json:"enterprise_project_id"`
	OSEXTSRVATTRUserData             string                             `json:"OS-EXT-SRV-ATTR:user_data"`
	OSSRVUSGLaunchedAt               string                             `json:"OS-SRV-USG:launched_at"`
	OSEXTSRVATTRKernelID             string                             `json:"OS-EXT-SRV-ATTR:kernel_id"`
	OSEXTSRVATTRLaunchIndex          int64                              `json:"OS-EXT-SRV-ATTR:launch_index"`
	HostStatus                       string                             `json:"host_status"`
	OSEXTSRVATTRReservationID        string                             `json:"OS-EXT-SRV-ATTR:reservation_id"`
	OSEXTSRVATTRHostname             string                             `json:"OS-EXT-SRV-ATTR:hostname"`
	OSSRVUSGTerminatedAt             string                             `json:"OS-SRV-USG:terminated_at"`
	SysTags                          []SysTag                           `json:"sys_tags"`
	SecurityGroups                   []SecurityGroup                    `json:"security_groups"`
}

func compareSet(currentSet []string, newSet []string) (add []string, remove []string, keep []string) {
	sort.Strings(currentSet)
	sort.Strings(newSet)

	i, j := 0, 0
	for i < len(currentSet) || j < len(newSet) {
		if i < len(currentSet) && j < len(newSet) {
			if currentSet[i] == newSet[j] {
				keep = append(keep, currentSet[i])
				i += 1
				j += 1
			} else if currentSet[i] < newSet[j] {
				remove = append(remove, currentSet[i])
				i += 1
			} else {
				add = append(add, newSet[j])
				j += 1
			}
		} else if i >= len(currentSet) {
			add = append(add, newSet[j])
			j += 1
		} else if j >= len(newSet) {
			remove = append(remove, currentSet[i])
			i += 1
		}
	}

	return add, remove, keep
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

	_secIds, err := self.host.zone.region.GetInstanceSecrityGroupIds(self.GetId())
	if err != nil {
		log.Errorf(err.Error())
	} else {
		secgroupIds := jsonutils.NewArray()
		for _, secId := range _secIds {
			secgroupIds.Add(jsonutils.NewString(secId))
		}
		data.Add(secgroupIds, "secgroupIds")
	}
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
	return self.AssignSecurityGroups([]string{secgroupId})
}

func (self *SInstance) AssignSecurityGroups(secgroupIds []string) error {
	currentSecgroups, err := self.host.zone.region.GetInstanceSecrityGroupIds(self.GetId())
	if err != nil {
		return err
	}

	add, remove, _ := compareSet(currentSecgroups, secgroupIds)
	err = self.host.zone.region.assignSecurityGroups(add, self.GetId())
	if err != nil {
		return err
	}

	return self.host.zone.region.unassignSecurityGroups(remove, self.GetId())
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_HUAWEI
}

func (self *SInstance) StartVM(ctx context.Context) error {
	if self.Status == InstanceStatusRunning {
		return nil
	}

	timeout := 300 * time.Second
	interval := 15 * time.Second

	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := self.Refresh()
		if err != nil {
			return err
		}

		if self.GetStatus() == models.VM_RUNNING {
			return nil
		} else if self.GetStatus() == models.VM_READY {
			err := self.host.zone.region.StartVM(self.GetId())
			if err != nil {
				return err
			}
		}
		time.Sleep(interval)
	}
	return cloudprovider.ErrTimeout
}

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	if self.Status == InstanceStatusStopped {
		return nil
	}

	err := self.host.zone.region.StopVM(self.GetId(), isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second) // 5mintues
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	for {
		err := self.host.zone.region.DeleteVM(self.GetId())
		if err != nil && self.Status != InstanceStatusTerminated {
			log.Errorf("DeleteVM fail: %s", err)
			return err
		} else {
			break
		}
	}

	return cloudprovider.WaitDeleted(self, 10*time.Second, 300*time.Second) // 5minutes
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return self.host.zone.region.UpdateVM(self.GetId(), name)
}

// https://support.huaweicloud.com/usermanual-ecs/zh-cn_topic_0032380449.html
// 创建云服务器过程中注入用户数据。支持注入文本、文本文件或gzip文件。
// 注入内容，需要进行base64格式编码。注入内容（编码之前的内容）最大长度32KB。
// 对于Linux弹性云服务器，adminPass参数传入时，user_data参数不生效。
func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067876349.html 使用原镜像重装
// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067876971.html 更换系统盘操作系统
// 不支持调整系统盘大小
// todo: 支持注入user_data
func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	var err error
	if self.Image.ID == imageId {
		err = self.host.zone.region.RebuildRoot(ctx, self.GetId(), passwd, publicKey)
		if err != nil {
			return "", err
		}
	} else {
		err = self.host.zone.region.ChangeRoot(ctx, self.GetId(), imageId, passwd, publicKey)
		if err != nil {
			return "", err
		}
	}

	// todo: wait job finished here
	return "", nil
}

func (self *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.DeployVM(self.GetId(), name, password, publicKey, deleteKeypair, description)
}

func (self *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return self.host.zone.region.ChangeVMConfig(self.OSEXTAZAvailabilityZone, self.GetId(), ncpu, vmem, nil)
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return self.host.zone.region.ChangeVMConfig2(self.OSEXTAZAvailabilityZone, self.GetId(), instanceType, nil)
}

// todo:// 返回jsonobject感觉很诡异。不能直接知道内部细节
func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return self.host.zone.region.GetInstanceVNCUrl(self.GetId())
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	// todo: calc device
	return self.host.zone.region.AttachDisk(self.GetId(), diskId, "")
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.GetId(), diskId)
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return self.host.zone.region.RenewInstance(self.GetId(), bc)
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

/*
系统盘大小取值范围：1-1024 GB，且必须不小于镜像min_disk.
*/
type SServerCreate struct {
	AvailabilityZone string            `json:"availability_zone"`
	Name             string            `json:"name"`
	ImageRef         string            `json:"imageRef"`
	RootVolume       RootVolume        `json:"root_volume"`
	DataVolumes      []DataVolume      `json:"data_volumes"`
	FlavorRef        string            `json:"flavorRef"`
	UserData         string            `json:"user_data"`
	Vpcid            string            `json:"vpcid"`
	SecurityGroups   []SecGroup        `json:"security_groups"`
	Nics             []NIC             `json:"nics"`
	KeyName          string            `json:"key_name"`
	AdminPass        string            `json:"adminPass"`
	Count            int64             `json:"count"`
	Extendparam      ServerExtendparam `json:"extendparam"`
	ServerTags       []ServerTag       `json:"server_tags"`
	Description      string            `json:"description"`
}

type DataVolume struct {
	Volumetype    string                 `json:"volumetype"`
	SizeGB        int                    `json:"size"`
	Extendparam   *DataVolumeExtendparam `json:"extendparam,omitempty"`
	Multiattach   *bool                  `json:"multiattach,omitempty"`
	HwPassthrough *string                `json:"hw:passthrough,omitempty"`
}

type DataVolumeExtendparam struct {
	SnapshotID string `json:"snapshotId"`
}

type ServerExtendparam struct {
	ChargingMode string `json:"chargingMode"` // 计费模式 prePaid|postPaid
	PeriodType   string `json:"periodType"`   // 周期类型：month|year
	PeriodNum    string `json:"periodNum"`    // 订购周期数：periodType=month（周期类型为月）时，取值为[1，9]。periodType=year（周期类型为年）时，取值为1。
	IsAutoRenew  string `json:"isAutoRenew"`  // 是否自动续订  true|false
	IsAutoPay    string `json:"isAutoPay"`    // 是否自动从客户的账户中支付 true|false
	RegionID     string `json:"regionID"`
}

type NIC struct {
	SubnetID  string `json:"subnet_id"` // 网络ID. 与 SNetwork里的ID对应。统一使用这个ID
	IpAddress string `json:"ip_address"`
}

type RootVolume struct {
	Volumetype string `json:"volumetype"`
	SizeGB     int    `json:"size"`
}

type SecGroup struct {
	ID string `json:"id"`
}

type ServerTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

/*
包月机器退订规则： https://support.huaweicloud.com/usermanual-billing/zh-cn_topic_0083138805.html
5天无理由全额退订：新购资源（不包含续费资源）在开通的五天内且退订次数不超过10次（每账号每年10次）的符合5天无理由全额退订。
非5天无理由退订：不符合5天无理由全额退订条件的退订，都属于非5天无理由退订。非5天无理由退订，不限制退订次数，但需要收取退订手续费。

退订资源的方法： https://support.huaweicloud.com/usermanual-billing/zh-cn_topic_0072297197.html
*/
func (self *SRegion) CreateInstance(name string, imageId string, instanceType string, SubnetId string,
	securityGroupId string, vpcId string, zoneId string, desc string, disks []SDisk, ipAddr string,
	keypair string, passwd string, userData string, bc *billing.SBillingCycle) (string, error) {
	params := SServerCreate{}
	params.AvailabilityZone = zoneId
	params.Name = name
	params.FlavorRef = instanceType
	params.ImageRef = imageId
	params.KeyName = keypair
	params.AdminPass = passwd
	params.UserData = userData
	params.Description = desc
	params.Count = 1
	params.Nics = []NIC{{SubnetID: SubnetId, IpAddress: ipAddr}}
	params.SecurityGroups = []SecGroup{{ID: securityGroupId}}
	params.Vpcid = vpcId

	for i, disk := range disks {
		if i == 0 {
			params.RootVolume.Volumetype = disk.VolumeType
			params.RootVolume.SizeGB = disk.SizeGB
		} else {
			dataVolume := DataVolume{}
			dataVolume.Volumetype = disk.VolumeType
			dataVolume.SizeGB = disk.SizeGB
			params.DataVolumes = append(params.DataVolumes, dataVolume)
		}
	}

	// billing type
	if bc != nil {
		params.Extendparam.ChargingMode = PRE_PAID
		if bc.GetMonths() <= 9 {
			params.Extendparam.PeriodNum = strconv.Itoa(bc.GetMonths())
			params.Extendparam.PeriodType = "month"
		} else {
			params.Extendparam.PeriodNum = strconv.Itoa(bc.GetYears())
			params.Extendparam.PeriodType = "year"
		}

		params.Extendparam.RegionID = self.GetId()
		params.Extendparam.IsAutoRenew = "false"
		params.Extendparam.IsAutoPay = "true"
	} else {
		params.Extendparam.ChargingMode = POST_PAID
	}

	serverObj := jsonutils.Marshal(params)
	createParams := jsonutils.NewDict()
	createParams.Add(serverObj, "server")
	_id, err := self.ecsClient.Servers.AsyncCreate(createParams)
	if err != nil {
		return "", err
	}

	var ids []string
	if params.Extendparam.ChargingMode == POST_PAID {
		// 按需计费
		ids, err = self.GetAllSubTaskEntityIDs(self.ecsClient.Servers.ServiceType(), _id, "server_id")
	} else {
		// 包年包月
		err = cloudprovider.WaitCreated(10*time.Second, 180*time.Second, func() bool {
			ids, err = self.getAllResIdsByType(_id, RESOURCE_TYPE_VM)
			if err != nil {
				log.Debugf(err.Error())
				return false
			}

			if len(ids) > 0 {
				return true
			}

			return false
		})
	}

	if err != nil {
		return "", err
	} else if len(ids) == 0 {
		return "", fmt.Errorf("CreateInstance job %s result is emtpy", _id)
	} else if len(ids) == 1 {
		return ids[0], nil
	} else {
		return "", fmt.Errorf("CreateInstance job %s mutliple instance id returned. %s", _id, ids)
	}
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067161469.html
// 添加多个安全组时，建议最多为弹性云服务器添加5个安全组。
// todo: 确认是否需要先删除，再进行添加操作
func (self *SRegion) assignSecurityGroups(secgroupIds []string, instanceId string) error {
	_, err := self.GetInstanceByID(instanceId)
	if err != nil {
		return err
	}

	for i := range secgroupIds {
		secId := secgroupIds[i]
		params := jsonutils.NewDict()
		secgroupObj := jsonutils.NewDict()
		secgroupObj.Add(jsonutils.NewString(secId), "name")
		params.Add(secgroupObj, "addSecurityGroup")

		_, err := self.ecsClient.NovaServers.PerformAction("action", instanceId, params)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0067161717.html
func (self *SRegion) unassignSecurityGroups(secgroupIds []string, instanceId string) error {
	for i := range secgroupIds {
		secId := secgroupIds[i]
		params := jsonutils.NewDict()
		secgroupObj := jsonutils.NewDict()
		secgroupObj.Add(jsonutils.NewString(secId), "name")
		params.Add(secgroupObj, "removeSecurityGroup")

		_, err := self.ecsClient.NovaServers.PerformAction("action", instanceId, params)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstanceByID(instanceId)
	if err != nil {
		return "", err
	}
	return instance.Status, nil
}

func (self *SRegion) instanceStatusChecking(instanceId, status string) error {
	remoteStatus, err := self.GetInstanceStatus(instanceId)
	if err != nil {
		log.Errorf("Fail to get instance status: %s", err)
		return err
	}
	if status != remoteStatus {
		log.Errorf("instanceStatusChecking: vm status is %s expect %s", remoteStatus, status)
		return cloudprovider.ErrInvalidStatus
	}

	return nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212207.html
func (self *SRegion) StartVM(instanceId string) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusStopped); err != nil {
		return err
	}

	params := jsonutils.NewDict()
	startObj := jsonutils.NewDict()
	serversObj := jsonutils.NewArray()
	serverObj := jsonutils.NewDict()
	serverObj.Add(jsonutils.NewString(instanceId), "id")
	serversObj.Add(serverObj)
	startObj.Add(serversObj, "servers")
	params.Add(startObj, "os-start")
	_, err := self.ecsClient.Servers.PerformAction2("action", "", params, "")
	return err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212651.html
func (self *SRegion) StopVM(instanceId string, isForce bool) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusRunning); err != nil {
		return err
	}

	params := jsonutils.NewDict()
	stopObj := jsonutils.NewDict()
	serversObj := jsonutils.NewArray()
	serverObj := jsonutils.NewDict()
	serverObj.Add(jsonutils.NewString(instanceId), "id")
	serversObj.Add(serverObj)
	stopObj.Add(serversObj, "servers")
	if isForce {
		stopObj.Add(jsonutils.NewString("HARD"), "type")
	} else {
		stopObj.Add(jsonutils.NewString("SOFT"), "type")
	}
	params.Add(stopObj, "os-stop")
	_, err := self.ecsClient.Servers.PerformAction2("action", "", params, "")
	return err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212679.html
// 只删除主机。弹性IP和数据盘需要单独删除
func (self *SRegion) DeleteVM(instanceId string) error {
	if err := self.instanceStatusChecking(instanceId, InstanceStatusStopped); err != nil {
		return err
	}

	params := jsonutils.NewDict()
	serversObj := jsonutils.NewArray()
	serverObj := jsonutils.NewDict()
	serverObj.Add(jsonutils.NewString(instanceId), "id")
	serversObj.Add(serverObj)
	params.Add(serversObj, "servers")
	params.Add(jsonutils.NewBool(false), "delete_publicip")
	params.Add(jsonutils.NewBool(false), "delete_volume")

	_, err := self.ecsClient.Servers.PerformAction2("delete", "", params, "")
	return err
}

func (self *SRegion) UpdateVM(instanceId, name string) error {
	params := jsonutils.NewDict()
	serversObj := jsonutils.NewArray()
	serverObj := jsonutils.NewDict()
	serverObj.Add(jsonutils.NewString(instanceId), "id")
	serversObj.Add(serverObj)
	params.Add(serversObj, "servers")
	params.Add(jsonutils.NewString(name), "name")

	_, err := self.ecsClient.Servers.PerformAction2("server-name", "", params, "")
	return err
}

func (self *SRegion) RebuildRoot(ctx context.Context, instanceId, passwd, publicKeyName string) error {
	params := jsonutils.NewDict()
	reinstallObj := jsonutils.NewDict()
	// meta := jsonutils.NewDict()
	// meta.Add(jsonutils.NewString(""), "user_data")
	if len(passwd) > 0 {
		reinstallObj.Add(jsonutils.NewString(passwd), "adminpass")
	} else if len(publicKeyName) > 0 {
		reinstallObj.Add(jsonutils.NewString(publicKeyName), "keyname")
	} else {
		return fmt.Errorf("both password and publicKey are empty.")
	}

	params.Add(reinstallObj, "os-reinstall")
	_, err := self.ecsClient.Servers.PerformAction2("reinstallos", instanceId, params, "")
	return err
}

func (self *SRegion) ChangeRoot(ctx context.Context, instanceId, imageId, passwd, publicKeyName string) error {
	params := jsonutils.NewDict()
	changeOsObj := jsonutils.NewDict()
	// meta := jsonutils.NewDict()
	// meta.Add(jsonutils.NewString(""), "user_data")
	if len(passwd) > 0 {
		changeOsObj.Add(jsonutils.NewString(passwd), "adminpass")
	} else if len(publicKeyName) > 0 {
		changeOsObj.Add(jsonutils.NewString(publicKeyName), "keyname")
	} else {
		return fmt.Errorf("both password and publicKey are empty.")
	}

	changeOsObj.Add(jsonutils.NewString(imageId), "imageid")
	params.Add(changeOsObj, "os-change")

	_, err := self.ecsClient.Servers.PerformAction2("changeos", instanceId, params, "")
	return err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212692.html
// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0110109377.html
// 一键式重置密码 需要安装安装一键式重置密码插件 https://support.huaweicloud.com/usermanual-ecs/zh-cn_topic_0068095385.html
// 目前不支持直接重置密钥
func (self *SRegion) DeployVM(instanceId string, name string, password string, keypairName string, deleteKeypair bool, description string) error {
	serverObj := jsonutils.NewDict()
	if len(name) > 0 {
		serverObj.Add(jsonutils.NewString(name), "name")
	}

	if len(description) > 0 {
		serverObj.Add(jsonutils.NewString(description), "description")
	}

	if serverObj.Size() > 0 {
		params := jsonutils.NewDict()
		params.Add(serverObj, "server")
		// 这里华为返回的image字段是字符串。和SInstance的定义的image是字典结构不一致。
		err := DoUpdate(self.ecsClient.NovaServers.Update, instanceId, params, nil)
		if err != nil {
			return err
		}
	}

	if len(password) > 0 {
		params := jsonutils.NewDict()
		passwdObj := jsonutils.NewDict()
		passwdObj.Add(jsonutils.NewString(password), "new_password")
		params.Add(passwdObj, "reset-password")

		err := DoUpdateWithSpec(self.ecsClient.NovaServers.UpdateInContextWithSpec, instanceId, "os-reset-password", params)
		if err != nil {
			return err
		}
	}

	return nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212653.html
func (self *SRegion) ChangeVMConfig(zoneId string, instanceId string, ncpu int, vmem int, disks []*SDisk) error {
	instanceTypes, err := self.GetMatchInstanceTypes(ncpu, vmem, zoneId)
	if err != nil {
		return err
	}

	for _, t := range instanceTypes {
		params := jsonutils.NewDict()
		resizeObj := jsonutils.NewDict()
		resizeObj.Add(jsonutils.NewString(t.ID), "flavorRef")
		params.Add(resizeObj, "resize")
		_, err := self.ecsClient.Servers.PerformAction2("resize", instanceId, params, "")
		if err != nil {
			log.Errorf("Failed for %s: %s", t.ID, err)
		} else {
			return nil
		}
	}

	return fmt.Errorf("Failed to change vm config, specification not supported")
}

func (self *SRegion) ChangeVMConfig2(zoneId string, instanceId string, instanceType string, disks []*SDisk) error {
	params := jsonutils.NewDict()
	resizeObj := jsonutils.NewDict()
	resizeObj.Add(jsonutils.NewString(instanceType), "flavorRef")
	params.Add(resizeObj, "resize")

	_, err := self.ecsClient.Servers.PerformAction2("resize", instanceId, params, "")
	return err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0142763126.html
// 微版本2.6及以上?
func (self *SRegion) GetInstanceVNCUrl(instanceId string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	vncObj := jsonutils.NewDict()
	vncObj.Add(jsonutils.NewString("novnc"), "type")
	vncObj.Add(jsonutils.NewString("vnc"), "protocol")
	params.Add(vncObj, "remote_console")

	ret, err := self.ecsClient.NovaServers.PerformAction2("remote-consoles", instanceId, params, "")
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0022472987.html
// todo: 指定device
// XEN平台虚拟机device为必选参数。
func (self *SRegion) AttachDisk(instanceId string, diskId string, device string) error {
	params := jsonutils.NewDict()
	volumeObj := jsonutils.NewDict()
	volumeObj.Add(jsonutils.NewString(diskId), "volumeId")
	params.Add(volumeObj, "volumeAttachment")

	_, err := self.ecsClient.Servers.PerformAction2("attachvolume", instanceId, params, "")
	return err
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0022472988.html
// 默认非强制卸载。delete_flag=0
func (self *SRegion) DetachDisk(instanceId string, diskId string) error {
	path := fmt.Sprintf("%s/detachvolume/%s", instanceId, diskId)
	return DoDelete(self.ecsClient.Servers.Delete, path, nil, nil)
}

// 目前无接口支持
func (region *SRegion) RenewInstance(instanceId string, bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0065817702.html
func (self *SRegion) GetInstanceSecrityGroupIds(instanceId string) ([]string, error) {
	if len(instanceId) == 0 {
		return nil, fmt.Errorf("GetInstanceSecrityGroups instanceId is empty")
	}

	securitygroups := make([]SSecurityGroup, 0)
	ctx := &modules.ManagerContext{InstanceManager: self.ecsClient.NovaServers, InstanceId: instanceId}
	err := DoListInContext(self.ecsClient.NovaSecurityGroups.ListInContext, ctx, nil, &securitygroups)
	if err != nil {
		return nil, err
	}

	securitygroupIds := []string{}
	for _, secgroup := range securitygroups {
		securitygroupIds = append(securitygroupIds, secgroup.GetId())
	}

	return securitygroupIds, nil
}

// https://support.huaweicloud.com/api-oce/zh-cn_topic_0082522030.html
func (self *SRegion) UnsubscribeInstance(instanceId string, domianId string) (jsonutils.JSONObject, error) {
	unsubObj := jsonutils.NewDict()
	unsubObj.Add(jsonutils.NewInt(1), "unSubType")
	unsubObj.Add(jsonutils.NewInt(5), "unsubscribeReasonType")
	unsubObj.Add(jsonutils.NewString("no reason"), "unsubscribeReason")
	resList := jsonutils.NewArray()
	resList.Add(jsonutils.NewString(instanceId))
	unsubObj.Add(resList, "resourceIds")

	self.ecsClient.Orders.SetDomainId(domianId)
	return self.ecsClient.Orders.PerformAction("resources/delete", "", unsubObj)
}
