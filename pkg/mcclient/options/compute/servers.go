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

package compute

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/cgrouputils"
)

var ErrEmtptyUpdate = errors.New("No valid update data")

type ServerListOptions struct {
	Zone               string   `help:"Zone ID or Name"`
	Wire               string   `help:"Wire ID or Name"`
	Network            string   `help:"Network ID or Name"`
	Disk               string   `help:"Disk ID or Name"`
	Host               string   `help:"Host ID or Name"`
	Baremetal          *bool    `help:"Show baremetal servers"`
	Gpu                *bool    `help:"Show gpu servers"`
	Secgroup           string   `help:"Secgroup ID or Name"`
	AdminSecgroup      string   `help:"AdminSecgroup ID or Name"`
	Hypervisor         string   `help:"Show server of hypervisor" choices:"kvm|esxi|container|baremetal|aliyun|azure|aws|huawei|ucloud|volcengine|zstack|openstack|google|ctyun|incloudsphere|nutanix|bingocloud|cloudpods|ecloud|jdcloud|remotefile|h3c|hcs|hcso|hcsop|proxmox|ksyun|baidu|cucloud|qingcloud"`
	Region             string   `help:"Show servers in cloudregion"`
	WithEip            *bool    `help:"Show Servers with EIP"`
	WithoutEip         *bool    `help:"Show Servers without EIP"`
	OsType             string   `help:"OS Type" choices:"linux|windows|vmware"`
	Vpc                []string `help:"Vpc id or name"`
	UsableServerForEip string   `help:"Eip id or name"`
	WithoutUserMeta    *bool    `help:"Show Servers without user metadata"`
	EipAssociable      *bool    `help:"Show Servers can associate with eip"`
	Group              string   `help:"Instance Group ID or Name"`
	HostSn             string   `help:"Host SN"`
	IpAddr             string   `help:"Fileter by ip"`
	IpAddrs            []string `help:"Fileter by ips"`

	OrderByDisk    string `help:"Order by disk size" choices:"asc|desc"`
	OrderByOsDist  string `help:"Order by os distribution" choices:"asc|desc"`
	OrderByHost    string `help:"Order by host name" choices:"asc|desc"`
	OrderByNetwork string `help:"Order by network name" choices:"asc|desc"`
	OrderByIp      string `help:"Order by ip" choices:"asc|desc"`

	ResourceType string `help:"Resource type" choices:"shared|prepaid|dedicated"`

	BillingType string `help:"billing type" choices:"postpaid|prepaid"`

	ScalingGroup string `help:"ScalingGroup's id or name'"`

	options.BaseListOptions
	options.MultiArchListOptions

	VpcProvider string `help:"filter by vpc's provider" json:"vpc_provider"`

	WithMeta *bool `help:"filter by metadata" negative:"without_meta"`

	WithUserMeta *bool `help:"filter by user metadata" negative:"without_user_meta"`

	WithHost *bool `help:"filter guest with host or not" negative:"without_host"`
}

func (o *ServerListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type ServerIdOptions struct {
	ID string `help:"ID or name of the server" json:"-"`
}

func (o *ServerIdOptions) GetId() string {
	return o.ID
}

func (o *ServerIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ServerLoginInfoOptions struct {
	ID  string `help:"ID or name of the server" json:"-"`
	Key string `help:"File name of private key, if password is encrypted by key"`
}

type ServerSSHLoginOptions struct {
	ServerLoginInfoOptions
	Host         string `help:"IP address or hostname of the server"`
	Port         int    `help:"SSH service port" default:"22"`
	User         string `help:"SSH login user"`
	Password     string `help:"SSH password"`
	UseCloudroot bool   `help:"SSH login with cloudroot"`
}

type ServerConvertToKvmOptions struct {
	ServerIdsOptions

	PreferHost string `help:"Perfer host id or name" json:"prefer_host"`
}

func (o *ServerConvertToKvmOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerStartOptions struct {
	ServerIdsOptions

	QemuVersion string `help:"prefer qemu version" json:"qemu_version"`
}

func (o *ServerStartOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerIdsOptions struct {
	ID []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
}

func (o *ServerIdsOptions) GetIds() []string {
	return o.ID
}

func (o *ServerIdsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ServerDeleteBackupOptions struct {
	ID    string `help:"ID of the server" json:"-"`
	Purge *bool  `help:"Purge Guest Backup" json:"purge"`
}

func (o *ServerDeleteBackupOptions) GetId() string {
	return o.ID
}

func (o *ServerDeleteBackupOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSwitchToBackupOptions struct {
	ID        string `help:"ID of the server" json:"-"`
	AutoStart bool   `help:"Start guest after switch to backup" json:"auto_start"`
}

func (o *ServerSwitchToBackupOptions) GetId() string {
	return o.ID
}

func (o *ServerSwitchToBackupOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerSwitchToBackupOptions) Description() string {
	return "Switch geust master to backup host"
}

type ServerCreateBackupOptions struct {
	ID        string `help:"ID of the server" json:"-"`
	AutoStart bool   `help:"Start guest after create backup guest" json:"auto_start"`
}

func (o *ServerCreateBackupOptions) GetId() string {
	return o.ID
}

func (o *ServerCreateBackupOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerShowOptions struct {
	options.BaseShowOptions `id->help:"ID or name of the server"`
}

func (o *ServerShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerShowOptions) GetId() string {
	return o.ID
}

type ServerChangeOwnerCandidateDomainsOptions struct {
	ServerShowOptions
}

func (o *ServerChangeOwnerCandidateDomainsOptions) Description() string {
	return "Get change owner candidate domain list"
}

func ParseServerDeployInfo(info string) (*computeapi.DeployConfig, error) {
	sdi := &computeapi.DeployConfig{}
	colon := strings.IndexByte(info, ':')
	if colon <= 0 {
		return nil, fmt.Errorf("malformed deploy info: %s", info)
	}
	if info[0] == '+' {
		sdi.Action = "append"
		sdi.Path = info[1:colon]
	} else {
		sdi.Action = "create"
		sdi.Path = info[:colon]
	}
	nameOrContent := info[colon+1:]
	data, err := ioutil.ReadFile(nameOrContent)
	if err != nil {
		sdi.Content = nameOrContent
	} else {
		sdi.Content = string(data)
	}
	return sdi, nil
}

func ParseServerDeployInfoList(list []string) ([]*computeapi.DeployConfig, error) {
	ret := make([]*computeapi.DeployConfig, 0)
	for _, info := range list {
		deployInfo, err := ParseServerDeployInfo(info)
		if err != nil {
			return nil, err
		}
		ret = append(ret, deployInfo)
	}
	return ret, nil
}

type ServerCreateCommonConfig struct {
	Manager string `help:"Preferred cloudprovider where virtual server should bd created" json:"prefer_manager"`
	Region  string `help:"Preferred region where virtual server should be created" json:"prefer_region"`
	Zone    string `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Wire    string `help:"Preferred wire where virtual server should be created" json:"prefer_wire"`
	Host    string `help:"Preferred host where virtual server should be created" json:"prefer_host"`

	ResourceType   string   `help:"Resource type" choices:"shared|prepaid|dedicated"`
	Schedtag       []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Net            []string `help:"Network descriptions" metavar:"NETWORK"`
	NetSchedtag    []string `help:"Network schedtag description, e.g. '0:<tag>:<strategy>'"`
	IsolatedDevice []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
	Project        string   `help:"'Owner project ID or Name" json:"tenant"`
	User           string   `help:"Owner user ID or Name"`
	Count          int      `help:"Create multiple simultaneously" default:"1"`
	Disk           []string `help:"
	Disk descriptions
	size: 500M, 10G
	fs: swap, ext2, ext3, ext4, xfs, ntfs, fat, hfsplus
	format: qcow2, raw, docker, iso, vmdk, vmdkflatver1, vmdkflatver2, vmdkflat, vmdksparse, vmdksparsever1, vmdksparsever2, vmdksesparse, vhd
	driver: virtio, ide, scsi, sata, pvscsi
	cache_mod: writeback, none, writethrough
	medium: rotate, ssd, hybrid
	disk_type: sys, data
	mountpoint: /, /opt
	storage_type: local, rbd, nas, nfs
	snapshot_id: use snapshot-list get snapshot id
	disk_id: use disk-list get disk id
	storage_id: use storage-list get storage id
	image_id: use image-list get image id
	for example:
		--disk 'image_id=c2be02a4-7ff2-43e6-8a00-a489e04d2d6f,size=10G,driver=ide,storage_type=rbd'
		--disk 'size=500M'
		--disk 'snpahost_id=1ceb8c6d-6571-451d-8957-4bd3a871af85'
	" nargs:"+"`
	DiskSchedtag []string `help:"Disk schedtag description, e.g. '0:<tag>:<strategy>'"`
}

func (o ServerCreateCommonConfig) Data() (*computeapi.ServerConfigs, error) {
	data := &computeapi.ServerConfigs{
		PreferManager: o.Manager,
		PreferRegion:  o.Region,
		PreferZone:    o.Zone,
		PreferWire:    o.Wire,
		PreferHost:    o.Host,
		ResourceType:  o.ResourceType,
		Count:         o.Count,
		Networks:      make([]*computeapi.NetworkConfig, 0),
		Disks:         make([]*computeapi.DiskConfig, 0),
	}
	for i, n := range o.Net {
		net, err := cmdline.ParseNetworkConfig(n, i)
		if err != nil {
			return nil, err
		}
		data.Networks = append(data.Networks, net)
	}
	for _, ntag := range o.NetSchedtag {
		idx, tag, err := cmdline.ParseResourceSchedtagConfig(ntag)
		if err != nil {
			return nil, fmt.Errorf("ParseDiskSchedtag: %v", err)
		}
		if idx >= len(data.Networks) {
			return nil, fmt.Errorf("Invalid network index: %d", idx)
		}
		n := data.Networks[idx]
		n.Schedtags = append(n.Schedtags, tag)
	}
	for i, g := range o.IsolatedDevice {
		dev, err := cmdline.ParseIsolatedDevice(g, i)
		if err != nil {
			return nil, err
		}
		data.IsolatedDevices = append(data.IsolatedDevices, dev)
	}
	for _, tag := range o.Schedtag {
		schedtag, err := cmdline.ParseSchedtagConfig(tag)
		if err != nil {
			return nil, err
		}
		data.Schedtags = append(data.Schedtags, schedtag)
	}
	for i, d := range o.Disk {
		disk, err := cmdline.ParseDiskConfig(d, i)
		if err != nil {
			return nil, err
		}
		data.Disks = append(data.Disks, disk)
	}
	for _, dtag := range o.DiskSchedtag {
		idx, tag, err := cmdline.ParseResourceSchedtagConfig(dtag)
		if err != nil {
			return nil, fmt.Errorf("ParseDiskSchedtag: %v", err)
		}
		if idx >= len(data.Disks) {
			return nil, fmt.Errorf("Invalid disk index: %d", idx)
		}
		d := data.Disks[idx]
		d.Schedtags = append(d.Schedtags, tag)
	}
	return data, nil
}

type ServerConfigs struct {
	ServerCreateCommonConfig
	Hypervisor                   string `help:"Hypervisor type" choices:"kvm|pod|esxi|baremetal|container|aliyun|azure|qcloud|aws|huawei|openstack|ucloud|volcengine|zstack|google|ctyun|incloudsphere|bingocloud|cloudpods|ecloud|jdcloud|remotefile|h3c|hcs|hcso|hcsop|proxmox"`
	Backup                       bool   `help:"Create server with backup server"`
	BackupHost                   string `help:"Perfered host where virtual backup server should be created"`
	AutoSwitchToBackupOnHostDown bool   `help:"Auto switch to backup server on host down"`
	Daemon                       *bool  `help:"Set as a daemon server" json:"is_daemon"`

	RaidConfig []string `help:"Baremetal raid config" json:"-"`
}

func (o ServerConfigs) Data() (*computeapi.ServerConfigs, error) {
	data, err := o.ServerCreateCommonConfig.Data()
	if err != nil {
		return nil, err
	}
	data.Backup = o.Backup
	data.PreferBackupHost = o.BackupHost
	data.IsDaemon = o.Daemon
	data.Hypervisor = o.Hypervisor
	if len(o.RaidConfig) > 0 {
		// if data.Hypervisor != "baremetal" {
		// 	return nil, fmt.Errorf("RaidConfig is applicable to baremetal ONLY")
		// }
		for _, conf := range o.RaidConfig {
			raidConf, err := cmdline.ParseBaremetalDiskConfig(conf)
			if err != nil {
				return nil, err
			}
			data.BaremetalDiskConfigs = append(data.BaremetalDiskConfigs, raidConf)
		}
	}
	return data, nil
}

type ServerCloneOptions struct {
	SOURCE      string `help:"Source server id or name"  json:"-"`
	TARGET_NAME string `help:"Name of newly server" json:"name"`
	AutoStart   bool   `help:"Auto start server after it is created"`

	EipBw         int    `help:"allocate EIP with bandwidth in MB when server is created" json:"eip_bw,omitzero"`
	EipChargeType string `help:"newly allocated EIP charge type" choices:"traffic|bandwidth" json:"eip_charge_type,omitempty"`
	Eip           string `help:"associate with an existing EIP when server is created" json:"eip,omitempty"`
}

func (o *ServerCloneOptions) GetId() string {
	return o.SOURCE
}

func (o *ServerCloneOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *ServerCloneOptions) Description() string {
	return "Clone a server"
}

type ServerCreateFromInstanceSnapshot struct {
	InstaceSnapshotId string `help:"Instace snapshot id or name"`
	NAME              string `help:"Name of newly server" json:"name"`
	AutoStart         bool   `help:"Auto start server after it is created"`
	AllowDelete       bool   `help:"Unlock server to allow deleting"`

	EipBw         int    `help:"allocate EIP with bandwidth in MB when server is created" json:"eip_bw,omitzero"`
	EipChargeType string `help:"newly allocated EIP charge type" choices:"traffic|bandwidth" json:"eip_charge_type,omitempty"`
	Eip           string `help:"associate with an existing EIP when server is created" json:"eip,omitempty"`
}

type ServerCreateOptions struct {
	ServerCreateOptionalOptions

	NAME string `help:"Name of server" json:"-"`
}

type ServerCreateOptionalOptions struct {
	ServerConfigs

	MemSpec        string `help:"Memory size Or Instance Type" metavar:"MEMSPEC" json:"-"`
	CpuSockets     int    `help:"Cpu sockets"`
	EnableMemclean bool   `help:"clean guest memory after guest exit" json:"enable_memclean"`

	Keypair          string   `help:"SSH Keypair"`
	Password         string   `help:"Default user password"`
	LoginAccount     string   `help:"Guest login account"`
	Iso              string   `help:"ISO image ID" metavar:"IMAGE_ID" json:"cdrom"`
	IsoBootIndex     *int8    `help:"Iso bootindex" metavar:"IMAGE_BOOT_INDEX" json:"cdrom_boot_index"`
	VcpuCount        int      `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	InstanceType     string   `help:"instance flavor"`
	Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl|virtio"`
	Vdi              string   `help:"VDI protocool" choices:"vnc|spice"`
	Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
	Machine          string   `help:"Machine type" choices:"pc|q35"`
	Desc             string   `help:"Description" metavar:"<DESCRIPTION>" json:"description"`
	Boot             string   `help:"Boot device" metavar:"<BOOT_DEVICE>" choices:"disk|cdrom" json:"-"`
	EnableCloudInit  bool     `help:"Enable cloud-init service"`
	NoAccountInit    *bool    `help:"Not reset account password"`
	AllowDelete      *bool    `help:"Unlock server to allow deleting" json:"-"`
	ShutdownBehavior string   `help:"Behavior after VM server shutdown" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate"`
	AutoStart        bool     `help:"Auto start server after it is created"`
	Deploy           []string `help:"Specify deploy files in virtual server file system" json:"-"`
	DeployTelegraf   bool     `help:"Deploy telegraf agent if guest os is supported"`
	Group            []string `help:"Group ID or Name of virtual server"`
	System           bool     `help:"Create a system VM, sysadmin ONLY option" json:"is_system"`
	TaskNotify       *bool    `help:"Setup task notify" json:"-"`
	FakeCreate       *bool    `help:"Fake create server"`
	DryRun           *bool    `help:"Dry run to test scheduler" json:"-"`
	UserDataFile     string   `help:"user_data file path" json:"-"`
	InstanceSnapshot string   `help:"instance snapshot" json:"instance_snapshot"`
	Secgroups        []string `help:"secgroups" json:"secgroups"`

	OsType string `help:"os type, e.g. Linux, Windows, etc."`

	Duration  string `help:"valid duration of the server, e.g. 1H, 1D, 1W, 1M, 1Y, ADMIN ONLY option"`
	AutoRenew bool   `help:"auto renew for prepaid server"`

	AutoPrepaidRecycle bool `help:"automatically enable prepaid recycling after server is created successfully" json:"auto_prepaid_recycle,omitfalse"`

	GenerateName bool `help:"name is generated by pattern" json:"-"`

	EipBw         int    `help:"allocate EIP with bandwidth in MB when server is created" json:"eip_bw,omitzero"`
	EipBgpType    string `help:"desired BGP type of newly alloated EIP" json:"eip_bgp_type,omitzero"`
	EipChargeType string `help:"newly allocated EIP charge type" choices:"traffic|bandwidth" json:"eip_charge_type,omitempty"`
	Eip           string `help:"associate with an existing EIP when server is created" json:"eip,omitempty"`

	PublicIpBw         int    `help:"associate public ip with bandwidth in MB where server is created" json:"public_ip_bw,omitzero"`
	PublicIpChargeType string `help:"newly allocated public ip charge type" choices:"traffic|bandwidth" json:"public_ip_charge_type,omitempty"`

	GuestImageID string `help:"create from guest image, need to specify the guest image id"`

	EncryptKey string `help:"encryption key"`
}

func (o *ServerCreateOptions) ToScheduleInput() (*schedapi.ScheduleInput, error) {
	// so serious error
	data := new(schedapi.ServerConfig)
	data.ServerConfigs = computeapi.NewServerConfigs()

	// only support digit number as for now
	memSize, err := strconv.Atoi(o.MemSpec)
	if err != nil {
		return nil, err
	}
	data.Memory = memSize
	if o.VcpuCount > 0 {
		data.Ncpu = o.VcpuCount
	}
	for i, d := range o.Disk {
		disk, err := cmdline.ParseDiskConfig(d, i)
		if err != nil {
			return nil, err
		}
		data.Disks = append(data.Disks, disk)
	}
	for i, n := range o.Net {
		net, err := cmdline.ParseNetworkConfig(n, i)
		if err != nil {
			return nil, err
		}
		data.Networks = append(data.Networks, net)
	}
	for i, g := range o.IsolatedDevice {
		dev, err := cmdline.ParseIsolatedDevice(g, i)
		if err != nil {
			return nil, err
		}
		data.IsolatedDevices = append(data.IsolatedDevices, dev)
	}
	count := 1
	if o.Count > 1 {
		count = o.Count
	}
	input := new(schedapi.ScheduleInput)

	data.Count = count
	data.InstanceGroupIds = o.Group
	input.ServerConfig = *data

	if o.DryRun != nil && *o.DryRun {
		input.Details = true
	}
	return input, nil
}

func (opts *ServerCreateOptionalOptions) OptionalParams() (*computeapi.ServerCreateInput, error) {
	config, err := opts.ServerConfigs.Data()
	if err != nil {
		return nil, err
	}

	params := &computeapi.ServerCreateInput{
		ServerConfigs:      config,
		VcpuCount:          opts.VcpuCount,
		KeypairId:          opts.Keypair,
		Password:           opts.Password,
		LoginAccount:       opts.LoginAccount,
		Cdrom:              opts.Iso,
		CdromBootIndex:     opts.IsoBootIndex,
		Vga:                opts.Vga,
		Vdi:                opts.Vdi,
		Bios:               opts.Bios,
		Machine:            opts.Machine,
		ShutdownBehavior:   opts.ShutdownBehavior,
		AutoStart:          opts.AutoStart,
		Duration:           opts.Duration,
		AutoRenew:          opts.AutoRenew,
		AutoPrepaidRecycle: opts.AutoPrepaidRecycle,
		EipBw:              opts.EipBw,
		EipBgpType:         opts.EipBgpType,
		EipChargeType:      opts.EipChargeType,
		PublicIpBw:         opts.PublicIpBw,
		PublicIpChargeType: opts.PublicIpChargeType,
		Eip:                opts.Eip,
		EnableCloudInit:    opts.EnableCloudInit,
		OsType:             opts.OsType,
		GuestImageID:       opts.GuestImageID,
		Secgroups:          opts.Secgroups,
		EnableMemclean:     opts.EnableMemclean,
	}

	params.ProjectId = opts.Project

	if opts.FakeCreate != nil {
		params.FakeCreate = *opts.FakeCreate
	}

	if len(opts.EncryptKey) > 0 {
		params.EncryptKeyId = &opts.EncryptKey
	}

	if regutils.MatchSize(opts.MemSpec) {
		memSize, err := fileutils.GetSizeMb(opts.MemSpec, 'M', 1024)
		if err != nil {
			return nil, err
		}
		params.VmemSize = memSize
	} else {
		params.InstanceType = opts.InstanceType
	}

	deployInfos, err := ParseServerDeployInfoList(opts.Deploy)
	if err != nil {
		return nil, err
	}
	params.DeployConfigs = deployInfos
	params.DeployTelegraf = opts.DeployTelegraf

	if len(opts.Boot) > 0 {
		if opts.Boot == "disk" {
			params.BootOrder = "cdn"
		} else {
			params.BootOrder = "dcn"
		}
	}

	resetPasswd := false
	if opts.NoAccountInit != nil && *opts.NoAccountInit {
		params.ResetPassword = &resetPasswd
	} else {
		params.ResetPassword = nil
	}

	if len(opts.UserDataFile) > 0 {
		userdata, err := ioutil.ReadFile(opts.UserDataFile)
		if err != nil {
			return nil, err
		}
		params.UserData = string(userdata)
	}

	if options.BoolV(opts.AllowDelete) {
		disableDelete := false
		params.DisableDelete = &disableDelete
	}

	if options.BoolV(opts.DryRun) {
		params.Suggestion = true
	}

	// group
	params.InstanceGroupIds = opts.Group
	// set description
	params.Description = opts.Desc

	params.IsSystem = &opts.System

	return params, nil
}

func (opts *ServerCreateOptions) Params() (*computeapi.ServerCreateInput, error) {
	params, err := opts.OptionalParams()
	if err != nil {
		return nil, err
	}

	if opts.GenerateName {
		params.GenerateName = opts.NAME
	} else {
		params.Name = opts.NAME
	}

	return params, nil
}

type ServerStopOptions struct {
	ID           []string `help:"ID or Name of server" json:"-"`
	Force        *bool    `help:"Stop server forcefully" json:"is_force"`
	StopCharging *bool    `help:"Stop charging when server stop"`
}

func (o *ServerStopOptions) GetIds() []string {
	return o.ID
}

func (o *ServerStopOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerUpdateOptions struct {
	ServerIdsOptions
	Name             string `help:"New name to change"`
	Vmem             string `help:"Memory size" json:"vmem_size"`
	Ncpu             *int   `help:"CPU count" json:"vcpu_count"`
	Vga              string `help:"VGA driver" choices:"std|vmware|cirrus|qxl|virtio"`
	Vdi              string `help:"VDI protocol" choices:"vnc|spice"`
	Bios             string `help:"BIOS" choices:"BIOS|UEFI"`
	Desc             string `help:"Description" json:"description"`
	Boot             string `help:"Boot device" choices:"disk|cdrom"`
	Delete           string `help:"Lock server to prevent from deleting" choices:"enable|disable" json:"-"`
	ShutdownBehavior string `help:"Behavior after VM server shutdown" choices:"stop|terminate"`
	Machine          string `help:"Machine type" choices:"q35|pc"`

	IsDaemon *bool `help:"Daemon server" negative:"no-daemon"`

	PendingDeletedAt string `help:"change pending deleted time"`

	Hostname string `help:"host name of server"`
}

func (opts *ServerUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}

	if len(opts.Boot) > 0 {
		if opts.Boot == "disk" {
			params.Set("boot_order", jsonutils.NewString("cdn"))
		} else {
			params.Set("boot_order", jsonutils.NewString("dcn"))
		}
	}
	if len(opts.Delete) > 0 {
		if opts.Delete == "disable" {
			params.Set("disable_delete", jsonutils.JSONTrue)
		} else {
			params.Set("disable_delete", jsonutils.JSONFalse)
		}
	}
	if params.Size() == 0 {
		return nil, ErrEmtptyUpdate
	}
	return params, nil
}

type ServerDeleteOptions struct {
	ServerIdsOptions
	OverridePendingDelete *bool `help:"Delete server directly instead of pending delete" short-token:"f"`
	DeleteSnapshots       *bool `help:"Delete server snapshots"`
	DeleteDisks           *bool `help:"Delete server disks"`
	DeleteEip             *bool `help:"Delete eip"`
}

func (o *ServerDeleteOptions) QueryParams() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerCancelDeleteOptions struct {
	ServerIdsOptions
}

func (o *ServerCancelDeleteOptions) Description() string {
	return "Cancel pending delete servers"
}

type ServerDeployOptions struct {
	ServerIdOptions
	Keypair        string   `help:"ssh Keypair used for login" json:"-"`
	DeleteKeypair  bool     `help:"Remove ssh Keypairs" json:"-"`
	Deploy         []string `help:"Specify deploy files in virtual server file system" json:"-"`
	ResetPassword  bool     `help:"Force reset password"`
	Password       string   `help:"Default user password"`
	AutoStart      bool     `help:"Auto start server after deployed"`
	DeployTelegraf bool     `help:"Deploy telegraf if guest os supported"`
}

func (opts *ServerDeployOptions) Params() (jsonutils.JSONObject, error) {
	params := new(computeapi.ServerDeployInput)
	{
		if opts.DeleteKeypair == true {
			params.DeleteKeypair = opts.DeleteKeypair
		} else if len(opts.Keypair) > 0 {
			params.Keypair = opts.Keypair
		}
		params.AutoStart = opts.AutoStart
		params.ResetPassword = opts.ResetPassword
		params.Password = opts.Password
		params.DeployTelegraf = opts.DeployTelegraf
	}
	{
		deployInfos, err := ParseServerDeployInfoList(opts.Deploy)
		if err != nil {
			return nil, err
		}
		params.DeployConfigs = deployInfos
	}
	return params.JSON(params), nil
}

func (opts *ServerDeployOptions) Description() string {
	return "Deploy hostname and keypair to a stopped virtual server"
}

type ServerSecGroupOptions struct {
	ID     string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	Secgrp string `help:"ID of Security Group" metavar:"Security Group" positional:"true"`
}

func (o *ServerSecGroupOptions) GetId() string {
	return o.ID
}

func (o *ServerSecGroupOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSecGroupsOptions struct {
	ID          string   `help:"ID or Name of server" metavar:"Guest" json:"-"`
	SecgroupIds []string `help:"Ids of Security Groups" metavar:"Security Groups" positional:"true"`
}

func (o *ServerSecGroupsOptions) GetId() string {
	return o.ID
}

func (opts *ServerSecGroupsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string][]string{"secgroup_ids": opts.SecgroupIds}), nil
}

type ServerModifySrcCheckOptions struct {
	ID          string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	SrcIpCheck  string `help:"Turn on/off src ip check" choices:"on|off"`
	SrcMacCheck string `help:"Turn on/off src mac check" choices:"on|off"`
}

func (o *ServerModifySrcCheckOptions) GetId() string {
	return o.ID
}

func (o *ServerModifySrcCheckOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerModifySrcCheckOptions) Description() string {
	return "Modify src ip, mac check settings"
}

type ServerSendKeyOptions struct {
	ID   string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	KEYS string `help:"Special keys to send, eg. ctrl, alt, f12, shift, etc, separated by \"-\""`
	Hold *uint  `help:"Hold key for specified milliseconds" json:"duration"`
}

func (o *ServerSendKeyOptions) GetId() string {
	return o.ID
}

func (o *ServerSendKeyOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerSendKeyOptions) Description() string {
	return "Send keys to server"
}

type ServerMonitorOptions struct {
	ServerIdOptions

	Qmp     bool   `help:"Use qmp protocol, default is hmp"`
	COMMAND string `help:"Qemu Monitor command to send"`
}

func (o *ServerMonitorOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerQgaSetPassword struct {
	ServerIdOptions

	USERNAME string `help:"Which user to set password" json:"username"`
	PASSWORD string `help:"Password content" json:"password"`
}

func (o *ServerQgaSetPassword) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerQgaCommand struct {
	ServerIdOptions

	COMMAND string `help:"qga command"`
	Timeout int    `help:"qga command execute timeout (ms)"`
}

func (o *ServerQgaCommand) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerQgaPing struct {
	ServerIdOptions

	Timeout int `help:"qga command execute timeout (ms)"`
}

func (o *ServerQgaPing) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerQgaGuestInfoTask struct {
	ServerIdOptions
}

func (o *ServerQgaGuestInfoTask) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerQgaGetNetwork struct {
	ServerIdOptions
}

func (o *ServerQgaGetNetwork) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSetPasswordOptions struct {
	ServerIdOptions

	Username      string `help:"Which user to set password" json:"username"`
	Password      string `help:"Password content" json:"password"`
	ResetPassword bool   `help:"Force reset password"`
	AutoStart     bool   `help:"Auto start server after reset password"`
}

func (o *ServerSetPasswordOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSetBootIndexOptions struct {
	ServerIdOptions
	Disks  map[string]int8 `help:"Disk index and boot index" json:"disks"`
	Cdroms map[string]int8 `help:"Cdrom ordinal and boot index" json:"cdroms"`
}

func (o *ServerSetBootIndexOptions) Params() (jsonutils.JSONObject, error) {
	for k, _ := range o.Disks {
		if i, e := strconv.Atoi(k); e != nil {
			return nil, e
		} else if i > 127 {
			return nil, fmt.Errorf("disk index grate than 127")
		}
	}
	for k, _ := range o.Cdroms {
		if _, e := strconv.Atoi(k); e != nil {
			return nil, e
		}
	}

	return options.StructToParams(o)
}

type ServerNicTrafficLimitOptions struct {
	ServerIdOptions
	MAC            string `help:"guest network mac address"`
	RxTrafficLimit *int64 `help:" rx traffic limit, unit Byte"`
	TxTrafficLimit *int64 `help:" tx traffic limit, unit Byte"`
}

func (o *ServerNicTrafficLimitOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSaveImageOptions struct {
	ServerIdOptions
	IMAGE     string `help:"Image name" json:"name"`
	Public    *bool  `help:"Make the image public available" json:"is_public"`
	Format    string `help:"image format" choices:"vmdk|qcow2"`
	Notes     string `help:"Notes about the image"`
	AutoStart *bool  `help:"Auto start server after image saved"`
}

func (o *ServerSaveImageOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerSaveImageOptions) Description() string {
	return "Save root disk to new image and upload to glance."
}

type ServerSaveGuestImageOptions struct {
	ServerIdOptions
	IMAGE     string `help:"Image name" json:"name"`
	AutoStart *bool  `help:"Auto start server after image saved"`
}

func (o *ServerSaveGuestImageOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerSaveGuestImageOptions) Description() string {
	return "Save root disk and data disks to new images and upload to glance."
}

type ServerChangeOwnerOptions struct {
	ID      string `help:"Server to change owner" json:"-"`
	PROJECT string `help:"Project ID or change" json:"tenant"`
}

func (o *ServerChangeOwnerOptions) GetId() string {
	return o.ID
}

func (o *ServerChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerRebuildRootOptions struct {
	ID            string `help:"Server to rebuild root" json:"-"`
	ImageId       string `help:"New root Image template ID" json:"image_id" token:"image"`
	Keypair       string `help:"ssh Keypair used for login"`
	Password      string `help:"Default user password"`
	NoAccountInit *bool  `help:"Not reset account password"`
	AutoStart     *bool  `help:"Auto start server after it is created"`
	AllDisks      *bool  `help:"Rebuild all disks including data disks"`
	UserData      string `hlep:"user data scripts"`
}

func (o *ServerRebuildRootOptions) GetId() string {
	return o.ID
}

func (o *ServerRebuildRootOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	if o.NoAccountInit != nil && *o.NoAccountInit {
		params.Add(jsonutils.JSONFalse, "reset_password")
	}
	return params, nil
}

func (o *ServerRebuildRootOptions) Description() string {
	return "Rebuild VM root image with new template"
}

type ServerChangeConfigOptions struct {
	ServerIdOptions
	VcpuCount  *int     `help:"New number of Virtual CPU cores" json:"vcpu_count" token:"ncpu"`
	CpuSockets *int     `help:"Cpu sockets"`
	VmemSize   string   `help:"New memory size" json:"vmem_size" token:"vmem"`
	Disk       []string `help:"Data disk description, from the 1st data disk to the last one, empty string if no change for this data disk"`

	InstanceType string `help:"Instance Type, e.g. S2.SMALL2 for qcloud"`

	ResetTrafficLimits []string `help:"reset traffic limits, mac,rx,tx"`
	SetTrafficLimits   []string `help:"set traffic limits, mac,rx,tx"`
}

func (o *ServerChangeConfigOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	if len(o.Disk) > 0 {
		params.Remove("disk.0")
		disksConf := make([]*computeapi.DiskConfig, 0)
		for i, d := range o.Disk {
			// params.Set(key, value)
			diskConfig, err := cmdline.ParseDiskConfig(d, i+1)
			if err != nil {
				return nil, err
			}
			disksConf = append(disksConf, diskConfig)
		}
		params.Set("disks", jsonutils.Marshal(disksConf))
	}
	if err != nil {
		return nil, err
	}

	if len(o.ResetTrafficLimits) > 0 {
		// mac,rx_limit,tx_limit
		// ab:bc:cd:ef:ad:fa,12312312,1231233
		resetLimits := []*jsonutils.JSONDict{}
		for i := range o.ResetTrafficLimits {
			resetLimit := jsonutils.NewDict()
			segs := strings.Split(o.ResetTrafficLimits[i], ",")
			if len(segs) != 3 {
				return nil, fmt.Errorf("invalid reset traffic limit input %s", o.ResetTrafficLimits[i])
			}
			resetLimit.Set("mac", jsonutils.NewString(segs[0]))
			rx, err := strconv.Atoi(segs[1])
			if err != nil {
				return nil, fmt.Errorf("invalid reset traffic limit input %s: %s", o.ResetTrafficLimits[i], err)
			}
			resetLimit.Set("rx_traffic_limit", jsonutils.NewInt(int64(rx)))
			tx, err := strconv.Atoi(segs[1])
			if err != nil {
				return nil, fmt.Errorf("invalid reset traffic limit input %s: %s", o.ResetTrafficLimits[i], err)
			}
			resetLimit.Set("tx_traffic_limit", jsonutils.NewInt(int64(tx)))
			resetLimits = append(resetLimits, resetLimit)
		}
		params.Set("reset_traffic_limits", jsonutils.Marshal(resetLimits))
	}

	if len(o.SetTrafficLimits) > 0 {
		// mac,rx_limit,tx_limit
		// ab:bc:cd:ef:ad:fa,12312312,1231233
		setLimits := []*jsonutils.JSONDict{}
		for i := range o.SetTrafficLimits {
			setLimit := jsonutils.NewDict()
			segs := strings.Split(o.SetTrafficLimits[i], ",")
			if len(segs) != 3 {
				return nil, fmt.Errorf("invalid reset traffic limit input %s", o.SetTrafficLimits[i])
			}
			setLimit.Set("mac", jsonutils.NewString(segs[0]))
			rx, err := strconv.Atoi(segs[1])
			if err != nil {
				return nil, fmt.Errorf("invalid reset traffic limit input %s: %s", o.SetTrafficLimits[i], err)
			}
			setLimit.Set("rx_traffic_limit", jsonutils.NewInt(int64(rx)))
			tx, err := strconv.Atoi(segs[1])
			if err != nil {
				return nil, fmt.Errorf("invalid reset traffic limit input %s: %s", o.SetTrafficLimits[i], err)
			}
			setLimit.Set("tx_traffic_limit", jsonutils.NewInt(int64(tx)))
			setLimits = append(setLimits, setLimit)
		}
		params.Set("set_traffic_limits", jsonutils.Marshal(setLimits))
	}

	if params.Size() == 0 {
		return nil, ErrEmtptyUpdate
	}
	return params, nil
}

func (o *ServerChangeConfigOptions) Description() string {
	return "Change configuration of VM"
}

type ServerResetOptions struct {
	ID   []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	Hard *bool    `help:"Hard reset or not; default soft" json:"is_hard"`
}

func (o *ServerResetOptions) GetIds() []string {
	return o.ID
}

func (o *ServerResetOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerRestartOptions struct {
	ID      []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	IsForce *bool    `help:"Force reset or not; default false" json:"is_force"`
}

func (o *ServerRestartOptions) GetIds() []string {
	return o.ID
}

func (o *ServerRestartOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerMigrateForecastOptions struct {
	ID              string `help:"ID of server" json:"-"`
	PreferHost      string `help:"Server migration prefer host id or name" json:"prefer_host"`
	LiveMigrate     *bool  `help:"Use live migrate"`
	SkipCpuCheck    *bool  `help:"Skip check CPU mode of the target host" json:"skip_cpu_check"`
	SkipKernelCheck *bool  `help:"Skip target kernel version check" json:"skip_kernel_check"`
}

func (o *ServerMigrateForecastOptions) GetId() string {
	return o.ID
}

func (o *ServerMigrateForecastOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerMigrateOptions struct {
	ID         string `help:"ID of server" json:"-"`
	PreferHost string `help:"Server migration prefer host id or name" json:"prefer_host"`
	AutoStart  *bool  `help:"Server auto start after migrate" json:"auto_start"`
	RescueMode *bool  `help:"Migrate server in rescue mode, all disks must reside on shared storage" json:"rescue_mode"`
}

func (o *ServerMigrateOptions) GetId() string {
	return o.ID
}

func (o *ServerMigrateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerLiveMigrateOptions struct {
	ID              string `help:"ID of server" json:"-"`
	PreferHost      string `help:"Server migration prefer host id or name" json:"prefer_host"`
	SkipCpuCheck    *bool  `help:"Skip check CPU mode of the target host" json:"skip_cpu_check"`
	SkipKernelCheck *bool  `help:"Skip target kernel version check" json:"skip_kernel_check"`
	EnableTLS       *bool  `help:"Enable tls migration" json:"enable_tls"`
	QuicklyFinish   *bool  `help:"quickly finish, fix downtime after a few rounds of memory synchronization"`
	MaxBandwidthMb  *int64 `help:"live migrate downtime, unit MB"`

	KeepDestGuestOnFailed *bool `help:"do not delete dest guest on migrate failed, for debug"`
}

func (o *ServerLiveMigrateOptions) GetId() string {
	return o.ID
}

func (o *ServerLiveMigrateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerSetLiveMigrateParamsOptions struct {
	ID              string `help:"ID of server" json:"-"`
	MaxBandwidthMB  *int64 `help:"live migrate downtime, unit MB"`
	DowntimeLimitMS *int64 `help:"live migrate downtime limit"`
}

func (o *ServerSetLiveMigrateParamsOptions) GetId() string {
	return o.ID
}

func (o *ServerSetLiveMigrateParamsOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type ServerBatchMetadataOptions struct {
	Guests []string `help:"IDs or names of server" json:"-"`
	TAGS   []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
}

func (opts *ServerBatchMetadataOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Guests) == 0 {
		return nil, fmt.Errorf("missing guest option")
	}
	params.Add(jsonutils.Marshal(opts.Guests), "guests")
	metadata := jsonutils.NewDict()
	for _, tag := range opts.TAGS {
		info := strings.Split(tag, "=")
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			metadata.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			metadata.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	params.Add(metadata, "metadata")
	return params, nil
}

type ServerAssociateEipOptions struct {
	ServerIdOptions
	EIP string `help:"ID or name of EIP to associate"`
}

func (o *ServerAssociateEipOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.EIP), "eip")
	return params, nil
}

func (o *ServerAssociateEipOptions) Description() string {
	return "Associate a server and an eip"
}

type ServerDissociateEipOptions struct {
	ServerIdOptions
	AutoDelete bool `help:"automatically delete the dissociate EIP" json:"auto_delete,omitfalse"`
}

func (o *ServerDissociateEipOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *ServerDissociateEipOptions) Description() string {
	return "Dissociate an eip from a server"
}

type ServerRenewOptions struct {
	ID       string `help:"ID or name of server to renew"`
	DURATION string `help:"Duration of renew, ADMIN only command"`
}

func (o *ServerRenewOptions) GetId() string {
	return o.ID
}

func (o *ServerRenewOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(o.DURATION), "duration")
	return params, nil
}

type ServerPrepaidRecycleOptions struct {
	ServerIdOptions
	AutoDelete bool `help:"after joining the pool, remove the server automatically"`
}

func (o *ServerPrepaidRecycleOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if o.AutoDelete {
		params.Add(jsonutils.JSONTrue, "auto_delete")
	}
	return params, nil
}

type ServerIoThrottle struct {
	ServerIdOptions

	DiskBps  map[string]int `help:"disk bps of throttle, input diskId=BPS" json:"bps"`
	DiskIOPS map[string]int `help:"disk iops of throttle, input diskId=IOPS" json:"iops"`
}

func (o *ServerIoThrottle) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *ServerIoThrottle) Description() string {
	return "Guest io set throttle"
}

type ServerPublicipToEip struct {
	ServerIdOptions
	AutoStart bool `help:"Auto start new guest"`
}

func (o *ServerPublicipToEip) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("auto_start", jsonutils.NewBool(o.AutoStart))
	return params, nil
}

func (o *ServerPublicipToEip) Description() string {
	return "Convert PublicIp to Eip for server"
}

type ServerSetAutoRenew struct {
	ServerIdOptions
	AutoRenew bool   `help:"Set server auto renew or manual renew"`
	Duration  string `help:"Duration for renew" default:"1M"`
}

func (o *ServerSetAutoRenew) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("auto_renew", jsonutils.NewBool(o.AutoRenew))
	params.Set("duration", jsonutils.NewString(o.Duration))
	return params, nil
}

func (o *ServerSetAutoRenew) Description() string {
	return "Set autorenew for server"
}

type ServerSaveTemplateOptions struct {
	ServerIdOptions
	TemplateName string `help:"The name of guest template"`
}

func (o *ServerSaveTemplateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	dict.Set("name", jsonutils.NewString(o.TemplateName))
	return dict, nil
}

func (o *ServerSaveTemplateOptions) Description() string {
	return "Save Guest Template of this Server"
}

type ServerRemoteUpdateOptions struct {
	ServerIdOptions
	computeapi.ServerRemoteUpdateInput
}

type ServerCreateEipOptions struct {
	options.BaseIdOptions
	Bandwidth  int     `help:"EIP bandwidth in Mbps" default:"5"`
	BgpType    *string `help:"desired BGP type"`
	ChargeType *string `help:"bandwidth charge type" choices:"traffic|bandwidth"`
}

func (opts *ServerCreateEipOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ServerMakeSshableOptions struct {
	options.BaseIdOptions

	User       string `help:"ssh username for ssh connection" default:"root"`
	PrivateKey string `help:"ssh privatekey for ssh connection"`
	Password   string `help:"ssh password for ssh connection"`
	Port       int    `help:"ssh port for ssh connection"`
}

func (opts *ServerMakeSshableOptions) Params() (jsonutils.JSONObject, error) {
	if opts.User == "" {
		return nil, fmt.Errorf("ssh username must be set")
	}
	if opts.PrivateKey == "" && opts.Password == "" {
		return nil, fmt.Errorf("either --private-key or --password must be set")
	}
	return jsonutils.Marshal(opts), nil
}

type ServerSetSshportOptions struct {
	options.BaseIdOptions

	Port int `help:"ssh port" default:"22"`
}

func (opts *ServerSetSshportOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ServerHaveAgentOptions struct {
	options.BaseIdOptions
}

type ServerMigrateNetworkOptions struct {
	options.BaseIdOptions

	computeapi.ServerMigrateNetworkInput
}

func (opts *ServerMigrateNetworkOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type ServerStatusStatisticsOptions struct {
	ServerListOptions
	options.StatusStatisticsOptions
}

type ServerProjectStatisticsOptions struct {
	ServerListOptions
	options.ProjectStatisticsOptions
}

type ServerDomainStatisticsOptions struct {
	ServerListOptions
	options.DomainStatisticsOptions
}

type ServerChangeDiskStorageOptions struct {
	options.BaseIdOptions
	DISKID         string `json:"disk_id" help:"Disk id or name"`
	TARGETSTORAGE  string `json:"target_storage_id" help:"Target storage id or name"`
	KeepOriginDisk bool   `json:"keep_origin_disk" help:"Keep origin disk when changed"`
}

func (o *ServerChangeDiskStorageOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerChangeStorageOptions struct {
	options.BaseIdOptions
	TARGETSTORAGE  string `json:"target_storage_id" help:"Target storage id or name"`
	KeepOriginDisk bool   `json:"keep_origin_disk" help:"Keep origin disk when changed"`
}

func (o *ServerChangeStorageOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerCPUSetOptions struct {
	options.BaseIdOptions
	SETS string `help:"Cgroup cpusets CPUs spec string, e.g. '0-2,16'"`
}

func (o *ServerCPUSetOptions) Params() (jsonutils.JSONObject, error) {
	sets := cgrouputils.ParseCpusetStr(o.SETS)
	parts := strings.Split(sets, ",")
	if len(parts) == 0 {
		return nil, errors.New(fmt.Sprintf("Invalid cpu sets %q", o.SETS))
	}
	input := &computeapi.ServerCPUSetInput{
		CPUS: make([]int, 0),
	}
	for _, s := range parts {
		sd, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Not digit part %q", s))
		}
		input.CPUS = append(input.CPUS, sd)
	}
	return jsonutils.Marshal(input), nil
}

type ServerVncOptions struct {
	ServerIdOptions
	Origin bool
}

func (o *ServerVncOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerIsoOptions struct {
	ServerIdOptions
	Ordinal int `help:"server iso ordinal, default 0"`
}

func (o *ServerIsoOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerAddSubIpsOptions struct {
	ServerIdOptions

	computeapi.GuestAddSubIpsInput
}

func (o *ServerAddSubIpsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ServerSetOSInfoOptions struct {
	ServerIdsOptions

	computeapi.ServerSetOSInfoInput
}

func (o *ServerSetOSInfoOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
