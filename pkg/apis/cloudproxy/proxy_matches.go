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

package cloudproxy

import "yunion.io/x/onecloud/pkg/apis"

type ProxyMatchListInput struct {
	apis.VirtualResourceListInput

	// 代理节点（ID或Name）
	ProxyEndpointId string `json:"proxy_endpoint_id"`
	// swagger:ignore
	// Deprecated
	// Filter by proxy endpoint Id
	ProxyEndpoint string `json:"proxy_endpoint" yunion-deprecated-by:"proxy_endpoint_id"`
}
