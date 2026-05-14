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
	"yunion.io/x/onecloud/pkg/apis"
)

type AiRoutingModelListInput struct {
	apis.StandaloneResourceListInput

	AiRoutingId  string `json:"ai_routing_id"`
	AiProviderId string `json:"ai_provider_id"`
	AiModelId    string `json:"ai_model_id"`
	Enabled      *bool  `json:"enabled"`
}

type AiRoutingModelCreateInput struct {
	apis.StandaloneResourceCreateInput

	AiRoutingId  string `json:"ai_routing_id"`
	AiProviderId string `json:"ai_provider_id"`
	AiModelId    string `json:"ai_model_id"`
	Priority     int    `json:"priority"`
	ModelPattern string `json:"model_pattern"`
	Enabled      *bool  `json:"enabled"`
}

type AiRoutingModelUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	AiRoutingId  string `json:"ai_routing_id,omitempty"`
	AiProviderId string `json:"ai_provider_id,omitempty"`
	AiModelId    string `json:"ai_model_id,omitempty"`
	Priority     int    `json:"priority,omitzero"`
	ModelPattern string `json:"model_pattern,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type AiRoutingModelDetails struct {
	apis.StandaloneResourceDetails

	AiRoutingId  string `json:"ai_routing_id"`
	AiProviderId string `json:"ai_provider_id"`
	AiModelId    string `json:"ai_model_id"`
	Priority     int    `json:"priority"`
	ModelPattern string `json:"model_pattern"`
	Enabled      bool   `json:"enabled"`
}
