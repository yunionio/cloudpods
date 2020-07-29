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

package yunionconf

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type ParameterListInput struct {
	apis.ResourceBaseListInput

	NamespaceId string `json:"namespace_id"`

	// 服务名称或ID
	ServiceId string `json:"service_id"`

	// Deprecated
	// swagger:ignore
	Service string `json:"service" "yunion:deprecated-by":"service_id"`

	// 用户名称或ID
	UserId string `json:"user_id"`

	// Deprecated
	// swagger:ignore
	User string `json:"user" "yunion:deprecated-by":"user_id"`

	// filter by name
	Name []string `json:"name"`
}
