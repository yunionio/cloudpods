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

package responses

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

// UsesAnthropicTranslation reports whether Responses requests are translated to Anthropic Messages upstream.
func UsesAnthropicTranslation(providerKey, apiMode string) bool {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	mode := strings.ToLower(strings.TrimSpace(apiMode))
	if mode == "" {
		mode = api.ProviderAPIModeOpenAI
	}
	if key == api.ProviderKeyAnthropic {
		return true
	}
	return mode == api.ProviderAPIModeAnthropic && api.SupportsDualAPIMode(key)
}

// GetAdapter returns the ResponsesAdapter for a resolved catalog provider_key and api_mode.
func GetAdapter(providerKey, apiMode string) (providerapi.ResponsesAdapter, error) {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	mode := strings.ToLower(strings.TrimSpace(apiMode))
	if mode == "" {
		mode = api.ProviderAPIModeOpenAI
	}
	if api.IsNativeResponsesProvider(key) {
		if key == api.ProviderKeyAzure {
			return azurePassthrough{}, nil
		}
		return passthroughAdapter{}, nil
	}
	if UsesAnthropicTranslation(providerKey, apiMode) {
		return anthropicTranslationAdapter{}, nil
	}
	switch key {
	case api.ProviderKeyGemini:
		return geminiTranslationAdapter{}, nil
	default:
		return chatTranslationAdapter{providerKey: providerKey, apiMode: mode}, nil
	}
}

// GetAdapterOrError rejects unknown native-only providers explicitly.
func GetAdapterOrError(providerKey, apiMode string) (providerapi.ResponsesAdapter, error) {
	key := strings.ToLower(strings.TrimSpace(providerKey))
	if key == "" {
		return nil, fmt.Errorf("empty provider_key")
	}
	return GetAdapter(providerKey, apiMode)
}
