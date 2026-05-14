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

type AiModelListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	AiProviderId string `json:"ai_provider_id"`
	ModelKey     string `json:"model_key"`
}

type AiModelCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	AiProviderId string `json:"ai_provider_id"`
	ModelKey     string `json:"model_key"`
}

type AiModelUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	AiProviderId string `json:"ai_provider_id"`
	ModelKey     string `json:"model_key"`
	Enabled      *bool  `json:"enabled"`
}

type AiModelDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	AiProviderId   string `json:"ai_provider_id"`
	AiProviderName string `json:"ai_provider_name"`
	ModelKey       string `json:"model_key"`
}
