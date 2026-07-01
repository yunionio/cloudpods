package llm

import (
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	InstantModelSourceHuggingFace = "huggingface"
	InstantModelSourceModelScope  = "model_scope"
)

type InstantModelListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput

	ModelName string `json:"model_name"`
	ModelTag  string `json:"model_tag"`
	ModelId   string `json:"model_id"`
	LlmType   string `json:"llm_type"`
	Image     string `json:"image"`

	Mounts string `json:"mounts"`

	AutoCache *bool `json:"auto_cache"`
}

type InstantModelImportInput struct {
	ModelName string           `json:"model_name"`
	ModelTag  string           `json:"model_tag"`
	LlmType   LLMContainerType `json:"llm_type"`
	Source    string           `json:"source,omitempty"`
	RepoId    string           `json:"repo_id,omitempty"`
	Revision  string           `json:"revision,omitempty"`
	FilePath  string           `json:"file_path,omitempty"` // model_scope file glob; empty means full snapshot
}

type InstantModelCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	LlmType   LLMContainerType `json:"llm_type"`
	ModelName string           `json:"model_name"`
	ModelTag  string           `json:"model_tag"`
	Source    string           `json:"source,omitempty"`
	RepoId    string           `json:"repo_id,omitempty"`
	Revision  string           `json:"revision,omitempty"`
	ImageId   string           `json:"image_id"`
	Size      int64            `json:"size"`
	ModelId   string           `json:"model_id"`

	ActualSizeMb int32 `json:"actual_size_mb"`

	Mounts []string `json:"mounts"`

	DoNotImport *bool `json:"do_not_import,omitempty"`
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

	MountedByLLMs []MountedByLLMInfo `json:"mounted_by_llms"`

	GPUMemoryRequired int64 `json:"gpu_memory_required"`
}

type MountedByLLMInfo struct {
	LlmId   string `json:"llm_id"`
	LlmName string `json:"llm_name"`
}

type InstantModelSyncstatusInput struct {
}

// InstantModelVramRequirement reports the heuristic VRAM needed to run this
// model, mirroring GPUStack's `estimate_model_vram`. Returns
// `vram_required_mb=0` when `weight_size_bytes` is unknown — callers should
// treat 0 as "no constraint", same as the scheduler does.
type InstantModelVramRequirement struct {
	LlmType         string `json:"llm_type"`
	WeightSizeBytes int64  `json:"weight_size_bytes"`
	VramRequiredMb  int    `json:"vram_required_mb"`
}

type InstantModelBackfillVramInput struct {
	// DryRun reports what would change but does not write to the database.
	DryRun bool `json:"dry_run,omitempty"`
}

type InstantModelBackfillVramItem struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	ModelName       string `json:"model_name"`
	ModelTag        string `json:"model_tag"`
	WeightSizeBytes int64  `json:"weight_size_bytes"`
	Status          string `json:"status"` // updated, skipped, failed
	Reason          string `json:"reason,omitempty"`
}

type InstantModelBackfillVramOutput struct {
	DryRun  bool                           `json:"dry_run"`
	Scanned int                            `json:"scanned"`
	Updated int                            `json:"updated"`
	Skipped int                            `json:"skipped"`
	Failed  int                            `json:"failed"`
	Items   []InstantModelBackfillVramItem `json:"items,omitempty"`
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
