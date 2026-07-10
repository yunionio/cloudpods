// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aiproxy

import "strings"

// Built-in provider_key values (registered in providers runtime registry).
const (
	ProviderKeyAliyun      = "aliyun"
	ProviderKeyAnthropic   = "anthropic"
	ProviderKeyAzure       = "azure"
	ProviderKeyBaidu       = "baidu"
	ProviderKeyBedrock     = "bedrock"
	ProviderKeyCerebras    = "cerebras"
	ProviderKeyCohere      = "cohere"
	ProviderKeyCustom      = "custom"
	ProviderKeyDeepseek    = "deepseek"
	ProviderKeyElevenlabs  = "elevenlabs"
	ProviderKeyFireworks   = "fireworks"
	ProviderKeyGemini      = "gemini"
	ProviderKeyGroq        = "groq"
	ProviderKeyHuggingface = "huggingface"
	ProviderKeyMistral     = "mistral"
	ProviderKeyMoonshot    = "moonshot"
	ProviderKeyNebius      = "nebius"
	ProviderKeyOllama      = "ollama"
	ProviderKeyOpenAI      = "openai"
	ProviderKeyOpenrouter  = "openrouter"
	ProviderKeyParasail    = "parasail"
	ProviderKeyPerplexity  = "perplexity"
	ProviderKeyReplicate   = "replicate"
	ProviderKeyRunway      = "runway"
	ProviderKeySGLang      = "sglang"
	ProviderKeyVertex      = "vertex"
	ProviderKeyVLLM        = "vllm"
	ProviderKeyXai         = "xai"
	ProviderKeyXiaomi      = "xiaomi"
)

// OpenAICompatProviderKeys are catalog keys routed through openai.NewCompat.
var OpenAICompatProviderKeys = []string{
	ProviderKeyOpenAI,
	ProviderKeyGroq,
	ProviderKeyMistral,
	// ProviderKeyCerebras, // uncommon
	ProviderKeyDeepseek,
	// ProviderKeyPerplexity, // uncommon
	ProviderKeyOpenrouter,
	// ProviderKeyFireworks, // uncommon
	// ProviderKeyNebius, // uncommon
	// ProviderKeyXai, // uncommon
	// ProviderKeyParasail, // uncommon
	ProviderKeySGLang,
	ProviderKeyHuggingface,
	ProviderKeyOllama,
	ProviderKeyXiaomi,
	ProviderKeyMoonshot,
}

var nativeMessagesAdapterProviderKeys = map[string]struct{}{
	ProviderKeyGemini: {},
	// ProviderKeyCohere: {}, // uncommon
	// ProviderKeyBaidu:  {}, // uncommon
	// ProviderKeyAliyun: {}, // uncommon
}

var nativeResponsesProviderKeys = map[string]struct{}{
	ProviderKeyOpenAI: {},
	ProviderKeyAzure:  {},
}

// IsNativeResponsesProvider reports whether provider_key proxies upstream /v1/responses directly.
func IsNativeResponsesProvider(providerKey string) bool {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	_, ok := nativeResponsesProviderKeys[key]
	return ok
}

// IsNativeMessagesAdapterProvider reports whether provider_key uses a dedicated
// non-OpenAI-chat upstream API that cannot be reached via Anthropic-to-OpenAI translation.
func IsNativeMessagesAdapterProvider(providerKey string) bool {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	_, ok := nativeMessagesAdapterProviderKeys[key]
	return ok
}

const (
	ProviderAPIModeOpenAI    = "openai"
	ProviderAPIModeAnthropic = "anthropic"
)

// DualAPIProviderKeys lists provider_key values that support openai and anthropic upstream APIs.
var DualAPIProviderKeys = map[string]struct{}{
	ProviderKeyCustom:   {},
	ProviderKeyDeepseek: {},
}

// IsCustomProviderKey reports whether provider_key is the user-defined custom gateway type.
func IsCustomProviderKey(providerKey string) bool {
	return strings.ToLower(strings.TrimSpace(providerKey)) == ProviderKeyCustom
}

// SupportsDualAPIMode reports whether provider_key may use config.api_mode.
func SupportsDualAPIMode(providerKey string) bool {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	_, ok := DualAPIProviderKeys[key]
	return ok
}

// IsValidProviderAPIMode reports whether mode is a supported api_mode value.
func IsValidProviderAPIMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ProviderAPIModeOpenAI, ProviderAPIModeAnthropic:
		return true
	default:
		return false
	}
}
