package llm

import "time"

const (
	LLM_VLLM              = "vllm"
	LLM_VLLM_DEFAULT_PORT = 8000
	LLM_VLLM_EXEC_PATH    = "python3 -m vllm.entrypoints.openai.api_server"

	LLM_VLLM_HF_ENDPOINT = "https://hf-mirror.com"
	LLM_VLLM_CACHE_DIR   = "/root/.cache/huggingface"
	LLM_VLLM_BASE_PATH   = "/data/models"
	LLM_VLLM_MODELS_PATH = "/data/models/huggingface"

	LLM_VLLM_HEALTH_CHECK_TIMEOUT  = 120 * time.Second
	LLM_VLLM_HEALTH_CHECK_INTERVAL = 10 * time.Second
)
