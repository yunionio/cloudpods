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

package openstack

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/version"
)

type SQuota struct {
	FixedIps           int
	Floatingips        int
	Networks           int
	Port               int
	RbacPolicy         int
	Router             int
	SecurityGroups     int
	SecurityGroupRules int
}

func (region *SRegion) GetQuota() (*SQuota, error) {
	_, resp, err := region.Get("compute", "/os-quota-sets/"+region.client.tokenCredential.GetTenantId(), "", nil)
	if err != nil {
		return nil, err
	}
	quota := &SQuota{}
	return quota, resp.Unmarshal(quota, "quota_set")
}

func (region *SRegion) SetQuota(quota *SQuota) error {
	_, maxVersion, _ := region.GetVersion("compute")
	params := map[string]map[string]interface{}{
		"quota_set": {
			"force": "True",
		},
	}

	if version.GE(maxVersion, "2.35") {
		if quota.Floatingips > 0 {
			params["quota_set"]["floating_ips"] = quota.Floatingips
		}

		if quota.SecurityGroups > 0 {
			params["quota_set"]["security_group"] = quota.SecurityGroups
		}

		if quota.SecurityGroupRules > 0 {
			params["quota_set"]["security_group_rules"] = quota.SecurityGroupRules
		}

		if quota.FixedIps > 0 {
			params["quota_set"]["fixed_ips"] = quota.FixedIps
		}

		if quota.Networks > 0 {
			params["quota_set"]["networks"] = quota.Networks
		}

	}

	_, _, err := region.Update("compute", "/os-quota-sets/"+region.client.tokenCredential.GetTenantId(), maxVersion, jsonutils.Marshal(params))
	return err
}
