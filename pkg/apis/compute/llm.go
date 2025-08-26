package compute

const (
	LLM_OLLAMA_EXEC_PATH        = "/bin/ollama"
	LLM_OLLAMA_PULL_ACTION      = "pull"
	LLM_OLLAMA_LIST_ACTION      = "list"
	LLM_OLLAMA_EXPORT_ENV_KEY   = "OLLAMA_HOST"
	LLM_OLLAMA_EXPORT_ENV_VALUE = "0.0.0.0:11434"
)

const (
	LLM_STATUS_CREATING_POD      = "creating_pod"
	LLM_STATUS_CREAT_POD_FAILED  = "creat_pod_failed"
	LLM_STATUS_CREATED_POD       = "created_pod"
	LLM_STATUS_PULLING_MODEL     = "pulling_model"
	LLM_STATUS_PULL_MODEL_FAILED = "pull_model_failed"
	LLM_STATUS_PULLED_MODEL      = "pulled_model"
)
