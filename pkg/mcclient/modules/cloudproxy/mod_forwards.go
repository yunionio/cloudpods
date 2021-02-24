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

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ForwardManager struct {
	modulebase.ResourceManager
}

var (
	Forwards ForwardManager
)

func init() {
	Forwards = ForwardManager{
		NewCloudProxyManager(
			"forward",
			"forwards",
			[]string{
				"id",
				"name",
				"type",
				"remote_addr",
				"remote_port",
				"bind_port",
				"last_seen",
			},
			[]string{
				"proxy_endpoint_id",
				"proxy_agent_id",
				"bind_port_req",
				"tenant",
			},
		),
	}
	registerV2(&Forwards)
}
