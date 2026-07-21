package llm_container

import (
	"path"
	"path/filepath"
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestModelsPathIsFlatBase(t *testing.T) {
	if api.LLM_VLLM_MODELS_PATH != api.LLM_VLLM_BASE_PATH {
		t.Fatalf("LLM_VLLM_MODELS_PATH = %q, want %q", api.LLM_VLLM_MODELS_PATH, api.LLM_VLLM_BASE_PATH)
	}
	if api.LLM_SGLANG_MODELS_PATH != api.LLM_SGLANG_BASE_PATH {
		t.Fatalf("LLM_SGLANG_MODELS_PATH = %q, want %q", api.LLM_SGLANG_MODELS_PATH, api.LLM_SGLANG_BASE_PATH)
	}
	if api.LLM_VLLM_MODELS_PATH != "/data/models" {
		t.Fatalf("LLM_VLLM_MODELS_PATH = %q, want /data/models", api.LLM_VLLM_MODELS_PATH)
	}
}

func TestImportModelFlatLayout(t *testing.T) {
	tmpDir := "/tmp/import-work"
	modelBase := filepath.Base("Qwen/Qwen3-0.6B")
	localDir := filepath.Join(tmpDir, modelBase)
	wantLocal := filepath.Join(tmpDir, "Qwen3-0.6B")
	if localDir != wantLocal {
		t.Fatalf("localDir = %q, want %q", localDir, wantLocal)
	}

	mount := path.Join(api.LLM_VLLM_MODELS_PATH, modelBase)
	wantMount := "/data/models/Qwen3-0.6B"
	if mount != wantMount {
		t.Fatalf("mount = %q, want %q", mount, wantMount)
	}

	// Mirrors GetImageInternalPathMounts: TrimPrefix(BASE) → path_map key.
	trimmed := strings.TrimPrefix(mount, api.LLM_VLLM_BASE_PATH)
	if trimmed != "/Qwen3-0.6B" {
		t.Fatalf("path_map key = %q, want /Qwen3-0.6B", trimmed)
	}
	mapped := path.Join(api.LLM_VLLM, trimmed)
	if mapped != "vllm/Qwen3-0.6B" {
		t.Fatalf("path_map value = %q, want vllm/Qwen3-0.6B", mapped)
	}
}
