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
	"strings"

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
	ContextWindow    int             `json:"context_window"`
	VisualProviderId string          `json:"visual_provider_id"`
	VisualModelKey   string          `json:"visual_model_key"`
	Config           *SAiModelConfig `json:"config"`
}

type AiModelUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	AiProviderId     string          `json:"ai_provider_id"`
	ModelKey         string          `json:"model_key"`
	ContextWindow    *int            `json:"context_window"`
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
	InputModalities    []string        `json:"input_modalities,omitempty"`
}

// SAiModelConfig stores per-model capability and extension settings.
type SAiModelConfig struct {
	// InputModalities declares native model modalities (e.g. text, image).
	InputModalities []string            `json:"input_modalities,omitempty"`
	Extensions      *SAiModelExtensions `json:"extensions,omitempty"`
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
	if len(c.InputModalities) > 0 {
		return false
	}
	return c.Extensions == nil || c.Extensions.Visual == nil
}

// VisualEnabled reports whether visual extension is configured and enabled.
func (cfg *SAiModelConfig) VisualEnabled() bool {
	return cfg != nil && cfg.Extensions != nil && cfg.Extensions.Visual != nil && cfg.Extensions.Visual.Enabled
}

// HasImageInput reports whether stored input_modalities includes image.
func (cfg *SAiModelConfig) HasImageInput() bool {
	for _, m := range cfg.NormalizedInputModalities() {
		if m == "image" {
			return true
		}
	}
	return false
}

// NormalizedInputModalities returns trimmed lowercase modality ids.
func (cfg *SAiModelConfig) NormalizedInputModalities() []string {
	if cfg == nil || len(cfg.InputModalities) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(cfg.InputModalities))
	out := make([]string, 0, len(cfg.InputModalities))
	for _, raw := range cfg.InputModalities {
		m := strings.ToLower(strings.TrimSpace(raw))
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

// InputModalitiesForCatalog returns Codex/OpenAI input_modalities for this model config.
// Prefers explicit input_modalities; falls back to visual-delegation implying text+image.
func (cfg *SAiModelConfig) InputModalitiesForCatalog() []string {
	if mods := cfg.NormalizedInputModalities(); len(mods) > 0 {
		return mods
	}
	if cfg.VisualEnabled() {
		return []string{"text", "image"}
	}
	return nil
}
