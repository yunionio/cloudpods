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
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SCdnDomainNames struct {
	DomainNames []string `json:"domainNames"`
}

type SDomainInfo struct {
	DomainCname string    `json:"DomainCname"`
	Status      string    `json:"Status"`
	CreateTime  time.Time `json:"CreateTime"`
	UpdateTime  time.Time `json:"UpdateTime"`
	DomainName  string    `json:"DomainName"`
}

type SCdnDomainInfos struct {
	DomainInfo []SDomainInfo `json:"domainInfo"`
}

type SCdnDomainsData struct {
	Source      string          `json:"Source"`
	Domains     SCdnDomainNames `json:"Domains"`
	DomainInfos SCdnDomainInfos `json:"DomainInfos"`
}

type SCdnDomainsList struct {
	DomainsData []SCdnDomainsData `json:"DomainsData"`
}

type SCdnSource struct {
	Port     int    `json:"Port"`
	Weight   string `json:"Weight"`
	Type     string `json:"Type"`
	Content  string `json:"Content"`
	Priority string `json:"Priority"`
}

type SCdnSources struct {
	Source []SCdnSource `json:"Source"`
}

type SCdnDomain struct {
	multicloud.SCDNDomainBase

	client *SAliyunClient

	Cname           string     `json:"Cname"`
	Description     string     `json:"Description"`
	CdnType         string     `json:"CdnType"`
	ResourceGroupID string     `json:"ResourceGroupId"`
	DomainStatus    string     `json:"DomainStatus"`
	SslProtocol     string     `json:"SslProtocol"`
	DomainName      string     `json:"DomainName"`
	Coverage        string     `json:"Coverage"`
	Sources         SCdnSource `json:"Sources"`
	GmtModified     string     `json:"GmtModified"`
	Sandbox         string     `json:"Sandbox"`
	GmtCreated      time.Time  `json:"GmtCreated"`
	SourceModels    struct {
		SourceModel []struct {
			Content  string
			Type     string
			Port     int
			Enabled  string
			Priority int
			Weight   string
		}
	}
}

func (self *SCdnDomain) GetArea() string {
	switch self.Coverage {
	case "domestic":
		return api.CDN_DOMAIN_AREA_MAINLAND
	case "global":
		return api.CDN_DOMAIN_AREA_GLOBAL
	case "overseas":
		return api.CDN_DOMAIN_AREA_OVERSEAS
	default:
		return self.Coverage
	}
}

func (self *SCdnDomain) GetCname() string {
	return self.Cname
}

func (self *SCdnDomain) GetEnabled() bool {
	return self.DomainStatus != "offline"
}

func (self *SCdnDomain) GetId() string {
	return self.DomainName
}

func (self *SCdnDomain) GetGlobalId() string {
	return self.DomainName
}

func (self *SCdnDomain) GetName() string {
	return self.DomainName
}

func (self *SCdnDomain) Refresh() error {
	domain, err := self.client.GetCdnDomain(self.DomainName)
	if err != nil {
		return errors.Wrapf(err, "GetCdnDomain")
	}
	self.DomainStatus = domain.DomainStatus
	return nil
}

func (self *SCdnDomain) GetTags() (map[string]string, error) {
	_, tags, err := self.client.GetRegion("").ListSysAndUserTags(ALIYUN_SERVICE_CDN, "DOMAIN", self.DomainName)
	if err != nil {
		return nil, errors.Wrapf(err, "ListTags")
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps, nil
}

func (self *SCdnDomain) GetSysTags() map[string]string {
	tags, _, err := self.client.GetRegion("").ListSysAndUserTags(ALIYUN_SERVICE_CDN, "DOMAIN", self.DomainName)
	if err != nil {
		return nil
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps
}

func (self *SCdnDomain) SetTags(tags map[string]string, replace bool) error {
	return self.client.GetRegion("").SetResourceTags(ALIYUN_SERVICE_CDN, "DOMAIN", self.DomainName, tags, replace)
}

func (self *SCdnDomain) GetOrigins() *cloudprovider.SCdnOrigins {
	domain, err := self.client.GetCdnDomain(self.DomainName)
	if err != nil {
		return nil
	}
	ret := cloudprovider.SCdnOrigins{}
	for _, origin := range domain.SourceModels.SourceModel {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Type:     origin.Type,
			Origin:   origin.Content,
			Port:     origin.Port,
			Enabled:  origin.Enabled,
			Priority: origin.Priority,
		})
	}
	return &ret
}

func (self *SCdnDomain) GetServiceType() string {
	return self.CdnType
}

func (self *SCdnDomain) GetStatus() string {
	switch self.DomainStatus {
	case "online", "offline":
		return self.DomainStatus
	case "configuring", "checking":
		return api.CDN_DOMAIN_STATUS_PROCESSING
	case "configure_failed":
		return self.DomainStatus
	case "check_failed":
		return api.CDN_DOMAIN_STATUS_REJECTED
	}
	return self.DomainStatus
}

func (self *SCdnDomain) Delete() error {
	params := map[string]string{
		"DomainName": self.DomainName,
	}
	_, err := self.client.cdnRequest("DeleteCdnDomain", params)
	return errors.Wrapf(err, "DeleteCdnDomain")
}

func (self *SAliyunClient) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	domains, err := self.GetCdnDomains()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomains")
	}
	ret := []cloudprovider.ICloudCDNDomain{}
	for i := range domains {
		domains[i].client = self
		ret = append(ret, &domains[i])
	}
	return ret, nil
}

func (self *SAliyunClient) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	return self.GetCDNDomainByName(name)
}

func (self *SAliyunClient) GetCDNDomainByName(name string) (*SCdnDomain, error) {
	domains, total, err := self.DescribeUserDomains(name, 1, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomain")
	}
	if total == 1 {
		domains[0].client = self
		return &domains[0], nil
	}
	if total == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, name)
}

func (client *SAliyunClient) DescribeDomainsBySource(origin string) (SCdnDomainsList, error) {
	sproducts := SCdnDomainsList{}
	params := map[string]string{}
	params["Action"] = "DescribeDomainsBySource"
	params["Sources"] = origin
	resp, err := client.cdnRequest("DescribeDomainsBySource", params)
	if err != nil {
		return sproducts, errors.Wrap(err, "DescribeDomainsBySource")
	}
	err = resp.Unmarshal(&sproducts, "DomainsList")
	if err != nil {
		return sproducts, errors.Wrap(err, "resp.Unmarshal")
	}
	return sproducts, nil
}

func (self *SAliyunClient) GetCdnDomains() ([]SCdnDomain, error) {
	domains := []SCdnDomain{}
	for {
		part, total, err := self.DescribeUserDomains("", 50, len(domains)/50+1)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeUserDomains")
		}
		domains = append(domains, part...)
		if len(domains) >= total || len(part) == 0 {
			break
		}
	}
	return domains, nil
}

func (client *SAliyunClient) DescribeUserDomains(domain string, pageSize, pageNumber int) ([]SCdnDomain, int, error) {
	if pageSize < 1 || pageSize > 50 {
		pageSize = 50
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	params := map[string]string{
		"PageSize":   fmt.Sprintf("%d", pageSize),
		"PageNumber": fmt.Sprintf("%d", pageNumber),
	}
	if len(domain) > 0 {
		params["DomainName"] = domain
		params["DomainSearchType"] = "full_match"
	}
	resp, err := client.cdnRequest("DescribeUserDomains", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeUserDomains")
	}
	domains := []SCdnDomain{}
	err = resp.Unmarshal(&domains, "Domains", "PageData")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("TotalCount")
	return domains, int(totalCount), nil
}

func (self *SAliyunClient) GetCdnDomain(domainName string) (*SCdnDomain, error) {
	params := map[string]string{
		"DomainName": domainName,
	}
	resp, err := self.cdnRequest("DescribeCdnDomainDetail", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCdnDomainDetail")
	}
	domain := &SCdnDomain{client: self}
	err = resp.Unmarshal(domain, "GetDomainDetailModel")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return domain, nil
}
