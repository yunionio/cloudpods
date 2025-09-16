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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type DnsZoneConfig struct {
	Comment     string `xml:"Comment"`
	PrivateZone bool   `xml:"PrivateZone"`
}

type AssociatedVPC struct {
	VPCId     string `xml:"VPCId"`
	VPCRegion string `xml:"VPCRegion"`
}

type SDnsZone struct {
	multicloud.SVirtualResourceBase
	AwsTags
	client *SAwsClient

	Id                     string        `xml:"Id"`
	Name                   string        `xml:"Name"`
	CallerReference        string        `xml:"CallerReference"`
	Config                 DnsZoneConfig `xml:"Config"`
	ResourceRecordSetCount int64         `xml:"ResourceRecordSetCount"`

	locations []GeoLocationDetails
}

type SDnsZoneDetails struct {
	HostedZone SDnsZone `xml:"HostedZone"`
	VPC        []struct {
		VPCRegion string `xml:"VPCRegion"`
		VPCId     string `xml:"VPCId"`
	} `xml:"VPCs>VPC"`
}

func (self *SDnsZone) GetId() string {
	return self.Id
}

func (self *SDnsZone) GetName() string {
	return strings.TrimSuffix(self.Name, ".")
}

func (self *SDnsZone) GetGlobalId() string {
	return self.Id
}

func (self *SDnsZone) GetStatus() string {
	return api.DNS_ZONE_STATUS_AVAILABLE
}

func (self *SDnsZone) Refresh() error {
	zone, err := self.client.GetDnsZone(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone(%s)", self.Id)
	}
	return jsonutils.Update(self, zone.HostedZone)
}

type GeoLocationDetails struct {
	ContinentCode   string `xml:"ContinentCode"`
	ContinentName   string `xml:"ContinentName"`
	CountryCode     string `xml:"CountryCode"`
	CountryName     string `xml:"CountryName"`
	SubdivisionCode string `xml:"SubdivisionCode"`
	SubdivisionName string `xml:"SubdivisionName"`
}

func (self GeoLocationDetails) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	if len(self.SubdivisionName) > 0 {
		return cloudprovider.TDnsPolicyValue(self.SubdivisionName)
	}
	if len(self.ContinentName) > 0 {
		return cloudprovider.TDnsPolicyValue(self.ContinentName)
	}
	return cloudprovider.TDnsPolicyValue(self.CountryName)
}

func (self GeoLocationDetails) equals(lo GeoLocationDetails) bool {
	return self.CountryCode == lo.CountryCode && self.ContinentCode == self.ContinentCode && self.SubdivisionCode == self.SubdivisionCode
}

func (client *SAwsClient) ListGeoLocations() ([]GeoLocationDetails, error) {
	ret := []GeoLocationDetails{}
	params := map[string]string{
		"maxitems": "100",
	}
	for {
		part := struct {
			Locations           []GeoLocationDetails `xml:"GeoLocationDetailsList>GeoLocationDetails"`
			NextCountryCode     string               `xml:"NextCountryCode"`
			NextContinentCode   string               `xml:"NextContinentCode"`
			NextSubdivisionCode string               `xml:"NextSubdivisionCode"`
		}{}
		err := client.dnsRequest("ListGeoLocations", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Locations...)
		if len(part.NextCountryCode) == 0 && len(part.NextContinentCode) == 0 && len(part.NextSubdivisionCode) == 0 {
			break
		}
		for k, v := range map[string]string{
			"startcountrycode":     part.NextCountryCode,
			"startcontinentcode":   part.NextContinentCode,
			"startsubdivisioncode": part.NextSubdivisionCode,
		} {
			if len(v) > 0 {
				params[k] = v
			}
		}
	}
	return ret, nil
}

func (client *SAwsClient) CreateDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (*SDnsZone, error) {
	params := map[string]string{
		"CallerReference":              time.Now().Format(time.RFC3339),
		"HostedZoneConfig.PrivateZone": "true",
		"Name":                         opts.Name,
	}
	if opts.ZoneType == cloudprovider.PrivateZone {
		params["HostedZoneConfig.PrivateZone"] = "true"
	}
	if len(opts.Desc) > 0 {
		params["HostedZoneConfig.Comment"] = opts.Desc
	}
	for i, vpc := range opts.Vpcs {
		if i == 0 {
			params["VPC.VPCRegion"] = vpc.RegionId
			params["VPC.VPCId"] = vpc.Id
		}
	}

	ret := SDnsZoneDetails{}
	ret.HostedZone.client = client
	err := client.dnsRequest("CreateHostedZone", params, &ret)
	if err != nil {
		return nil, err
	}
	for i, vpc := range opts.Vpcs {
		if i != 0 {
			err = ret.HostedZone.AddVpc(&vpc)
			if err != nil {
				log.Errorf("add vpc %s(%s) error: %v", vpc.Id, vpc.RegionId, err)
			}
		}
	}
	return &ret.HostedZone, nil
}

func (client *SAwsClient) DeleteDnsZone(Id string) error {
	params := map[string]string{
		"Id": Id,
	}
	return client.dnsRequest("DeleteHostedZone", params, nil)
}

func (client *SAwsClient) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return client.CreateDnsZone(opts)
}

func (client *SAwsClient) GetDnsZones() ([]SDnsZone, error) {
	params := map[string]string{
		"maxitems": "1",
	}
	ret := []SDnsZone{}
	for {
		part := struct {
			DnsZones   []SDnsZone `xml:"HostedZones>HostedZone"`
			NextMarker string     `xml:"NextMarker"`
		}{}
		err := client.dnsRequest("ListHostedZones", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.DnsZones...)
		if len(part.NextMarker) == 0 {
			break
		}
		params["marker"] = part.NextMarker
	}
	return ret, nil
}

func (client *SAwsClient) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	zones, err := client.GetDnsZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetDnsZones()")
	}
	result := []cloudprovider.ICloudDnsZone{}
	for i := 0; i < len(zones); i++ {
		zones[i].client = client
		result = append(result, &zones[i])
	}
	return result, nil
}

func (client *SAwsClient) GetDnsZone(id string) (*SDnsZoneDetails, error) {
	params := map[string]string{"Id": id}
	ret := SDnsZoneDetails{}
	err := client.dnsRequest("GetHostedZone", params, &ret)
	if err != nil {
		return nil, err
	}
	ret.HostedZone.client = client
	return &ret, nil
}

func (client *SAwsClient) AssociateVPCWithHostedZone(vpcId string, regionId string, zoneId string) error {
	params := map[string]string{
		"Id":            zoneId,
		"VPC.VPCId":     vpcId,
		"VPC.VPCRegion": regionId,
	}
	ret := struct{}{}
	return client.dnsRequest("AssociateVPCWithHostedZone", params, &ret)
}

func (client *SAwsClient) DisassociateVPCFromHostedZone(vpcId string, regionId string, zoneId string) error {
	params := map[string]string{
		"Id":            zoneId,
		"VPC.VPCId":     vpcId,
		"VPC.VPCRegion": regionId,
	}
	ret := struct{}{}
	return client.dnsRequest("DisassociateVPCFromHostedZone", params, &ret)
}

func (self *SDnsZone) Delete() error {
	records, err := self.client.ListResourceRecordSet(self.Id)
	if err != nil {
		return errors.Wrapf(err, "ListResourceRecordSet")
	}
	for i := range records {
		if records[i].Type == "NS" || records[i].Type == "SOA" {
			continue
		}
		records[i].zone = self
		err = records[i].Delete()
		if err != nil {
			return errors.Wrapf(err, "Delete record %s", records[i].GetGlobalId())
		}
	}
	return self.client.DeleteDnsZone(self.Id)
}

func (self *SDnsZone) GetZoneType() cloudprovider.TDnsZoneType {
	if self.Config.PrivateZone {
		return cloudprovider.PrivateZone
	}
	return cloudprovider.PublicZone
}

func (self *SDnsZone) GetICloudVpcIds() ([]string, error) {
	zone, err := self.client.GetDnsZone(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for _, vpc := range zone.VPC {
		ret = append(ret, vpc.VPCId)
	}
	return ret, nil
}

func (self *SDnsZone) AddVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return self.client.AssociateVPCWithHostedZone(vpc.Id, vpc.RegionId, self.Id)
}

func (self *SDnsZone) RemoveVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return self.client.DisassociateVPCFromHostedZone(vpc.Id, vpc.RegionId, self.Id)
}

func (self *SDnsZone) GetIDnsRecords() ([]cloudprovider.ICloudDnsRecord, error) {
	recordSets, err := self.client.ListResourceRecordSet(self.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "ListResourceRecordSet(%s)", self.Id)
	}

	result := []cloudprovider.ICloudDnsRecord{}
	for i := 0; i < len(recordSets); i++ {
		recordSets[i].zone = self
		result = append(result, &recordSets[i])
	}
	return result, nil
}

func (self *SDnsZone) GetIDnsRecordById(id string) (cloudprovider.ICloudDnsRecord, error) {
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

func (self *SDnsZone) GetGeoLocations() ([]GeoLocationDetails, error) {
	if len(self.locations) > 0 {
		return self.locations, nil
	}
	var err error
	self.locations, err = self.client.ListGeoLocations()
	return self.locations, err
}

func (self *SDnsZone) AddDnsRecord(opts *cloudprovider.DnsRecord) (string, error) {
	name := opts.DnsName
	if len(opts.DnsName) > 1 && opts.DnsName != "@" {
		name = opts.DnsName + "." + self.Name
	}
	id := stringutils.UUID4()
	return self.client.ChangeResourceRecordSets("CREATE", self.Id, name, id, *opts)
}

func (self *SDnsZone) GetDnsProductType() cloudprovider.TDnsProductType {
	return ""
}
