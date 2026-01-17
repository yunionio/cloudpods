package llm

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

const (
	SERVICE_TYPE = "llm"
)

type LLMBaseListDetails struct {
	apis.VirtualResourceDetails

	// AccessInfo []AccessInfoListOutput
	Volume Volume `json:"volume"`

	LLMImage      string `json:"llm_image"`
	LLMImageLable string `json:"llm_image_lable"`
	LLMImageName  string `json:"llm_image_name"`

	VcpuCount  int      `json:"vcpu_count"`
	VmemSizeMb int      `json:"vmem_size_mb"`
	Devices    *Devices `json:"devices"`

	NetworkType string `json:"network_type"`
	NetworkId   string `json:"network_id"`
	Network     string `json:"network"`

	EffectBandwidthMbps int       `json:"effect_bandwidth_mbps"`
	StartTime           time.Time `json:"start_time"`

	LLMStatus string `json:"llm_status"`

	Server string `json:"server"`

	HostInfo

	Zone   string `json:"zone"`
	ZoneId string `json:"zone_id"`

	AdbPublic string `json:"adb_public"`
	AdbAccess string `json:"adb_access"`
}

type MountedModelInfo struct {
	FullName string `json:"fullname"` // 模型全名，如: qwen3:8b
	Id       string `json:"id"`       // 模型ID，如: 500a1f067a9f
}

type LLMListDetails struct {
	LLMBaseListDetails

	LLMSku string

	MountedModels []MountedModelInfo
}

type LLMBaseCreateInput struct {
	apis.VirtualResourceCreateInput

	PreferHost string `json:"prefer_host"`
	AutoStart  bool   `json:"auto_start"`

	NetworkType string `json:"network_type"`
	NetworkId   string `json:"network_id"`

	BandwidthMB   int  `json:"bandwidth_mb"`
	DebugMode     bool `json:"debug_mode"`
	RootfsUnlimit bool `json:"rootfs_unlimit"`
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

	NetworkType string `json:"network_type"`
	NetworkId   string `json:"network_id"`

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
	// 秒装模型 LLM 类型
	LlmType string `json:"llm_type"`
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
