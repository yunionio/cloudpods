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
	"strings"
	"sync"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/anthropic" // "yunion.io/x/onecloud/pkg/aiproxy/providers/azure" // uncommon
	// "yunion.io/x/onecloud/pkg/aiproxy/providers/baidu" // uncommon
	// "yunion.io/x/onecloud/pkg/aiproxy/providers/cohere" // uncommon
	"yunion.io/x/onecloud/pkg/aiproxy/providers/gemini"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/vllm"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy" // "yunion.io/x/onecloud/pkg/aiproxy/providers/aliyun" // uncommon
)

var (
	registryMu sync.RWMutex
	registry   = map[string]providerapi.Provider{}
	defaultP   providerapi.Provider
)

func init() {
	defaultP = openai.NewCompat("")
	register(defaultP)
	for _, key := range api.OpenAICompatProviderKeys {
		register(openai.NewCompat(key))
	}
	// register(cohere.New()) // uncommon
	// register(aliyun.New()) // uncommon
	// register(baidu.New()) // uncommon
	register(anthropic.New())
	register(gemini.New())
	// register(azure.New()) // uncommon
	register(vllm.New())
	register(openai.NewCompat(api.ProviderKeyCustom))
}

// Register adds or replaces a provider implementation for its Key().
func Register(p Provider) {
	if p == nil {
		return
	}
	register(p)
}

func register(p providerapi.Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	k := normalizeKey(p.Key())
	if k == "" {
		registry[""] = p
		return
	}
	registry[k] = p
}

// Get returns the provider for providerKey, or the default OpenAI-compatible passthrough.
func Get(providerKey string) Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	k := normalizeKey(providerKey)
	if p, ok := registry[k]; ok {
		return p
	}
	if defaultP != nil {
		return defaultP
	}
	return openai.NewCompat("")
}

func normalizeKey(k string) string {
	return strings.ToLower(strings.TrimSpace(k))
}
