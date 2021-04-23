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
	"github.com/aws/aws-sdk-go/service/wafv2"

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
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	ret := []SWafRuleGroup{}
	input := wafv2.ListAvailableManagedRuleGroupsInput{}
	input.SetScope(scope)
	for {
		resp, err := client.ListAvailableManagedRuleGroups(&input)
		if err != nil {
			return nil, errors.Wrapf(err, "ListAvailableManagedRuleGroups")
		}
		part := []SWafRuleGroup{}
		jsonutils.Update(&part, resp.ManagedRuleGroups)
		ret = append(ret, part...)
		if resp.NextMarker == nil || len(*resp.NextMarker) == 0 {
			break
		}
		input.SetNextMarker(*resp.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) DescribeManagedRuleGroup(name, scope, vendorName string) (*SWafRuleGroup, error) {
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.DescribeManagedRuleGroupInput{}
	input.SetName(name)
	input.SetScope(scope)
	input.SetVendorName(vendorName)
	resp, err := client.DescribeManagedRuleGroup(&input)
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
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	ret := []SWafRuleGroup{}
	input := wafv2.ListRuleGroupsInput{}
	input.SetScope(scope)
	for {
		resp, err := client.ListRuleGroups(&input)
		if err != nil {
			return nil, errors.Wrapf(err, "ListRuleGroups")
		}
		part := []SWafRuleGroup{}
		jsonutils.Update(&part, resp.RuleGroups)
		ret = append(ret, part...)
		if resp.NextMarker == nil || len(*resp.NextMarker) == 0 {
			break
		}
		input.SetNextMarker(*resp.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) GetRuleGroup(id, name, scope string) (*SWafRuleGroup, error) {
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.GetRuleGroupInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	resp, err := client.GetRuleGroup(&input)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRuleGroup")
	}
	ret := &SWafRuleGroup{}
	return ret, jsonutils.Update(ret, resp.RuleGroup)
}

func (self *SRegion) DeleteRuleGroup(id, name, scope, lockToken string) error {
	client, err := self.getWafClient()
	if err != nil {
		return errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.DeleteRuleGroupInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	input.SetLockToken(lockToken)
	_, err = client.DeleteRuleGroup(&input)
	return errors.Wrapf(err, "DeleteRuleGroup")
}
