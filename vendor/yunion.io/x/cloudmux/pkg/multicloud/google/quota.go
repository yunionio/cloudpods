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

package google

import (
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SQuota struct {
	Metric string
	Limit  int
	Usage  int
	Owner  string
}

func (q *SQuota) GetGlobalId() string {
	return strings.ToLower(q.Metric)
}

func (q *SQuota) GetName() string {
	return q.Metric
}

func (q *SQuota) GetDesc() string {
	return q.Metric
}

func (q *SQuota) GetQuotaType() string {
	return q.Metric
}

func (q *SQuota) GetMaxQuotaCount() int {
	return q.Limit
}

func (q *SQuota) GetCurrentQuotaUsedCount() int {
	return q.Usage
}

func (region *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	ret := []cloudprovider.ICloudQuota{}
	for i := range region.Quotas {
		ret = append(ret, &region.Quotas[i])
	}
	return ret, nil
}
