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

import (
	"encoding/json"
	"errors"
	"strings"

	"yunion.io/x/onecloud/pkg/apis"
)

// SAiProviderConfig holds JSON-serialized provider connectivity settings for an ai_provider row.
type SAiProviderConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIMode string `json:"api_mode,omitempty"`
}

// UnmarshalJSON rejects legacy config.api_key and decodes supported fields only.
func (c *SAiProviderConfig) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if msg, ok := raw["api_key"]; ok {
		var key string
		_ = json.Unmarshal(msg, &key)
		if strings.TrimSpace(key) != "" {
			return errors.New("config.api_key is not supported, use secret and ai_keys")
		}
	}
	type cfgAlias SAiProviderConfig
	var alias cfgAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*c = SAiProviderConfig(alias)
	return nil
}

// ResolvedBaseURL returns config.base_url.
func (c *SAiProviderConfig) ResolvedBaseURL() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.BaseURL)
}

// ResolvedAPIMode returns config.api_mode (default openai).
func (c *SAiProviderConfig) ResolvedAPIMode() string {
	if c == nil {
		return ProviderAPIModeOpenAI
	}
	mode := strings.ToLower(strings.TrimSpace(c.APIMode))
	if mode == "" {
		return ProviderAPIModeOpenAI
	}
	return mode
}

// EffectiveBaseURL returns the upstream base URL adjusted for api_mode and provider_key.
func (c *SAiProviderConfig) EffectiveBaseURL(providerKey string) string {
	base := c.ResolvedBaseURL()
	if base == "" {
		base = DefaultPublicBaseURL(providerKey)
	}
	if base == "" {
		return ""
	}
	if c.ResolvedAPIMode() != ProviderAPIModeAnthropic {
		return base
	}
	pk := strings.ToLower(strings.TrimSpace(providerKey))
	switch pk {
	case ProviderKeyDeepseek:
		base = strings.TrimRight(base, "/")
		if strings.HasSuffix(strings.ToLower(base), "/anthropic") {
			return base
		}
		return base + "/anthropic"
	case ProviderKeyZhipu:
		base = strings.TrimRight(base, "/")
		if strings.HasSuffix(strings.ToLower(base), "/api/anthropic") {
			return base
		}
		openaiDefault := strings.TrimRight(DefaultPublicBaseURL(ProviderKeyZhipu), "/")
		if base == "" || strings.EqualFold(base, openaiDefault) {
			return DefaultZhipuAnthropicBaseURL()
		}
		return base
	default:
		return base
	}
}

// String implements gotypes.ISerializable for sqlchemy JSON/compound columns.
func (c *SAiProviderConfig) String() string {
	if c == nil {
		return "{}"
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// IsZero implements gotypes.ISerializable.
func (c *SAiProviderConfig) IsZero() bool {
	if c == nil {
		return true
	}
	return c.ResolvedBaseURL() == "" && strings.TrimSpace(c.APIMode) == ""
}

type AiProviderListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	ProviderKey     string `json:"provider_key"`
	LlmDeploymentId string `json:"llm_deployment_id"`
	LlmId           string `json:"llm_id"`
}

type AiProviderCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	Secret          string             `json:"secret"`
	ModelKeys       []string           `json:"model_keys"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
}

type AiProviderUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
	Enabled         *bool              `json:"enabled"`
}

type AiProviderDetails struct {
	apis.VirtualResourceDetails

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
}

type AiProviderTestConnectivityInput struct {
	ProviderKey string             `json:"provider_key"`
	Secret      string             `json:"secret"`
	Config      *SAiProviderConfig `json:"config"`
}

const (
	AiProviderModelsSourceUpstream = "upstream"
	AiProviderModelsSourceCatalog  = "catalog"
)

type AiProviderUpstreamModel struct {
	ModelKey string `json:"model_key"`
}

type AiProviderTestConnectivityOutput struct {
	Ok           bool                      `json:"ok"`
	Message      string                    `json:"message"`
	ModelsSource string                    `json:"models_source"`
	Models       []AiProviderUpstreamModel `json:"models"`
}
