package llm

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
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
	ModelId  string `json:"model_id"` // 模型ID，如: 500a1f067a9f
	Id       string `json:"id"`       // 秒装包的 ID 主键
}

type LLMListDetails struct {
	LLMBaseListDetails

	LLMSku  string `json:"llm_sku"`
	LLMType string `json:"llm_type"`

	MountedModels []MountedModelInfo `json:"mounted_models"`
}

type LLMBaseCreateInput struct {
	apis.VirtualResourceCreateInput

	PreferHost string `json:"prefer_host"`
	AutoStart  bool   `json:"auto_start"`

	Nets []*computeapi.NetworkConfig `json:"nets"`

	BandwidthMB   int  `json:"bandwidth_mb"`
	DebugMode     bool `json:"debug_mode"`
	RootfsUnlimit bool `json:"rootfs_unlimit"`
}

type LLMCreateInput struct {
	LLMBaseCreateInput

	LLMSkuId   string   `json:"llm_sku_id"`
	LLMImageId string   `json:"llm_image_id"`
	LLMSpec    *LLMSpec `json:"llm_spec,omitempty"`
}

// LLMUpdateInput is the request body for updating an LLM (including llm_spec overrides).
type LLMUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	InstantModelQuotaGb *int     `json:"instant_model_quota_gb,omitempty"`
	LLMSpec             *LLMSpec `json:"llm_spec,omitempty"`
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

	LLMSku   string   `json:"llm_sku"`
	LLMImage string   `json:"llm_image"`
	LLMTypes []string `json:"llm_types"` // filter by linked SKU's llm_types (e.g. [dify, openclaw])
	LLMType  string   `json:"llm_type"`  // filter by linked SKU's llm_type (e.g. dify)
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

type LLMMountDirInfo struct {
	ImageId   string
	Host      string
	Container string
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

type LLMRestartInput struct {
}

type LLMRestartTaskInput struct {
	LLMId         string
	ResetDataDisk bool
	LLMStatus     string
	SkuId         string
	ImageId       string
	BackupName    string
	Property      []string

	RebindVolumeId string
	OnlyStop       bool
}

type LLMChangeNetworkInput struct {
	BandwidthMb   int      `json:"bandwidth_mb"`
	WhitePrefxies []string `json:"white_prefxies"`
}

type LLMVolumeInput struct {
	LLMId     string `json:"llm_id"`
	VolumeId  string `json:"volume_id"`
	AutoStart bool   `json:"auto_start"`
}

type LLMAccessUrlInfo struct {
	LoginUrl    string `json:"login_url"`
	PublicUrl   string `json:"public_url"`
	InternalUrl string `json:"internal_url"`
}

// LLMAccessInfo is the response for GET /llms/<id>/login-info: login URL and credentials.
type LLMAccessInfo struct {
	LLMAccessUrlInfo

	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type LLMProviderModelsInput struct {
	URL          string        `json:"url"`
	ProviderType LLMClientType `json:"provider_type"`
}

type LLMProviderModelsOutput struct {
	ProviderType LLMClientType `json:"provider_type"`
	URL          string        `json:"url"`
	Models       []string      `json:"models"`
}
