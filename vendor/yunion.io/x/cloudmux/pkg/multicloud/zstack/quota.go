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

package zstack

import (
	"net/url"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

/*
   "uuid":"132e0ad32b2242c4a648440be917ec4a",
   "name":"pci.num",
   "identityUuid":"2dce5dc485554d21a3796500c1db007a",
   "identityType":"AccountVO",
   "value":0,
   "lastOpDate":"Dec 28, 2019 2:26:03 PM",
   "createDate":"Dec 28, 2019 2:26:03 PM"
*/

type SQuota struct {
	Uuid         string
	Name         string
	IdentityUuid string
	IdentityType string
	Value        int
}

func (q *SQuota) GetGlobalId() string {
	return q.Uuid
}

func (q *SQuota) GetName() string {
	return q.Name
}

func (q *SQuota) GetDesc() string {
	return ""
}

func (q *SQuota) GetQuotaType() string {
	return q.Name
}

func (q *SQuota) GetMaxQuotaCount() int {
	return q.Value
}

func (q *SQuota) GetCurrentQuotaUsedCount() int {
	return -1
}

func (region *SRegion) GetQuotas() ([]SQuota, error) {
	quotas := []SQuota{}
	params := url.Values{}
	err := region.client.listAll("accounts/quotas", params, &quotas)
	if err != nil {
		return nil, err
	}
	return quotas, nil
}

type SUserAccount struct {
	Uuid string
}

func (region *SRegion) GetUserAccount(name string) ([]SUserAccount, error) {
	users := []SUserAccount{}
	params := url.Values{}
	if len(name) > 0 {
		params.Add("q", "name="+name)
	}
	err := region.client.listAll("accounts/users", params, &users)
	if err != nil {
		return nil, errors.Wrap(err, "users.list")
	}
	return users, nil
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
