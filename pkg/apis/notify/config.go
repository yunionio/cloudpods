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

package notify

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type ConfigCreateInput struct {
	apis.StandaloneResourceCreateInput

	// description: config type
	// required: true
	// example: feishu
	Type string `json:"type"`

	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content jsonutils.JSONObject `json:"content"`
}

type ConfigUpdateInput struct {
	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content jsonutils.JSONObject `json:"content"`
}

type ConfigDetails struct {
	apis.StandaloneResourceDetails

	SConfig
}

type ConfigListInput struct {
	apis.StandaloneResourceListInput
	Type string `json:"type"`
}

type ConfigValidateInput struct {
	// description: config type
	// required: true
	// example: feishu
	Type string `json:"type"`

	// description: config content
	// required: true
	// example: {"app_id": "123456", "app_secret": "feishu_nihao"}
	Content jsonutils.JSONObject `json:"content"`
}

type ConfigValidateOutput struct {
	IsValid bool   `json:"is_valid"`
	Message string `json:"message"`
}

type ConfigManagerGetTypesInput struct {
	// description: Filter about robot
	// enum: no,yes,only
	// example: yes
	Robot string `json:"robot"`
}

type ConfigManagerGetTypesOutput struct {
	Types []string `json:"types"`
}
