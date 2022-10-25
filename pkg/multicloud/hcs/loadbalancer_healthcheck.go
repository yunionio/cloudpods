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

package hcs

type SElbHealthCheck struct {
	region *SRegion

	Name          string `json:"name"`
	AdminStateUp  bool   `json:"admin_state_up"`
	TenantId      string `json:"tenant_id"`
	ProjectId     string `json:"project_id"`
	DomainName    string `json:"domain_name"`
	Delay         int    `json:"delay"`
	ExpectedCodes string `json:"expected_codes"`
	MaxRetries    int    `json:"max_retries"`
	HTTPMethod    string `json:"http_method"`
	Timeout       int    `json:"timeout"`
	Pools         []Pool `json:"pools"`
	URLPath       string `json:"url_path"`
	Type          string `json:"type"`
	Id            string `json:"id"`
	MonitorPort   int    `json:"monitor_port"`
}
