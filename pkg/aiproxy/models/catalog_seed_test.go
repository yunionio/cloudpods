package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestCatalogSeedModelsForPublicProviders(t *testing.T) {
	publicKeys := []string{
		api.ProviderKeyOpenAI,
		api.ProviderKeyDeepseek,
		api.ProviderKeyAnthropic,
		api.ProviderKeyGemini,
		api.ProviderKeyMoonshot,
		api.ProviderKeyXiaomi,
		api.ProviderKeyZhipu,
	}
	for _, pk := range publicKeys {
		if !api.HasDefaultPublicBaseURL(pk) {
			t.Fatalf("%q should have default public base URL", pk)
		}
		entries := catalogSeedModelsForProvider(pk)
		if len(entries) == 0 {
			t.Fatalf("expected catalog models for public provider %q", pk)
		}
	}
}

func TestCatalogSeedModelsSkippedForSelfHostedProviders(t *testing.T) {
	selfHosted := []string{api.ProviderKeyOllama, api.ProviderKeyVLLM, api.ProviderKeySGLang}
	for _, pk := range selfHosted {
		if api.HasDefaultPublicBaseURL(pk) {
			t.Fatalf("%q should not be treated as public SaaS provider", pk)
		}
	}
}

func TestCatalogContextWindow(t *testing.T) {
	cases := []struct {
		modelKey string
		want     int
	}{
		{modelKey: "deepseek-v4-pro", want: 1_000_000},
		{modelKey: "deepseek-v4-flash", want: 1_000_000},
		{modelKey: "deepseek-chat", want: 1_000_000},
		{modelKey: "glm-5.2", want: 1_000_000},
		{modelKey: "glm-5.3", want: 1_000_000},
		{modelKey: "kimi-k3", want: 1_000_000},
		{modelKey: "mimo-v2.5", want: 1_000_000},
		{modelKey: "gpt-4.1", want: 1_000_000},
		{modelKey: "gpt-5.6", want: 1_050_000},
		{modelKey: "gemini-2.0-flash", want: 1_000_000},
		{modelKey: "gemini-1.5-pro", want: 2_000_000},
		{modelKey: "claude-opus-4-20250514", want: 1_000_000},
		{modelKey: "claude-sonnet-5", want: 1_000_000},
		{modelKey: "glm-5.1", want: 0},
		{modelKey: "gpt-5.2", want: 0},
		{modelKey: "kimi-k2.6", want: 0},
		{modelKey: "", want: 0},
	}
	for _, tc := range cases {
		got := CatalogContextWindow(tc.modelKey)
		if tc.want >= 1_000_000 {
			if got != tc.want {
				t.Errorf("CatalogContextWindow(%q) = %d, want %d", tc.modelKey, got, tc.want)
			}
			continue
		}
		if got >= 1_000_000 {
			t.Errorf("CatalogContextWindow(%q) = %d, want 0 or < 1M", tc.modelKey, got)
		}
	}
}
