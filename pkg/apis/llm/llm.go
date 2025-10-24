package llm

import "yunion.io/x/onecloud/pkg/apis"

const (
	SERVICE_TYPE = "llm"
)

type LLMBaseCreateInput struct {
	apis.VirtualResourceCreateInput

	PreferHost    string
	AutoStart     bool
	BandwidthMB   int  `json:"bandwidth_mb"`
	DebugMode     bool `json:"debug_mode"`
	RootfsUnlimit bool `json:"rootfs_unlimit"`
}

type LLMCreateInput struct {
	LLMBaseCreateInput

	LLMModelId string
	LLMImageId string
}

type LLMBaseListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	Host   string   `json:"host"`
	Status []string `json:"status"`

	NoVolume   *bool  `json:"no_volume"`
	ListenPort int    `json:"listen_port"`
	PublicIp   string `json:"public_ip"`
	VolumeId   string `json:"volume_id"`
	Unused     *bool  `json:"unused"`
}

type LLMListInput struct {
	LLMBaseListInput

	LLMModel string `json:"llm_model"`
	LLMImage string `json:"llm_image"`
}
