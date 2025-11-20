package llm

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type InstantModelListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput

	ModelName string `json:"model_name"`
	Tag       string `json:"tag"`
	ModelId   string `json:"model_id"`
	Image     string `json:"image"`

	Mounts string `json:"mounts"`

	AutoCache *bool `json:"auto_cache"`
}

type InstantModelCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	LLMType   LLMContainerType `json:"llm_type"`
	ModelName string           `json:"model_name"`
	Tag       string           `json:"tag"`
	ImageId   string           `json:"image_id"`
	Size      int64            `json:"size"`
	ModelId   string           `json:"model_id"`

	ActualSizeMb int32 `json:"actual_size_mb"`

	Mounts []string `json:"mounts"`
}

type InstantModelUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ImageId string `json:"image_id"`
	Size    int64  `json:"size"`

	ActualSizeMb int32 `json:"actual_size_mb"`

	Mounts []string `json:"mounts"`
}

type InstantModelDetails struct {
	apis.SharableVirtualResourceDetails

	Image string

	CacheCount  int
	CachedCount int

	IconBase64 string `json:"icon_base64"`
}

type InstantModelImportInput struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	SignVer   string `json:"sign_ver"`
}

func (input InstantModelImportInput) Invalid() bool {
	if len(input.Endpoint) == 0 || len(input.AccessKey) == 0 || len(input.SecretKey) == 0 || len(input.Bucket) == 0 || len(input.Key) == 0 {
		return true
	}
	return false
}

type InstantModelSyncstatusInput struct {
}

type InstantAppCacheInput struct {
}

type InstantModelEnableAutoCacheInput struct {
	AutoCache bool `json:"auto_cache"`
}

type MountedModelResourceListInput struct {
	MountedModels []string `json:"mounted_models"`
}

type MountedModelResourceCreateInput struct {
	MountedModels []string `json:"mounted_models"`
}

type MountedModelResourceUpdateInput struct {
	MountedModels []string `json:"mounted_models"`
}
