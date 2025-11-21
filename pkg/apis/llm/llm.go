package llm

import (
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

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

	LLMSkuId   string
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

	LLMSku   string `json:"llm_sku"`
	LLMImage string `json:"llm_image"`
}

type ModelInfo struct {
	// 秒装模型ID
	Id string `json:"id"`
	// 秒装模型 ModelId
	ModelId string `json:"model_id"`
	// 秒装模型展示的名称，如: Qwen-7B
	DisplayName string `json:"display_name"`
	// 秒装模型 tag，如: 7b
	Tag string `json:"tag"`
}

type LLMPerformQuickModelsInput struct {
	Models []ModelInfo
	Method TQuickModelMethod
}

type LLMBatchPerformOutput struct {
	Data []LLMPerformOutput
	Task *taskman.STask
}

type LLMPerformOutput struct {
	Id            string
	Name          string
	RequestStatus int
	Msg           string
	TaskId        string
}

type LLMSyncModelTaskInput struct {
	LLMPerformQuickModelsInput

	LLMStatus         string   `json:"llm_status"`
	InstallModelIds   []string `json:"install_model_ids"`
	InstallDirs       []string `json:"install_dirs"`
	UninstallModelIds []string `json:"uninstall_model_ids"`
}
