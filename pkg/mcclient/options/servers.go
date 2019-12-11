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

package options

import (
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
)

type ServerListOptions struct {
	Zone               string `help:"Zone ID or Name"`
	Wire               string `help:"Wire ID or Name"`
	Network            string `help:"Network ID or Name"`
	Disk               string `help:"Disk ID or Name"`
	Host               string `help:"Host ID or Name"`
	Baremetal          *bool  `help:"Show baremetal servers"`
	Gpu                *bool  `help:"Show gpu servers"`
	Secgroup           string `help:"Secgroup ID or Name"`
	AdminSecgroup      string `help:"AdminSecgroup ID or Name"`
	Hypervisor         string `help:"Show server of hypervisor" choices:"kvm|esxi|container|baremetal|aliyun|azure|aws|huawei|ucloud|zstack|openstack|ctyun"`
	Region             string `help:"Show servers in cloudregion"`
	WithEip            *bool  `help:"Show Servers with EIP"`
	WithoutEip         *bool  `help:"Show Servers without EIP"`
	OsType             string `help:"OS Type" choices:"linux|windows|vmware"`
	OrderByDisk        string `help:"Order by disk size" choices:"asc|desc"`
	OrderByHost        string `help:"Order by host name" choices:"asc|desc"`
	Vpc                string `help:"Vpc id or name"`
	UsableServerForEip string `help:"Eip id or name"`
	WithoutUserMeta    *bool  `help:"Show Servers without user metadata"`
	Group              string `help:"Instance Group ID or Name"`

	ResourceType string `help:"Resource type" choices:"shared|prepaid|dedicated"`

	BillingType string `help:"billing type" choices:"postpaid|prepaid"`

	BaseListOptions
}

type ServerIdOptions struct {
	ID string `help:"ID or name of the server" json:"-"`
}

type ServerLoginInfoOptions struct {
	ID  string `help:"ID or name of the server" json:"-"`
	Key string `help:"File name of private key, if password is encrypted by key"`
}

type ServerIdsOptions struct {
	ID []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
}

type ServerDeleteBackupOptions struct {
	ID    string `help:"ID of the server" json:"-"`
	Purge *bool  `help:"Purge Guest Backup" json:"purge"`
}

type ServerSwitchToBackupOptions struct {
	ID           string `help:"ID of the server" json:"-"`
	PurgeBackup  *bool  `help:"Purge Guest Backup" json:"purge_backup"`
	DeleteBackup *bool  `help:"Delete Guest Backup" json:"delete_backup"`
}

type ServerShowOptions struct {
	ID       string `help:"ID or name of the server" json:"-"`
	WithMeta *bool  `help:"With meta data"`
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

type ServerConfigs struct {
	Region     string `help:"Preferred region where virtual server should be created" json:"prefer_region"`
	Zone       string `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Wire       string `help:"Preferred wire where virtual server should be created" json:"prefer_wire"`
	Host       string `help:"Preferred host where virtual server should be created" json:"prefer_host"`
	BackupHost string `help:"Perfered host where virtual backup server should be created"`

	Hypervisor   string `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun|azure|qcloud|aws|huawei|openstack|ucloud|zstack|ctyun"`
	ResourceType string `help:"Resource type" choices:"shared|prepaid|dedicated"`
	Backup       bool   `help:"Create server with backup server"`

	Schedtag       []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Disk           []string `help:"Disk descriptions" nargs:"+"`
	DiskSchedtag   []string `help:"Disk schedtag description, e.g. '0:<tag>:<strategy>'"`
	Net            []string `help:"Network descriptions" metavar:"NETWORK"`
	NetSchedtag    []string `help:"Network schedtag description, e.g. '0:<tag>:<strategy>'"`
	IsolatedDevice []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
	RaidConfig     []string `help:"Baremetal raid config" json:"-"`
	Project        string   `help:"'Owner project ID or Name" json:"tenant"`
	User           string   `help:"Owner user ID or Name"`
	Count          int      `help:"Create multiple simultaneously" default:"1"`
}

func (o ServerConfigs) Data() (*computeapi.ServerConfigs, error) {
	data := &computeapi.ServerConfigs{
		PreferRegion:     o.Region,
		PreferZone:       o.Zone,
		PreferWire:       o.Wire,
		PreferHost:       o.Host,
		PreferBackupHost: o.BackupHost,
		Hypervisor:       o.Hypervisor,
		ResourceType:     o.ResourceType,
		Backup:           o.Backup,
		Count:            o.Count,
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
	for _, tag := range o.Schedtag {
		schedtag, err := cmdline.ParseSchedtagConfig(tag)
		if err != nil {
			return nil, err
		}
		data.Schedtags = append(data.Schedtags, schedtag)
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

	MemSpec string `help:"Memory size Or Instance Type" metavar:"MEMSPEC" json:"-"`

	Keypair          string   `help:"SSH Keypair"`
	Password         string   `help:"Default user password"`
	Iso              string   `help:"ISO image ID" metavar:"IMAGE_ID" json:"cdrom"`
	VcpuCount        int      `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	InstanceType     string   `help:"instance flavor"`
	Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl"`
	Vdi              string   `help:"VDI protocool" choices:"vnc|spice"`
	Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
	Desc             string   `help:"Description" metavar:"<DESCRIPTION>" json:"description"`
	Boot             string   `help:"Boot device" metavar:"<BOOT_DEVICE>" choices:"disk|cdrom" json:"-"`
	EnableCloudInit  bool     `help:"Enable cloud-init service"`
	NoAccountInit    *bool    `help:"Not reset account password"`
	AllowDelete      *bool    `help:"Unlock server to allow deleting" json:"-"`
	ShutdownBehavior string   `help:"Behavior after VM server shutdown" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate"`
	AutoStart        bool     `help:"Auto start server after it is created"`
	Deploy           []string `help:"Specify deploy files in virtual server file system" json:"-"`
	Group            []string `help:"Group ID or Name of virtual server"`
	System           bool     `help:"Create a system VM, sysadmin ONLY option" json:"is_system"`
	TaskNotify       *bool    `help:"Setup task notify" json:"-"`
	DryRun           *bool    `help:"Dry run to test scheduler" json:"-"`
	UserDataFile     string   `help:"user_data file path" json:"-"`
	InstanceSnapshot string   `help:"instance snapshot" json:"instance_snapshot"`
	Secgroups        []string `help:"secgroups" json:"secgroups"`

	OsType string `help:"os type, e.g. Linux, Windows, etc."`

	Duration string `help:"valid duration of the server, e.g. 1H, 1D, 1W, 1M, 1Y, ADMIN ONLY option"`

	AutoPrepaidRecycle bool `help:"automatically enable prepaid recycling after server is created successfully" json:"auto_prepaid_recycle,omitfalse"`

	GenerateName bool `help:"name is generated by pattern" json:"-"`

	EipBw         int    `help:"allocate EIP with bandwidth in MB when server is created" json:"eip_bw,omitzero"`
	EipChargeType string `help:"newly allocated EIP charge type" choices:"traffic|bandwidth" json:"eip_charge_type,omitempty"`
	Eip           string `help:"associate with an existing EIP when server is created" json:"eip,omitempty"`

	GuestImageID string `help:"create from guest image, need to specify the guest image id"`
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
		Cdrom:              opts.Iso,
		Vga:                opts.Vga,
		Vdi:                opts.Vdi,
		Bios:               opts.Bios,
		ShutdownBehavior:   opts.ShutdownBehavior,
		AutoStart:          opts.AutoStart,
		Duration:           opts.Duration,
		AutoPrepaidRecycle: opts.AutoPrepaidRecycle,
		EipBw:              opts.EipBw,
		EipChargeType:      opts.EipChargeType,
		Eip:                opts.Eip,
		EnableCloudInit:    opts.EnableCloudInit,
		OsType:             opts.OsType,
		GuestImageID:       opts.GuestImageID,
		Secgroups:          opts.Secgroups,
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

	if BoolV(opts.AllowDelete) {
		disableDelete := false
		params.DisableDelete = &disableDelete
	}

	if BoolV(opts.DryRun) {
		params.Suggestion = true
	}

	// group
	params.InstanceGroupIds = opts.Group

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
	ID    []string `help:"ID or Name of server" json:"-"`
	Force *bool    `help:"Stop server forcefully" json:"is_force"`
}

type ServerUpdateOptions struct {
	ID               []string `help:"IDs or Names of servers to update" json:"-"`
	Name             string   `help:"New name to change"`
	Vmem             string   `help:"Memory size" json:"vmem_size"`
	Ncpu             *int     `help:"CPU count" json:"vcpu_count"`
	Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl"`
	Vdi              string   `help:"VDI protocol" choices:"vnc|spice"`
	Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
	Desc             string   `help:"Description" json:"description"`
	Boot             string   `help:"Boot device" choices:"disk|cdrom"`
	Delete           string   `help:"Lock server to prevent from deleting" choices:"enable|disable" json:"-"`
	ShutdownBehavior string   `help:"Behavior after VM server shutdown" choices:"stop|terminate"`
}

func (opts *ServerUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
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
	return params, nil
}

type ServerDeleteOptions struct {
	ID                    []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	OverridePendingDelete *bool    `help:"Delete server directly instead of pending delete" short-token:"f"`
	DeleteSnapshots       *bool    `help:"Delete server snapshots"`
}

type ServerDeployOptions struct {
	ID            string   `help:"ID or Name of server" json:"-"`
	Keypair       string   `help:"ssh Keypair used for login" json:"-"`
	DeleteKeypair *bool    `help:"Remove ssh Keypairs" json:"-"`
	Deploy        []string `help:"Specify deploy files in virtual server file system" json:"-"`
	ResetPassword *bool    `help:"Force reset password"`
	Password      string   `help:"Default user password"`
	AutoStart     *bool    `help:"Auto start server after deployed"`
}

func (opts *ServerDeployOptions) Params() (*computeapi.ServerDeployInput, error) {
	params := new(computeapi.ServerDeployInput)
	{
		if opts.DeleteKeypair != nil {
			params.DeleteKeypair = opts.DeleteKeypair
		} else if len(opts.Keypair) > 0 {
			params.Keypair = opts.Keypair
		}
		params.AutoStart = opts.AutoStart
		params.ResetPassword = opts.ResetPassword
		params.Password = opts.Password
	}
	{
		deployInfos, err := ParseServerDeployInfoList(opts.Deploy)
		if err != nil {
			return nil, err
		}
		params.DeployConfigs = deployInfos
	}
	return params, nil
}

type ServerSecGroupOptions struct {
	ID     string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	Secgrp string `help:"ID of Security Group" metavar:"Security Group" positional:"true"`
}

type ServerSecGroupsOptions struct {
	ID     string   `help:"ID or Name of server" metavar:"Guest" json:"-"`
	Secgrp []string `help:"Ids of Security Groups" metavar:"Security Groups" positional:"true"`
}

type ServerModifySrcCheckOptions struct {
	ID          string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	SrcIpCheck  string `help:"Turn on/off src ip check" choices:"on|off"`
	SrcMacCheck string `help:"Turn on/off src mac check" choices:"on|off"`
}

type ServerSendKeyOptions struct {
	ID   string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	KEYS string `help:"Special keys to send, eg. ctrl, alt, f12, shift, etc, separated by \"-\""`
	Hold *uint  `help:"Hold key for specified milliseconds" json:"duration"`
}

type ServerMonitorOptions struct {
	ID string `help:"ID or Name of server" json:"-"`

	COMMAND string `help:"Qemu Monitor command to send"`
	Admin   *bool  `help:"Is this an admin call?"`
}

type ServerSaveImageOptions struct {
	ID        string `help:"ID or name of server" json:"-"`
	IMAGE     string `help:"Image name" json:"name"`
	Public    *bool  `help:"Make the image public available" json:"is_public"`
	Format    string `help:"image format" choices:"vmdk|qcow2"`
	Notes     string `help:"Notes about the image"`
	AutoStart *bool  `help:"Auto start server after image saved"`
}

type ServerSaveGuestImageOptions struct {
	ID        string `help:"ID or name of server" json:"-"`
	IMAGE     string `help:"Image name" json:"name"`
	AutoStart *bool  `help:"Auto start server after image saved"`
}

type ServerRebuildRootOptions struct {
	ID            string `help:"Server to rebuild root" json:"-"`
	ImageId       string `help:"New root Image template ID" json:"image_id" token:"image"`
	Keypair       string `help:"ssh Keypair used for login"`
	Password      string `help:"Default user password"`
	NoAccountInit *bool  `help:"Not reset account password"`
	AutoStart     *bool  `help:"Auto start server after it is created"`
	AllDisks      *bool  `help:"Rebuild all disks including data disks"`
}

type ServerChangeConfigOptions struct {
	ID        string   `help:"Server to rebuild root" json:"-"`
	VcpuCount *int     `help:"New number of Virtual CPU cores" json:"vcpu_count" token:"ncpu"`
	VmemSize  string   `help:"New memory size" json:"vmem_size" token:"vmem"`
	Disk      []string `help:"Data disk description, from the 1st data disk to the last one, empty string if no change for this data disk"`

	InstanceType string `help:"Instance Type, e.g. S2.SMALL2 for qcloud"`
}

type ServerResetOptions struct {
	ID   []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	Hard *bool    `help:"Hard reset or not; default soft" json:"is_hard"`
}

type ServerRestartOptions struct {
	ID      []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	IsForce *bool    `help:"Force reset or not; default false" json:"is_force"`
}

type ServerMigrateOptions struct {
	ID         string `help:"ID of server" json:"-"`
	PreferHost string `help:"Server migration prefer host id or name" json:"prefer_host"`
	AutoStart  *bool  `help:"Server auto start after migrate" json:"auto_start"`
	RescueMode *bool  `help:"Migrate server in rescue mode,
					  all disk must store in shared storage;
					  default false" json:"rescue_mode"`
}

type ServerLiveMigrateOptions struct {
	ID         string `help:"ID of server" json:"-"`
	PreferHost string `help:"Server migration prefer host id or name" json:"prefer_host"`
}

type ServerMetadataOptions struct {
	ID   string   `help:"ID or name of server" json:"-"`
	TAGS []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
}

func (opts *ServerMetadataOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	for _, tag := range opts.TAGS {
		info := strings.Split(tag, "=")
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			params.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			params.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	return params, nil
}

type ServerBatchMetadataOptions struct {
	Guests []string `help:"IDs or names of server" json:"-"`
	TAGS   []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
}

func (opts *ServerBatchMetadataOptions) Params() (*jsonutils.JSONDict, error) {
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
