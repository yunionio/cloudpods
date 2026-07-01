package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestBuildLLMSkuImportFromModelSpecModelScope(t *testing.T) {
	input := &api.LLMSkuCreateInput{
		LLMType: "vllm",
		ModelSpec: &api.InstantModelImportInput{
			ModelName: "Qwen/Qwen3-0.6B",
			ModelTag:  "master",
			LlmType:   api.LLM_CONTAINER_VLLM,
			Source:    api.InstantModelSourceModelScope,
			RepoId:    "Qwen/Qwen3-0.6B",
			Revision:  "master",
			FilePath:  "*.safetensors",
		},
	}
	out, err := buildLLMSkuImportFromModelSpec(input)
	if err != nil {
		t.Fatalf("buildLLMSkuImportFromModelSpec: %v", err)
	}
	if out.Source != api.InstantModelSourceModelScope {
		t.Fatalf("expected source model_scope, got %q", out.Source)
	}
	if input.Source != api.LLM_MODEL_SOURCE_MODEL_SCOPE {
		t.Fatalf("expected sku source model_scope, got %q", input.Source)
	}
	if input.ModelScopeModelId != "Qwen/Qwen3-0.6B" {
		t.Fatalf("unexpected model_scope_model_id: %q", input.ModelScopeModelId)
	}
	if input.ModelScopeFilePath != "*.safetensors" {
		t.Fatalf("unexpected model_scope_file_path: %q", input.ModelScopeFilePath)
	}
	if out.FilePath != "*.safetensors" {
		t.Fatalf("unexpected file_path: %q", out.FilePath)
	}
}

func TestResolveImportRepoAndRevisionModelScope(t *testing.T) {
	source, repo, rev := resolveImportRepoAndRevision(api.InstantModelImportInput{
		Source:    api.InstantModelSourceModelScope,
		RepoId:    "Qwen/Qwen3-8B",
		ModelName: "Qwen/Qwen3-8B",
	})
	if source != api.InstantModelSourceModelScope {
		t.Fatalf("source=%q", source)
	}
	if repo != "Qwen/Qwen3-8B" {
		t.Fatalf("repo=%q", repo)
	}
	if rev != defaultModelScopeRevision {
		t.Fatalf("rev=%q want %q", rev, defaultModelScopeRevision)
	}
}
