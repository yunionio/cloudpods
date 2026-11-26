// Copyright 2019 Yunion
package llm

import "testing"

func TestNormalizeAndValidateLLMImageCapabilities(t *testing.T) {
	if err := ValidateLLMImageCapabilities([]string{"omni", "Omni"}); err != nil {
		t.Fatalf("valid caps rejected: %v", err)
	}
	got := NormalizeLLMImageCapabilities([]string{" Omni ", "omni", ""})
	if len(got) != 1 || got[0] != "omni" {
		t.Fatalf("normalize: %v", got)
	}
	if err := ValidateLLMImageCapabilities([]string{"nope"}); err == nil {
		t.Fatal("expected invalid capability error")
	}
	if !HasLLMImageCapability([]string{"omni"}, LLM_IMAGE_CAPABILITY_OMNI) {
		t.Fatal("HasLLMImageCapability missed omni")
	}
}

func TestIsLLMOmniCategory(t *testing.T) {
	if !IsLLMOmniCategory([]string{LLM_MODEL_CATEGORY_TEXT_TO_SPEECH}) {
		t.Fatal("tts should be omni")
	}
	if !IsLLMOmniCategory([]string{LLM_MODEL_CATEGORY_IMAGE}) {
		t.Fatal("image should be omni")
	}
	if IsLLMOmniCategory([]string{LLM_MODEL_CATEGORY_LLM}) {
		t.Fatal("llm should not be omni")
	}
}
