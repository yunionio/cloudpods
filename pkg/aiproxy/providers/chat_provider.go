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

package providers

import (
	"yunion.io/x/onecloud/pkg/aiproxy/providers/anthropic"
	apapi "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

// ChatProviderForUpstream returns the provider adapter for chat/completions upstream calls.
func ChatProviderForUpstream(providerKey, apiMode string) Provider {
	if apiMode == apapi.ProviderAPIModeAnthropic && apapi.SupportsDualAPIMode(providerKey) {
		return anthropic.OpenAINativeBridge()
	}
	return Get(providerKey)
}

// ChatContextFromUpstream builds a provider ChatContext from resolved upstream fields.
func ChatContextFromUpstream(providerKey, baseURL, apiKey, upstreamModel, apiMode string) *ChatContext {
	return &ChatContext{
		ProviderKey:   providerKey,
		BaseURL:       baseURL,
		APIKey:        apiKey,
		UpstreamModel: upstreamModel,
		APIMode:       apiMode,
	}
}
