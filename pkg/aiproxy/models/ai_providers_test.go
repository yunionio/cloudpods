package models

import (
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestRejectProviderConfigAPIKeyInJSON(t *testing.T) {
	obj, err := jsonutils.Parse([]byte(`{"config":{"api_key":"sk-test"}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = rejectProviderConfigAPIKeyInJSON(obj)
	if err == nil {
		t.Fatal("expected error for config.api_key")
	}
	if !strings.Contains(err.Error(), "config.api_key is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRejectProviderConfigAPIKeyInJSONAllowsValidConfig(t *testing.T) {
	obj, err := jsonutils.Parse([]byte(`{"config":{"base_url":"https://api.openai.com"}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = rejectProviderConfigAPIKeyInJSON(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAiProviderCreateInputDefaultsEnabled(t *testing.T) {
	input := api.AiProviderCreateInput{
		ProviderKey: "deepseek",
		Secret:      "sk-test",
	}
	if input.Enabled != nil {
		t.Fatal("expected enabled unset before defaulting")
	}
	if input.Disabled != nil {
		t.Fatal("expected disabled unset before defaulting")
	}
	input.SetEnabled()
	if input.Enabled == nil || !*input.Enabled {
		t.Fatal("expected enabled=true after SetEnabled")
	}
}

func TestResolveUpstreamAPIKeyNilProvider(t *testing.T) {
	_, err := resolveUpstreamAPIKey(nil, "gpt-4o")
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestValidateAiProviderConfigCustomRequiresBaseURL(t *testing.T) {
	err := validateAiProviderConfig(nil, api.ProviderKeyCustom)
	if err == nil {
		t.Fatal("expected error when custom has no config")
	}
	err = validateAiProviderConfig(&api.SAiProviderConfig{}, api.ProviderKeyCustom)
	if err == nil {
		t.Fatal("expected error when custom base_url empty")
	}
}

func TestValidateAiProviderConfigCustomAnthropic(t *testing.T) {
	cfg := &api.SAiProviderConfig{
		BaseURL: "https://llm.example.com/anthropic",
		APIMode: api.ProviderAPIModeAnthropic,
	}
	if err := validateAiProviderConfig(cfg, api.ProviderKeyCustom); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
