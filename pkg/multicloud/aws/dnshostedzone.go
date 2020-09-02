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
	"time"

	"github.com/aws/aws-sdk-go/service/route53"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type HostedZoneConfig struct {
	Comment     string `json:"Comment"`
	PrivateZone bool   `json:"PrivateZone"`
}

type AssociatedVPC struct {
	VPCId     string `json:"VPCId"`
	VPCRegion string `json:"VPCRegion"`
}

type SHostedZone struct {
	multicloud.SResourceBase
	client *SAwsClient

	ID                     string           `json:"Id"`
	Name                   string           `json:"Name"`
	Config                 HostedZoneConfig `json:"Config"`
	ResourceRecordSetCount int64            `json:"ResourceRecordSetCount"`
	VPCs                   []AssociatedVPC  `json:"VPCs"`
}

func (self *SHostedZone) GetId() string {
	return self.ID
}

func (self *SHostedZone) GetName() string {
	return self.Name
}

func (self *SHostedZone) GetGlobalId() string {
	return self.ID
}

func (self *SHostedZone) GetStatus() string {
	return ""
}

func (self *SHostedZone) Refresh() error {
	hostedZone, err := self.client.GetHostedZoneById(self.ID)
	if err != nil {
		return errors.Wrapf(err, "self.client.GetHostedZoneById(%s)", self.ID)
	}

	return jsonutils.Update(self, hostedZone)
}

func (client *SAwsClient) CreateHostedZone(opts *cloudprovider.SDnsZoneCreateOptions) (*SHostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.CreateHostedZoneInput{}
	timeStirng := time.Now().String()
	params.CallerReference = &timeStirng
	params.Name = &opts.Name

	Config := route53.HostedZoneConfig{}
	var IsPrivate bool
	if opts.ZoneType == cloudprovider.PrivateZone {
		IsPrivate = true
	}
	Config.Comment = &opts.Desc
	Config.PrivateZone = &IsPrivate
	params.HostedZoneConfig = &Config

	if len(opts.Vpcs) > 0 {
		vpc := route53.VPC{}
		vpc.VPCId = &opts.Vpcs[0].Id
		vpc.VPCRegion = &opts.Vpcs[0].RegionId
		params.SetVPC(&vpc)
	}

	ret, err := route53Client.CreateHostedZone(&params)
	if err != nil {
		return nil, errors.Wrap(err, "route53Client.GetHostedZone()")
	}
	hostedzone := SHostedZone{}
	err = unmarshalAwsOutput(ret, "HostedZone", &hostedzone)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZone)")
	}
	for i := 1; i < len(opts.Vpcs); i++ {
		err := client.AssociateVPCWithHostedZone(opts.Vpcs[i].Id, opts.Vpcs[i].RegionId, hostedzone.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "client.AssociateVPCWithHostedZone(%s,%s,%s)", opts.Vpcs[i].Id, opts.Vpcs[i].RegionId, hostedzone.ID)
		}
	}
	return client.GetHostedZoneById(hostedzone.ID)
}

func (client *SAwsClient) DeleteHostedZone(Id string) error {
	// client
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	// fetch records
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(Id)
	if err != nil {
		return errors.Wrapf(err, "client.GetRoute53ResourceRecordSets(%s)", Id)
	}
	// prepare batch and delete
	deleteRecordSets := []*route53.ResourceRecordSet{}
	for i := 0; i < len(resourceRecordSets); i++ {
		var dnsType string
		if resourceRecordSets[i].Type != nil {
			dnsType = *resourceRecordSets[i].Type
		}
		if dnsType == "NS" || dnsType == "SOA" {
			continue
		}
		deleteRecordSets = append(deleteRecordSets, resourceRecordSets[i])
	}
	if len(deleteRecordSets) > 0 {
		err = client.ChangeResourceRecordSets("DELETE", Id, deleteRecordSets...)
		if err != nil {
			return errors.Wrapf(err, "client.ChangeResourceRecordSets(DELETE, %s, deleteRecordSets)", Id)
		}
	}
	// delete hostedzone
	params := route53.DeleteHostedZoneInput{}
	params.Id = &Id
	_, err = route53Client.DeleteHostedZone(&params)
	if err != nil {
		return errors.Wrapf(err, "route53Client.DeleteHostedZone(%s)", Id)
	}
	return nil
}

func (client *SAwsClient) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return client.CreateHostedZone(opts)
}

func (client *SAwsClient) GetHostedZones() ([]SHostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	result := []SHostedZone{}
	Marker := ""
	MaxItems := "100"
	params := route53.ListHostedZonesInput{}
	for true {
		if len(Marker) > 0 {
			params.Marker = &Marker
		}
		params.MaxItems = &MaxItems
		ret, err := route53Client.ListHostedZones(&params)
		if err != nil {
			return nil, errors.Wrap(err, "route53Client.ListHostedZones(nil)")
		}
		hostedZones := []SHostedZone{}
		err = unmarshalAwsOutput(ret, "HostedZones", &hostedZones)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZones)")
		}
		result = append(result, hostedZones...)
		if !*ret.IsTruncated {
			break
		}
		if ret.Marker != nil {
			Marker = *ret.Marker
		}

	}
	for i := 0; i < len(result); i++ {
		result[i].client = client
	}

	return result, nil
}

func (client *SAwsClient) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	hostedZones, err := client.GetHostedZones()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetHostedZones()")
	}
	result := []cloudprovider.ICloudDnsZone{}
	for i := 0; i < len(hostedZones); i++ {
		hostedZones[i].client = client
		result = append(result, &hostedZones[i])
	}
	return result, nil
}

func (client *SAwsClient) GetHostedZoneById(ID string) (*SHostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.GetHostedZoneInput{}
	params.Id = &ID
	ret, err := route53Client.GetHostedZone(&params)
	if err != nil {
		return nil, errors.Wrap(err, "route53Client.GetHostedZone()")
	}

	result := SHostedZone{client: client}
	err = unmarshalAwsOutput(ret, "HostedZone", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZone)")
	}
	return &result, nil
}

func (client *SAwsClient) AssociateVPCWithHostedZone(vpcId string, regionId string, hostedZoneId string) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.AssociateVPCWithHostedZoneInput{}
	vpcParams := route53.VPC{}
	vpcParams.VPCId = &vpcId
	vpcParams.VPCRegion = &regionId
	params.VPC = &vpcParams
	params.HostedZoneId = &hostedZoneId

	_, err = route53Client.AssociateVPCWithHostedZone(&params)
	if err != nil {
		return errors.Wrap(err, "route53Client.AssociateVPCWithHostedZone()")
	}
	return nil
}

func (client *SAwsClient) DisassociateVPCFromHostedZone(vpcId string, regionId string, hostedZoneId string) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.DisassociateVPCFromHostedZoneInput{}
	vpcParams := route53.VPC{}
	vpcParams.VPCId = &vpcId
	vpcParams.VPCRegion = &regionId
	params.VPC = &vpcParams
	params.HostedZoneId = &hostedZoneId

	_, err = route53Client.DisassociateVPCFromHostedZone(&params)
	if err != nil {
		return errors.Wrap(err, "route53Client.AssociateVPCWithHostedZone()")
	}
	return nil
}

func (self *SHostedZone) Delete() error {
	return self.client.DeleteHostedZone(self.ID)
}

func (self *SHostedZone) GetZoneType() cloudprovider.TDnsZoneType {
	if self.Config.PrivateZone {
		return cloudprovider.PrivateZone
	}
	return cloudprovider.PublicZone
}

func (self *SHostedZone) GetOptions() *jsonutils.JSONDict {
	return nil
}

func (self *SHostedZone) GetICloudVpcIds() ([]string, error) {
	vpcs := []string{}
	if self.Config.PrivateZone {
		for i := 0; i < len(self.VPCs); i++ {
			vpcs = append(vpcs, self.VPCs[i].VPCId)
		}
		return vpcs, nil
	}
	return vpcs, errors.Wrapf(cloudprovider.ErrNotSupported, "not a private hostedzone")
}

func (self *SHostedZone) AddVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	if self.Config.PrivateZone {
		err := self.client.AssociateVPCWithHostedZone(vpc.Id, vpc.RegionId, self.ID)
		if err != nil {
			return errors.Wrapf(err, "self.client.associateVPCWithHostedZone(%s,%s,%s)", vpc.Id, vpc.RegionId, self.ID)
		}
	} else {
		return errors.Wrap(cloudprovider.ErrNotSupported, "public hostedZone not support associate vpc")
	}
	return nil
}

func (self *SHostedZone) RemoveVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	if self.Config.PrivateZone {
		err := self.client.DisassociateVPCFromHostedZone(vpc.Id, vpc.RegionId, self.ID)
		if err != nil {
			return errors.Wrapf(err, "self.client.disassociateVPCFromHostedZone(%s,%s,%s)", vpc.Id, vpc.RegionId, self.ID)
		}
	} else {
		return errors.Wrap(cloudprovider.ErrNotSupported, "public hostedZone not support disassociate vpc")
	}
	return nil
}

func (self *SHostedZone) GetIDnsRecordSets() ([]cloudprovider.ICloudDnsRecordSet, error) {
	recordSets, err := self.client.GetSdnsRecordSets(self.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "self.client.GetSdnsRecordSets(%s)", self.ID)
	}

	result := []cloudprovider.ICloudDnsRecordSet{}
	for i := 0; i < len(recordSets); i++ {
		recordSets[i].hostedZone = self
		result = append(result, &recordSets[i])
	}
	return result, nil
}

func (self *SHostedZone) SyncDnsRecordSets(common, add, del, update []cloudprovider.DnsRecordSet) error {
	for i := 0; i < len(add); i++ {
		err := self.AddDnsRecordSet(&add[i])
		if err != nil {
			return errors.Wrap(err, "self.AddDnsRecordSet()")
		}
	}
	for i := 0; i < len(del); i++ {
		err := self.RemoveDnsRecordSet(&del[i])
		if err != nil {
			return errors.Wrap(err, "self.RemoveDnsRecordSet()")
		}
	}
	for i := 0; i < len(update); i++ {
		err := self.UpdateDnsRecordSet(&update[i])
		if err != nil {
			return errors.Wrap(err, "self.UpdateDnsRecordSet()")
		}
	}
	return nil
}

func (self *SHostedZone) AddDnsRecordSet(opts *cloudprovider.DnsRecordSet) error {
	if len(opts.DnsName) < 1 || opts.DnsName == "@" {
		opts.DnsName = self.Name
	} else {
		opts.DnsName = opts.DnsName + "." + self.Name
	}
	return self.client.AddDnsRecordSet(self.ID, opts)
}

func (self *SHostedZone) UpdateDnsRecordSet(opts *cloudprovider.DnsRecordSet) error {
	if len(opts.DnsName) < 1 || opts.DnsName == "@" {
		opts.DnsName = self.Name
	} else {
		opts.DnsName = opts.DnsName + "." + self.Name
	}
	return self.client.UpdateDnsRecordSet(self.ID, opts)
}

func (self *SHostedZone) RemoveDnsRecordSet(opts *cloudprovider.DnsRecordSet) error {
	if len(opts.DnsName) < 1 || opts.DnsName == "@" {
		opts.DnsName = self.Name
	} else {
		opts.DnsName = opts.DnsName + "." + self.Name
	}
	return self.client.RemoveDnsRecordSet(self.ID, opts)
}
