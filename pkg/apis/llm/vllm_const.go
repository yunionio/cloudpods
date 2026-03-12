package llm

import "time"

const (
	LLM_VLLM              = "vllm"
	LLM_VLLM_DEFAULT_PORT = 8000
	LLM_VLLM_EXEC_PATH    = "python3 -m vllm.entrypoints.openai.api_server"

	LLM_VLLM_HF_ENDPOINT = "https://hf-mirror.com"

	// Directory constants
	LLM_VLLM_CACHE_DIR   = "/root/.cache/huggingface"
	LLM_VLLM_BASE_PATH   = "/data/models"
	LLM_VLLM_MODELS_PATH = "/data/models/huggingface"

	// Health check
	LLM_VLLM_HEALTH_CHECK_TIMEOUT  = 120 * time.Second // 2 minutes
	LLM_VLLM_HEALTH_CHECK_INTERVAL = 10 * time.Second  // 10 seconds

	// Default vLLM memory params when Python estimation fails (conservative to avoid OOM)
	LLM_VLLM_DEFAULT_GPU_MEMORY_UTIL = 0.9
	LLM_VLLM_DEFAULT_MAX_MODEL_LEN   = 2048
	LLM_VLLM_DEFAULT_MAX_NUM_SEQS    = 1

	// Prefixes for parsing resolveModelAndParams output line (KEY=value)
	LLM_VLLM_RESOLVE_OUTPUT_PREFIX_GPU_UTIL    = "GPU_MEMORY_UTIL="
	LLM_VLLM_RESOLVE_OUTPUT_PREFIX_MAX_LEN     = "MAX_MODEL_LEN="
	LLM_VLLM_RESOLVE_OUTPUT_PREFIX_MAX_NUM_SEQ = "MAX_NUM_SEQS="
)

const (

	// vllmEstimateParamsScript is a Python script run inside the container to estimate
	// --gpu-memory-utilization, --max-model-len, and --max-num-seqs from GPU memory and model config.
	// Args: sys.argv[1]=model path, sys.argv[2]=tensor_parallel_size.
	// Prints one line: GPU_MEMORY_UTIL=0.9 MAX_MODEL_LEN=2624 MAX_NUM_SEQS=1 (eval-safe).
	LLM_VLLM_ESTIMATE_PARAMS_SCRIPT = `
import sys, json, os
model_path = sys.argv[1] if len(sys.argv) > 1 else ""
tp = int(sys.argv[2]) if len(sys.argv) > 2 else 1
if not model_path or not os.path.isdir(model_path):
    sys.exit(1)
config_path = os.path.join(model_path, "config.json")
if not os.path.isfile(config_path):
    sys.exit(1)
with open(config_path) as f:
    config = json.load(f)
def get_nested(d, *keys):
    for k in keys:
        d = d.get(k) if isinstance(d, dict) else None
        if d is None:
            return None
    return d
num_layers = config.get("num_hidden_layers") or config.get("n_layer") or get_nested(config, "text_config", "num_hidden_layers") or 0
num_heads = config.get("num_attention_heads") or config.get("n_head") or get_nested(config, "text_config", "num_attention_heads") or 0
num_kv_heads = config.get("num_key_value_heads") or get_nested(config, "text_config", "num_key_value_heads") or num_heads
hidden_size = config.get("hidden_size") or config.get("n_embd") or get_nested(config, "text_config", "hidden_size") or 0
head_dim = hidden_size // num_heads if num_heads else (hidden_size // 64)
max_pos = config.get("max_position_embeddings") or config.get("n_positions") or get_nested(config, "text_config", "max_position_embeddings") or 4096
if not num_layers or not num_kv_heads or not head_dim:
    sys.exit(1)
try:
    import torch
    total_mem = sum(torch.cuda.get_device_properties(i).total_memory for i in range(torch.cuda.device_count()))
except Exception:
    sys.exit(1)
gpu_util = 0.9
activation_overhead = 2 * (1024**3)
num_params = config.get("num_parameters") or config.get("num_params") or get_nested(config, "text_config", "num_parameters")
if num_params is not None:
    model_bytes = num_params * 2
else:
    model_bytes = 12 * (1024**3)
available_kv = total_mem * gpu_util - model_bytes - activation_overhead
if available_kv <= 0:
    available_kv = total_mem * 0.5
kv_per_token = num_layers * 2 * num_kv_heads * head_dim * 2
max_num_seqs = 4
max_model_len = int(available_kv / (kv_per_token * max_num_seqs))
max_model_len = max(1, min(max_model_len, max_pos))
if max_model_len < 256:
    max_num_seqs = 1
    max_model_len = int(available_kv / (kv_per_token * max_num_seqs))
    max_model_len = max(1, min(max_model_len, max_pos))
print("GPU_MEMORY_UTIL=%s MAX_MODEL_LEN=%d MAX_NUM_SEQS=%d" % (gpu_util, max_model_len, max_num_seqs))
`
)
