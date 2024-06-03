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

package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SWafInstance struct {
	multicloud.SResourceBase
	QcloudTags
	region *SRegion

	AccessStatus      string
	AlbType           string
	ApiStatus         int
	AppId             string
	BotStatus         string
	CCList            []string
	CdcClusters       string
	CloudType         string
	ClsStatus         string
	Cname             string
	IsCdn             *int
	CreateTime        time.Time
	Domain            string
	DomainId          string
	Edition           string
	Engine            int
	FlowMode          int
	InstanceId        string
	InstanceName      string
	UpstreamScheme    *string
	HttpsUpstreamPort *int
	Ipv6Status        int
	Level             int
	Mode              int
	IpHeaders         *[]string
	Note              string
	Ports             []struct {
		NginxServerId    string
		Port             int
		Protocol         string
		UpstreamPort     int
		UpstreamProtocol string
	}
	PostCKafkaStatus   int
	PostCLSStatus      string
	Region             string
	RsList             []string
	SgDetail           string
	SrcList            []string
	Status             int
	UpstreamDomainList []string
	SSLId              *string
}

func (self *SWafInstance) GetName() string {
	return self.Domain
}

func (self *SWafInstance) GetGlobalId() string {
	return self.Domain
}

func (self *SWafInstance) GetId() string {
	return self.Domain
}

func (self *SWafInstance) GetWafType() cloudprovider.TWafType {
	switch self.Edition {
	case "sparta-waf":
		return cloudprovider.WafTypeSaaS
	case "clb-waf", "cdc-clb-waf":
		return cloudprovider.WafTypeLoadbalancer
	default:
		return cloudprovider.TWafType(self.Edition)
	}
}

func (self *SWafInstance) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SWafInstance) GetDefaultAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{}
}

func (self *SWafInstance) GetRules() ([]cloudprovider.ICloudWafRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWafInstance) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWafInstance) GetCloudResources() ([]cloudprovider.SCloudResource, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SWafInstance) Refresh() error {
	waf, err := self.region.GetWafInstance(self.Domain, self.DomainId, self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, waf)
}

func (self *SWafInstance) GetEnabled() bool {
	return true
}

func (self *SWafInstance) GetStatus() string {
	return apis.STATUS_AVAILABLE
}

func (self *SWafInstance) GetCname() string {
	return self.Cname
}

func (self *SWafInstance) GetIsAccessProduct() bool {
	if self.IsCdn == nil {
		err := self.Refresh()
		if err != nil {
			return false
		}
	}
	if self.IsCdn != nil {
		return *self.IsCdn != 0
	}
	return false
}

func (self *SWafInstance) GetAccessHeaders() []string {
	if !self.GetIsAccessProduct() {
		return []string{}
	}
	switch *self.IsCdn {
	case 1:
		return []string{"X-Forwarded-For"}
	default:
		if self.IpHeaders != nil {
			return *self.IpHeaders
		}
	}
	return []string{}
}

func (self *SWafInstance) GetHttpPorts() []int {
	ret := []int{}
	for _, port := range self.Ports {
		if port.Protocol == "http" {
			ret = append(ret, port.Port)
		}
	}
	return ret
}

func (self *SWafInstance) GetHttpsPorts() []int {
	ret := []int{}
	for _, port := range self.Ports {
		if port.Protocol == "https" {
			ret = append(ret, port.Port)
		}
	}
	return ret
}

func (self *SWafInstance) GetCertId() string {
	if self.SSLId == nil {
		self.Refresh()
	}
	if self.SSLId != nil {
		return *self.SSLId
	}
	return ""
}

func (self *SWafInstance) GetCertName() string {
	sslId := self.GetCertId()
	if len(sslId) > 0 {
		cert, err := self.region.client.GetCertificate(sslId)
		if err != nil {
			return ""
		}
		return cert.GetName()
	}
	return ""
}

func (self *SWafInstance) GetSourceIps() []string {
	return append(self.SrcList, self.UpstreamDomainList...)
}

func (self *SWafInstance) GetCcList() []string {
	return self.CCList
}

func (self *SWafInstance) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SWafInstance) GetUpstreamScheme() string {
	if self.UpstreamScheme == nil {
		self.Refresh()
	}
	if self.UpstreamScheme != nil {
		return *self.UpstreamScheme
	}
	return ""
}

func (self *SWafInstance) GetUpstreamPort() int {
	if self.HttpsUpstreamPort == nil {
		self.Refresh()
	}
	if self.HttpsUpstreamPort != nil {
		return *self.HttpsUpstreamPort
	}
	return 0
}

func (self *SRegion) GetWafInstances() ([]SWafInstance, error) {
	params := map[string]string{
		"Limit": "1000",
	}
	ret := []SWafInstance{}
	for {
		params["Offset"] = fmt.Sprintf("%d", len(ret))
		resp, err := self.wafRequest("DescribeDomains", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Domains []SWafInstance
			Total   int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Domains...)
		if len(ret) >= part.Total || len(part.Domains) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SRegion) GetICloudWafInstances() ([]cloudprovider.ICloudWafInstance, error) {
	wafs, err := self.GetWafInstances()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWafInstance{}
	for i := range wafs {
		wafs[i].region = self
		ret = append(ret, &wafs[i])
	}
	return ret, nil
}

func (self *SRegion) GetICloudWafInstanceById(id string) (cloudprovider.ICloudWafInstance, error) {
	wafs, err := self.GetWafInstances()
	if err != nil {
		return nil, err
	}
	for i := range wafs {
		wafs[i].region = self
		if wafs[i].GetGlobalId() == id {
			return &wafs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetWafInstance(domain, domainId, instanceId string) (*SWafInstance, error) {
	params := map[string]string{
		"Domain":     domain,
		"DomainId":   domainId,
		"InstanceId": instanceId,
	}
	resp, err := self.wafRequest("DescribeDomainDetailsSaas", params)
	if err != nil {
		return nil, err
	}
	ret := &SWafInstance{}
	err = resp.Unmarshal(ret, "DomainsPartInfo")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
