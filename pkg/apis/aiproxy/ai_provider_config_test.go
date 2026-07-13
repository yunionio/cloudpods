package aiproxy

import "testing"

func TestResolvedAPIModeDefault(t *testing.T) {
	cfg := &SAiProviderConfig{}
	if got := cfg.ResolvedAPIMode(); got != ProviderAPIModeOpenAI {
		t.Fatalf("ResolvedAPIMode() = %q, want %q", got, ProviderAPIModeOpenAI)
	}
}

func TestEffectiveBaseURLDeepseekAnthropic(t *testing.T) {
	cfg := &SAiProviderConfig{
		BaseURL: "https://api.deepseek.com",
		APIMode: ProviderAPIModeAnthropic,
	}
	got := cfg.EffectiveBaseURL(ProviderKeyDeepseek)
	want := "https://api.deepseek.com/anthropic"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestEffectiveBaseURLDeepseekOpenAI(t *testing.T) {
	cfg := &SAiProviderConfig{
		BaseURL: "https://api.deepseek.com/anthropic",
		APIMode: ProviderAPIModeOpenAI,
	}
	got := cfg.EffectiveBaseURL(ProviderKeyDeepseek)
	want := "https://api.deepseek.com/anthropic"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestSupportsDualAPIMode(t *testing.T) {
	if !SupportsDualAPIMode(ProviderKeyDeepseek) {
		t.Fatal("deepseek should support dual api mode")
	}
	if !SupportsDualAPIMode(ProviderKeyCustom) {
		t.Fatal("custom should support dual api mode")
	}
	if !SupportsDualAPIMode(ProviderKeyZhipu) {
		t.Fatal("zhipu should support dual api mode")
	}
	if SupportsDualAPIMode(ProviderKeyOpenAI) {
		t.Fatal("openai should not support dual api mode")
	}
}

func TestIsCustomProviderKey(t *testing.T) {
	if !IsCustomProviderKey(ProviderKeyCustom) {
		t.Fatal("custom key should match")
	}
	if IsCustomProviderKey(ProviderKeyOpenAI) {
		t.Fatal("openai should not be custom")
	}
}

func TestEffectiveBaseURLFallbackOpenAI(t *testing.T) {
	cfg := &SAiProviderConfig{}
	got := cfg.EffectiveBaseURL(ProviderKeyOpenAI)
	want := "https://api.openai.com"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestEffectiveBaseURLFallbackDeepseekAnthropic(t *testing.T) {
	cfg := &SAiProviderConfig{APIMode: ProviderAPIModeAnthropic}
	got := cfg.EffectiveBaseURL(ProviderKeyDeepseek)
	want := "https://api.deepseek.com/anthropic"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestEffectiveBaseURLCustomNoAnthropicSuffix(t *testing.T) {
	cfg := &SAiProviderConfig{
		BaseURL: "https://llm.example.com/v1",
		APIMode: ProviderAPIModeAnthropic,
	}
	got := cfg.EffectiveBaseURL(ProviderKeyCustom)
	want := "https://llm.example.com/v1"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestHasDefaultPublicBaseURL(t *testing.T) {
	if !HasDefaultPublicBaseURL(ProviderKeyOpenAI) {
		t.Fatal("openai should have default base url")
	}
	if HasDefaultPublicBaseURL(ProviderKeyAzure) {
		t.Fatal("azure should not have default base url")
	}
	if HasDefaultPublicBaseURL(ProviderKeyCustom) {
		t.Fatal("custom should not have default base url")
	}
	if !HasDefaultPublicBaseURL(ProviderKeyMoonshot) {
		t.Fatal("moonshot should have default base url")
	}
	if DefaultPublicBaseURL(ProviderKeyMoonshot) != "https://api.moonshot.cn" {
		t.Fatalf("moonshot default base = %q", DefaultPublicBaseURL(ProviderKeyMoonshot))
	}
	if !HasDefaultPublicBaseURL(ProviderKeyZhipu) {
		t.Fatal("zhipu should have default base url")
	}
	if DefaultPublicBaseURL(ProviderKeyZhipu) != "https://open.bigmodel.cn/api/paas/v4" {
		t.Fatalf("zhipu default base = %q", DefaultPublicBaseURL(ProviderKeyZhipu))
	}
}

func TestEffectiveBaseURLZhipuAnthropic(t *testing.T) {
	cfg := &SAiProviderConfig{
		BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		APIMode: ProviderAPIModeAnthropic,
	}
	got := cfg.EffectiveBaseURL(ProviderKeyZhipu)
	want := DefaultZhipuAnthropicBaseURL()
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestEffectiveBaseURLZhipuAnthropicFallback(t *testing.T) {
	cfg := &SAiProviderConfig{APIMode: ProviderAPIModeAnthropic}
	got := cfg.EffectiveBaseURL(ProviderKeyZhipu)
	want := DefaultZhipuAnthropicBaseURL()
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}

func TestEffectiveBaseURLZhipuOpenAI(t *testing.T) {
	cfg := &SAiProviderConfig{}
	got := cfg.EffectiveBaseURL(ProviderKeyZhipu)
	want := "https://open.bigmodel.cn/api/paas/v4"
	if got != want {
		t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
	}
}
