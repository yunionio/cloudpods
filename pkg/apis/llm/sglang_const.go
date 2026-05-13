package llm

const (
	LLM_SGLANG              = "sglang"
	LLM_SGLANG_DEFAULT_PORT = 30000
	LLM_SGLANG_EXEC_PATH    = "python3 -m sglang.launch_server"

	LLM_SGLANG_HF_ENDPOINT = LLM_VLLM_HF_ENDPOINT
	LLM_SGLANG_CACHE_DIR   = "/root/.cache/huggingface"
	LLM_SGLANG_BASE_PATH   = "/data/models"
	LLM_SGLANG_MODELS_PATH = "/data/models/huggingface"
)
