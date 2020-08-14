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
	Service string `json:"service"`

	// Deprecated
	// swagger:ignore
	ServiceId string `json:"service_id" yunion-deprecated-by:"service"`

	// 用户名称或ID
	User string `json:"user"`

	// Deprecated
	// swagger:ignore
	UserId string `json:"user_id" yunion-deprecated-by:"user"`

	// filter by name
	Name []string `json:"name"`
}
