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

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SQuota struct {
	Min   int
	Quota int
	Type  string
	Used  int
}

func (q *SQuota) GetGlobalId() string {
	return q.Type
}

func (q *SQuota) GetQuotaType() string {
	return q.Type
}

func (q *SQuota) GetName() string {
	return q.Type
}

func (q *SQuota) GetDesc() string {
	return ""
}

func (q *SQuota) GetMaxQuotaCount() int {
	return q.Quota
}

func (q *SQuota) GetCurrentQuotaUsedCount() int {
	return q.Used
}

func (self *SRegion) GetQuotas() ([]SQuota, error) {
	quotas := []SQuota{}
	params := map[string]string{}
	result, err := self.ecsClient.Quotas.Get("", params)
	if err != nil {
		return nil, errors.Wrap(err, "Quotas.List")
	}

	err = result.Unmarshal(&quotas, "resources")
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}

	return quotas, nil
}

func (region *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	quotas, err := region.GetQuotas()
	if err != nil {
		return nil, errors.Wrap(err, "GetQuotas")
	}
	ret := []cloudprovider.ICloudQuota{}
	for i := range quotas {
		ret = append(ret, &quotas[i])
	}
	return ret, nil
}
