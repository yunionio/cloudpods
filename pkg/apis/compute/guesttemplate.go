package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
)

type GuesttemplateCreateInput struct {
	apis.Meta

	// description: guest template name
	// unique: true
	// required: true
	// example: hello
	Name string `json:"name"`

	// description: the content of guest template
	// required: true
	Content jsonutils.JSONObject
}

type GuesttemplateDetails struct {
	apis.Meta
	apis.SharableVirtualResourceDetails
	SGuestTemplate

	Config GuesttemplateConfigInfo
}

type GuesttemplateConfigInfo struct {

	Region string
	Zone string
	Hypervisor string
	OsType string
	Sku string
	Disks []GuesttemplateDisk
	Keypair string
	Nets []string
	Secgroup string
	IsolatedDeviceConfig  []IsolatedDeviceConfig
	Image string
}

type GuesttemplateDisk struct {
	Backend string
	DiskType string
	Index int
	SizeMb int
}
