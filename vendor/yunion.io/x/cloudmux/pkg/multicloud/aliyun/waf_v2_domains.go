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

package aliyun

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SWafDomainV2 struct {
	multicloud.SResourceBase
	AliyunTags
	region *SRegion
	insId  string

	Domain      string
	Status      string
	Cname       string
	ListenPorts struct {
		Http  []int
		Https []int
	}
	Backeds struct {
		Http []struct {
			Backend string
		}
		Https []struct {
			Backend string
		}
	}
	Listen struct {
		HttpPorts     []int
		HttpsPorts    []int
		Http2Enabled  bool
		CertId        string
		XffHeaderMode int
		XffHeaders    []string
	}
	Redirect struct {
		Backends []struct {
			Backend string
		}
	}
	CertDetail *struct {
		Id   string
		Name string
	}
	ResourceManagerResourceGroupId string
}

func (self *SWafDomainV2) GetId() string {
	return self.Domain
}

func (self *SWafDomainV2) GetStatus() string {
	return api.WAF_STATUS_AVAILABLE
}

func (self *SWafDomainV2) GetWafType() cloudprovider.TWafType {
	return cloudprovider.WafTypeDefault
}

func (self *SWafDomainV2) GetEnabled() bool {
	return true
}

func (self *SWafDomainV2) GetName() string {
	return self.Domain
}

func (self *SWafDomainV2) GetGlobalId() string {
	return self.Domain
}

func (self *SWafDomainV2) GetIsAccessProduct() bool {
	return self.Listen.XffHeaderMode != 0
}

func (self *SWafDomainV2) GetAccessHeaders() []string {
	return self.Listen.XffHeaders
}

func (self *SWafDomainV2) GetHttpPorts() []int {
	return self.Listen.HttpPorts
}

func (self *SWafDomainV2) GetHttpsPorts() []int {
	return self.Listen.HttpsPorts
}

func (self *SWafDomainV2) GetCname() string {
	return self.Cname
}

func (self *SWafDomainV2) GetCertId() string {
	if self.CertDetail == nil {
		self.Refresh()
	}
	if self.CertDetail != nil {
		return self.CertDetail.Id
	}
	return ""
}

func (self *SWafDomainV2) GetCertName() string {
	if self.CertDetail == nil {
		self.Refresh()
	}
	if self.CertDetail != nil {
		return self.CertDetail.Name
	}
	return ""
}

func (self *SWafDomainV2) GetUpstreamPort() int {
	return 0
}

func (self *SWafDomainV2) GetUpstreamScheme() string {
	return ""
}

func (self *SWafDomainV2) GetSourceIps() []string {
	ret := []string{}
	for _, backend := range self.Redirect.Backends {
		ret = append(ret, backend.Backend)
	}
	return ret
}

func (self *SWafDomainV2) GetCcList() []string {
	return []string{}
}

func (self *SWafDomainV2) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWafDomainV2) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "AddRule")
}

func (self *SWafDomainV2) Refresh() error {
	domain, err := self.region.DescribeDomain(self.insId, self.Domain)
	if err != nil {
		return errors.Wrapf(err, "DescribeDomain")
	}
	return jsonutils.Update(self, domain)
}

func (self *SWafDomainV2) GetCloudResources() ([]cloudprovider.SCloudResource, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWafDomainV2) GetDefaultAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{
		Action:        cloudprovider.WafActionAllow,
		InsertHeaders: map[string]string{},
	}
}

func (self *SWafDomainV2) GetRules() ([]cloudprovider.ICloudWafRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) DescribeWafDomains(insId string) ([]SWafDomainV2, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": insId,
		"PageNumber": "1",
		"PageSize":   "50",
	}
	pageNum := 1
	ret := []SWafDomainV2{}
	for {
		resp, err := self.wafv2Request("DescribeDomains", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeDomains")
		}
		part := struct {
			TotalCount int
			Domains    []SWafDomainV2
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Domains...)
		if len(ret) >= part.TotalCount || len(part.Domains) == 0 {
			break
		}
		pageNum++
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
	}
	return ret, nil
}

func (self *SRegion) DescribeDomainV2(id, domain string) (*SWafDomainV2, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
		"Domain":     domain,
	}
	resp, err := self.wafv2Request("DescribeDomainDetail", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomainDetail")
	}
	ret := &SWafDomainV2{Domain: domain, region: self, insId: id}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}
