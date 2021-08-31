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

package hcso

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845574.html
// 租户列表
type SDomain struct {
	Contacts       string `json:"contacts"`
	Description    string `json:"description"`
	Enabled        bool   `json:"enabled"`
	EnterpriseName string `json:"enterpriseName"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Tagflag        int    `json:"tagflag"`
}

func (self *SHuaweiClient) GetDomains() ([]SDomain, error) {
	huawei, _ := self.newGeneralAPIClient()
	domains := make([]SDomain, 0)
	err := doListAll(huawei.Domains.List, nil, &domains)
	return domains, err
}

func (self *SHuaweiClient) getEnabledDomains() ([]SDomain, error) {
	domains, err := self.GetDomains()

	enabledDomains := make([]SDomain, 0)
	for i := range domains {
		if domains[i].Enabled {
			enabledDomains = append(enabledDomains, domains[i])
		}
	}

	return enabledDomains, err
}
