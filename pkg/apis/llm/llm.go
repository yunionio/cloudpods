package llm

import "yunion.io/x/onecloud/pkg/apis"

const (
	SERVICE_TYPE = "llm"
)

type LLMCreateInput struct {
	apis.VirtualResourceCreateInput

	LLMModelId    string
	LLMImageId    string
	PreferHost    string
	AutoStart     bool
	BandwidthMB   int  `json:"bandwidth_mb"`
	DebugMode     bool `json:"debug_mode"`
	RootfsUnlimit bool `json:"rootfs_unlimit"`
}
