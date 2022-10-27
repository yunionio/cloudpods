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
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/route53"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type resourceRecord struct {
	Value string `json:"Value"`
}

type SGeoLocationCode struct {
	// The two-letter code for the continent.
	//
	// Valid values: AF | AN | AS | EU | OC | NA | SA
	//
	// Constraint: Specifying ContinentCode with either CountryCode or SubdivisionCode
	// returns an InvalidInput error.
	ContinentCode string `json:"ContinentCode"`

	// The two-letter code for the country.
	CountryCode string `json:"CountryCode"`

	// The code for the subdivision. Route 53 currently supports only states in
	// the United States.
	SubdivisionCode string `json:"SubdivisionCode"`
}

type SAliasTarget struct {
	DNSName              string `json:"DNSName"`
	EvaluateTargetHealth *bool  `json:"EvaluateTargetHealth"`
	HostedZoneId         string `json:"HostedZoneId"`
}

type SdnsRecordSet struct {
	hostedZone              *SHostedZone
	AliasTarget             SAliasTarget     `json:"AliasTarget"`
	Name                    string           `json:"Name"`
	ResourceRecords         []resourceRecord `json:"ResourceRecords"`
	TTL                     int64            `json:"TTL"`
	TrafficPolicyInstanceId string           `json:"TrafficPolicyInstanceId"`
	Type                    string           `json:"Type"`
	SetIdentifier           string           `json:"SetIdentifier"` // 区别 多值 等名称重复的记录
	// policy info
	Failover         string            `json:"Failover"`
	GeoLocation      *SGeoLocationCode `json:"GeoLocation"`
	Region           string            `json:"Region"` // latency based
	MultiValueAnswer *bool             `json:"MultiValueAnswer"`
	Weight           *int64            `json:"Weight"`

	HealthCheckId string `json:"HealthCheckId"`
}

func (client *SAwsClient) GetSdnsRecordSets(HostedZoneId string) ([]SdnsRecordSet, error) {
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(HostedZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "client.GetRoute53ResourceRecordSets(%s)", HostedZoneId)
	}
	result := []SdnsRecordSet{}
	err = unmarshalAwsOutput(resourceRecordSets, "", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
	}

	return result, nil
}

func (client *SAwsClient) GetRoute53ResourceRecordSets(HostedZoneId string) ([]*route53.ResourceRecordSet, error) {
	// client
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	// fetch records
	resourceRecordSets := []*route53.ResourceRecordSet{}
	listParams := route53.ListResourceRecordSetsInput{}
	StartRecordName := ""
	MaxItems := "100"
	for true {
		if len(StartRecordName) > 0 {
			listParams.StartRecordName = &StartRecordName
		}
		listParams.MaxItems = &MaxItems
		listParams.HostedZoneId = &HostedZoneId
		ret, err := route53Client.ListResourceRecordSets(&listParams)
		if err != nil {
			return nil, errors.Wrap(err, "route53Client.ListResourceRecordSets()")
		}
		resourceRecordSets = append(resourceRecordSets, ret.ResourceRecordSets...)
		if ret.IsTruncated == nil || !*ret.IsTruncated {
			break
		}
		StartRecordName = *ret.NextRecordName
	}
	return resourceRecordSets, nil
}

// CREATE, DELETE, UPSERT
func (client *SAwsClient) ChangeResourceRecordSets(action string, hostedZoneId string, resourceRecordSets ...*route53.ResourceRecordSet) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "client.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	ChangeBatch := route53.ChangeBatch{}
	for i := 0; i < len(resourceRecordSets); i++ {
		change := route53.Change{}
		change.Action = &action
		change.ResourceRecordSet = resourceRecordSets[i]
		ChangeBatch.Changes = append(ChangeBatch.Changes, &change)
	}

	changeParams := route53.ChangeResourceRecordSetsInput{}
	changeParams.HostedZoneId = &hostedZoneId
	changeParams.ChangeBatch = &ChangeBatch
	_, err = route53Client.ChangeResourceRecordSets(&changeParams)
	if err != nil {
		return errors.Wrap(err, "route53Client.ChangeResourceRecordSets(&params)")
	}
	return nil
}

func Getroute53ResourceRecordSet(client *SAwsClient, opts *cloudprovider.DnsRecordSet) (*route53.ResourceRecordSet, error) {
	resourceRecordSet := route53.ResourceRecordSet{}
	resourceRecordSet.SetName(opts.DnsName)
	resourceRecordSet.SetTTL(opts.Ttl)
	resourceRecordSet.SetType(string(opts.DnsType))
	if len(opts.ExternalId) > 0 {
		resourceRecordSet.SetSetIdentifier(opts.ExternalId)
	}
	records := []*route53.ResourceRecord{}
	values := strings.Split(opts.DnsValue, "\n")
	for i := 0; i < len(values); i++ {
		value := values[i]
		if opts.DnsType == cloudprovider.DnsTypeTXT || opts.DnsType == cloudprovider.DnsTypeSPF {
			value = "\"" + value + "\""
		}
		if opts.DnsType == cloudprovider.DnsTypeMX {
			value = strconv.FormatInt(opts.MxPriority, 10) + " " + value
		}
		records = append(records, &route53.ResourceRecord{Value: &value})
	}
	resourceRecordSet.SetResourceRecords(records)

	// traffic policy info--------------------------------------------
	if opts.PolicyType == cloudprovider.DnsPolicyTypeSimple {
		return &resourceRecordSet, nil
	}
	// SetIdentifier 设置policy需要 ,也可以通过externalId设置
	if resourceRecordSet.SetIdentifier == nil {
		resourceRecordSet.SetSetIdentifier(stringutils.UUID4())
	}
	// addition option(health check)
	if opts.PolicyOptions != nil {
		health := struct {
			HealthCheckId string
		}{}
		opts.PolicyOptions.Unmarshal(&health)
		if len(health.HealthCheckId) > 0 {
			resourceRecordSet.SetHealthCheckId(health.HealthCheckId)
		}
	}

	// failover choice:PRIMARY|SECONDARY
	if opts.PolicyType == cloudprovider.DnsPolicyTypeFailover {
		resourceRecordSet.SetFailover(string(opts.PolicyValue))
	}
	// geolocation
	if opts.PolicyType == cloudprovider.DnsPolicyTypeByGeoLocation {
		Geo := route53.GeoLocation{}
		locations, err := client.ListGeoLocations()
		if err != nil {
			return nil, errors.Wrap(err, "client.ListGeoLocations()")
		}
		matchedIndex := -1
		for i := 0; i < len(locations); i++ {
			if locations[i].SubdivisionName != nil {
				if string(opts.PolicyValue) == *locations[i].SubdivisionName {
					matchedIndex = i
					break
				}
			}
			if locations[i].CountryName != nil {
				if string(opts.PolicyValue) == *locations[i].CountryName {
					matchedIndex = i
					break
				}
			}
			if locations[i].ContinentCode != nil {
				if string(opts.PolicyValue) == *locations[i].ContinentCode {
					matchedIndex = i
					break
				}
			}
		}
		if matchedIndex < 0 || matchedIndex >= len(locations) {
			return nil, errors.Wrap(cloudprovider.ErrNotSupported, "Can't find Support for this location")
		}
		Geo.ContinentCode = locations[matchedIndex].ContinentCode
		Geo.CountryCode = locations[matchedIndex].CountryCode
		Geo.SubdivisionCode = locations[matchedIndex].SubdivisionCode
		resourceRecordSet.SetGeoLocation(&Geo)
	}
	//  latency ,region based
	if opts.PolicyType == cloudprovider.DnsPolicyTypeLatency {
		resourceRecordSet.SetRegion(string(opts.PolicyValue))
	}
	// MultiValueAnswer ,bool
	if opts.PolicyType == cloudprovider.DnsPolicyTypeMultiValueAnswer {
		var multiValueAnswer bool = true
		resourceRecordSet.SetMultiValueAnswer(multiValueAnswer)
	}
	// Weighted.,int64 value
	if opts.PolicyType == cloudprovider.DnsPolicyTypeWeighted {
		weight, _ := strconv.Atoi(string(opts.PolicyValue))
		resourceRecordSet.SetWeight(int64(weight))
	}

	return &resourceRecordSet, nil
}

func (client *SAwsClient) AddDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSet, err := Getroute53ResourceRecordSet(client, opts)
	if err != nil {
		return errors.Wrapf(err, "Getroute53ResourceRecordSet(%s)", jsonutils.Marshal(opts).String())
	}
	err = client.ChangeResourceRecordSets("CREATE", hostedZoneId, resourceRecordSet)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "CREATE",%s)`, hostedZoneId)
	}
	return nil
}

func (client *SAwsClient) UpdateDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSet, err := Getroute53ResourceRecordSet(client, opts)
	if err != nil {
		return errors.Wrapf(err, "Getroute53ResourceRecordSet(%s)", jsonutils.Marshal(opts).String())
	}
	err = client.ChangeResourceRecordSets("UPSERT", hostedZoneId, resourceRecordSet)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "CREATE",%s)`, hostedZoneId)
	}
	return nil
}

func (client *SAwsClient) RemoveDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(hostedZoneId)
	if err != nil {
		return errors.Wrapf(err, "self.client.GetRoute53ResourceRecordSets(%s)", hostedZoneId)
	}
	for i := 0; i < len(resourceRecordSets); i++ {
		srecordSet := SdnsRecordSet{}
		err = unmarshalAwsOutput(resourceRecordSets[i], "", &srecordSet)
		if err != nil {
			return errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
		}
		if srecordSet.match(opts) {
			err := client.ChangeResourceRecordSets("DELETE", hostedZoneId, resourceRecordSets[i])
			if err != nil {
				return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "DELETE",%s)`, hostedZoneId)
			}
			return nil
		}
	}
	return nil
}

func (self *SdnsRecordSet) GetStatus() string {
	return api.DNS_RECORDSET_STATUS_AVAILABLE
}

func (self *SdnsRecordSet) GetEnabled() bool {
	return true
}

func (self *SdnsRecordSet) GetGlobalId() string {
	return self.SetIdentifier
}

func (self *SdnsRecordSet) GetDnsName() string {
	if self.hostedZone == nil {
		return self.Name
	}
	if self.Name == self.hostedZone.Name {
		return "@"
	}
	return strings.TrimSuffix(self.Name, "."+self.hostedZone.Name)
}

func (self *SdnsRecordSet) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SdnsRecordSet) GetDnsValue() string {
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

func (self *SdnsRecordSet) GetTTL() int64 {
	return self.TTL
}

func (self *SdnsRecordSet) GetMxPriority() int64 {
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
	log.Errorf("can't parse mxpriority:%s", self.GetDnsValue())
	return 0
}

// trafficpolicy 信息
func (self *SdnsRecordSet) GetPolicyType() cloudprovider.TDnsPolicyType {
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
	if self.GeoLocation != nil {
		return cloudprovider.DnsPolicyTypeByGeoLocation
	}
	if len(self.Region) > 0 {
		return cloudprovider.DnsPolicyTypeLatency
	}
	if self.MultiValueAnswer != nil {
		return cloudprovider.DnsPolicyTypeMultiValueAnswer
	}
	if self.Weight != nil {
		return cloudprovider.DnsPolicyTypeWeighted
	}
	return cloudprovider.DnsPolicyTypeSimple

}

func (self *SdnsRecordSet) GetPolicyOptions() *jsonutils.JSONDict {
	options := jsonutils.NewDict()
	if len(self.HealthCheckId) > 0 {
		options.Add(jsonutils.NewString(self.HealthCheckId), "health_check_id")
	}
	return options
}

func CodeMatch(s string, d *string) bool {
	if d == nil {
		return len(s) == 0
	}
	return s == *d
}

func (self *SdnsRecordSet) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	if len(self.Failover) > 0 {
		return cloudprovider.TDnsPolicyValue(self.Failover)
	}
	if self.GeoLocation != nil {
		locations, err := self.hostedZone.client.ListGeoLocations()
		log.Errorf("List aws route53 locations failed!")
		if err != nil {
			return ""
		}
		for i := 0; i < len(locations); i++ {
			if CodeMatch(self.GeoLocation.SubdivisionCode, locations[i].SubdivisionCode) &&
				CodeMatch(self.GeoLocation.CountryCode, locations[i].CountryCode) &&
				CodeMatch(self.GeoLocation.ContinentCode, locations[i].ContinentCode) {
				if locations[i].SubdivisionCode != nil {
					return cloudprovider.TDnsPolicyValue(*locations[i].SubdivisionName)
				}
				if locations[i].CountryCode != nil {
					return cloudprovider.TDnsPolicyValue(*locations[i].CountryName)
				}
				if locations[i].ContinentCode != nil {
					return cloudprovider.TDnsPolicyValue(*locations[i].ContinentName)
				}
			}
		}
		return ""
	}
	if len(self.Region) > 0 {
		return cloudprovider.TDnsPolicyValue(self.Region)
	}
	if self.MultiValueAnswer != nil {
		return cloudprovider.DnsPolicyValueEmpty
	}
	if self.Weight != nil {
		return cloudprovider.TDnsPolicyValue(strconv.FormatInt(*self.Weight, 10))
	}
	return cloudprovider.DnsPolicyValueEmpty
}

func (self *SdnsRecordSet) match(change *cloudprovider.DnsRecordSet) bool {
	if change.DnsName != self.GetDnsName() {
		return false
	}
	if change.DnsValue != self.GetDnsValue() {
		return false
	}
	if change.Ttl != self.GetTTL() {
		return false
	}
	if change.DnsType != self.GetDnsType() {
		return false
	}
	if change.ExternalId != self.GetGlobalId() {
		return false
	}
	return true
}
