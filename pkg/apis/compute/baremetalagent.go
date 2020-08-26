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

package compute

import "yunion.io/x/onecloud/pkg/apis"

type BaremetalagentDetails struct {
	apis.StandaloneResourceDetails
	ZoneResourceInfo

	SBaremetalagent
}

type BaremetalagentCreateInput struct {
	apis.StandaloneResourceCreateInput

	ZoneResourceInput

	AccessIp   string `json:"access_ip"`
	ManagerUri string `json:"manager_uri"`
	AgentType  string `json:"agent_type"`
	Version    string `json:"version"`
}

type BaremetalagentUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	ZoneResourceInput

	AccessIp   string `json:"access_ip"`
	ManagerUri string `json:"manager_uri"`
	AgentType  string `json:"agent_type"`
	Version    string `json:"version"`
}
