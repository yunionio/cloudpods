package desc

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SGuestPorjectDesc struct {
	Tenant        string
	TenantId      string
	DomainId      string
	ProjectDomain string
}

type SGuestRegionDesc struct {
	Zone     string
	Domain   string
	HostId   string
	Hostname string
}

type SGuestControlDesc struct {
	IsDaemon bool
	IsMaster bool
	IsSlave  bool

	ScalingGroupId     string
	SecurityRules      string
	AdminSecurityRules string

	EncryptKeyId string
}

type SGuestHardwareDesc struct {
	Cpu       int64
	Mem       int64
	Machine   string
	Bios      string
	Vga       string
	Vdi       string
	BootOrder string

	Cdrom           *api.GuestcdromJsonDesc
	Disks           []*api.GuestdiskJsonDesc
	Nics            []*api.GuestnetworkJsonDesc
	NicsStandby     []*api.GuestnetworkJsonDesc
	IsolatedDevices []*api.IsolatedDeviceJsonDesc
}

type SGuestDesc struct {
	SGuestPorjectDesc
	SGuestRegionDesc
	SGuestControlDesc
	SGuestHardwareDesc

	Name         string
	Uuid         string
	OsName       string
	Pubkey       string
	Keypair      string
	Secgroup     string
	Flavor       string
	UserData     string
	Metadata     map[string]string
	ExtraOptions map[string]jsonutils.JSONObject
}
