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

type AiRoutingListInput struct {
	apis.SharableVirtualResourceListInput
	apis.EnabledResourceBaseListInput

	ModelPattern  string `json:"model_pattern"`
	AiProxyNodeId string `json:"ai_proxy_node_id"`
}

// AiRoutingModelItem is one catalog model binding when creating ai_routing.
// Priority orders models within the routing (lower = higher priority). Weight is an alias for Priority.
type AiRoutingModelItem struct {
	AiProviderId string `json:"ai_provider_id"`
	AiModelId    string `json:"ai_model_id"`
	Priority     int    `json:"priority,omitempty"`
	Weight       int    `json:"weight,omitempty"`
	ModelPattern string `json:"model_pattern,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type AiRoutingCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	Priority      int                  `json:"priority"`
	ModelPattern  string               `json:"model_pattern"`
	AiProxyNodeId string               `json:"ai_proxy_node_id"`
	Models        []AiRoutingModelItem `json:"models"`
}

type AiRoutingUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	Priority      int    `json:"priority"`
	ModelPattern  string `json:"model_pattern"`
	AiProxyNodeId string `json:"ai_proxy_node_id"`
	Enabled       *bool  `json:"enabled"`
}

type AiRoutingDetails struct {
	apis.SharableVirtualResourceDetails

	Priority      int                     `json:"priority"`
	ModelPattern  string                  `json:"model_pattern"`
	AiProxyNodeId string                  `json:"ai_proxy_node_id"`
	Enabled       bool                    `json:"enabled"`
	RoutingModels []AiRoutingModelDetails `json:"routing_models,omitempty"`
}

// AiRoutingSetModelsInput replaces all ai_routing_models for an ai_routing.
type AiRoutingSetModelsInput struct {
	Models []AiRoutingModelItem `json:"models"`
}
