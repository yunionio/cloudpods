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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SRecordCreateRet struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Weight int    `json:"weight"`
}

type SRecordCountInfo struct {
	RecordTotal string `json:"record_total"`
	RecordsNum  string `json:"records_num"`
	SubDomains  string `json:"sub_domains"`
}

type SDnsRecord struct {
	domain *SDomian

	Line          string `json:"Line"`
	LineId        string `json:"LineId"`
	MX            int64  `json:"MX"`
	MonitorStatus string `json:"MonitorStatus"`
	Name          string `json:"Name"`
	RecordId      int64  `json:"RecordId"`
	Remark        string `json:"Remark"`
	Status        string `json:"Status"`
	TTL           int64  `json:"TTL"`
	Type          string `json:"Type"`
	UpdatedOn     string `json:"UpdatedOn"`
	Value         string `json:"Value"`
}

func (self *SQcloudClient) GetDnsRecords(domain string, offset int, limit int) ([]SDnsRecord, int, error) {
	params := map[string]string{}
	params["Domain"] = domain
	params["Offset"] = strconv.Itoa(offset)
	if limit > 0 {
		params["Limit"] = strconv.Itoa(limit)
	}
	resp, err := self.dnsRequest("DescribeRecordList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeRecordList")
	}
	ret := []SDnsRecord{}
	err = resp.Unmarshal(&ret, "RecordList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal RecordList")
	}
	total, err := resp.Float("RecordCountInfo", "TotalCount")
	return ret, int(total), err
}

func GetRecordLineLineType(policyinfo cloudprovider.TDnsPolicyValue) string {
	switch policyinfo {
	case cloudprovider.DnsPolicyValueMainland:
		return "境内"
	case cloudprovider.DnsPolicyValueOversea:
		return "境外"
	case cloudprovider.DnsPolicyValueTelecom:
		return "电信"
	case cloudprovider.DnsPolicyValueUnicom:
		return "联通"
	case cloudprovider.DnsPolicyValueChinaMobile:
		return "移动"
	case cloudprovider.DnsPolicyValueCernet:
		return "教育网"

	case cloudprovider.DnsPolicyValueBaidu:
		return "百度"
	case cloudprovider.DnsPolicyValueGoogle:
		return "谷歌"
	case cloudprovider.DnsPolicyValueYoudao:
		return "有道"
	case cloudprovider.DnsPolicyValueBing:
		return "必应"
	case cloudprovider.DnsPolicyValueSousou:
		return "搜搜"
	case cloudprovider.DnsPolicyValueSougou:
		return "搜狗"
	case cloudprovider.DnsPolicyValueQihu360:
		return "奇虎"
	default:
		return "默认"
	}
}

// https://cloud.tencent.com/document/api/1427/56180
func (self *SQcloudClient) CreateDnsRecord(opts *cloudprovider.DnsRecord, domainName string) (string, error) {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyValue)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	if len(opts.DnsName) < 1 {
		opts.DnsName = "@"
	}
	params["Domain"] = domainName
	params["SubDomain"] = opts.DnsName
	params["RecordType"] = string(opts.DnsType)
	params["TTL"] = strconv.FormatInt(opts.Ttl, 10)
	params["Value"] = opts.DnsValue
	params["RecordLine"] = recordline
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["MX"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	if !opts.Enabled {
		params["Status"] = "DISABLE"
	}
	resp, err := self.dnsRequest("CreateRecord", params)
	if err != nil {
		return "", errors.Wrapf(err, "CreateRecord")
	}
	return resp.GetString("RecordId")
}

// https://cloud.tencent.com/document/api/1427/56157
func (self *SQcloudClient) ModifyDnsRecord(domainName, recordId string, opts *cloudprovider.DnsRecord) error {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyValue)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	subDomain := strings.TrimSuffix(opts.DnsName, "."+domainName)
	if len(subDomain) < 1 {
		subDomain = "@"
	}
	params["Domain"] = domainName
	params["RecordId"] = recordId
	params["SubDomain"] = subDomain
	params["RecordType"] = string(opts.DnsType)
	params["TTL"] = strconv.FormatInt(opts.Ttl, 10)
	params["Value"] = opts.DnsValue
	params["RecordLine"] = recordline
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["MX"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	if !opts.Enabled {
		params["Status"] = "DISABLE"
	}
	_, err := self.dnsRequest("ModifyRecord", params)
	if err != nil {
		return errors.Wrapf(err, "ModifyRecord")
	}
	return nil
}

// https://cloud.tencent.com/document/product/302/8519
func (client *SQcloudClient) ModifyRecordStatus(status, recordId, domain string) error {
	params := map[string]string{}
	params["Domain"] = domain
	params["RecordId"] = recordId
	params["Status"] = strings.ToUpper(status) // “disable” 和 “enable”
	_, err := client.dnsRequest("ModifyRecordStatus", params)
	if err != nil {
		return errors.Wrapf(err, "ModifyRecordStatus")
	}
	return nil
}

// https://cloud.tencent.com/document/api/1427/56176
func (client *SQcloudClient) DeleteDnsRecord(recordId string, domainName string) error {
	params := map[string]string{}
	params["Domain"] = domainName
	params["RecordId"] = recordId
	_, err := client.dnsRequest("DeleteRecord", params)
	if err != nil {
		return errors.Wrapf(err, "DeleteRecord")
	}
	return nil
}

func (self *SDnsRecord) GetGlobalId() string {
	return fmt.Sprintf("%d", self.RecordId)
}

func (self *SDnsRecord) GetDnsName() string {
	return self.Name
}

func (self *SDnsRecord) GetStatus() string {
	if self.Status != "SPAM" {
		return api.DNS_RECORDSET_STATUS_AVAILABLE
	}
	return api.DNS_ZONE_STATUS_UNKNOWN
}

func (self *SDnsRecord) GetEnabled() bool {
	return self.Status == "ENABLE"
}

func (self *SDnsRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SDnsRecord) GetDnsValue() string {
	if self.GetDnsType() == cloudprovider.DnsTypeMX || self.GetDnsType() == cloudprovider.DnsTypeCNAME || self.GetDnsType() == cloudprovider.DnsTypeSRV {
		return self.Value[:len(self.Value)-1]
	}
	return self.Value
}

func (self *SDnsRecord) GetTTL() int64 {
	return int64(self.TTL)
}

func (self *SDnsRecord) GetMxPriority() int64 {
	if self.GetDnsType() == cloudprovider.DnsTypeMX {
		return self.MX
	}
	return 0
}

func (self *SDnsRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	switch self.Line {
	case "境内", "境外":
		return cloudprovider.DnsPolicyTypeByGeoLocation
	case "电信", "联通", "移动", "教育网":
		return cloudprovider.DnsPolicyTypeByCarrier
	case "百度", "谷歌", "有道", "必应", "搜搜", "搜狗", "奇虎":
		return cloudprovider.DnsPolicyTypeBySearchEngine
	default:
		return cloudprovider.DnsPolicyTypeSimple
	}
}

func (self *SDnsRecord) Delete() error {
	return self.domain.client.DeleteDnsRecord(self.GetGlobalId(), self.domain.Name)
}

func (self *SDnsRecord) GetExtraAddresses() ([]string, error) {
	return []string{}, nil
}

func (self *SDnsRecord) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	switch self.Line {
	case "境内":
		return cloudprovider.DnsPolicyValueMainland
	case "境外":
		return cloudprovider.DnsPolicyValueOversea

	case "电信":
		return cloudprovider.DnsPolicyValueTelecom
	case "联通":
		return cloudprovider.DnsPolicyValueUnicom
	case "移动":
		return cloudprovider.DnsPolicyValueChinaMobile
	case "教育网":
		return cloudprovider.DnsPolicyValueCernet

	case "百度":
		return cloudprovider.DnsPolicyValueBaidu
	case "谷歌":
		return cloudprovider.DnsPolicyValueGoogle
	case "有道":
		return cloudprovider.DnsPolicyValueYoudao
	case "必应":
		return cloudprovider.DnsPolicyValueBing
	case "搜搜":
		return cloudprovider.DnsPolicyValueSousou
	case "搜狗":
		return cloudprovider.DnsPolicyValueSougou
	case "奇虎":
		return cloudprovider.DnsPolicyValueQihu360
	default:
		return cloudprovider.DnsPolicyValueEmpty
	}
}

func (self *SDnsRecord) Enable() error {
	return self.domain.client.ModifyRecordStatus("enable", self.GetGlobalId(), self.domain.Name)
}

func (self *SDnsRecord) Disable() error {
	return self.domain.client.ModifyRecordStatus("disable", self.GetGlobalId(), self.domain.Name)
}

func (self *SDnsRecord) Update(opts *cloudprovider.DnsRecord) error {
	return self.domain.client.ModifyDnsRecord(self.domain.Name, self.GetGlobalId(), opts)
}
