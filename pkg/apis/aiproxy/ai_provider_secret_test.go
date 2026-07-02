package aiproxy

import (
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestSAiProviderConfigRejectsAPIKey(t *testing.T) {
	cfg := &SAiProviderConfig{}
	obj, err := jsonutils.Parse([]byte(`{"base_url":"https://api.openai.com","api_key":"sk-test"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = obj.Unmarshal(cfg)
	if err == nil {
		t.Fatal("expected error for config.api_key")
	}
	if !strings.Contains(err.Error(), "config.api_key is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSAiProviderConfigAllowsWithoutAPIKey(t *testing.T) {
	cfg := &SAiProviderConfig{}
	obj, err := jsonutils.Parse([]byte(`{"base_url":"https://api.openai.com","api_mode":"openai"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := obj.Unmarshal(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ResolvedBaseURL() != "https://api.openai.com" {
		t.Fatalf("base_url = %q", cfg.ResolvedBaseURL())
	}
}
