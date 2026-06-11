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

// SAiProviderConfig holds JSON-serialized provider connectivity settings for an ai_provider row.
type SAiProviderConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

// ResolvedBaseURL returns config.base_url.
func (c *SAiProviderConfig) ResolvedBaseURL() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.BaseURL)
}

// ResolvedAPIKey returns config.api_key.
func (c *SAiProviderConfig) ResolvedAPIKey() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.APIKey)
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
	return c.ResolvedBaseURL() == "" && c.ResolvedAPIKey() == ""
}

type AiProviderListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	ProviderKey       string `json:"provider_key"`
	LlmDeploymentId   string `json:"llm_deployment_id"`
	LlmId             string `json:"llm_id"`
}

type AiProviderCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
}

type AiProviderUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
	Enabled         *bool              `json:"enabled"`
}

type AiProviderDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	ProviderKey     string             `json:"provider_key"`
	Config          *SAiProviderConfig `json:"config"`
	LlmDeploymentId string             `json:"llm_deployment_id"`
	LlmId           string             `json:"llm_id"`
}
