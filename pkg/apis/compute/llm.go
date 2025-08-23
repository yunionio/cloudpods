package compute

const (
	LLM_OLLAMA_EXEC_PATH   = "/bin/ollama"
	LLM_OLLAMA_PULL_ACTION = "pull"
	LLM_OLLAMA_LIST_ACTION = "list"
)

const (
	LLM_STATUS_PULLING_MODEL     = "pulling_model"
	LLM_STATUS_PULL_MODEL_FAILED = "pull_model_failed"
	LLM_STATUS_PULLED_MODEL      = "pulled_model"
)
