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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SDomainRecords struct {
	//RequestID     string         `json:"RequestId"`
	TotalCount    int            `json:"TotalCount"`
	PageNumber    int            `json:"PageNumber"`
	PageSize      int            `json:"PageSize"`
	DomainRecords sDomainRecords `json:"DomainRecords"`
}

// https://help.aliyun.com/document_detail/29777.html?spm=a2c4g.11186623.6.666.aa4832307YdopF
type SDomainRecord struct {
	DomainId   string `json:"DomainId"`
	GroupId    string `json:"GroupId"`
	GroupName  string `json:"GroupName"`
	PunyCode   string `json:"PunyCode"`
	RR         string `json:"RR"`
	Status     string `json:"Status"`
	Value      string `json:"Value"`
	RecordID   string `json:"RecordId"`
	Type       string `json:"Type"`
	RequestID  string `json:"RequestId"`
	DomainName string `json:"DomainName"`
	Locked     bool   `json:"Locked"`
	Line       string `json:"Line"`
	TTL        int64  `json:"TTL"`
	Priority   int64  `json:"Priority"`
	Remark     string
}

type sDomainRecords struct {
	Record []SDomainRecord `json:"Record"`
}

func (client *SAliyunClient) DescribeDomainRecords(domainName string, pageNumber int, pageSize int) (SDomainRecords, error) {
	srecords := SDomainRecords{}
	params := map[string]string{}
	params["Action"] = "DescribeDomainRecords"
	params["DomainName"] = domainName
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.alidnsRequest("DescribeDomainRecords", params)
	if err != nil {
		return srecords, errors.Wrap(err, "DescribeDomainRecords")
	}
	err = resp.Unmarshal(&srecords)
	if err != nil {
		return srecords, errors.Wrap(err, "resp.Unmarshal")
	}
	return srecords, nil
}

func (client *SAliyunClient) GetAllDomainRecords(domainName string) ([]SDomainRecord, error) {
	srecords := []SDomainRecord{}
	pageNumber := 0
	for {
		pageNumber++
		records, err := client.DescribeDomainRecords(domainName, pageNumber, 2)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeDomainRecords(%d, 20)", len(srecords))
		}
		srecords = append(srecords, records.DomainRecords.Record...)
		if len(srecords) >= records.TotalCount {
			break
		}
	}
	return srecords, nil
}

func (client *SAliyunClient) DescribeDomainRecordInfo(recordId string) (*SDomainRecord, error) {
	srecord := SDomainRecord{}
	params := map[string]string{}
	params["Action"] = "DescribeDomainRecordInfo"
	params["RecordId"] = recordId

	resp, err := client.alidnsRequest("DescribeDomainRecordInfo", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeDomainRecordInfo")
	}
	err = resp.Unmarshal(&srecord)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return &srecord, nil
}

func GetRecordLineLineType(policyinfo cloudprovider.TDnsPolicyValue) string {
	switch policyinfo {
	case cloudprovider.DnsPolicyValueOversea:
		return "oversea"
	case cloudprovider.DnsPolicyValueTelecom:
		return "telecom"
	case cloudprovider.DnsPolicyValueUnicom:
		return "unicom"
	case cloudprovider.DnsPolicyValueChinaMobile:
		return "mobile"
	case cloudprovider.DnsPolicyValueCernet:
		return "edu"
	case cloudprovider.DnsPolicyValueDrPeng:
		return "drpeng"
	case cloudprovider.DnsPolicyValueBtvn:
		return "btvn"

	case cloudprovider.DnsPolicyValueBaidu:
		return "baidu"
	case cloudprovider.DnsPolicyValueGoogle:
		return "google"
	case cloudprovider.DnsPolicyValueYoudao:
		return "youdao"
	case cloudprovider.DnsPolicyValueBing:
		return "biying"
	default:
		return "default"
	}
}

func (client *SAliyunClient) AddDomainRecord(domainName string, opts cloudprovider.DnsRecordSet) (string, error) {
	line := GetRecordLineLineType(opts.PolicyValue)
	params := map[string]string{}
	params["Action"] = "AddDomainRecord"
	params["RR"] = opts.DnsName
	params["Type"] = string(opts.DnsType)
	params["Value"] = opts.DnsValue
	params["DomainName"] = domainName
	params["TTL"] = strconv.FormatInt(opts.Ttl, 10)
	params["Line"] = line
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["Priority"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	ret, err := client.alidnsRequest("AddDomainRecord", params)
	if err != nil {
		return "", errors.Wrap(err, "AddDomainRecord")
	}
	recordId := ""
	return recordId, ret.Unmarshal(&recordId, "RecordId")
}

// line
func (client *SAliyunClient) UpdateDomainRecord(opts cloudprovider.DnsRecordSet) error {
	line := GetRecordLineLineType(opts.PolicyValue)
	params := map[string]string{}
	params["Action"] = "UpdateDomainRecord"
	params["RR"] = opts.DnsName
	params["RecordId"] = opts.ExternalId
	params["Type"] = string(opts.DnsType)
	params["Value"] = opts.DnsValue
	params["TTL"] = strconv.FormatInt(opts.Ttl, 10)
	params["Line"] = line
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["Priority"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	_, err := client.alidnsRequest("UpdateDomainRecord", params)
	if err != nil {
		return errors.Wrap(err, "UpdateDomainRecord")
	}
	return nil
}

func (client *SAliyunClient) UpdateDomainRecordRemark(recordId string, remark string) error {
	params := map[string]string{}
	params["RecordId"] = recordId
	params["Remark"] = remark

	_, err := client.alidnsRequest("UpdateDomainRecordRemark", params)
	if err != nil {
		return errors.Wrap(err, "UpdateDomainRecordRemark")
	}
	return nil
}

// Enable: 启用解析 Disable: 暂停解析
func (client *SAliyunClient) SetDomainRecordStatus(recordId, status string) error {
	params := map[string]string{}
	params["Action"] = "SetDomainRecordStatus"
	params["RecordId"] = recordId
	params["Status"] = status
	_, err := client.alidnsRequest("SetDomainRecordStatus", params)
	if err != nil {
		return errors.Wrap(err, "SetDomainRecordStatus")
	}
	return nil
}

func (client *SAliyunClient) DeleteDomainRecord(recordId string) error {
	params := map[string]string{}
	params["Action"] = "DeleteDomainRecord"
	params["RecordId"] = recordId
	_, err := client.alidnsRequest("DeleteDomainRecord", params)
	if err != nil {
		return errors.Wrap(err, "DeleteDomainRecord")
	}
	return nil
}

func (self *SDomainRecord) GetGlobalId() string {
	return self.RecordID
}

func (self *SDomainRecord) GetDnsName() string {
	return self.RR
}

func (self *SDomainRecord) GetStatus() string {
	return api.DNS_RECORDSET_STATUS_AVAILABLE
}

func (self *SDomainRecord) GetEnabled() bool {
	return self.Status == "ENABLE"
}

func (self *SDomainRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SDomainRecord) GetDnsValue() string {
	return self.Value
}

func (self *SDomainRecord) GetTTL() int64 {
	return self.TTL
}

func (self *SDomainRecord) GetMxPriority() int64 {
	if self.GetDnsType() == cloudprovider.DnsTypeMX {
		return self.Priority
	}
	return 0
}

func (self *SDomainRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	switch self.Line {
	case "telecom", "unicom", "mobile", "edu", "drpeng", "btvn":
		return cloudprovider.DnsPolicyTypeByCarrier
	case "google", "baidu", "biying", "youdao":
		return cloudprovider.DnsPolicyTypeBySearchEngine
	case "oversea":
		return cloudprovider.DnsPolicyTypeByGeoLocation
	default:
		return cloudprovider.DnsPolicyTypeSimple
	}
}

func (self *SDomainRecord) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	switch self.Line {
	case "telecom":
		return cloudprovider.DnsPolicyValueTelecom
	case "unicom":
		return cloudprovider.DnsPolicyValueUnicom
	case "mobile":
		return cloudprovider.DnsPolicyValueChinaMobile
	case "oversea":
		return cloudprovider.DnsPolicyValueOversea
	case "edu":
		return cloudprovider.DnsPolicyValueCernet

	case "drpeng":
		return cloudprovider.DnsPolicyValueDrPeng
	case "btvn":
		return cloudprovider.DnsPolicyValueBtvn
	case "google":
		return cloudprovider.DnsPolicyValueGoogle
	case "baidu":
		return cloudprovider.DnsPolicyValueBaidu
	case "biying":
		return cloudprovider.DnsPolicyValueBing
	case "youdao":
		return cloudprovider.DnsPolicyValueYoudao

	}
	return ""
}

func (self *SDomainRecord) GetPolicyOptions() *jsonutils.JSONDict {
	return nil
}

func (self *SDomainRecord) match(update cloudprovider.DnsRecordSet) bool {
	if update.DnsName != self.GetDnsName() {
		return false
	}
	if update.DnsType != self.GetDnsType() {
		return false
	}
	if update.DnsValue != self.GetDnsValue() {
		return false
	}
	if update.PolicyType != self.GetPolicyType() {
		return false
	}
	if update.PolicyValue != self.GetPolicyValue() {
		return false
	}
	if update.Ttl != self.GetTTL() {
		return false
	}
	return true
}
