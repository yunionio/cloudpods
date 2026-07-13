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
