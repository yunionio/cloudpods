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

	PreferHost    string `json:"prefer_host"`
	AutoStart     bool   `json:"auto_start"`
	BandwidthMB   int    `json:"bandwidth_mb"`
	DebugMode     bool   `json:"debug_mode"`
	RootfsUnlimit bool   `json:"rootfs_unlimit"`
}

type LLMCreateInput struct {
	LLMBaseCreateInput

	LLMSkuId   string `json:"llm_sku_id"`
	LLMImageId string `json:"llm_image_id"`
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
	Models []ModelInfo       `json:"models"`
	Method TQuickModelMethod `json:"method"`
}

type LLMBatchPerformOutput struct {
	Data []LLMPerformOutput `json:"data"`
	Task *taskman.STask     `json:"task"`
}

type LLMPerformOutput struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	RequestStatus int    `json:"request_status"`
	Msg           string `json:"msg"`
	TaskId        string `json:"task_id"`
}

type LLMSyncModelTaskInput struct {
	LLMPerformQuickModelsInput

	LLMStatus         string   `json:"llm_status"`
	InstallModelIds   []string `json:"install_model_ids"`
	InstallDirs       []string `json:"install_dirs"`
	UninstallModelIds []string `json:"uninstall_model_ids"`
}

type LLMMountDirInfo struct {
	ImageId   string `json:"image_id"`
	Host      string `json:"host"`
	Container string `json:"container"`
}

func (info LLMMountDirInfo) ToOverlay() apis.ContainerVolumeMountDiskPostOverlay {
	// uid := int64(1000)
	// gid := int64(1000)
	if len(info.ImageId) > 0 {
		return apis.ContainerVolumeMountDiskPostOverlay{
			Image: &apis.ContainerVolumeMountDiskPostImageOverlay{
				Id: info.ImageId,
			},
			// FsUser:  &uid,
			// FsGroup: &gid,
		}
	}
	return apis.ContainerVolumeMountDiskPostOverlay{
		ContainerTargetDir: info.Container,
		HostLowerDir:       []string{info.Host},
		// FsUser:             &uid,
		// FsGroup:            &gid,
	}
}

type LLMSyncStatusInput struct {
}
