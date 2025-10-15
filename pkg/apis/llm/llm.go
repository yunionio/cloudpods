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

type LLMListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	Host      string   `json:"host"`
	LLMModel  string   `json:"llm_model"`
	LLMImage  string   `json:"llm_image"`
	LLMStatus []string `json:"llm_status"`

	NoVolume   *bool  `json:"no_volume"`
	ListenPort int    `json:"listen_port"`
	PublicIp   string `json:"public_ip"`
	VolumeId   string `json:"volume_id"`
	Unused     *bool  `json:"unused"`
}
