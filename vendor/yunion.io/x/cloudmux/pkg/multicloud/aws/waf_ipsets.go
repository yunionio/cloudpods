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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SWafIPSet struct {
	region      *SRegion
	scope       string
	Addresses   []string
	ARN         string
	Description string
	Id          string
	LockToken   string
	Name        string
}

func (self *SWafIPSet) GetName() string {
	return self.Name
}

func (self *SWafIPSet) GetDesc() string {
	return self.Description
}

func (self *SWafIPSet) GetGlobalId() string {
	return self.ARN
}

func (self *SWafIPSet) GetType() cloudprovider.TWafType {
	switch self.scope {
	case SCOPE_CLOUDFRONT:
		return cloudprovider.WafTypeCloudFront
	case SCOPE_REGIONAL:
		return cloudprovider.WafTypeRegional
	}
	return cloudprovider.TWafType(self.scope)
}

func (self *SWafIPSet) GetAddresses() cloudprovider.WafAddresses {
	if len(self.Addresses) == 0 {
		ipSet, err := self.region.GetIPSet(self.Id, self.Name, self.scope)
		if err != nil {
			return cloudprovider.WafAddresses{}
		}
		return ipSet.Addresses
	}
	return self.Addresses
}

func (self *SWafIPSet) Delete() error {
	return self.region.DeleteIPSet(self.Id, self.Name, self.scope, self.LockToken)
}

func (self *SRegion) ListIPSets(scope string) ([]SWafIPSet, error) {
	if scope == SCOPE_CLOUDFRONT && self.RegionId != "us-east-1" {
		return []SWafIPSet{}, nil
	}
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	ret := []SWafIPSet{}
	input := wafv2.ListIPSetsInput{}
	input.SetScope(scope)
	for {
		resp, err := client.ListIPSets(&input)
		if err != nil {
			return nil, errors.Wrapf(err, "ListIPSets")
		}
		part := []SWafIPSet{}
		jsonutils.Update(&part, resp.IPSets)
		ret = append(ret, part...)
		if resp.NextMarker == nil || len(*resp.NextMarker) == 0 {
			break
		}
		input.SetNextMarker(*resp.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) GetIPSet(id, name, scope string) (*SWafIPSet, error) {
	client, err := self.getWafClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.GetIPSetInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	resp, err := client.GetIPSet(&input)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIPSet")
	}
	ret := &SWafIPSet{LockToken: *resp.LockToken}
	return ret, jsonutils.Update(ret, resp.IPSet)
}

func (self *SRegion) DeleteIPSet(id, name, scope, lockToken string) error {
	client, err := self.getWafClient()
	if err != nil {
		return errors.Wrapf(err, "getWafClient")
	}
	input := wafv2.DeleteIPSetInput{}
	input.SetId(id)
	input.SetName(name)
	input.SetScope(scope)
	input.SetLockToken(lockToken)
	_, err = client.DeleteIPSet(&input)
	return errors.Wrapf(err, "DeleteIPSet")
}

func (self *SRegion) GetICloudWafIPSets() ([]cloudprovider.ICloudWafIPSet, error) {
	ret := []cloudprovider.ICloudWafIPSet{}
	for _, scope := range WAF_SCOPES {
		part, err := self.ListIPSets(scope)
		if err != nil {
			return nil, errors.Wrapf(err, "ListIPSets(%s)", scope)
		}
		for i := range part {
			part[i].scope = scope
			part[i].region = self
			ret = append(ret, &part[i])
		}
	}
	return ret, nil
}
