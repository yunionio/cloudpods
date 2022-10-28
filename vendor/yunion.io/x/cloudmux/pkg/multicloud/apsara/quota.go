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

package apsara

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type ValueItem struct {
	Value        string
	DiskCategory string
}

type AttributeValues struct {
	ValueItem []ValueItem
}

type SAccountAttributeItem struct {
	AttributeValues AttributeValues
	AttributeName   string
}

type SQuota struct {
	Name      string
	UsedCount int
	MaxCount  int
}

func (q *SQuota) GetGlobalId() string {
	return q.Name
}

func (q *SQuota) GetDesc() string {
	return q.Name
}

func (q *SQuota) GetQuotaType() string {
	return q.Name
}

func (q *SQuota) GetMaxQuotaCount() int {
	return q.MaxCount
}

func (q *SQuota) GetCurrentQuotaUsedCount() int {
	return q.UsedCount
}

func (region *SRegion) GetAccountAttributes() ([]SAccountAttributeItem, error) {
	params := map[string]string{
		"RegionId": region.RegionId,
	}
	resp, err := region.ecsRequest("DescribeAccountAttributes", params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsRequest")
	}
	quotas := []SAccountAttributeItem{}
	err = resp.Unmarshal(&quotas, "AccountAttributeItems", "AccountAttributeItem")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return quotas, nil
}

func (region *SRegion) GetQuotas() ([]SQuota, error) {
	attrs, err := region.GetAccountAttributes()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccountAttributes")
	}
	quotas := map[string]SQuota{}
	for _, attr := range attrs {
		for _, item := range attr.AttributeValues.ValueItem {
			value, err := strconv.ParseInt(item.Value, 10, 64)
			if err != nil {
				continue
			}
			used := false
			name := attr.AttributeName
			if strings.HasPrefix(name, "used-") {
				used = true
				name = strings.TrimPrefix(name, "used-")
			}
			if len(item.DiskCategory) > 0 {
				name = fmt.Sprintf("%s/%s", name, item.DiskCategory)
			}
			if _, ok := quotas[name]; !ok {
				quotas[name] = SQuota{
					Name:      name,
					UsedCount: -1,
				}
			}
			quota := quotas[name]
			if used {
				quota.UsedCount = int(value)
			} else {
				quota.MaxCount = int(value)
			}
			quotas[name] = quota
		}
	}
	ret := []SQuota{}
	for _, quota := range quotas {
		ret = append(ret, quota)
	}
	return ret, nil
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
