package llm_container

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/hub"
)

func TestBuildModelScopeFilesURL(t *testing.T) {
	got := hub.BuildModelScopeFilesURL("https://www.modelscope.cn", "Qwen/Qwen3-0.6B", "master")
	want := "https://www.modelscope.cn/api/v1/models/Qwen/Qwen3-0.6B/repo/files?Recursive=true&Revision=master"
	if got != want {
		t.Fatalf("BuildModelScopeFilesURL=%q want %q", got, want)
	}
}

func TestBuildModelScopeFileDownloadURL(t *testing.T) {
	got := hub.BuildModelScopeFileDownloadURL("https://www.modelscope.cn", "Qwen/Qwen3-0.6B", "config.json")
	if got == "" {
		t.Fatal("empty download url")
	}
}

func TestMatchModelScopeFilePaths(t *testing.T) {
	files := []hub.ModelScopeFileEntry{
		{Path: "config.json", Size: 100},
		{Path: "model.safetensors", Size: 200},
	}
	matched, err := hub.MatchModelScopeFilePaths(files, "*.safetensors")
	if err != nil {
		t.Fatalf("MatchModelScopeFilePaths: %v", err)
	}
	if len(matched) != 1 || matched[0].Path != "model.safetensors" {
		t.Fatalf("unexpected matched: %+v", matched)
	}
}

func TestResolveImportRevisionModelScope(t *testing.T) {
	rev := resolveImportRevision(api.InstantModelImportInput{
		Source: api.InstantModelSourceModelScope,
	}, hub.DefaultModelScopeRevision)
	if rev != hub.DefaultModelScopeRevision {
		t.Fatalf("rev=%q", rev)
	}
}
