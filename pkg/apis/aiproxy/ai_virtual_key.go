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

	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/apis"
)

// SAiVirtualKeyLimits constrains which catalog providers a virtual key may route to,
// caps client max_tokens, and configures request rate (approximate per-minute token bucket).
//
// AllowedAiProviderIds: when non-empty, the resolved ai_provider must match one entry by id or name (case-insensitive for name).
// When empty, any provider allowed by routing applies.
//
// MaxTokensPerRequest: when > 0, caps the JSON body max_tokens (missing max_tokens is set to this value).
//
// RequestsPerMinute: when > 0, enforces an approximate per-minute limit per virtual key row (in-process; multi-replica deployments need external limiting).
type SAiVirtualKeyLimits struct {
	AllowedAiProviderIds []string `json:"allowed_ai_provider_ids,omitempty"`
	MaxTokensPerRequest  int      `json:"max_tokens_per_request,omitempty"`
	RequestsPerMinute    int      `json:"requests_per_minute,omitempty"`
}

// String implements gotypes.ISerializable for sqlchemy JSON columns.
func (l *SAiVirtualKeyLimits) String() string {
	if l == nil {
		return "{}"
	}
	b, err := json.Marshal(l)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// IsZero implements gotypes.ISerializable.
func (l *SAiVirtualKeyLimits) IsZero() bool {
	if l == nil {
		return true
	}
	return len(l.AllowedAiProviderIds) == 0 && l.MaxTokensPerRequest == 0 && l.RequestsPerMinute == 0
}

type AiVirtualKeyListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	VirtualKey string `json:"virtual_key"`
	UserId     string `json:"user_id"`
}

type AiVirtualKeyCreateInput struct {
	apis.VirtualResourceCreateInput

	// OwnerId is the owning user; defaults to the creating user when empty.
	OwnerId string `json:"owner_id"`
	// VirtualKey is optional; when empty a unique sk- prefixed key is generated.
	VirtualKey string               `json:"virtual_key"`
	Limits     *SAiVirtualKeyLimits `json:"limits"`
	Enabled    tristate.TriState    `json:"enabled"`
}

type AiVirtualKeyUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	OwnerId    string               `json:"owner_id"`
	VirtualKey string               `json:"virtual_key"`
	Limits     *SAiVirtualKeyLimits `json:"limits"`
	Enabled    tristate.TriState    `json:"enabled"`
}

type AiVirtualKeyDetails struct {
	apis.VirtualResourceDetails

	OwnerId    string               `json:"owner_id"`
	OwnerName  string               `json:"owner_name"`
	VirtualKey string               `json:"virtual_key"`
	Limits     *SAiVirtualKeyLimits `json:"limits"`
	Enabled    bool                 `json:"enabled"`
}
