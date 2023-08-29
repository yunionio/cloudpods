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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDomian struct {
	multicloud.SVirtualResourceBase
	QcloudTags
	client *SQcloudClient

	CNAMESpeedup     string   `json:"CNAMESpeedup"`
	CreatedOn        string   `json:"CreatedOn"`
	DNSStatus        string   `json:"DNSStatus"`
	DomainId         int      `json:"DomainId"`
	EffectiveDNS     []string `json:"EffectiveDNS"`
	Grade            string   `json:"Grade"`
	GradeLevel       int64    `json:"GradeLevel"`
	GradeTitle       string   `json:"GradeTitle"`
	GroupId          int64    `json:"GroupId"`
	IsVip            string   `json:"IsVip"`
	Name             string   `json:"Name"`
	Owner            string   `json:"Owner"`
	Punycode         string   `json:"Punycode"`
	RecordCount      int64    `json:"RecordCount"`
	Remark           string   `json:"Remark"`
	SearchEnginePush string   `json:"SearchEnginePush"`
	Status           string   `json:"Status"`
	TTL              int64    `json:"TTL"`
	UpdatedOn        string   `json:"UpdatedOn"`
	VipAutoRenew     string   `json:"VipAutoRenew"`
	VipEndAt         string   `json:"VipEndAt"`
	VipStartAt       string   `json:"VipStartAt"`
}

func (self *SQcloudClient) GetDomains(key string, offset int, limit int) ([]SDomian, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	if limit > 0 {
		params["Limit"] = strconv.Itoa(limit)
	}
	if len(key) > 0 {
		params["Keyword"] = key
	}
	resp, err := self.dnsRequest("DescribeDomainList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDomainList")
	}
	domains := []SDomian{}
	err = resp.Unmarshal(&domains, "DomainList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal DomainList")
	}
	total, err := resp.Float("DomainCountInfo", "DomainTotal")
	return domains, int(total), err
}

func (self *SQcloudClient) GetDomain(domain string) (*SDomian, error) {
	domains, _, err := self.GetDomains(domain, 0, 0)
	if err != nil {
		return nil, err
	}
	for i := range domains {
		if domains[i].Name == domain {
			domains[i].client = self
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, domain)
}

func (self *SQcloudClient) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	result := []cloudprovider.ICloudDnsZone{}
	domains := []SDomian{}
	for {
		part, total, err := self.GetDomains("", len(domains), 1000)
		if err != nil {
			return nil, err
		}
		domains = append(domains, part...)
		if len(domains) >= total {
			break
		}
	}
	for i := 0; i < len(domains); i++ {
		domains[i].client = self
		result = append(result, &domains[i])
	}
	return result, nil
}

func (self *SDomian) GetIDnsRecordById(id string) (cloudprovider.ICloudDnsRecord, error) {
	records, err := self.GetIDnsRecords()
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].GetGlobalId() == id {
			return records[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SQcloudClient) CreateDomian(domianName string) (*SDomian, error) {
	params := map[string]string{}
	params["Domain"] = domianName
	_, err := self.dnsRequest("CreateDomain", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDomain")
	}
	for i := 0; i < 3; i++ {
		domain, err := self.GetDomain(domianName)
		if err == nil {
			return domain, nil
		}
		time.Sleep(time.Second * 10)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, domianName)
}

func (self *SQcloudClient) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	domain, err := self.CreateDomian(opts.Name)
	if err != nil {
		return nil, err
	}
	return domain, nil
}

func (client *SQcloudClient) DeleteDomian(domianName string) error {
	params := map[string]string{}
	params["Domain"] = domianName
	_, err := client.dnsRequest("DeleteDomain", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteDomain")
	}
	return nil
}

func (self *SDomian) GetId() string {
	return fmt.Sprintf("%d", self.DomainId)
}

func (self *SDomian) GetName() string {
	if len(self.Punycode) > 0 {
		return self.Punycode
	}
	return self.Name
}

func (self *SDomian) GetGlobalId() string {
	return self.Name
}

func (self *SDomian) GetStatus() string {
	switch self.Status {
	case "ENABLE":
		return api.DNS_ZONE_STATUS_AVAILABLE
	case "PAUSE":
		return api.DNS_ZONE_STATUS_AVAILABLE
	default:
		return api.DNS_ZONE_STATUS_UNKNOWN
	}
}

func (self *SDomian) GetEnabled() bool {
	if self.Status == "ENABLE" {
		return true
	}
	return false
}

func (self *SDomian) Refresh() error {
	domain, err := self.client.GetDomain(self.Name)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, domain)
}

func (self *SDomian) Delete() error {
	return self.client.DeleteDomian(self.Name)
}

func (self *SDomian) GetZoneType() cloudprovider.TDnsZoneType {
	return cloudprovider.PublicZone
}

func (self *SDomian) GetICloudVpcIds() ([]string, error) {
	return nil, nil
}

func (self *SDomian) AddVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDomian) RemoveVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDomian) GetIDnsRecords() ([]cloudprovider.ICloudDnsRecord, error) {
	records := []SDnsRecord{}
	for {
		part, total, err := self.client.GetDnsRecords(self.Name, len(records), 1000)
		if err != nil {
			return nil, err
		}
		records = append(records, part...)
		if len(records) >= total {
			break
		}
	}
	result := []cloudprovider.ICloudDnsRecord{}
	for i := 0; i < len(records); i++ {
		records[i].domain = self
		result = append(result, &records[i])
	}
	return result, nil
}

func (self *SDomian) AddDnsRecord(opts *cloudprovider.DnsRecord) (string, error) {
	return self.client.CreateDnsRecord(opts, self.Name)
}

func (self *SDomian) GetDnsProductType() cloudprovider.TDnsProductType {
	switch self.GradeTitle {
	case "企业旗舰版":
		return cloudprovider.DnsProductEnterpriseUltimate
	case "企业标准版":
		return cloudprovider.DnsProductEnterpriseStandard
	case "企业基础版":
		return cloudprovider.DnsProductEnterpriseBasic
	case "个人专业版":
		return cloudprovider.DnsProductPersonalProfessional
	case "免费版":
		return cloudprovider.DnsProductFree
	default:
		return cloudprovider.DnsProductFree
	}
}
