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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SCdnOrigin struct {
	Origins            []string `json:"Origins"`
	OriginType         string   `json:"OriginType"`
	ServerName         string   `json:"ServerName"`
	CosPrivateAccess   string   `json:"CosPrivateAccess"`
	OriginPullProtocol string   `json:"OriginPullProtocol"`
	BackupOrigins      []string `json:"BackupOrigins"`
	BackupOriginType   string   `json:"BackupOriginType"`
	BackupServerName   string   `json:"BackupServerName"`
}

type SCdnDomain struct {
	multicloud.SResourceBase
	QcloudTags

	client *SQcloudClient

	Area        string     `json:"Area"`
	Cname       string     `json:"Cname"`
	CreateTime  string     `json:"CreateTime"`
	Disable     string     `json:"Disable"`
	Domain      string     `json:"Domain"`
	Origin      SCdnOrigin `json:"Origin"`
	ProjectID   int        `json:"ProjectId"`
	Readonly    string     `json:"Readonly"`
	ResourceID  string     `json:"ResourceId"`
	ServiceType string     `json:"ServiceType"`
	Status      string     `json:"Status"`
	UpdateTime  string     `json:"UpdateTime"`
}

func (self *SCdnDomain) GetName() string {
	return self.Domain
}

func (self *SCdnDomain) GetGlobalId() string {
	return self.Domain
}

func (self *SCdnDomain) GetId() string {
	return self.Domain
}

func (self *SCdnDomain) GetStatus() string {
	return self.Status
}

func (self *SCdnDomain) GetEnabled() bool {
	return self.Disable == "normal"
}

func (self *SCdnDomain) GetCname() string {
	return self.Cname
}

func (self *SCdnDomain) GetOrigins() *cloudprovider.SCdnOrigins {
	ret := cloudprovider.SCdnOrigins{}
	if self.Origin.OriginType == "cos" {
		self.Origin.OriginType = api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET
	}
	for _, org := range self.Origin.Origins {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Type:       self.Origin.OriginType,
			ServerName: self.Origin.ServerName,
			Protocol:   self.Origin.OriginPullProtocol,
			Origin:     org,
		})
	}
	for _, org := range self.Origin.BackupOrigins {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Type:       self.Origin.BackupOriginType,
			ServerName: self.Origin.BackupServerName,
			Origin:     org,
		})
	}
	return &ret
}

func (self *SCdnDomain) GetArea() string {
	return self.Area
}

func (self *SCdnDomain) GetServiceType() string {
	return self.ServiceType
}

func (self *SQcloudClient) GetCdnDomain(domain string) (*SCdnDomain, error) {
	domains, _, err := self.DescribeCdnDomains([]string{domain}, nil, "", 0, 100)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCdnDomains")
	}
	for i := range domains {
		if domains[i].Domain == domain {
			domains[i].client = self
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, domain)
}

func (self *SCdnDomain) Refresh() error {
	domain, err := self.client.GetCdnDomain(self.Domain)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, domain)
}

func (self *SCdnDomain) Delete() error {
	err := self.client.StopCdnDomain(self.Domain)
	if err != nil {
		return errors.Wrapf(err, "StopCdnDomain")
	}
	return self.client.DeleteCdnDomain(self.Domain)
}

func (self *SCdnDomain) SetTags(tags map[string]string, replace bool) error {
	region, err := self.client.getDefaultRegion()
	if err != nil {
		return errors.Wrapf(err, "getDefaultRegion")
	}
	return region.SetResourceTags("cdn", "domain", []string{self.Domain}, tags, replace)
}

func (self *SQcloudClient) StopCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("StopCdnDomain", params)
	return errors.Wrapf(err, "StopCdnDomain")
}

func (self *SQcloudClient) StartCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("StartCdnDomain", params)
	return errors.Wrapf(err, "StartCdnDomain")
}

func (self *SQcloudClient) DeleteCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("DeleteCdnDomain", params)
	return errors.Wrapf(err, "DeleteCdnDomain")
}

type SDomains struct {
	RequestID   string       `json:"RequestId"`
	Domains     []SCdnDomain `json:"Domains"`
	TotalNumber int          `json:"TotalNumber"`
}

func (self *SQcloudClient) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	cdns, err := self.DescribeAllCdnDomains(nil, nil, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudCDNDomain{}
	for i := range cdns {
		cdns[i].client = self
		ret = append(ret, &cdns[i])
	}
	return ret, nil
}

func (self *SQcloudClient) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	domains, _, err := self.DescribeCdnDomains([]string{name}, nil, "", 0, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCdnDomains")
	}
	for i := range domains {
		if domains[i].Domain == name {
			domains[i].client = self
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (client *SQcloudClient) AddCdnDomain(domain string, originType string, origins []string, cosPrivateAccess string) error {
	params := map[string]string{}
	params["Domain"] = domain
	params["ServiceType"] = "web"
	for i := range origins {
		params[fmt.Sprintf("Origin.Origins.%d", i)] = origins[i]
	}
	params["Origin.OriginType"] = originType
	params["Origin.CosPrivateAccess"] = cosPrivateAccess
	_, err := client.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return errors.Wrapf(err, `AddCdnDomain %s`, params)
	}
	return nil
}

func (client *SQcloudClient) DescribeCdnDomains(domains, origins []string, domainType string, offset int, limit int) ([]SCdnDomain, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	filterIndex := 0
	if len(domains) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domain"
		for i := range domains {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = domains[i]
		}
		filterIndex++
	}
	if len(origins) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "origin"
		for i := range origins {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = origins[i]
		}
		filterIndex++
	}

	if len(domainType) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domainType"
		params[fmt.Sprintf("Filters.%d.Value.0", filterIndex)] = domainType
		filterIndex++
	}

	resp, err := client.cdnRequest("DescribeDomainsConfig", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDomainsConfig %s", params)
	}
	cdnDomains := []SCdnDomain{}
	err = resp.Unmarshal(&cdnDomains, "Domains")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalcount, _ := resp.Float("TotalNumber")
	return cdnDomains, int(totalcount), nil
}

func (client *SQcloudClient) DescribeAllCdnDomains(domains, origins []string, domainType string) ([]SCdnDomain, error) {
	cdnDomains := make([]SCdnDomain, 0)
	for {
		part, total, err := client.DescribeCdnDomains(domains, origins, domainType, len(cdnDomains), 50)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeCdnDomains")
		}
		cdnDomains = append(cdnDomains, part...)
		if len(cdnDomains) >= total {
			break
		}
	}
	return cdnDomains, nil
}

func (self *SQcloudClient) CreateCDNDomain(opts *cloudprovider.CdnCreateOptions) (*SCdnDomain, error) {
	params := map[string]string{
		"Domain":      opts.Domain,
		"ServiceType": opts.ServiceType,
	}
	if len(opts.Area) > 0 {
		params["Area"] = opts.Area
	}
	originTypes := map[string][]string{}
	for _, origin := range opts.Origins {
		_, ok := originTypes[origin.Type]
		if !ok {
			originTypes[origin.Type] = []string{}
		}
		originTypes[origin.Type] = append(originTypes[origin.Type], origin.Origin)
	}
	for _, origin := range opts.Origins {
		params["Origin.OriginType"] = origin.Type
		if origin.Type == api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET {
			params["Origin.OriginType"] = "cos"
			if len(origin.ServerName) > 0 {
				params["Origin.ServerName"] = origin.ServerName
			} else {
				params["Origin.ServerName"] = origin.Origin
			}
		}
		if len(origin.Protocol) > 0 {
			params["Origin.OriginPullProtocol"] = origin.Protocol
		}
		origins, ok := originTypes[origin.Type]
		if ok {
			for i, origin := range origins {
				params[fmt.Sprintf("Origin.Origins.%d", i)] = origin
			}
		}
	}
	_, err := self.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return nil, errors.Wrapf(err, "AddCdnDomain")
	}
	return self.GetCdnDomain(opts.Domain)
}
