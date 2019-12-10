package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type GuesttemplateCreateInput struct {
	apis.SharableVirutalResourceCreateInput
	// description: the content of guest template
	// required: true
	Content jsonutils.JSONObject `json:"content"`
}

type GuesttemplateDetails struct {
	apis.Meta
	apis.SharableVirtualResourceDetails
	SGuestTemplate

	ConfigInfo GuesttemplateConfigInfo `json:"config_info"`
}

type GuesttemplateConfigInfo struct {
	Region               string                 `json:"region"`
	Zone                 string                 `json:"zone"`
	Hypervisor           string                 `json:"hypervisor"`
	OsType               string                 `json:"os_type"`
	Sku                  GuesttemplateSku       `json:"sku"`
	Disks                []GuesttemplateDisk    `json:"disks"`
	Keypair              string                 `json:"keypair"`
	Nets                 []GuesttemplateNetwork `json:"nets"`
	Secgroup             string                 `json:"secgroup"`
	IsolatedDeviceConfig []IsolatedDeviceConfig `json:"isolated_device_config"`
	Image                string                 `json:"image"`
}

type GuesttemplateDisk struct {
	Backend  string `json:"backend"`
	DiskType string `json:"disk_type"`
	Index    int    `json:"index"`
	SizeMb   int    `json:"size_mb"`
}

type GuesttemplateNetwork struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	GuestIpStart string `json:"guest_ip_start"`
	GuestIpEnd   string `json:"guest_up_end"`
	VlanId       int    `json:"vlan_id"`
}

type GuesttemplateSku struct {
	Name                 string `json:"name"`
	CpuCoreCount         int    `json:"cpu_core_count"`
	MemorySizeMb         int    `json:"memory_size_mb"`
	InstanceTypeCategory string `json:"instance_type_category`
	InstanceTypeFamily   string `json:"instance_type_family"`
}

type GuesttemplatePublicInput struct {
	apis.Meta

	// description: the scope about public operator
	// required: true
	// example: system
	Scope string `json:"scope"`
}
