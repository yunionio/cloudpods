package llm

import (
	"strings"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

// Backend types for LLMModel
const (
	LLM_MODEL_BACKEND_VLLM   = "vllm"
	LLM_MODEL_BACKEND_OLLAMA = "ollama"
	LLM_MODEL_BACKEND_SGLANG = "sglang"
	LLM_MODEL_BACKEND_CUSTOM = "custom"
)

var LLM_MODEL_BACKENDS = sets.NewString(
	LLM_MODEL_BACKEND_VLLM,
	LLM_MODEL_BACKEND_OLLAMA,
	LLM_MODEL_BACKEND_SGLANG,
	LLM_MODEL_BACKEND_CUSTOM,
)

func IsLLMModelBackend(t string) bool {
	return LLM_MODEL_BACKENDS.Has(strings.ToLower(t))
}

// Placement strategies
const (
	LLM_MODEL_PLACEMENT_SPREAD  = "spread"
	LLM_MODEL_PLACEMENT_BINPACK = "binpack"
)

var LLM_MODEL_PLACEMENT_STRATEGIES = sets.NewString(
	LLM_MODEL_PLACEMENT_SPREAD,
	LLM_MODEL_PLACEMENT_BINPACK,
)

// Access policies
const (
	LLM_MODEL_ACCESS_PUBLIC        = "public"
	LLM_MODEL_ACCESS_AUTHED        = "authed"
	LLM_MODEL_ACCESS_ALLOWED_USERS = "allowed_users"
)

var LLM_MODEL_ACCESS_POLICIES = sets.NewString(
	LLM_MODEL_ACCESS_PUBLIC,
	LLM_MODEL_ACCESS_AUTHED,
	LLM_MODEL_ACCESS_ALLOWED_USERS,
)

// Model categories
const (
	LLM_MODEL_CATEGORY_LLM            = "llm"
	LLM_MODEL_CATEGORY_EMBEDDING      = "embedding"
	LLM_MODEL_CATEGORY_IMAGE          = "image"
	LLM_MODEL_CATEGORY_RERANKER       = "reranker"
	LLM_MODEL_CATEGORY_SPEECH_TO_TEXT = "speech_to_text"
	LLM_MODEL_CATEGORY_TEXT_TO_SPEECH = "text_to_speech"
)

var LLM_MODEL_CATEGORIES = sets.NewString(
	LLM_MODEL_CATEGORY_LLM,
	LLM_MODEL_CATEGORY_EMBEDDING,
	LLM_MODEL_CATEGORY_IMAGE,
	LLM_MODEL_CATEGORY_RERANKER,
	LLM_MODEL_CATEGORY_SPEECH_TO_TEXT,
	LLM_MODEL_CATEGORY_TEXT_TO_SPEECH,
)

// LLMDeploymentNets is a named slice type so it can be persisted on the
// SLLMDeployment row as a JSON column via sqlchemy.
type LLMDeploymentNets []*computeapi.NetworkConfig

// Intermediate deployment status values + their failure counterparts
const (
	LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL     = "importing_model"
	LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED = "import_model_failed"
	LLM_DEPLOYMENT_STATUS_CREATING_SKU        = "creating_sku"
	LLM_DEPLOYMENT_STATUS_CREATE_SKU_FAILED   = "create_sku_failed"
	// Instances created, waiting for them to come up to running.
	LLM_DEPLOYMENT_STATUS_DEPLOYING = "deploying"
	// Some replicas running but not all (e.g., one died, scale-up in progress).
	LLM_DEPLOYMENT_STATUS_PARTIAL = "partial"
	// Replica reconcile or syncstatus in progress.
	LLM_DEPLOYMENT_STATUS_SYNCING = "syncing"
)

// AiproxySyncStatus values stored on SLLMDeployment.AiproxySyncStatus.
const (
	AIPROXY_SYNC_STATUS_DISABLED = "disabled"
	AIPROXY_SYNC_STATUS_PENDING  = "pending"
	AIPROXY_SYNC_STATUS_SYNCING  = "syncing"
	AIPROXY_SYNC_STATUS_SYNCED   = "synced"
	AIPROXY_SYNC_STATUS_PARTIAL  = "partial"
	AIPROXY_SYNC_STATUS_FAILED   = "failed"
)

// LLMDeploymentCreateInput is the input for creating a new LLMDeployment deployment.
//
// Three modes are supported (mutually exclusive on LLMSkuId vs SkuSpec):
//
//	Mode A: LLMSkuId is set        → reuse existing SKU (ignore SkuSpec/ModelSpec)
//	Mode B: SkuSpec is set         → auto-create SKU (reuse existing mounted models)
//	Mode C: SkuSpec + ModelSpec    → auto-create InstantModel first, then SKU
type LLMDeploymentCreateInput struct {
	apis.VirtualResourceCreateInput

	// Mode A: reference existing SKU
	LLMSkuId string `json:"llm_sku_id,omitempty"`

	// Mode B: spec to auto-create SKU
	SkuSpec *LLMSkuCreateInput `json:"sku_spec,omitempty"`

	// Mode C: spec to auto-create InstantModel before SKU
	// (only meaningful when SkuSpec is set)
	ModelSpec *InstantModelImportInput `json:"model_spec,omitempty"`

	// Network config for instances (required)
	Nets []*computeapi.NetworkConfig `json:"nets"`
	// Auto start instances after creation
	AutoStart bool `json:"auto_start"`
	// Prefer specific host
	PreferHost string `json:"prefer_host"`
	// Prefer specific hosts for local_path scheduling (round-robin per replica).
	PreferHosts []string `json:"prefer_hosts,omitempty"`
	// Host path mounts for instances.
	HostPaths *HostPaths `json:"host_paths,omitempty"`

	// Expected number of replicas (SLLM instances)
	Replicas int `json:"replicas"`
	// Placement strategy: spread or binpack
	PlacementStrategy string `json:"placement_strategy"`
	// Allow CPU offloading for partial GPU models
	CpuOffloading *bool `json:"cpu_offloading"`
	// Allow distributed inference across multiple hosts
	DistributedInference *bool `json:"distributed_inference"`
	// Manual GPU selection (JSON)
	GpuSelector *GpuSelector `json:"gpu_selector"`
	// Explicit GPU memory utilization fraction for inference backend.
	GpuMemoryUtilization *float64 `json:"gpu_memory_utilization,omitempty"`
	// Calculate GPU memory utilization from mounted model VRAM and GPU memory
	// (default true for supported backends; pass false to disable).
	AutoGpuMemoryUtilization *bool `json:"auto_gpu_memory_utilization,omitempty"`
	// Host label selector for scheduling (JSON)
	WorkerSelector map[string]string `json:"worker_selector"`
	// Restart instance on error
	RestartOnError *bool `json:"restart_on_error"`
	// Extended KV cache config (JSON)
	ExtendedKVCache *ExtendedKVCacheConfig `json:"extended_kv_cache"`
	// Speculative decoding config (JSON)
	SpeculativeConfig *SpeculativeDecodingConfig `json:"speculative_config"`
	// Access policy: public, authed, allowed_users
	AccessPolicy string `json:"access_policy"`

	// AutoRegisterAiproxy registers running replicas with aiproxy catalog when true (default true; pass false to disable).
	AutoRegisterAiproxy *bool `json:"auto_register_aiproxy"`
	// AiproxyModelPrefix is deprecated and no longer affects client model alias.
	AiproxyModelPrefix string `json:"aiproxy_model_prefix"`
}

// LLMDeploymentUpdateInput is the input for updating an existing LLMModel.
type LLMDeploymentUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	Replicas                 *int                       `json:"replicas,omitempty"`
	PlacementStrategy        *string                    `json:"placement_strategy,omitempty"`
	CpuOffloading            *bool                      `json:"cpu_offloading,omitempty"`
	DistributedInference     *bool                      `json:"distributed_inference,omitempty"`
	GpuSelector              *GpuSelector               `json:"gpu_selector,omitempty"`
	GpuMemoryUtilization     *float64                   `json:"gpu_memory_utilization,omitempty"`
	AutoGpuMemoryUtilization *bool                      `json:"auto_gpu_memory_utilization,omitempty"`
	WorkerSelector           *map[string]string         `json:"worker_selector,omitempty"`
	RestartOnError           *bool                      `json:"restart_on_error,omitempty"`
	ExtendedKVCache          *ExtendedKVCacheConfig     `json:"extended_kv_cache,omitempty"`
	SpeculativeConfig        *SpeculativeDecodingConfig `json:"speculative_config,omitempty"`
	AccessPolicy             *string                    `json:"access_policy,omitempty"`
	AutoRegisterAiproxy      *bool                      `json:"auto_register_aiproxy,omitempty"`
	AiproxyModelPrefix       *string                    `json:"aiproxy_model_prefix,omitempty"`
}

type LLMDeploymentRestartInput struct {
}

type LLMDeploymentSyncstatusInput struct {
}

// Model source types
const (
	LLM_MODEL_SOURCE_HUGGINGFACE = "huggingface"
	LLM_MODEL_SOURCE_MODEL_SCOPE = "model_scope"
	LLM_MODEL_SOURCE_LOCAL_PATH  = "local_path"
)

var LLM_MODEL_SOURCES = sets.NewString(
	LLM_MODEL_SOURCE_HUGGINGFACE,
	LLM_MODEL_SOURCE_MODEL_SCOPE,
	LLM_MODEL_SOURCE_LOCAL_PATH,
)

// LLMDeploymentListInput is the input for listing LLMDeployments.
type LLMDeploymentListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	PlacementStrategy string `json:"placement_strategy"`
	AccessPolicy      string `json:"access_policy"`
	LLMSku            string `json:"llm_sku"` // filter by SKU id or name
}

// LLMDeploymentDetails is the output for LLMDeployment list/show responses.
type LLMDeploymentDetails struct {
	apis.VirtualResourceDetails

	Source                   string   `json:"source"`
	HuggingfaceRepoId        string   `json:"huggingface_repo_id"`
	HuggingfaceFilename      string   `json:"huggingface_filename"`
	ModelScopeModelId        string   `json:"model_scope_model_id"`
	ModelScopeFilePath       string   `json:"model_scope_file_path"`
	LocalPath                string   `json:"local_path"`
	PreferHosts              []string `json:"prefer_hosts,omitempty"`
	Categories               []string `json:"categories,omitempty"`
	Backend                  string   `json:"backend"`
	BackendVersion           string   `json:"backend_version"`
	Replicas                 int      `json:"replicas"`
	ReadyReplicas            int      `json:"ready_replicas"`
	PlacementStrategy        string   `json:"placement_strategy"`
	CpuOffloading            *bool    `json:"cpu_offloading,omitempty"`
	DistributedInference     *bool    `json:"distributed_inference,omitempty"`
	GpuMemoryUtilization     *float64 `json:"gpu_memory_utilization,omitempty"`
	AutoGpuMemoryUtilization *bool    `json:"auto_gpu_memory_utilization,omitempty"`
	RestartOnError           *bool    `json:"restart_on_error,omitempty"`
	AccessPolicy             string   `json:"access_policy"`

	// Computed: count of running SLLM instances
	RunningInstances int `json:"running_instances"`

	LLMSkuId string `json:"llm_sku_id"`
	LLMSku   string `json:"llm_sku"`

	AutoRegisterAiproxy bool   `json:"auto_register_aiproxy"`
	AiproxyModelPrefix  string `json:"aiproxy_model_prefix"`
	AiproxyRoutingId    string `json:"aiproxy_routing_id"`
	AiproxySyncStatus   string `json:"aiproxy_sync_status"`
}

// Per-replica aiproxy binding sync status (AiproxyInstanceBinding.sync_status).
const (
	AIPROXY_BINDING_SYNC_PENDING = "pending"
	AIPROXY_BINDING_SYNC_SYNCED  = "synced"
	AIPROXY_BINDING_SYNC_FAILED  = "failed"
)

// AiproxyInstanceBinding records one llm replica registered in aiproxy.
type AiproxyInstanceBinding struct {
	LlmId            string `json:"llm_id"`
	ClientModelAlias string `json:"client_model_alias"`
	AiProviderId     string `json:"ai_provider_id"`
	AiProviderName   string `json:"ai_provider_name"`
	BaseURL          string `json:"base_url"`
	SyncStatus       string `json:"sync_status"`
	LastError        string `json:"last_error,omitempty"`
}

// GpuSelector defines manual GPU selection for scheduling.
type GpuSelector struct {
	// GPU IDs in "host:type:index" format
	GpuIds []string `json:"gpu_ids,omitempty"`
	// Number of GPUs per replica
	GpusPerReplica int `json:"gpus_per_replica,omitempty"`
}

// ExtendedKVCacheConfig configures KV cache offloading to CPU memory.
type ExtendedKVCacheConfig struct {
	Enabled   bool     `json:"enabled"`
	RamRatio  *float64 `json:"ram_ratio,omitempty"`
	RamSize   *int     `json:"ram_size,omitempty"`
	ChunkSize *int     `json:"chunk_size,omitempty"`
}

// SpeculativeDecodingConfig configures speculative decoding.
type SpeculativeDecodingConfig struct {
	Enabled             bool   `json:"enabled"`
	Algorithm           string `json:"algorithm,omitempty"` // eagle3, mtp, ngram
	DraftModel          string `json:"draft_model,omitempty"`
	NumDraftTokens      *int   `json:"num_draft_tokens,omitempty"`
	NgramMinMatchLength *int   `json:"ngram_min_match_length,omitempty"`
	NgramMaxMatchLength *int   `json:"ngram_max_match_length,omitempty"`
}
