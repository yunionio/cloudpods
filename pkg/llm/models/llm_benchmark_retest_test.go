package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestBenchmarkRetestCreateInputSeparatesTarget(t *testing.T) {
	internal := &SLLMBenchmark{}
	internal.Id = "bench-internal"
	internal.LLMId = "llm-1"
	internal.LLMDeploymentId = "dep-1"
	internal.LLMImageId = "img-1"
	internal.BenchmarkPackageId = "pkg-1"
	internal.Backend = "vllm"
	internal.TargetUrl = "http://10.0.0.8:8000"
	internal.ApiKey = "stored-key"
	internal.Tokenizer = "Qwen/Qwen3-8B"
	internal.NetworkId = "net-1"
	got, err := benchmarkRetestCreateInput(internal)
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetUrl != "" || got.ApiKey != "" || got.Tokenizer != "" || got.NetworkId != "" {
		t.Fatalf("internal derived fields kept: %+v", got)
	}
	if got.LLMId != "" || got.LLMDeploymentId != "dep-1" {
		t.Fatalf("internal target = llm %q deployment %q", got.LLMId, got.LLMDeploymentId)
	}
	if got.BenchmarkImage != "img-1" || got.BenchmarkPackage != "pkg-1" {
		t.Fatalf("image %q package %q", got.BenchmarkImage, got.BenchmarkPackage)
	}

	llmOnly := &SLLMBenchmark{}
	llmOnly.LLMId = "llm-1"
	llmOnly.TargetUrl = "http://10.0.0.8:8000"
	llmOnly.Backend = "vllm"
	got, err = benchmarkRetestCreateInput(llmOnly)
	if err != nil {
		t.Fatal(err)
	}
	if got.LLMId != "llm-1" || got.LLMDeploymentId != "" || got.TargetUrl != "" {
		t.Fatalf("llm-only retest input = llm %q deployment %q url %q", got.LLMId, got.LLMDeploymentId, got.TargetUrl)
	}

	sealed, err := sealBenchmarkAPIKey("bench-external", "secret")
	if err != nil {
		t.Fatal(err)
	}
	external := &SLLMBenchmark{}
	external.Id = "bench-external"
	external.Backend = api.LLMBenchmarkBackendExternal
	external.LLMId = "stale-llm"
	external.LLMDeploymentId = "stale-dep"
	external.TargetUrl = "https://example.com/v1"
	external.ApiKey = sealed
	external.Tokenizer = "Qwen/Qwen3-8B"
	external.NetworkId = "net-ext"
	external.LLMImageId = "img-ext"
	got, err = benchmarkRetestCreateInput(external)
	if err != nil {
		t.Fatal(err)
	}
	if got.LLMId != "" || got.LLMDeploymentId != "" {
		t.Fatalf("external retest kept llm %q deployment %q", got.LLMId, got.LLMDeploymentId)
	}
	if got.TargetUrl != "https://example.com/v1" || got.ApiKey != "secret" || got.Tokenizer != "Qwen/Qwen3-8B" || got.NetworkId != "net-ext" {
		t.Fatalf("external retest input = %+v", got)
	}
	if got.BenchmarkImage != "img-ext" {
		t.Fatalf("external image %q", got.BenchmarkImage)
	}
}
