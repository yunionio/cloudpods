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
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SPvtzRecords struct {
	PageNumber int          `json:"PageNumber"`
	Records    sPvtzRecords `json:"Records"`
	PageSize   int          `json:"PageSize"`
	// RequestID  string       `json:"RequestId"`
	TotalItems int `json:"TotalItems"`
	TotalPages int `json:"TotalPages"`
}

type SPvtzRecord struct {
	szone    *SPrivateZone
	Status   string `json:"Status"`
	Value    string `json:"Value"`
	Rr       string `json:"Rr"`
	RecordId string `json:"RecordId"`
	TTL      int64  `json:"Ttl"`
	Type     string `json:"Type"`
	Priority int64  `json:"Priority"`
}

type sPvtzRecords struct {
	Record []SPvtzRecord `json:"Record"`
}

// https://help.aliyun.com/document_detail/66252.html?spm=a2c4g.11186623.6.595.3f3f243fU1AjOV
func (client *SAliyunClient) DescribeZoneRecords(ZoneId string, pageNumber int, pageSize int) (SPvtzRecords, error) {
	sRecords := SPvtzRecords{}
	params := map[string]string{}
	params["Action"] = "DescribeZoneRecords"
	params["ZoneId"] = ZoneId
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.pvtzRequest("DescribeZoneRecords", params)
	if err != nil {
		return sRecords, errors.Wrap(err, "DescribeZoneRecords")
	}
	err = resp.Unmarshal(&sRecords)
	if err != nil {
		return sRecords, errors.Wrap(err, "resp.Unmarshal")
	}
	return sRecords, nil
}

func (client *SAliyunClient) GetAllZoneRecords(ZoneId string) ([]SPvtzRecord, error) {
	count := 0
	pageNumber := 0
	srecords := []SPvtzRecord{}

	for {
		pageNumber++
		records, err := client.DescribeZoneRecords(ZoneId, pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeZones(%d, 20)", count)
		}
		count += len(records.Records.Record)

		srecords = append(srecords, records.Records.Record...)
		if count >= records.TotalItems {
			break
		}
	}
	return srecords, nil
}

func (client *SAliyunClient) AddZoneRecord(ZoneId string, opts *cloudprovider.DnsRecord) (string, error) {
	params := map[string]string{}
	params["Action"] = "AddZoneRecord"
	params["Rr"] = opts.DnsName
	params["Type"] = string(opts.DnsType)
	params["Value"] = opts.DnsValue
	params["ZoneId"] = ZoneId
	params["Ttl"] = strconv.FormatInt(opts.Ttl, 10)
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["Priority"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	ret, err := client.pvtzRequest("AddZoneRecord", params)
	if err != nil {
		return "", errors.Wrap(err, "AddZoneRecord")
	}
	return ret.GetString("RecordId")
}

// https://help.aliyun.com/document_detail/66251.html?spm=a2c4g.11186623.6.596.15d55563zpsSWE
// status ENABLE: 启用解析 DISABLE: 暂停解析
func (client *SAliyunClient) SetZoneRecordStatus(recordId string, status string) error {
	params := map[string]string{}
	params["Action"] = "SetZoneRecordStatus"
	params["RecordId"] = recordId
	params["Status"] = strings.ToUpper(status)
	_, err := client.pvtzRequest("SetZoneRecordStatus", params)
	if err != nil {
		return errors.Wrap(err, "SetZoneRecordStatus")
	}
	return nil
}

func (client *SAliyunClient) DeleteZoneRecord(RecordId string) error {
	params := map[string]string{}
	params["Action"] = "DeleteZoneRecord"
	params["RecordId"] = RecordId
	_, err := client.pvtzRequest("DeleteZoneRecord", params)
	if err != nil {
		return errors.Wrap(err, "DeleteZoneRecord")
	}
	return nil
}

func (self *SPvtzRecord) GetGlobalId() string {
	return self.RecordId
}

func (self *SPvtzRecord) GetDnsName() string {
	return self.Rr
}

func (self *SPvtzRecord) GetStatus() string {
	return api.DNS_RECORDSET_STATUS_AVAILABLE
}

func (self *SPvtzRecord) GetEnabled() bool {
	return self.Status == "ENABLE"
}

func (self *SPvtzRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SPvtzRecord) GetDnsValue() string {
	return self.Value
}

func (self *SPvtzRecord) GetTTL() int64 {
	return self.TTL
}

func (self *SPvtzRecord) Delete() error {
	return self.szone.client.DeleteZoneRecord(self.RecordId)
}

func (self *SPvtzRecord) GetExtraAddresses() ([]string, error) {
	return []string{}, nil
}

func (self *SPvtzRecord) GetMxPriority() int64 {
	if self.GetDnsType() == cloudprovider.DnsTypeMX {
		return self.Priority
	}
	return 0
}

func (self *SPvtzRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	return cloudprovider.DnsPolicyTypeSimple
}

func (self *SPvtzRecord) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	return ""
}

func (client *SAliyunClient) UpdateZoneRecord(id string, opts *cloudprovider.DnsRecord) error {
	params := map[string]string{}
	params["Action"] = "UpdateZoneRecord"
	params["RecordId"] = id
	params["Rr"] = opts.DnsName
	params["Type"] = string(opts.DnsType)
	params["Value"] = opts.DnsValue
	params["Ttl"] = strconv.FormatInt(opts.Ttl, 10)
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["Priority"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	_, err := client.pvtzRequest("UpdateZoneRecord", params)
	if err != nil {
		return errors.Wrap(err, "UpdateZoneRecord")
	}
	return nil
}

func (self *SPvtzRecord) Update(opts *cloudprovider.DnsRecord) error {
	return self.szone.client.UpdateZoneRecord(self.RecordId, opts)
}

func (self *SPvtzRecord) Enable() error {
	return self.szone.client.SetZoneRecordStatus(self.RecordId, "ENABLE")
}

func (self *SPvtzRecord) Disable() error {
	return self.szone.client.SetZoneRecordStatus(self.RecordId, "DISABLE")
}
