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
	"fmt"
	"strconv"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SAliasTarget struct {
	DNSName              string `xml:"DNSName"`
	EvaluateTargetHealth *bool  `xml:"EvaluateTargetHealth"`
	HostedZoneId         string `xml:"HostedZoneId"`
}

type SDnsRecord struct {
	zone *SDnsZone

	Name            string `xml:"Name"`
	Type            string `xml:"Type"`
	TTL             int64  `xml:"TTL"`
	ResourceRecords []struct {
		Value string `xml:"Value"`
	} `xml:"ResourceRecords>ResourceRecord"`

	AliasTarget             SAliasTarget       `xml:"AliasTarget"`
	GeoLocation             GeoLocationDetails `xml:"GeoLocation"`
	Region                  string             `xml:"Region"`
	Failover                string             `xml:"Failover"`
	HealthCheckId           string             `xml:"HealthCheckId"`
	MultiValueAnswer        *bool              `xml:"MultiValueAnswer"`
	SetIdentifier           string             `xml:"SetIdentifier"`
	TrafficPolicyInstanceId string             `xml:"TrafficPolicyInstanceId"`
	Weight                  int64              `xml:"Weight"`

	locations []GeoLocationDetails
}

func (client *SAwsClient) ListResourceRecordSet(id string) ([]SDnsRecord, error) {
	params := map[string]string{
		"Id": id,
	}

	ret := []SDnsRecord{}
	for {
		part := struct {
			ResourceRecordSets []SDnsRecord `xml:"ResourceRecordSets>ResourceRecordSet"`
			NextRecordName     string       `xml:"NextRecordName"`
			NextRecordType     string       `xml:"NextRecordType"`
		}{}

		err := client.dnsRequest("ListResourceRecordSets", params, &part)
		if err != nil {
			return nil, err
		}

		ret = append(ret, part.ResourceRecordSets...)
		if len(part.NextRecordName) == 0 && len(part.NextRecordType) == 0 {
			break
		}
		params["Id"] = id
		params["name"] = part.NextRecordName
		params["type"] = part.NextRecordType
	}
	return ret, nil
}

func (self *SDnsRecord) GetStatus() string {
	return api.DNS_RECORDSET_STATUS_AVAILABLE
}

func (self *SDnsRecord) GetEnabled() bool {
	return true
}

func (self *SDnsRecord) GetGlobalId() string {
	return fmt.Sprintf("%s %s %s %s", self.SetIdentifier, self.Type, self.Name, self.GetDnsValue())
}

func (self *SDnsRecord) GetDnsName() string {
	if self.zone == nil {
		return self.Name
	}
	if self.Name == self.zone.Name {
		return "@"
	}
	return strings.TrimSuffix(self.Name, "."+self.zone.Name)
}

func (self *SDnsRecord) Delete() error {
	values := []string{}
	for _, r := range self.ResourceRecords {
		values = append(values, r.Value)
	}
	_, err := self.zone.client.ChangeResourceRecordSets("DELETE", self.zone.Id, self.Name, self.SetIdentifier, cloudprovider.DnsRecord{
		DnsType:     cloudprovider.TDnsType(self.Type),
		Ttl:         self.TTL,
		DnsValue:    strings.Join(values, "\n"),
		PolicyType:  self.GetPolicyType(),
		PolicyValue: self.GetPolicyValue(),
	})
	return err
}

func (self *SDnsRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SDnsRecord) GetDnsValue() string {
	var records []string
	for i := 0; i < len(self.ResourceRecords); i++ {
		value := self.ResourceRecords[i].Value
		if self.Type == "TXT" || self.Type == "SPF" {
			value = value[1 : len(value)-1]
		}
		if self.Type == "MX" {
			strs := strings.Split(value, " ")
			if len(strs) >= 2 {
				value = strs[1]
			}
		}
		records = append(records, value)
	}
	return strings.Join(records, "\n")
}

func (self *SDnsRecord) GetTTL() int64 {
	return self.TTL
}

func (self *SDnsRecord) GetMxPriority() int64 {
	if self.GetDnsType() != cloudprovider.DnsTypeMX {
		return 0
	}
	strs := strings.Split(self.GetDnsValue(), " ")
	if len(strs) > 0 {
		mx, err := strconv.ParseInt(strs[0], 10, 64)
		if err == nil {
			return mx
		}
	}
	return 0
}

func (self *SAwsClient) ChangeResourceRecordSets(action, zoneId, name, id string, opts cloudprovider.DnsRecord) (string, error) {
	record := &SDnsRecord{Name: id, Type: string(opts.DnsType)}
	params := map[string]string{
		"Id":                                  zoneId,
		"ChangeBatch.Changes.0.Change.Action": action,
	}
	if len(name) > 0 {
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.Name"] = name
	}
	if len(id) > 0 && utils.IsInStringArray(string(opts.PolicyType), []string{
		string(cloudprovider.DnsPolicyTypeByGeoLocation),
		string(cloudprovider.DnsPolicyTypeFailover),
		string(cloudprovider.DnsPolicyTypeLatency),
		string(cloudprovider.DnsPolicyTypeWeighted),
		string(cloudprovider.DnsPolicyTypeMultiValueAnswer),
	}) {
		record.SetIdentifier = id
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.SetIdentifier"] = id
	}
	if len(string(opts.DnsType)) > 0 {
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.Type"] = string(opts.DnsType)
	}
	if opts.Ttl > 0 {
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.TTL"] = fmt.Sprintf("%d", opts.Ttl)
	}
	if len(opts.DnsValue) > 0 {
		values := strings.Split(opts.DnsValue, "\n")
		for i, v := range values {
			params[fmt.Sprintf("ChangeBatch.Changes.0.Change.ResourceRecordSet.ResourceRecords.0.ResourceRecord.%d.Value", i)] = v
		}
	}
	switch opts.PolicyType {
	case cloudprovider.DnsPolicyTypeByGeoLocation:
		locations, err := self.ListGeoLocations()
		if err != nil {
			return "", errors.Wrapf(err, "ListGeoLocations")
		}
		find := false
		for i := range locations {
			if locations[i].GetPolicyValue() == opts.PolicyValue {
				if len(locations[i].SubdivisionCode) > 0 {
					params["ChangeBatch.Changes.0.Change.ResourceRecordSet.GeoLocation.CountryCode"] = locations[i].CountryCode
					params["ChangeBatch.Changes.0.Change.ResourceRecordSet.GeoLocation.SubdivisionCode"] = locations[i].SubdivisionCode
				} else if len(locations[i].ContinentCode) > 0 {
					params["ChangeBatch.Changes.0.Change.ResourceRecordSet.GeoLocation.ContinentCode"] = locations[i].ContinentCode
				} else {
					params["ChangeBatch.Changes.0.Change.ResourceRecordSet.GeoLocation.CountryCode"] = locations[i].CountryCode
				}
				find = true
				break
			}
		}
		if !find {
			return "", errors.Errorf("invalid policy value %s %s", opts.PolicyType, opts.PolicyValue)
		}
	case cloudprovider.DnsPolicyTypeFailover:
		return "", cloudprovider.ErrNotImplemented
	case cloudprovider.DnsPolicyTypeWeighted:
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.Weight"] = string(opts.PolicyValue)
	case cloudprovider.DnsPolicyTypeMultiValueAnswer:
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.MultiValueAnswer"] = "true"
	case cloudprovider.DnsPolicyTypeLatency:
		params["ChangeBatch.Changes.0.Change.ResourceRecordSet.Region"] = string(opts.PolicyValue)
	}
	ret := struct {
		ChangeInfo struct {
			Id string `xml:"Id"`
		} `xml:"ChangeInfo"`
	}{}
	err := self.dnsRequest("ChangeResourceRecordSets", params, &ret)
	if err != nil {
		return "", err
	}
	return record.GetGlobalId(), nil
}

// trafficpolicy 信息
func (self *SDnsRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	/*
		Failover         string          `json:"Failover"`
		GeoLocation      GeoLocationCode `json:"GeoLocation"`
		Region           string          `json:"Region"` // latency based
		MultiValueAnswer *bool           `json:"MultiValueAnswer"`
		Weight           *int64          `json:"Weight"`
	*/
	if len(self.Failover) > 0 {
		return cloudprovider.DnsPolicyTypeFailover
	}
	if len(self.GeoLocation.CountryCode) > 0 || len(self.GeoLocation.ContinentCode) > 0 || len(self.GeoLocation.SubdivisionCode) > 0 {
		return cloudprovider.DnsPolicyTypeByGeoLocation
	}
	if len(self.Region) > 0 {
		return cloudprovider.DnsPolicyTypeLatency
	}
	if self.Weight > 0 {
		return cloudprovider.DnsPolicyTypeWeighted
	}
	if self.MultiValueAnswer != nil {
		return cloudprovider.DnsPolicyTypeMultiValueAnswer
	}
	return cloudprovider.DnsPolicyTypeSimple

}

func (self *SDnsRecord) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	if len(self.Failover) > 0 {
		return cloudprovider.TDnsPolicyValue(self.Failover)
	}
	if self.GetPolicyType() == cloudprovider.DnsPolicyTypeByGeoLocation {
		locations, err := self.zone.GetGeoLocations()
		if err != nil {
			return cloudprovider.DnsPolicyValueEmpty
		}
		for i := range locations {
			if locations[i].equals(self.GeoLocation) {
				return locations[i].GetPolicyValue()
			}
		}
	}
	if len(self.Region) > 0 {
		return cloudprovider.TDnsPolicyValue(self.Region)
	}
	if self.MultiValueAnswer != nil {
		return cloudprovider.DnsPolicyValueEmpty
	}
	if self.Weight > 0 {
		return cloudprovider.TDnsPolicyValue(strconv.FormatInt(self.Weight, 10))
	}
	return cloudprovider.DnsPolicyValueEmpty
}

func (self *SDnsRecord) Enable() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDnsRecord) Disable() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDnsRecord) Update(opts *cloudprovider.DnsRecord) error {
	_, err := self.zone.client.ChangeResourceRecordSets("UPSERT", self.zone.Id, self.Name, self.SetIdentifier, *opts)
	return err
}
