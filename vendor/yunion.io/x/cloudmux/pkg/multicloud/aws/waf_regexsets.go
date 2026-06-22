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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type RegularExpression struct {
	RegexString string
}

type SWafRegexSet struct {
	region                *SRegion
	scope                 string
	RegularExpressionList []RegularExpression
	ARN                   string
	Description           string
	Id                    string
	LockToken             string
	Name                  string
}

func (self *SWafRegexSet) GetName() string {
	return self.Name
}

func (self *SWafRegexSet) GetDesc() string {
	return self.Description
}

func (self *SWafRegexSet) GetGlobalId() string {
	return self.ARN
}

func (self *SWafRegexSet) GetType() cloudprovider.TWafType {
	switch self.scope {
	case SCOPE_REGIONAL:
		return cloudprovider.WafTypeRegional
	case SCOPE_CLOUDFRONT:
		return cloudprovider.WafTypeCloudFront
	default:
		return cloudprovider.TWafType(self.scope)
	}
}

func (self *SWafRegexSet) GetRegexPatterns() cloudprovider.WafRegexPatterns {
	if len(self.RegularExpressionList) == 0 {
		rSet, err := self.region.GetRegexSet(self.Id, self.Name, self.scope)
		if err != nil {
			return cloudprovider.WafRegexPatterns{}
		}
		jsonutils.Update(self, rSet)
	}
	ret := cloudprovider.WafRegexPatterns{}
	for _, r := range self.RegularExpressionList {
		ret = append(ret, r.RegexString)
	}
	return ret
}

func (self *SWafRegexSet) Delete() error {
	return self.region.DeleteRegexSet(self.Id, self.Name, self.scope, self.LockToken)
}

func (self *SRegion) ListRegexSets(scope string) ([]SWafRegexSet, error) {
	if scope == SCOPE_CLOUDFRONT && self.RegionId != "us-east-1" {
		return []SWafRegexSet{}, nil
	}
	ret := []SWafRegexSet{}
	params := map[string]interface{}{"Scope": scope}
	for {
		resp := struct {
			RegexPatternSets []SWafRegexSet
			NextMarker       string
		}{}
		err := self.wafRequest("ListRegexPatternSets", params, &resp)
		if err != nil {
			return nil, errors.Wrapf(err, "ListRegexPatternSets")
		}
		ret = append(ret, resp.RegexPatternSets...)
		if len(resp.NextMarker) == 0 {
			break
		}
		params["NextMarker"] = resp.NextMarker
	}
	return ret, nil
}

func (self *SRegion) GetRegexSet(id, name, scope string) (*SWafRegexSet, error) {
	params := map[string]interface{}{
		"Id":    id,
		"Name":  name,
		"Scope": scope,
	}
	resp := struct {
		RegexPatternSet map[string]interface{}
		LockToken       string
	}{}
	err := self.wafRequest("GetRegexPatternSet", params, &resp)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegexPatternSet")
	}
	ret := &SWafRegexSet{LockToken: resp.LockToken}
	return ret, jsonutils.Update(ret, resp.RegexPatternSet)
}

func (self *SRegion) DeleteRegexSet(id, name, scope, lockToken string) error {
	params := map[string]interface{}{
		"Id":        id,
		"Name":      name,
		"Scope":     scope,
		"LockToken": lockToken,
	}
	err := self.wafRequest("DeleteRegexPatternSet", params, nil)
	return errors.Wrapf(err, "DeleteRegexPatternSet")
}

func (self *SRegion) GetICloudWafRegexSets() ([]cloudprovider.ICloudWafRegexSet, error) {
	ret := []cloudprovider.ICloudWafRegexSet{}
	for _, scope := range WAF_SCOPES {
		part, err := self.ListRegexSets(scope)
		if err != nil {
			return nil, errors.Wrapf(err, "ListRegexSets(%s)", scope)
		}
		for i := range part {
			part[i].scope = scope
			part[i].region = self
			ret = append(ret, &part[i])
		}
	}
	return ret, nil
}
