package llm

const (
	LLM_COMFYUI_BASE_PATH             = "/root/ComfyUI"
	LLM_COMFYUI_MODELS_PATH           = LLM_COMFYUI_BASE_PATH + "/models"
	LLM_COMFYUI_HF_ENDPOINT           = LLM_VLLM_HF_ENDPOINT
	LLM_COMFYUI_CHECKPOINTS_DIR       = "checkpoints"
	LLM_COMFYUI_LORAS_DIR             = "loras"
	LLM_COMFYUI_VAE_DIR               = "vae"
	LLM_COMFYUI_CONTROLNET_DIR        = "controlnet"
	LLM_COMFYUI_CLIP_DIR              = "clip"
	LLM_COMFYUI_TEXT_ENCODERS_DIR     = "text_encoders"
	LLM_COMFYUI_CLIP_VISION_DIR       = "clip_vision"
	LLM_COMFYUI_DIFFUSION_MODELS_DIR  = "diffusion_models"
	LLM_COMFYUI_EMBEDDINGS_DIR        = "embeddings"
	LLM_COMFYUI_IPADAPTER_DIR         = "ipadapter"
	LLM_COMFYUI_STYLE_MODELS_DIR      = "style_models"
	LLM_COMFYUI_UNET_DIR              = "unet"
	LLM_COMFYUI_UPSCALE_MODELS_DIR    = "upscale_models"
	LLM_COMFYUI_HF_MODELS_DIR         = LLM_COMFYUI_CHECKPOINTS_DIR
	LLM_COMFYUI_HF_MODELS_PATH        = LLM_COMFYUI_MODELS_PATH + "/" + LLM_COMFYUI_HF_MODELS_DIR
	LLM_COMFYUI_MODELS_VOLUME_SUBDIR  = "storage-models/models"
	LLM_COMFYUI_STORAGE_VOLUME_SUBDIR = "storage"
)
