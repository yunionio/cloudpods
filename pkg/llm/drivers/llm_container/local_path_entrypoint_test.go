package llm_container

import (
	"strings"
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func TestBuildVLLMEntrypointScriptMountedModelsFlag(t *testing.T) {
	sleepScript := buildVLLMEntrypointScript(false, 1, nil, nil)
	if !strings.Contains(sleepScript, "sleep infinity") {
		t.Fatalf("expected idle script without mounted models, got %q", sleepScript)
	}
	serveScript := buildVLLMEntrypointScript(true, 1, nil, &api.LLMSpecVllm{PreferredModel: "Qwen3-8B"})
	if strings.Contains(serveScript, "sleep infinity") {
		t.Fatalf("expected serve script with mounted models, got %q", serveScript)
	}
	if !strings.Contains(serveScript, api.LLM_VLLM_EXEC_PATH) {
		t.Fatalf("expected vllm exec in serve script, got %q", serveScript)
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
	postOverlaysLen := 0
	hasMountedModels := postOverlaysLen > 0 || models.SkuHasLocalHostPathModel(sku)
	if !hasMountedModels {
		t.Fatal("expected local_path sku to count as having mounted models")
	}
}
