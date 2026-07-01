package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestMapLLMTypeToProviderKey(t *testing.T) {
	cases := []struct {
		in  string
		key string
		ok  bool
	}{
		{string(api.LLM_CONTAINER_VLLM), "vllm", true},
		{string(api.LLM_CONTAINER_OLLAMA), "ollama", true},
		{string(api.LLM_CONTAINER_SGLANG), "sgl", true},
		{"dify", "", false},
	}
	for _, c := range cases {
		key, ok := mapLLMTypeToProviderKey(c.in)
		if ok != c.ok || key != c.key {
			t.Fatalf("mapLLMTypeToProviderKey(%q) = (%q, %v), want (%q, %v)", c.in, key, ok, c.key, c.ok)
		}
	}
}

func TestSlugModelKey(t *testing.T) {
	if got := slugModelKey("Qwen/Qwen2.5-7B-Instruct"); got != "qwen-qwen2-5-7b-instruct" {
		t.Fatalf("slugModelKey got %q", got)
	}
}

func TestDeploymentClientModelAlias(t *testing.T) {
	dep := &SLLMDeployment{}
	dep.Name = "my-qwen"
	if got := deploymentClientModelAlias(dep, "Qwen3-0.6B"); got != "my-qwen-Qwen3-0.6B" {
		t.Fatalf("deploymentClientModelAlias got %q", got)
	}
	depEmpty := &SLLMDeployment{}
	depEmpty.Id = "dep-id-1"
	if got := deploymentClientModelAlias(depEmpty, ""); got != "dep-id-1" {
		t.Fatalf("deploymentClientModelAlias without model_key got %q", got)
	}
}

func TestDeploymentRoutingModelKey(t *testing.T) {
	dep := &SLLMDeployment{}
	dep.Name = "my-qwen"
	if got := deploymentRoutingModelKey(dep, "Qwen3-0.6B"); got != "my-qwen-Qwen3-0.6B" {
		t.Fatalf("deploymentRoutingModelKey got %q", got)
	}
}

func TestAiproxyResourceNames(t *testing.T) {
	dep := &SLLMDeployment{}
	dep.Name = "My-Qwen"
	dep.Id = "dep-id-1"
	if got := aiRoutingNameForDeployment(dep); got != "llm-dep-my-qwen" {
		t.Fatalf("aiRoutingNameForDeployment got %q", got)
	}

	depEmpty := &SLLMDeployment{}
	depEmpty.Id = "dep-id-2"
	if got := aiRoutingNameForDeployment(depEmpty); got != "llm-dep-dep-id-2" {
		t.Fatalf("aiRoutingNameForDeployment empty name got %q", got)
	}

	llm := &SLLM{}
	llm.Name = "my-qwen-0"
	llm.Id = "llm-id-1"
	if got := aiProviderNameForLlm(llm); got != "llm-my-qwen-0" {
		t.Fatalf("aiProviderNameForLlm got %q", got)
	}

	llmEmpty := &SLLM{}
	llmEmpty.Id = "llm-id-2"
	if got := aiProviderNameForLlm(llmEmpty); got != "llm-llm-id-2" {
		t.Fatalf("aiProviderNameForLlm empty name got %q", got)
	}

	if got := aiModelNameForLlm(llm, "Qwen/Qwen3-0.6B"); got != "llm-my-qwen-0-qwen-qwen3-0-6b" {
		t.Fatalf("aiModelNameForLlm got %q", got)
	}
}

func TestClearDeploymentAiproxyRegistrationState(t *testing.T) {
	dep := &SLLMDeployment{}
	dep.AutoRegisterAiproxy = true
	dep.AiproxyRoutingId = "routing-1"
	dep.AiproxyBindings = &api.AiproxyBindings{{LlmId: "llm-1"}}
	dep.AiproxySyncStatus = api.AIPROXY_SYNC_STATUS_SYNCED

	clearDeploymentAiproxyRegistrationState(dep)

	if dep.AutoRegisterAiproxy {
		t.Fatal("AutoRegisterAiproxy should be false")
	}
	if dep.AiproxyRoutingId != "" {
		t.Fatalf("AiproxyRoutingId should be empty, got %q", dep.AiproxyRoutingId)
	}
	if dep.AiproxyBindings != nil {
		t.Fatal("AiproxyBindings should be nil")
	}
	if dep.AiproxySyncStatus != api.AIPROXY_SYNC_STATUS_DISABLED {
		t.Fatalf("AiproxySyncStatus should be disabled, got %q", dep.AiproxySyncStatus)
	}
}

func TestResolveAiproxySyncStatusAfterReconcile(t *testing.T) {
	cases := []struct {
		name     string
		result   aiproxyBindingSyncResult
		wantStat string
	}{
		{
			name:     "fully synced",
			result:   aiproxyBindingSyncSynced,
			wantStat: api.AIPROXY_SYNC_STATUS_SYNCED,
		},
		{
			name:     "binding partial failure",
			result:   aiproxyBindingSyncPartial,
			wantStat: api.AIPROXY_SYNC_STATUS_PARTIAL,
		},
		{
			name:     "all bindings failed",
			result:   aiproxyBindingSyncFailed,
			wantStat: api.AIPROXY_SYNC_STATUS_FAILED,
		},
		{
			name:     "pending",
			result:   aiproxyBindingSyncPending,
			wantStat: api.AIPROXY_SYNC_STATUS_PENDING,
		},
	}
	for _, c := range cases {
		got := resolveAiproxySyncStatusAfterReconcile(c.result)
		if got != c.wantStat {
			t.Fatalf("%s: got %q want %q", c.name, got, c.wantStat)
		}
	}
}

func TestAiproxySyncFailureReason(t *testing.T) {
	dep := &SLLMDeployment{}
	dep.AiproxyBindings = &api.AiproxyBindings{
		{LlmId: "llm-1", SyncStatus: api.AIPROXY_BINDING_SYNC_SYNCED},
		{LlmId: "llm-2", SyncStatus: api.AIPROXY_BINDING_SYNC_FAILED, LastError: "provider upsert failed"},
	}
	got := AiproxySyncFailureReason(dep)
	want := "llm llm-2: provider upsert failed"
	if got != want {
		t.Fatalf("AiproxySyncFailureReason() = %q, want %q", got, want)
	}

	msg := aiproxySyncStatusMessage(dep, aiproxyBindingSyncFailed)
	if msg != want {
		t.Fatalf("aiproxySyncStatusMessage(failed) = %q, want %q", msg, want)
	}
}

func TestUpstreamModelKeyForBackend(t *testing.T) {
	cases := []struct {
		llmType   string
		modelName string
		modelTag  string
		want      string
	}{
		{string(api.LLM_CONTAINER_VLLM), "Qwen/Qwen3-0.6B", "main", "Qwen3-0.6B"},
		{string(api.LLM_CONTAINER_SGLANG), "Qwen/Qwen2.5-7B-Instruct", "main", "Qwen2.5-7B-Instruct"},
		{string(api.LLM_CONTAINER_VLLM), "Qwen3-0.6B", "main", "Qwen3-0.6B"},
		{string(api.LLM_CONTAINER_OLLAMA), "qwen3", "8b", "qwen3:8b"},
		{string(api.LLM_CONTAINER_OLLAMA), "qwen3", "", "qwen3"},
	}
	for _, c := range cases {
		got := upstreamModelKeyForBackend(c.llmType, c.modelName, c.modelTag)
		if got != c.want {
			t.Fatalf("upstreamModelKeyForBackend(%q, %q, %q) = %q, want %q",
				c.llmType, c.modelName, c.modelTag, got, c.want)
		}
	}
}
