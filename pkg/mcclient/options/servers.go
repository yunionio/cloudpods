package options

import (
	"fmt"
	"io/ioutil"
	"strings"

	"yunion.io/x/jsonutils"
)

type ServerListOptions struct {
	Zone          string `help:"Zone ID or Name"`
	Wire          string `help:"Wire ID or Name"`
	Network       string `help:"Network ID or Name"`
	Disk          string `help:"Disk ID or Name"`
	Host          string `help:"Host ID or Name"`
	Baremetal     *bool  `help:"Show baremetal servers"`
	Gpu           *bool  `help:"Show gpu servers"`
	Secgroup      string `help:"Secgroup ID or Name"`
	AdminSecgroup string `help:"AdminSecgroup ID or Name"`
	Hypervisor    string `help:"Show server of hypervisor" choices:"kvm|esxi|container|baremetal|aliyun|azure"`
	Manager       string `help:"Show servers imported from manager"`
	Region        string `help:"Show servers in cloudregion"`
	WithEip       *bool  `help:"Show Servers with EIP"`
	WithoutEip    *bool  `help:"Show Servers without EIP"`

	BaseListOptions
}

type ServerIdOptions struct {
	ID string `help:"ID or name of the server" json:"-"`
}

type ServerIdsOptions struct {
	ID []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
}

type ServerShowOptions struct {
	ID       string `help:"ID or name of the server" json:"-"`
	WithMeta *bool  `help:"With meta data"`
}

type ServerDeployInfo struct {
	Action  string
	Path    string
	Content string
}

func ParseServerDeployInfo(info string) (*ServerDeployInfo, error) {
	sdi := &ServerDeployInfo{}
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

func ParseServerDeployInfoList(list []string) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	for i, info := range list {
		ret, err := ParseServerDeployInfo(info)
		if err != nil {
			return nil, err
		}
		params.Set(fmt.Sprintf("deploy.%d.action", i), jsonutils.NewString(ret.Action))
		params.Set(fmt.Sprintf("deploy.%d.path", i), jsonutils.NewString(ret.Path))
		params.Set(fmt.Sprintf("deploy.%d.content", i), jsonutils.NewString(ret.Content))
	}
	return params, nil
}

type ServerCreateOptions struct {
	NAME             string   `help:"Name of server"`
	MEM              string   `help:"Memory size" metavar:"MEMORY" json:"vmem_size"`
	Disk             []string `help:"Disk descriptions" nargs:"+"`
	Net              []string `help:"Network descriptions" metavar:"NETWORK"`
	IsolatedDevice   []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
	Keypair          string   `help:"SSH Keypair"`
	Password         string   `help:"Default user password"`
	Iso              string   `help:"ISO image ID" metavar:"IMAGE_ID" json:"cdrom"`
	Ncpu             *int     `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count"`
	Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl"`
	Vdi              string   `help:"VDI protocool" choices:"vnc|spice"`
	Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
	Desc             string   `help:"Description" metavar:"<DESCRIPTION>" json:"description"`
	Boot             string   `help:"Boot device" metavar:"<BOOT_DEVICE>" choices:"disk|cdrom" json:"-"`
	NoAccountInit    *bool    `help:"Not reset account password"`
	AllowDelete      *bool    `help:"Unlock server to allow deleting" json:"-"`
	ShutdownBehavior string   `help:"Behavior after VM server shutdown, stop or terminate server" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate"`
	AutoStart        *bool    `help:"Auto start server after it is created"`
	Zone             string   `help:"Preferred zone where virtual server should be created" json:"prefer_zone"`
	Host             string   `help:"Preferred host where virtual server should be created" json:"prefer_host"`
	SchedTag         []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
	Deploy           []string `help:"Specify deploy files in virtual server file system" json:"-"`
	Group            []string `help:"Group of virtual server"`
	Project          string   `help:"'Owner project ID or Name" json:"tenant"`
	User             string   `help:"Owner user ID or Name"`
	System           *bool    `help:"Create a system VM, sysadmin ONLY option" json:"is_system"`
	Hypervisor       string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun|azure"`
	TaskNotify       *bool    `help:"Setup task notify" json:"-"`
	Count            *int     `help:"Create multiple simultaneously" default:"1" json:"-"`
	DryRun           *bool    `help:"Dry run to test scheduler" json:"-"`
	RaidConfig       []string `help:"Baremetal raid config" json:"-"`
}

func (opts *ServerCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}

	{
		deployParams, err := ParseServerDeployInfoList(opts.Deploy)
		if err != nil {
			return nil, err
		}
		params.Update(deployParams)
	}

	if len(opts.Boot) > 0 {
		if opts.Boot == "disk" {
			params.Set("boot_order", jsonutils.NewString("cdn"))
		} else {
			params.Set("boot_order", jsonutils.NewString("dcn"))
		}
	}

	if BoolV(opts.AllowDelete) {
		params.Set("disable_delete", jsonutils.JSONFalse)
	}

	if len(opts.RaidConfig) > 0 {
		if opts.Hypervisor != "baremetal" {
			return nil, fmt.Errorf("RaidConfig is applicable to baremetal ONLY")
		}
		for i, conf := range opts.RaidConfig {
			params.Set(fmt.Sprintf("baremetal_disk_config.%d", i), jsonutils.NewString(conf))
		}
	}

	if BoolV(opts.DryRun) {
		params.Set("suggestion", jsonutils.JSONTrue)
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
	ShutdownBehavior string   `help:"Behavior after VM server shutdown, stop or terminate server" choices:"stop|terminate"`
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
	OverridePendingDelete *bool    `help:"Delete server directly instead of pending delete"`
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

func (opts *ServerDeployOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	{
		deleteKeyPair := BoolV(opts.DeleteKeypair)
		if deleteKeyPair {
			params.Add(jsonutils.JSONTrue, "__delete_keypair__")
		} else if len(opts.Keypair) > 0 {
			params.Add(jsonutils.NewString(opts.Keypair), "keypair")
		}
	}
	{
		deployParams, err := ParseServerDeployInfoList(opts.Deploy)
		if err != nil {
			return nil, err
		}
		params.Update(deployParams)
	}
	return params, nil
}

type ServerSecGroupOptions struct {
	ID     string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	Secgrp string `help:"ID of Security Group" metavar:"Security Group" positional:"true"`
}

type ServerSendKeyOptions struct {
	ID   string `help:"ID or Name of server" metavar:"Guest" json:"-"`
	KEYS string `help:"Special keys to send, eg. ctrl, alt, f12, shift, etc, separated by \"-\""`
	Hold *uint  `help:"Hold key for specified milliseconds" json:"duration"`
}

type ServerMonitorOptions struct {
	ID  string `help:"ID or Name of server" json:"-"`
	CMD string `help:"Qemu Monitor command to send"`
}

type ServerSaveImageOptions struct {
	ID        string `help:"ID or name of server" json:"-"`
	IMAGE     string `help:"Image name" json:"name"`
	Public    *bool  `help:"Make the image public available" json:"is_public"`
	Format    string `help:"image format" choices:"vmdk|qcow2"`
	Notes     string `help:"Notes about the image"`
	AutoStart *bool  `help:"Auto start server after image saved"`
}

type ServerRebuildRootOptions struct {
	ID            string `help:"Server to rebuild root" json:"-"`
	Image         string `help:"New root Image template ID" json:"image_id"`
	Keypair       string `help:"ssh Keypair used for login"`
	Password      string `help:"Default user password"`
	NoAccountInit *bool  `help:"Not reset account password"`
	AutoStart     *bool  `help:"Auto start server after it is created"`
}

type ServerChangeConfigOptions struct {
	ID   string   `help:"Server to rebuild root" json:"-"`
	Ncpu *int     `help:"New number of Virtual CPU cores" json:"vcpu_count"`
	Vmem string   `help:"New memory size" json:"vmem_size"`
	Disk []string `help:"Data disk description, from the 1st data disk to the last one, empty string if no change for this data disk"`
}

type ServerResetOptions struct {
	ID   []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	Hard *bool    `help:"Hard reset or not; default soft" json:"is_hard"`
}

type ServerRestartOptions struct {
	ID      []string `help:"ID of servers to operate" metavar:"SERVER" json:"-"`
	IsForce *bool    `help:"Force reset or not; default false" json:"is_force"`
}
