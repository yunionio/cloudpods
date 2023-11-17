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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type AliasesType struct {
	Quantity int      `xml:"Quantity"`
	Items    []string `xml:"Items>CNAME"`
}

type Origin struct {
	DomainName            string `xml:"DomainName"`
	Id                    string `xml:"Id"`
	ConnectionAttempts    int    `xml:"ConnectionAttempts,omitempty"`
	ConnectionTimeout     int    `xml:"ConnectionTimeout,omitempty"`
	OriginAccessControlId string `xml:"OriginAccessControlId,omitempty"`
	OriginPath            string `xml:"OriginPath,omitempty"`
}

type OriginsType struct {
	Items    []Origin `xml:"Items>Origin"`
	Quantity int      `xml:"Quantity"`
}

type DistributionConfigType struct {
	CallerReference              string      `xml:"CallerReference"`
	Comment                      string      `xml:"Comment"`
	Enabled                      bool        `xml:"Enabled"`
	Origins                      OriginsType `xml:"Origins"`
	Aliases                      AliasesType `xml:"Aliases,omitempty"`
	ContinuousDeploymentPolicyId string      `xml:"ContinuousDeploymentPolicyId,omitempty"`
	DefaultRootObject            string      `xml:"DefaultRootObject,omitempty"`
	HttpVersion                  string      `xml:"HttpVersion,omitempty"`
	IsIPV6Enabled                bool        `xml:"IsIPV6Enabled,omitempty"`
	PriceClass                   string      `xml:"PriceClass,omitempty"`
	Staging                      bool        `xml:"Staging,omitempty"`
	WebACLId                     string      `xml:"WebACLId,omitempty"`
}

type SCdnDomain struct {
	multicloud.SResourceBase
	AwsTags

	client *SAwsClient

	Aliases            AliasesType            `xml:"Aliases"`
	ARN                string                 `xml:"ARN"`
	Comment            string                 `xml:"Comment"`
	DomainName         string                 `xml:"DomainName"`
	Enabled            bool                   `xml:"Enabled"`
	HttpVersion        string                 `xml:"HttpVersion"`
	Id                 string                 `xml:"Id"`
	IsIPV6Enabled      bool                   `xml:"IsIPV6Enabled"`
	LastModifiedTime   string                 `xml:"LastModifiedTime"` // or time.Time if parsed
	Origins            OriginsType            `xml:"Origins"`
	PriceClass         string                 `xml:"PriceClass"`
	Staging            bool                   `xml:"Staging"`
	Status             string                 `xml:"Status"`
	WebACLId           string                 `xml:"WebACLId"`
	DistributionConfig DistributionConfigType `xml:"DistributionConfig"`
}

func (cd *SCdnDomain) GetCacheKeys() (*cloudprovider.SCDNCacheKeys, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetRangeOriginPull() (*cloudprovider.SCDNRangeOriginPull, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetCache() (*cloudprovider.SCDNCache, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetHTTPS() (*cloudprovider.SCDNHttps, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetForceRedirect() (*cloudprovider.SCDNForceRedirect, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetReferer() (*cloudprovider.SCDNReferer, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetMaxAge() (*cloudprovider.SCDNMaxAge, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetArea() string {
	return "global"
}

func (cd *SCdnDomain) GetCname() string {
	res := make([]string, len(cd.Aliases.Items))
	for idx, item := range cd.Aliases.Items {
		res[idx] = item
	}
	return strings.Join(res, ",")
}

func (cd *SCdnDomain) GetEnabled() bool {
	return cd.Enabled
}

func (cd *SCdnDomain) GetId() string {
	return cd.DomainName
}

func (cd *SCdnDomain) GetGlobalId() string {
	return cd.DomainName
}

func (cd *SCdnDomain) GetName() string {
	return cd.DomainName
}

func (cd *SCdnDomain) Refresh() error {
	domain, err := cd.client.GetCdnDomain(cd.Id)
	if err != nil {
		return errors.Wrapf(err, "GetCdnDomain")
	}

	cd.Status = domain.Status
	return nil
}

func (cd *SCdnDomain) GetOrigins() *cloudprovider.SCdnOrigins {
	domain, err := cd.client.GetCdnDomain(cd.Id)
	if err != nil {
		return nil
	}

	ret := cloudprovider.SCdnOrigins{}
	for _, origin := range domain.Origins.Items {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Origin: origin.DomainName,
			Path:   origin.OriginPath,
		})
	}
	return &ret
}

func (cd *SCdnDomain) GetServiceType() string {
	return "wholeSite"
}

func (cd *SCdnDomain) GetStatus() string {
	if cd.Status == "Deployed" {
		return "online"
	} else {
		return "offline"
	}
}

func (cd *SCdnDomain) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (cd *SCdnDomain) GetProjectId() string {
	return ""
}

func (cd *SCdnDomain) GetDescription() string {
	return ""
}

func (ac *SAwsClient) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	domains, err := ac.GetCdnDomains()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomains")
	}

	ret := make([]cloudprovider.ICloudCDNDomain, 0)
	for i := range domains {
		domains[i].client = ac
		ret = append(ret, &domains[i])
	}
	return ret, nil
}

func (ac *SAwsClient) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	return ac.GetCDNDomainByName(name)
}

func (ac *SAwsClient) GetCDNDomainByName(name string) (*SCdnDomain, error) {
	domains, err := ac.GetCdnDomains()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomain")
	}

	for i := range domains {
		if domains[i].DomainName == name {
			domains[i].client = ac
			return &domains[i], nil
		}
	}

	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (ac *SAwsClient) GetCdnDomains() ([]SCdnDomain, error) {
	domains := make([]SCdnDomain, 0)
	marker := ""
	for {
		part, nextMarker, err := ac.DescribeUserDomains(marker, int64(100))
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeUserDomains")
		}

		domains = append(domains, part...)
		if nextMarker == "" {
			break
		} else {
			marker = nextMarker
		}
	}
	return domains, nil
}

func (ac *SAwsClient) DescribeUserDomains(marker string, pageSize int64) ([]SCdnDomain, string, error) {
	resp, NextMarker, err := ac.cdnList(marker, pageSize)
	if err != nil {
		return nil, "", errors.Wrap(err, "DescribeUserDomains")
	}

	domains := make([]SCdnDomain, 0)
	for i := range resp {
		resp[i].client = ac
		domains = append(domains, resp[i])
	}
	return domains, NextMarker, nil
}

func (ac *SAwsClient) GetCdnDomain(domainID string) (*SCdnDomain, error) {
	resp, err := ac.cdnGet(domainID)
	if err != nil {
		return nil, errors.Wrapf(err, "ShowDomainDetail")
	}

	domain := &SCdnDomain{client: ac}
	domain.Status = resp.Status
	domain.Origins = resp.DistributionConfig.Origins
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}

	return domain, nil
}
