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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type UsageName struct {
	Value          string
	LocalizedValue string
}

// {"value":[{"unit":"Count","currentValue":0,"limit":250,"name":{"value":"StorageAccounts","localizedValue":"Storage Accounts"}}]}
type SUsage struct {
	Unit         string
	CurrentValue int
	Limit        int
	Name         UsageName
}

func (u *SUsage) GetGlobalId() string {
	return strings.ToLower(u.Name.Value)
}

func (u *SUsage) GetQuotaType() string {
	return u.Name.Value
}

func (u *SUsage) GetDesc() string {
	return u.Name.LocalizedValue
}

func (u *SUsage) GetMaxQuotaCount() int {
	return u.Limit
}

func (u *SUsage) GetCurrentQuotaUsedCount() int {
	return u.CurrentValue
}

func (region *SRegion) GetUsage(resourceType string) ([]SUsage, error) {
	usage := []SUsage{}
	resource := fmt.Sprintf("%s/locations/%s/usages", resourceType, region.Name)
	err := region.client.list(resource, url.Values{}, &usage)
	if err != nil {
		return nil, errors.Wrapf(err, "ListAll(%s)", resource)
	}
	return usage, nil
}

func (region *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	ret := []cloudprovider.ICloudQuota{}
	for _, resourceType := range []string{"Microsoft.Network", "Microsoft.Storage", "Microsoft.Compute"} {
		usages, err := region.GetUsage(resourceType)
		if err != nil {
			return nil, errors.Wrap(err, "GetUsage")
		}
		for i := range usages {
			ret = append(ret, &usages[i])
		}
	}
	return ret, nil
}
