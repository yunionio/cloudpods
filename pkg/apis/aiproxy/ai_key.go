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

// SAiKeyRouting constrains which client request models may use an ai_key, and relative priority.
// Matching uses the JSON body "model" string (same as ai_routing.model_pattern: exact match, case-insensitive,
// or prefix* glob when the pattern ends with "*").
//
// AllowedModelKeys: when non-empty, the requested model must match at least one entry.
// When empty, any model is allowed unless blocked by BlockedModelKeys.
//
// BlockedModelKeys: requested model must not match any entry (evaluated after allow-list pass).
//
// Weight in routing is legacy; prefer the ai_key.weight column. When ai_key.weight is unset (0), routing.weight is used.
type SAiKeyRouting struct {
	AllowedModelKeys []string `json:"allowed_model_keys,omitempty"`
	BlockedModelKeys []string `json:"blocked_model_keys,omitempty"`
	Weight           int      `json:"weight,omitempty"`
}

// String implements gotypes.ISerializable for sqlchemy JSON/compound columns.
func (r *SAiKeyRouting) String() string {
	if r == nil {
		return "{}"
	}
	b, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// IsZero implements gotypes.ISerializable.
func (r *SAiKeyRouting) IsZero() bool {
	if r == nil {
		return true
	}
	return len(r.AllowedModelKeys) == 0 && len(r.BlockedModelKeys) == 0 && r.Weight == 0
}

type AiKeyListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	AiProviderId string `json:"ai_provider_id"`
}

type AiKeyCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	AiProviderId string         `json:"ai_provider_id"`
	Secret       string         `json:"secret"`
	Weight       int            `json:"weight"`
	Routing      *SAiKeyRouting `json:"routing"`
}

type AiKeyUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	AiProviderId string         `json:"ai_provider_id"`
	Secret       string         `json:"secret"`
	Weight       int            `json:"weight,omitzero"`
	Routing      *SAiKeyRouting `json:"routing"`
	Enabled      *bool          `json:"enabled"`
}

type AiKeyDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	AiProviderId   string         `json:"ai_provider_id"`
	AiProviderName string         `json:"ai_provider_name"`
	Weight         int            `json:"weight"`
	Routing        *SAiKeyRouting `json:"routing"`
}
