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

package aws

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SWafRuleGroup struct {
	Description string
	Name        string
	VendorName  string
	Capacity    int `json:"Capacity"`
	Rules       []SWafRule
}

func (self *SRegion) ListAvailableManagedRuleGroups(scope string) ([]SWafRuleGroup, error) {
	if scope == SCOPE_CLOUDFRONT && self.RegionId != "us-east-1" {
		return []SWafRuleGroup{}, nil
	}
	ret := []SWafRuleGroup{}
	params := map[string]interface{}{"Scope": scope}
	for {
		resp := struct {
			ManagedRuleGroups []SWafRuleGroup
			NextMarker        string
		}{}
		err := self.wafRequest("ListAvailableManagedRuleGroups", params, &resp)
		if err != nil {
			return nil, errors.Wrapf(err, "ListAvailableManagedRuleGroups")
		}
		ret = append(ret, resp.ManagedRuleGroups...)
		if len(resp.NextMarker) == 0 {
			break
		}
		params["NextMarker"] = resp.NextMarker
	}
	return ret, nil
}

func (self *SRegion) DescribeManagedRuleGroup(name, scope, vendorName string) (*SWafRuleGroup, error) {
	params := map[string]interface{}{
		"Name":       name,
		"Scope":      scope,
		"VendorName": vendorName,
	}
	resp := map[string]interface{}{}
	err := self.wafRequest("DescribeManagedRuleGroup", params, &resp)
	if err != nil {
		return nil, err
	}
	ret := &SWafRuleGroup{
		Name:       name,
		VendorName: vendorName,
	}
	return ret, jsonutils.Update(ret, resp)
}

func (self *SRegion) ListRuleGroups(scope string) ([]SWafRuleGroup, error) {
	if scope == SCOPE_CLOUDFRONT && self.RegionId != "us-east-1" {
		return []SWafRuleGroup{}, nil
	}
	ret := []SWafRuleGroup{}
	params := map[string]interface{}{"Scope": scope}
	for {
		resp := struct {
			RuleGroups []SWafRuleGroup
			NextMarker string
		}{}
		err := self.wafRequest("ListRuleGroups", params, &resp)
		if err != nil {
			return nil, errors.Wrapf(err, "ListRuleGroups")
		}
		ret = append(ret, resp.RuleGroups...)
		if len(resp.NextMarker) == 0 {
			break
		}
		params["NextMarker"] = resp.NextMarker
	}
	return ret, nil
}

func (self *SRegion) GetRuleGroup(id, name, scope string) (*SWafRuleGroup, error) {
	params := map[string]interface{}{
		"Id":    id,
		"Name":  name,
		"Scope": scope,
	}
	resp := struct {
		RuleGroup map[string]interface{}
	}{}
	err := self.wafRequest("GetRuleGroup", params, &resp)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRuleGroup")
	}
	ret := &SWafRuleGroup{}
	return ret, jsonutils.Update(ret, resp.RuleGroup)
}

func (self *SRegion) DeleteRuleGroup(id, name, scope, lockToken string) error {
	params := map[string]interface{}{
		"Id":        id,
		"Name":      name,
		"Scope":     scope,
		"LockToken": lockToken,
	}
	err := self.wafRequest("DeleteRuleGroup", params, nil)
	return errors.Wrapf(err, "DeleteRuleGroup")
}
