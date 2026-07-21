package llm_container

import (
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func TestBuildVLLMEntrypointScriptMountedModelsFlag(t *testing.T) {
	sleepScript := buildVLLMEntrypointScript("", 1, nil, nil)
	if !strings.Contains(sleepScript, "sleep infinity") {
		t.Fatalf("expected idle script without model path, got %q", sleepScript)
	}
	if strings.Contains(sleepScript, "find ") {
		t.Fatalf("idle script should not find models, got %q", sleepScript)
	}

	nested := "/data/models/huggingface/Qwen3-8B"
	serveScript := buildVLLMEntrypointScript(nested, 1, nil, &api.LLMSpecVllm{PreferredModel: "Qwen3-8B"})
	if strings.Contains(serveScript, "sleep infinity") {
		t.Fatalf("expected serve script with model path, got %q", serveScript)
	}
	if !strings.Contains(serveScript, api.LLM_VLLM_EXEC_PATH) {
		t.Fatalf("expected vllm exec in serve script, got %q", serveScript)
	}
	if !strings.Contains(serveScript, nested) {
		t.Fatalf("expected nested model path in serve script, got %q", serveScript)
	}
	if strings.Contains(serveScript, "find ") {
		t.Fatalf("serve script should not find under MODELS_PATH, got %q", serveScript)
	}
	if strings.Contains(serveScript, api.LLM_VLLM_MODELS_PATH+"'") || strings.Contains(serveScript, "mkdir -p '"+api.LLM_VLLM_MODELS_PATH) {
		t.Fatalf("serve script should not select via MODELS_PATH root, got %q", serveScript)
	}
}

func TestBuildSGLangEntrypointScriptNestedModelPath(t *testing.T) {
	sleepScript := buildSGLangEntrypointScript("", 1, nil, nil)
	if !strings.Contains(sleepScript, "sleep infinity") {
		t.Fatalf("expected idle script without model path, got %q", sleepScript)
	}

	nested := "/data/models/huggingface/Qwen3-8B"
	serveScript := buildSGLangEntrypointScript(nested, 1, nil, &api.LLMSpecSGLang{PreferredModel: "Qwen3-8B"})
	if strings.Contains(serveScript, "sleep infinity") {
		t.Fatalf("expected serve script with model path, got %q", serveScript)
	}
	if !strings.Contains(serveScript, api.LLM_SGLANG_EXEC_PATH) {
		t.Fatalf("expected sglang exec in serve script, got %q", serveScript)
	}
	if !strings.Contains(serveScript, nested) {
		t.Fatalf("expected nested model path in serve script, got %q", serveScript)
	}
	if strings.Contains(serveScript, "find ") {
		t.Fatalf("serve script should not find under MODELS_PATH, got %q", serveScript)
	}
}

func TestLocalPathSkuEnablesServeEntrypoint(t *testing.T) {
	hostPaths := api.HostPaths{
		{
			Type: "directory",
			Path: "/data/models/Qwen3-8B",
			Containers: api.ContainerHostPathRelations{
				"0": {MountPath: "/data/models/huggingface/Qwen3-8B"},
			},
		},
	}
	sku := &models.SLLMSku{
		LLMType:   string(api.LLM_CONTAINER_VLLM),
		Source:    api.LLM_MODEL_SOURCE_LOCAL_PATH,
		LocalPath: "/data/models/Qwen3-8B",
	}
	sku.HostPaths = &hostPaths
	paths := models.CollectContainerModelMountPaths(nil, sku)
	modelPath := models.PickContainerModelMountPath(paths, "Qwen3-8B")
	if modelPath != "/data/models/huggingface/Qwen3-8B" {
		t.Fatalf("expected nested local_path mount, got %q", modelPath)
	}
	script := buildVLLMEntrypointScript(modelPath, 1, nil, nil)
	if !strings.Contains(script, modelPath) {
		t.Fatalf("expected script to embed mount path, got %q", script)
	}
}
