package ucloud

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"
)

type SInstance struct {
	host *SHost

	UHostID            string    `json:"UHostId"`
	Zone               string    `json:"Zone"`
	LifeCycle          string    `json:"LifeCycle"`
	OSName             string    `json:"OsName"`
	ImageID            string    `json:"ImageId"`
	BasicImageID       string    `json:"BasicImageId"`
	BasicImageName     string    `json:"BasicImageName"`
	Tag                string    `json:"Tag"`
	Name               string    `json:"Name"`
	Remark             string    `json:"Remark"`
	State              string    `json:"State"`
	NetworkState       string    `json:"NetworkState"`
	HostType           string    `json:"HostType"`
	StorageType        string    `json:"StorageType"`
	TotalDiskSpace     int       `json:"TotalDiskSpace"`
	DiskSet            []DiskSet `json:"DiskSet"`
	NetCapability      string    `json:"NetCapability"`
	IPSet              []IPSet   `json:"IPSet"`
	SubnetType         string    `json:"SubnetType"`
	ChargeType         string    `json:"ChargeType"`
	ExpireTime         int64     `json:"ExpireTime"`
	AutoRenew          string    `json:"AutoRenew"`
	IsExpire           string    `json:"IsExpire"`
	UHostType          string    `json:"UHostType"`
	OSType             string    `json:"OsType"`
	CreateTime         int64     `json:"CreateTime"`
	CPU                int       `json:"CPU"`
	GPU                int       `json:"GPU"`
	MemoryMB           int       `json:"Memory"`
	TimemachineFeature string    `json:"TimemachineFeature"`
	HotplugFeature     bool      `json:"HotplugFeature"`
	NetCapFeature      bool      `json:"NetCapFeature"`
	BootDiskState      string    `json:"BootDiskState"`
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	secgroups, err := self.GetSecurityGroups()
	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	secgroupIds := make([]string, 0)
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.GetId())
	}

	return secgroupIds, nil
}

func (self *SInstance) GetProjectId() string {
	return self.host.zone.region.client.projectId
}

func (self *SInstance) GetError() error {
	return cloudprovider.ErrNotImplemented
}

type DiskSet struct {
	DiskID    string `json:"DiskId"`
	Drive     string `json:"Drive"`
	Size      int    `json:"Size"`
	Encrypted string `json:"Encrypted"`
	Type      string `json:"Type"`
}

type IPSet struct {
	Type     string `json:"Type"`
	IP       string `json:"IP"`
	IPId     string `json:"IPId"` // IP资源ID (内网IP无对应的资源ID)
	MAC      string `json:"Mac"`
	VPCID    string `json:"VPCId"`
	SubnetID string `json:"SubnetId"`
}

type SVncInfo struct {
	VNCIP       string `json:"VncIP"`
	VNCPassword string `json:"VncPassword"`
	UHostID     string `json:"UHostId"`
	Action      string `json:"Action"`
	VNCPort     int64  `json:"VncPort"`
}

func (self *SInstance) GetId() string {
	return self.UHostID
}

func (self *SInstance) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

// 实例状态，枚举值：
// >初始化: Initializing;
// >启动中: Starting;
// > 运行中: Running;
// > 关机中: Stopping;
// >关机: Stopped
// >安装失败: Install Fail;
// >重启中: Rebooting
func (self *SInstance) GetStatus() string {
	switch self.State {
	case "Running":
		return models.VM_RUNNING
	case "Stopped":
		return models.VM_READY
	case "Rebooting":
		return models.VM_STOPPING
	case "Initializing":
		return models.VM_INIT
	case "Starting":
		return models.VM_STARTING
	case "Stopping":
		return models.VM_STOPPING
	case "Install Fail":
		return models.VM_CREATE_FAILED
	default:
		return models.VM_UNKNOWN
	}
}

func (self *SInstance) Refresh() error {
	new, err := self.host.zone.region.GetInstanceByID(self.GetId())
	if err != nil {
		return err
	}

	new.host = self.host
	return jsonutils.Update(self, new)
}

func (self *SInstance) IsEmulated() bool {
	return false
}

func (self *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	// todo: add price key here
	data.Add(jsonutils.NewString(self.host.zone.GetGlobalId()), "zone_ext_id")
	if len(self.BasicImageID) > 0 {
		if image, err := self.host.zone.region.GetImage(self.BasicImageID); err != nil {
			log.Errorf("Failed to find image %s for instance %s", self.BasicImageID, self.GetName())
		} else if meta := image.GetMetadata(); meta != nil {
			data.Update(meta)
		}
	}

	return data
}

// 计费模式，枚举值为： Year，按年付费； Month，按月付费； Dynamic，按需付费（需开启权限）；
func (self *SInstance) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return models.BILLING_TYPE_PREPAID
	default:
		return models.BILLING_TYPE_POSTPAID
	}
}

func (self *SInstance) GetExpiredAt() time.Time {
	return time.Unix(self.ExpireTime, 0)
}

func (self *SInstance) GetCreateTime() time.Time {
	return time.Unix(self.CreateTime, 0)
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	diskIds := make([]string, 0)
	for _, disk := range self.DiskSet {
		diskIds = append(diskIds, disk.DiskID)
	}

	disks, err := self.host.zone.region.GetDisks("", "", diskIds)
	if err != nil {
		return nil, err
	}

	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		idisks[i] = &disks[i]
		// 将系统盘放到第0个位置
		if disks[i].GetDiskType() == models.DISK_TYPE_SYS {
			idisks[0], idisks[i] = idisks[i], idisks[0]
		}
	}

	return idisks, nil
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	nics := make([]cloudprovider.ICloudNic, 0)

	for _, ip := range self.IPSet {
		if len(ip.SubnetID) == 0 {
			continue
		}

		nic := SInstanceNic{instance: self, ipAddr: ip.IP}
		nics = append(nics, &nic)
	}

	return nics, nil
}

// 国际: Internation，BGP: BGP，内网: Private
func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	for _, ip := range self.IPSet {
		if len(ip.IPId) > 0 {
			eip, err := self.host.zone.region.GetEipById(ip.IPId)
			if err != nil {
				return nil, err
			}

			return &eip, nil
		}
	}

	return nil, nil
}

func (self *SInstance) GetVcpuCount() int8 {
	return int8(self.CPU)
}

func (self *SInstance) GetVmemSizeMB() int {
	return self.MemoryMB
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
	return osprofile.NormalizeOSType(self.OSType)
}

func (self *SInstance) GetOSName() string {
	return self.OSName
}

func (self *SInstance) GetBios() string {
	return "BIOS"
}

func (self *SInstance) GetMachine() string {
	return "pc"
}

func (self *SInstance) GetInstanceType() string {
	return self.UHostType
}

func (self *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetHypervisor() string {
	return models.HYPERVISOR_UCLOUD
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	return self.host.zone.region.GetInstanceVNCUrl(self.GetId())
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstance) GetSecurityGroups() ([]SSecurityGroup, error) {
	return self.host.zone.region.GetSecurityGroups("", self.GetId())
}

// https://docs.ucloud.cn/api/uhost-api/get_uhost_instance_vnc_info
func (self *SRegion) GetInstanceVNCUrl(instanceId string) (jsonutils.JSONObject, error) {
	params := NewUcloudParams()
	params.Set("UHostId", instanceId)
	vnc := SVncInfo{}
	err := self.DoAction("GetUHostInstanceVncInfo", params, &vnc)
	if err != nil {
		return nil, err
	}

	vncInfo := jsonutils.NewDict()
	vncInfo.Add(jsonutils.NewString(vnc.VNCIP), "host")
	vncInfo.Add(jsonutils.NewInt(vnc.VNCPort), "port")
	vncInfo.Add(jsonutils.NewString("vnc"), "protocol")
	vncInfo.Add(jsonutils.NewString(vnc.VNCPassword), "vncPassword")
	vncInfo.Add(jsonutils.NewString(vnc.VNCIP), "host")
	return vncInfo, nil
}
