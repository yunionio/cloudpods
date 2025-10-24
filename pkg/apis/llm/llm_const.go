package llm

const (
	SERVICE_TYPE = "llm"
)

const (
	LLM_OLLAMA                  = "ollama"
	LLM_OLLAMA_EXEC_PATH        = "/bin/ollama"
	LLM_OLLAMA_PULL_ACTION      = "pull"
	LLM_OLLAMA_LIST_ACTION      = "list"
	LLM_OLLAMA_CREATE_ACTION    = "create"
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
	LLM_OLLAMA_GGUF_DIR                    = "/gguf"
	LLM_OLLAMA_GGUF_SOURCE_HOST            = "host"
	LLM_OLLAMA_GGUF_SOURCE_WEB             = "web"
	LLM_OLLAMA_MODELFILE_NAME              = "modelfile"
	LLM_OLLAMA_GGUF_FROM                   = "FROM %s\n"
	LLM_OLLAMA_GGUF_PARAMETER              = "PARAMETER %s %s\n"
	LLM_OLLAMA_GGUF_TEMPLATE               = "TEMPLATE \"\"\"%s\"\"\"\n"
	LLM_OLLAMA_GGUF_SYSTEM                 = "SYSTEM %s\n"
	LLM_OLLAMA_GGUF_ADAPTER                = "ADAPTER %s\n"
	LLM_OLLAMA_GGUF_LICENSE                = "LICENSE \"\"\"%s\"\"\"\n"
	LLM_OLLAMA_GGUF_MESSAGE                = "MESSAGE %s %s\n"
	LLM_OLLAMA_GGUF_MESSAGE_ROLE_SYSTEM    = "system"
	LLM_OLLAMA_GGUF_MESSAGE_ROLE_USER      = "user"
	LLM_OLLAMA_GGUF_MESSAGE_ROLE_ASSISTANT = "assistant"
)

const (
	LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX        = "num_ctx"
	LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N  = "repeat_last_n"
	LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY = "repeat_penalty"
	LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE    = "temperature"
	LLM_OLLAMA_MODELFILE_PARAMETER_SEED           = "seed"
	LLM_OLLAMA_MODELFILE_PARAMETER_STOP           = "stop"
	LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT    = "num_predict"
	LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K          = "top_k"
	LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P          = "top_p"
	LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P          = "min_p"
)

const (
	LLM_STATUS_CREATING_POD             = "creating_pod"
	LLM_STATUS_CREAT_POD_FAILED         = "creat_pod_failed"
	LLM_STATUS_PULLING_MODEL            = "pulling_model"
	LLM_STATUS_GET_MANIFESTS_FAILED     = "get_manifests_failed"
	LLM_STATUS_DOWNLOADING_BLOBS        = "downloading_blobs"
	LLM_STATUS_DOWNLOADING_BLOBS_FAILED = "downloading_blobs_failed"
	LLM_STATUS_FETCHING_GGUF_FILE       = "fetching_gguf_file"
	LLM_STATUS_FETCH_GGUF_FILE_FAILED   = "fetch_gguf_failed"
	LLM_STATUS_CREATING_GGUF_MODEL      = "creating_gguf_model"
	LLM_STATUS_CREATE_GGUF_MODEL_FAILED = "create_gguf_model_failed"
	LLM_STATUS_PULLED_MODEL             = "pulled_model"
)
