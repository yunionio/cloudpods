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

	"yunion.io/x/onecloud/pkg/apis"
)

type AiModelListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	AiProviderId string `json:"ai_provider_id"`
	AiRoutingId  string `json:"ai_routing_id"`
	ModelKey     string `json:"model_key"`
}

type AiModelCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	AiProviderId     string          `json:"ai_provider_id"`
	ModelKey         string          `json:"model_key"`
	VisualProviderId string          `json:"visual_provider_id"`
	VisualModelKey   string          `json:"visual_model_key"`
	Config           *SAiModelConfig `json:"config"`
}

type AiModelUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	AiProviderId     string          `json:"ai_provider_id"`
	ModelKey         string          `json:"model_key"`
	VisualProviderId string          `json:"visual_provider_id"`
	VisualModelKey   string          `json:"visual_model_key"`
	Enabled          *bool           `json:"enabled"`
	Config           *SAiModelConfig `json:"config"`
}

type AiModelDetails struct {
	apis.VirtualResourceDetails

	AiProviderId       string          `json:"ai_provider_id"`
	AiProviderName     string          `json:"ai_provider_name"`
	ModelKey           string          `json:"model_key"`
	VisualProviderId   string          `json:"visual_provider_id"`
	VisualProviderName string          `json:"visual_provider_name"`
	VisualProviderKey  string          `json:"visual_provider_key"`
	VisualModelKey     string          `json:"visual_model_key"`
	VisualActive       bool            `json:"visual_active"`
	Config             *SAiModelConfig `json:"config"`
	ContextWindow      int             `json:"context_window,omitempty"`
}

// SAiModelConfig stores per-model extension settings.
type SAiModelConfig struct {
	Extensions *SAiModelExtensions `json:"extensions,omitempty"`
}

type SAiModelExtensions struct {
	Visual *SAiModelVisualConfig `json:"visual,omitempty"`
}

// SAiModelVisualConfig enables tool-delegated vision for text-only upstream models.
// visual_provider_id / visual_model_key live on ai_model columns, not in this JSON.
type SAiModelVisualConfig struct {
	Enabled   bool `json:"enabled"`
	MaxRounds int  `json:"max_rounds,omitempty"`
	MaxTokens int  `json:"max_tokens,omitempty"`
}

// String implements gotypes.ISerializable for sqlchemy JSON/compound columns.
func (c *SAiModelConfig) String() string {
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
func (c *SAiModelConfig) IsZero() bool {
	if c == nil {
		return true
	}
	return c.Extensions == nil || c.Extensions.Visual == nil
}

// VisualEnabled reports whether visual extension is configured and enabled.
func (cfg *SAiModelConfig) VisualEnabled() bool {
	return cfg != nil && cfg.Extensions != nil && cfg.Extensions.Visual != nil && cfg.Extensions.Visual.Enabled
}

// InputModalitiesForCatalog returns Codex input_modalities for this model config.
func (cfg *SAiModelConfig) InputModalitiesForCatalog() []string {
	if cfg.VisualEnabled() {
		return []string{"text", "image"}
	}
	return nil
}
