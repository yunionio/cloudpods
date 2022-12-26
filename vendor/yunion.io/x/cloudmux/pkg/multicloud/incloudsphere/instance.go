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

package incloudsphere

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/imagetools"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type Cdrom struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	Connected      bool   `json:"connected"`
	StartConnected bool   `json:"startConnected"`
	CifsDto        string `json:"cifsDto"`
	DataStore      struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"dataStore"`
}

type Floppy struct {
	Path      string `json:"path"`
	DataStore string `json:"dataStore"`
	VfdType   string `json:"vfdType"`
}

type FtpServerHost struct {
	Id string `json:"id"`
	Ip string `json:"ip"`
}

type VvSourceDto struct {
	SourceType    string        `json:"sourceType"`
	FilePath      string        `json:"filePath"`
	Name          string        `json:"name"`
	FtpServerHost FtpServerHost `json:"ftpServerHost"`
}

type Volume struct {
	Id                 string      `json:"id"`
	UUID               string      `json:"uuid"`
	Size               float64     `json:"size"`
	RealSize           float64     `json:"realSize"`
	Name               string      `json:"name"`
	FileName           string      `json:"fileName"`
	Offset             int         `json:"offset"`
	Shared             bool        `json:"shared"`
	DeleteModel        string      `json:"deleteModel"`
	VolumePolicy       string      `json:"volumePolicy"`
	Format             string      `json:"format"`
	BlockDeviceId      string      `json:"blockDeviceId"`
	DiskType           string      `json:"diskType"`
	DataStoreId        string      `json:"dataStoreId"`
	DataStoreName      string      `json:"dataStoreName"`
	DataStoreSize      float64     `json:"dataStoreSize"`
	FreeStorage        float64     `json:"freeStorage"`
	DataStoreType      string      `json:"dataStoreType"`
	DataStoreReplicate int         `json:"dataStoreReplicate"`
	VMName             string      `json:"vmName"`
	VMStatus           string      `json:"vmStatus"`
	Type               string      `json:"type"`
	Description        string      `json:"description"`
	Bootable           bool        `json:"bootable"`
	VolumeStatus       string      `json:"volumeStatus"`
	MountedHostIds     string      `json:"mountedHostIds"`
	Md5                string      `json:"md5"`
	DataSize           int         `json:"dataSize"`
	OpenStackId        string      `json:"openStackId"`
	VvSourceDto        VvSourceDto `json:"vvSourceDto"`
	FormatDisk         bool        `json:"formatDisk"`
	ToBeConverted      bool        `json:"toBeConverted"`
	RelatedVms         string      `json:"relatedVms"`
	XactiveDataStoreId string      `json:"xactiveDataStoreId"`
	ClusterSize        int         `json:"clusterSize"`
	ScsiId             string      `json:"scsiId"`
	SecondaryUUID      string      `json:"secondaryUuid"`
	SecondaryVolumes   []struct{}  `json:"secondaryVolumes"`
}

type Disks struct {
	Id              string  `json:"id"`
	Label           string  `json:"label"`
	ScsiId          string  `json:"scsiId"`
	Enabled         bool    `json:"enabled"`
	WriteBps        int     `json:"writeBps"`
	ReadBps         int     `json:"readBps"`
	TotalBps        int     `json:"totalBps"`
	TotalIops       int     `json:"totalIops"`
	WriteIops       int     `json:"writeIops"`
	ReadIops        int     `json:"readIops"`
	Volume          Volume  `json:"volume"`
	BusModel        string  `json:"busModel"`
	Usage           float64 `json:"usage"`
	MonReadIops     float64 `json:"monReadIops"`
	MonWriteIops    float64 `json:"monWriteIops"`
	ReadThroughput  float64 `json:"readThroughput"`
	WriteThroughput float64 `json:"writeThroughput"`
	ReadWriteModel  string  `json:"readWriteModel"`
	EnableNativeIO  bool    `json:"enableNativeIO"`
	EnableKernelIO  bool    `json:"enableKernelIO"`
	L2CacheSize     int     `json:"l2CacheSize"`
	QueueNum        int     `json:"queueNum"`
}

type CdpInfo struct {
	CdpBackupDatastoreId string `json:"cdpBackupDatastoreId"`
	BackupDataStoreName  string `json:"backupDataStoreName"`
	StartTime            string `json:"startTime"`
	EndTime              string `json:"endTime"`
	EnableCDP            bool   `json:"enableCDP"`
	CdpAvgWriteMBps      int    `json:"cdpAvgWriteMBps"`
	CdpRemainTimes       int    `json:"cdpRemainTimes"`
	CdpLogSpaceSize      int    `json:"cdpLogSpaceSize"`
	IntervalTime         int    `json:"intervalTime"`
}

type GuestOSAuthInfo struct {
	UserName string `json:"userName"`
	UserPwd  string `json:"userPwd"`
}

type SInstance struct {
	multicloud.SInstanceBase
	InCloudSphereTags

	host *SHost

	osInfo *imagetools.ImageInfo

	Id                 string `json:"id"`
	CustomVMId         string `json:"customVmId"`
	Name               string `json:"name"`
	State              string `json:"state"`
	Status             string `json:"status"`
	HostId             string `json:"hostId"`
	HostName           string `json:"hostName"`
	HostIP             string `json:"hostIp"`
	HostStatus         string `json:"hostStatus"`
	HostMemory         int64  `json:"hostMemory"`
	DataCenterId       string `json:"dataCenterId"`
	HaEnabled          bool   `json:"haEnabled"`
	RouterFlag         bool   `json:"routerFlag"`
	Migratable         bool   `json:"migratable"`
	HostBinded         bool   `json:"hostBinded"`
	ToolsInstalled     bool   `json:"toolsInstalled"`
	ToolsVersion       string `json:"toolsVersion"`
	ToolsType          string `json:"toolsType"`
	ToolsVersionStatus string `json:"toolsVersionStatus"`
	ToolsRunningStatus string `json:"toolsRunningStatus"`
	ToolsNeedUpdate    bool   `json:"toolsNeedUpdate"`
	Description        string `json:"description"`
	HaMaxLimit         int64  `json:"haMaxLimit"`
	Template           bool   `json:"template"`
	Initialized        bool   `json:"initialized"`
	GuestosLabel       string `json:"guestosLabel"`
	GuestosType        string `json:"guestosType"`
	GuestOSInfo        struct {
		Model               string `json:"model"`
		SocketLimit         int    `json:"socketLimit"`
		SupportCpuHotPlug   bool   `json:"supportCpuHotPlug"`
		SupportDiskHotPlug  bool   `json:"supportDiskHotPlug"`
		SupportMemHotPlug   bool   `json:"supportMemHotPlug"`
		SupportUefiBootMode bool   `json:"supportUefiBootMode"`
	} `json:"guestOsInfo"`
	InnerName         string `json:"innerName"`
	UUId              string `json:"uuid"`
	MaxMemory         int64  `json:"maxMemory"`
	Memory            int    `json:"memory"`
	MemoryUsage       int64  `json:"memoryUsage"`
	MemHotplugEnabled bool   `json:"memHotplugEnabled"`
	EnableHugeMemPage bool   `json:"enableHugeMemPage"`
	CPUNum            int    `json:"cpuNum"`
	CPUSocket         int    `json:"cpuSocket"`
	CPUCore           int64  `json:"cpuCore"`
	CPUUsage          int64  `json:"cpuUsage"`
	MaxCPUNum         int64  `json:"maxCpuNum"`
	CPUHotplugEnabled bool   `json:"cpuHotplugEnabled"`
	CPUModelType      string `json:"cpuModelType"`
	CPUModelEnabled   bool   `json:"cpuModelEnabled"`
	RunningTime       int64  `json:"runningTime"`
	Boot              string `json:"boot"`
	BootMode          string `json:"bootMode"`
	SplashTime        int64  `json:"splashTime"`
	StoragePriority   int64  `json:"storagePriority"`
	USB               string `json:"usb"`

	//Usbs                     []interface{}   `json:"usbs"`
	Cdrom  Cdrom          `json:"cdrom"`
	Floppy Floppy         `json:"floppy"`
	Disks  []Disks        `json:"disks"`
	Nics   []SInstanceNic `json:"nics"`

	//Gpus                     []interface{}   `json:"gpus"`
	//VMPcis                   []interface{}   `json:"vmPcis"`
	ConfigLocation           string          `json:"configLocation"`
	HotplugEnabled           bool            `json:"hotplugEnabled"`
	VNCPort                  int64           `json:"vncPort"`
	VNCPasswd                string          `json:"vncPasswd"`
	VNCSharePolicy           string          `json:"vncSharePolicy"`
	CPUBindType              string          `json:"cpuBindType"`
	VcpuPin                  string          `json:"vcpuPin"`
	VcpuPins                 []string        `json:"vcpuPins"`
	CPUShares                int64           `json:"cpuShares"`
	PanickPolicy             string          `json:"panickPolicy"`
	DataStoreId              string          `json:"dataStoreId"`
	SdsdomainId              string          `json:"sdsdomainId"`
	ClockModel               string          `json:"clockModel"`
	CPULimit                 int64           `json:"cpuLimit"`
	MemShares                int64           `json:"memShares"`
	CPUReservation           int64           `json:"cpuReservation"`
	MemReservation           int64           `json:"memReservation"`
	LastBackup               string          `json:"lastBackup"`
	VMType                   string          `json:"vmType"`
	SystemVMType             string          `json:"systemVmType"`
	MemBalloonEnabled        bool            `json:"memBalloonEnabled"`
	Completed                bool            `json:"completed"`
	GraphicsCardModel        string          `json:"graphicsCardModel"`
	GraphicsCardMemory       int64           `json:"graphicsCardMemory"`
	GraphicsCards            string          `json:"graphicsCards"`
	VMHostName               string          `json:"vmHostName"`
	DiskTotalSize            int64           `json:"diskTotalSize"`
	DiskUsedSize             float64         `json:"diskUsedSize"`
	DiskUsage                float64         `json:"diskUsage"`
	StartPriority            string          `json:"startPriority"`
	OwnerName                string          `json:"ownerName"`
	Version                  string          `json:"version"`
	EnableReplicate          bool            `json:"enableReplicate"`
	ReplicationDatastoreId   string          `json:"replicationDatastoreId"`
	ReplicationDatastoreName string          `json:"replicationDatastoreName"`
	RecoveryFlag             bool            `json:"recoveryFlag"`
	SpiceUSBNum              int64           `json:"spiceUsbNum"`
	CDPInfo                  CdpInfo         `json:"cdpInfo"`
	GuestOSAuthInfo          GuestOSAuthInfo `json:"guestOSAuthInfo"`
	AwareNUMAEnabled         bool            `json:"awareNumaEnabled"`
	DrxEnabled               bool            `json:"drxEnabled"`
	SecretLevel              string          `json:"secretLevel"`
	//VmfpgaDevs               []interface{}   `json:"vmfpgaDevs"`
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetId() string {
	return self.Id
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

func (self *SInstance) Refresh() error {
	ins, err := self.host.zone.region.GetInstance(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ins)
}

func (self *SInstance) AssignSecurityGroup(id string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.Id, diskId)
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return self.host.zone.region.ChangeConfig(self.Id, opts.Cpu, opts.MemoryMB)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.Id)
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.ResetVmPassword(self.Id, username, password)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.Id, diskId)
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.BootMode)
}

func (self *SInstance) GetBootOrder() string {
	return strings.ToLower(self.Boot)
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) GetHostname() string {
	return self.HostName
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_INCLOUD_SPHERE
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	ret := []cloudprovider.ICloudDisk{}
	for i := range self.Disks {
		disk, err := self.host.zone.region.GetDisk(self.Disks[i].Volume.Id)
		if err != nil {
			return nil, err
		}
		ret = append(ret, disk)
	}
	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	for i := range self.Nics {
		self.Nics[i].ins = self
		ret = append(ret, &self.Nics[i])
	}
	return ret, nil
}

func (self *SInstance) GetInstanceType() string {
	return fmt.Sprintf("ecs.g1.c%dm%d", self.GetVcpuCount(), self.GetVmemSizeMB()/1024)
}

func (self *SInstance) GetMachine() string {
	return ""
}

/*
PENDING 开启准备中
STARTING 开启或恢复中
STARTED 开启的
PAUSING 暂停中
PAUSED 暂停的
RESTARTING 重启中
STOPPING 关机中
STOPPED 关闭的
HOT_SAVING 开机快照中
COLD_SAVING 关机快照中
PRE_MIGRATE 迁移准备中
HOT_MIGRATING 热迁移中
COLD_MIGRATING 冷迁移中
REVERTING 恢复快照中
EXPORTING 导出中
IMPORTING 导入中
COLD_BACKUPING 关机备份中
HOT_BACKUPING 开机备份中
COLD_MODIFYING 冷配置中
HOT_MODIFYING 热配置中
COLD_COPING 关机复制中
*/

func (self *SInstance) GetStatus() string {
	switch strings.ToUpper(self.Status) {
	case "COLD_COPING", "COLD_BACKUPING", "COLD_SAVING", "STOPPED":
		return api.VM_READY
	case "PENDING", "STARTING":
		return api.VM_START_START
	case "STARTED", "HOT_SAVING", "HOT_BACKUPING":
		return api.VM_RUNNING
	case "PAUSING", "RESTARTING", "STOPPING":
		return api.VM_START_STOP
	case "REVERTING", "EXPORTING", "IMPORTING", "COLD_MODIFYING", "HOT_MODIFYING":
		return api.VM_RESUMING
	case "PRE_MIGRATE", "HOT_MIGRATING", "COLD_MIGRATING":
		return api.VM_MIGRATING
	}
	return strings.ToLower(self.Status)
}

func (i *SInstance) GetFullOsName() string {
	return i.GuestOSInfo.Model
}

func (i *SInstance) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(i.GuestosType)
}

func (i *SInstance) GetOsArch() string {
	return i.GetIHost().GetCpuArchitecture()
}

func (i *SInstance) getNormalizedOsInfo() *imagetools.ImageInfo {
	if i.osInfo == nil {
		osInfo := imagetools.NormalizeImageInfo(i.GuestOSInfo.Model, "", "", "", "")
		i.osInfo = &osInfo
	}
	return i.osInfo
}

func (i *SInstance) GetOsDist() string {
	return i.getNormalizedOsInfo().OsDistro
}

func (i *SInstance) GetOsVersion() string {
	return i.getNormalizedOsInfo().OsVersion
}

func (i *SInstance) GetOsLang() string {
	return i.getNormalizedOsInfo().OsLang
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	ret := &cloudprovider.ServerVncOutput{
		Protocol: "vnc",
		Host:     self.host.Name,
		Port:     self.VNCPort,
		Id:       self.Id,
		Password: "1q@W3e$R",
	}
	return ret, nil
}

func (self *SInstance) GetVcpuCount() int {
	return int(self.CPUNum)
}

func (self *SInstance) GetVmemSizeMB() int {
	return int(self.Memory)
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.StartVm(self.Id, "", "")
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if self.GetStatus() == api.VM_READY {
		return nil
	}
	return self.host.zone.region.StopVm(self.Id)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetInstances(hostId string) ([]SInstance, error) {
	ret := []SInstance{}
	res := fmt.Sprintf("/hosts/%s/vms", hostId)
	return ret, self.list(res, url.Values{}, &ret)
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	ret := &SInstance{}
	res := fmt.Sprintf("/vms/%s", id)
	return ret, self.get(res, nil, ret)
}

func (self *SRegion) StartVm(id string, password, publicKey string) error {
	res := fmt.Sprintf("/vms/%s", id)
	params := url.Values{}
	params.Set("action", "poweron")
	if len(password) > 0 {
		params.Set("os-passwd", password)
	}
	if len(publicKey) > 0 {
		params.Set("publicKey", publicKey)
	}
	return self.put(res, params, nil, nil)
}

func (self *SRegion) StopVm(id string) error {
	res := fmt.Sprintf("/vms/%s", id)
	params := url.Values{}
	params.Set("action", "poweroff")
	return self.put(res, params, nil, nil)
}

func (self *SRegion) AttachDisk(vmId, diskId string) error {
	disk, err := self.GetDisk(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetDisk(%s)", diskId)
	}
	vm, err := self.GetInstance(vmId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", vmId)
	}
	vm.VNCPasswd = ""
	diskInfo := Disks{
		BusModel:       "VIRTIO",
		ReadWriteModel: "NONE",
		QueueNum:       1,
		Volume: Volume{
			Id:   disk.Id,
			Size: disk.Size,
		},
	}
	vm.Disks = append(vm.Disks, diskInfo)
	return self.put("/vms/"+vm.Id, nil, jsonutils.Marshal(vm), nil)
}

func (self *SRegion) DetachDisk(vmId, diskId string) error {
	vm, err := self.GetInstance(vmId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", vmId)
	}
	vm.VNCPasswd = ""
	disks := []Disks{}
	for i := 0; i < len(vm.Disks); i++ {
		if vm.Disks[i].Volume.Id == diskId {
			continue
		}
		disks = append(disks, vm.Disks[i])
	}
	vm.Disks = disks
	return self.put("/vms/"+vm.Id, nil, jsonutils.Marshal(vm), nil)
}

func (self *SRegion) ChangeConfig(vmId string, cpu int, memMb int) error {
	vm, err := self.GetInstance(vmId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", vmId)
	}
	vm.VNCPasswd = ""
	changed := false
	if cpu > 0 {
		vm.CPUCore = 1
		vm.CPUNum = cpu
		vm.CPUSocket = cpu
		changed = true
	}
	if memMb > 0 {
		vm.Memory = memMb
		changed = true
	}
	if !changed {
		return nil
	}
	return self.put("/vms/"+vm.Id, nil, jsonutils.Marshal(vm), nil)
}

func (self *SRegion) ResetVmPassword(vmId string, username, password string) error {
	params := url.Values{}
	params.Set("action", "updateOsAuth")
	body := map[string]interface{}{
		"userName": username,
		"userPwd":  password,
	}
	return self.put("/vms/"+vmId, params, jsonutils.Marshal(body), nil)
}

func (self *SRegion) DeleteVM(vmId string) error {
	params := url.Values{}
	params.Set("type", "force")
	return self.del("/vms/"+vmId, params, nil)
}
