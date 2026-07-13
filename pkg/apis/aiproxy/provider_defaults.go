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

// DefaultPublicBaseURL returns a well-known public API base for catalog providers.
// Empty string means no default (user must set config.base_url).
func DefaultPublicBaseURL(providerKey string) string {
	switch strings.ToLower(strings.TrimSpace(providerKey)) {
	case ProviderKeyOpenAI:
		return "https://api.openai.com"
	case ProviderKeyAnthropic:
		return "https://api.anthropic.com"
	case ProviderKeyAzure, ProviderKeyBedrock, ProviderKeySGLang, ProviderKeyOllama, ProviderKeyVLLM, ProviderKeyAliyun, ProviderKeyBaidu:
		return ""
	case ProviderKeyDeepseek:
		return "https://api.deepseek.com"
	case ProviderKeyGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ProviderKeyGroq:
		return "https://api.groq.com/openai"
	case ProviderKeyMistral:
		return "https://api.mistral.ai"
	case ProviderKeyOpenrouter:
		return "https://openrouter.ai/api"
	case ProviderKeyHuggingface:
		return "https://router.huggingface.co"
	case ProviderKeyXiaomi:
		return "https://api.xiaomimimo.com"
	case ProviderKeyMoonshot:
		return "https://api.moonshot.cn"
	case ProviderKeyZhipu:
		return "https://open.bigmodel.cn/api/paas/v4"
	default:
		return ""
	}
}

// DefaultZhipuAnthropicBaseURL is the upstream base for Zhipu Claude-compatible API.
func DefaultZhipuAnthropicBaseURL() string {
	return "https://open.bigmodel.cn/api/anthropic"
}

// HasDefaultPublicBaseURL reports whether provider_key has a built-in public base URL.
func HasDefaultPublicBaseURL(providerKey string) bool {
	return DefaultPublicBaseURL(providerKey) != ""
}
