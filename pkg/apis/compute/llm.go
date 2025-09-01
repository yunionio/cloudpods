package compute

const (
	LLM_OLLAMA                  = "ollama"
	LLM_OLLAMA_EXEC_PATH        = "/bin/ollama"
	LLM_OLLAMA_PULL_ACTION      = "pull"
	LLM_OLLAMA_LIST_ACTION      = "list"
	LLM_OLLAMA_EXPORT_ENV_KEY   = "OLLAMA_HOST"
	LLM_OLLAMA_EXPORT_ENV_VALUE = "0.0.0.0:11434"
)

const (
	LLM_OLLAMA_CACHE_DIR           = "/.llm_ollama_cache"
	LLM_OLLAMA_CACHE_MOUNT_PATH    = "/usr/local"
	LLM_OLLAMA_LIBRARY_BASE_URL    = `https://registry.ollama.ai/v2/library/%s`
	LLM_OLLAMA_BASE_PATH           = "/root/.ollama/models"
	LLM_OLLAMA_BLOBS_DIR           = "/blobs"
	LLM_OLLAMA_MANIFESTS_BASE_PATH = "/manifests/registry.ollama.ai/library"
)

const (
	LLM_STATUS_CREATING_POD             = "creating_pod"
	LLM_STATUS_CREAT_POD_FAILED         = "creat_pod_failed"
	LLM_STATUS_PULLING_MODEL            = "pulling_model"
	LLM_STATUS_GET_MANIFESTS_FAILED     = "get_manifests_failed"
	LLM_STATUS_DOWNLOADING_BLOBS        = "downloading_blobs"
	LLM_STATUS_DOWNLOADING_BLOBS_FAILED = "downloading_blobs_failed"
	LLM_STATUS_PULLED_MODEL             = "pulled_model"
)

type LLMCreateInput struct {
	ServerCreateInput
	Model string `json:"model"`
}

type LLMAccessCacheInput struct {
	ModelName string   `json:"model_name"`
	Blobs     []string `json:"blobs"`
}
