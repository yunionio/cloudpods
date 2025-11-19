package llm

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type InstantAppListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput

	ModelName string `json:"model_name"`
	Tag       string `json:"tag"`
	ModelId   string `json:"model_id"`
	Image     string `json:"image"`

	Mounts string `json:"mounts"`

	AutoCache *bool `json:"auto_cache"`
}

type InstantAppCreateInput struct {
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

type InstantAppUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ImageId string `json:"image_id"`
	Size    int64  `json:"size"`

	ActualSizeMb int32 `json:"actual_size_mb"`

	Mounts []string `json:"mounts"`
}

type InstantAppDetails struct {
	apis.SharableVirtualResourceDetails

	Image string

	CacheCount  int
	CachedCount int

	IconBase64 string `json:"icon_base64"`
}

type InstantAppImportInput struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	SignVer   string `json:"sign_ver"`
}

func (input InstantAppImportInput) Invalid() bool {
	if len(input.Endpoint) == 0 || len(input.AccessKey) == 0 || len(input.SecretKey) == 0 || len(input.Bucket) == 0 || len(input.Key) == 0 {
		return true
	}
	return false
}

type InstantAppSyncstatusInput struct {
}

type InstantAppCacheInput struct {
}

type InstantAppEnableAutoCacheInput struct {
	AutoCache bool `json:"auto_cache"`
}

type MountedAppResourceListInput struct {
	MountedApps []string `json:"mounted_apps"`
}

type MountedAppResourceCreateInput struct {
	MountedApps []string `json:"mounted_apps"`
}

type MountedAppResourceUpdateInput struct {
	MountedApps []string `json:"mounted_apps"`
}
