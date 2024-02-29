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

package huawei

import (
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// 公网NAT
type SNatgateway struct {
	multicloud.SNatGatewayBase
	HuaweiTags
	region *SRegion

	Id           string
	Name         string
	Description  string
	Spec         string
	Status       string
	AdminStateUp bool

	InternalNetworkId     string
	RouterId              string
	CreatedAt             string `json:"created_at"`
	EnterpriseProject_id  string
	NgportIpAddress       string
	BillingInfo           string
	DnatRulesLimit        int
	SnatRulePublicIpLimit int
}

func (gateway *SNatgateway) GetId() string {
	return gateway.Id
}

func (gateway *SNatgateway) GetName() string {
	return gateway.Name
}

func (gateway *SNatgateway) GetGlobalId() string {
	return gateway.GetId()
}

func (gateway *SNatgateway) GetStatus() string {
	return NatResouceStatusTransfer(gateway.Status)
}

func (self *SNatgateway) Delete() error {
	return self.region.DeleteNatgateway(self.Id)
}

func (self *SNatgateway) Refresh() error {
	nat, err := self.region.GetNatgateway(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetNatgateway(%s)", self.Id)
	}
	return jsonutils.Update(self, nat)
}

func (self *SNatgateway) GetINetworkId() string {
	return self.InternalNetworkId
}

func (self *SNatgateway) GetNetworkType() string {
	return api.NAT_NETWORK_TYPE_INTERNET
}

func (gateway *SNatgateway) GetNatSpec() string {
	switch gateway.Spec {
	case "1":
		return api.NAT_SPEC_SMALL
	case "2":
		return api.NAT_SPEC_MIDDLE
	case "3":
		return api.NAT_SPEC_LARGE
	case "4":
		return api.NAT_SPEC_XLARGE
	}
	return gateway.Spec
}

func (gateway *SNatgateway) GetDescription() string {
	return gateway.Description
}

func (gateway *SNatgateway) GetBillingType() string {
	if len(gateway.BillingInfo) > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (gateway *SNatgateway) GetCreatedAt() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05.000000", gateway.CreatedAt)
	return t
}

func (gateway *SNatgateway) GetExpiredAt() time.Time {
	// no support for expired time
	return time.Time{}
}

func (gateway *SNatgateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	ports, err := gateway.region.GetPorts(gateway.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPorts(%s)", gateway.Id)
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := range ports {
		eips, err := gateway.region.GetEips(ports[i].ID, nil)
		if err != nil {
			return nil, err
		}
		for i := range eips {
			eips[i].region = gateway.region
			ret = append(ret, &eips[i])
		}
	}
	return ret, nil
}

func (gateway *SNatgateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	dNatTable, err := gateway.region.GetPublicNatDTable(gateway.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPublicNatDTable")
	}
	ret := []cloudprovider.ICloudNatDEntry{}
	for i := range dNatTable {
		dNatTable[i].nat = gateway
		ret = append(ret, &dNatTable[i])
	}
	return ret, nil
}

func (gateway *SNatgateway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
	sNatTable, err := gateway.region.GetPublicNatSTable(gateway.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNatSTable")
	}
	ret := []cloudprovider.ICloudNatSEntry{}
	for i := range sNatTable {
		sNatTable[i].nat = gateway
		ret = append(ret, &sNatTable[i])
	}
	return ret, nil
}

func (gateway *SNatgateway) GetINatDEntryById(id string) (cloudprovider.ICloudNatDEntry, error) {
	dnat, err := gateway.region.GetNatDTable(id)
	if err != nil {
		return nil, err
	}
	dnat.nat = gateway
	return dnat, nil
}

func (gateway *SNatgateway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	snat, err := gateway.region.GetNatSEntry(id)
	if err != nil {
		return nil, err
	}
	snat.nat = gateway
	return snat, nil
}

func (region *SRegion) GetNatgateways(vpcId, id string) ([]SNatgateway, error) {
	query := url.Values{}
	if len(id) != 0 {
		query.Set("id", id)
	}
	if len(vpcId) != 0 {
		query.Set("router_id", vpcId)
	}
	ret := []SNatgateway{}
	for {
		resp, err := region.list(SERVICE_NAT_V2, "nat_gateways", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			NatGateways []SNatgateway
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.NatGateways...)
		if len(part.NatGateways) == 0 {
			break
		}
		query.Set("marker", part.NatGateways[len(part.NatGateways)-1].Id)
	}
	return ret, nil
}

func (region *SRegion) GetNatDTable(id string) (*SNatDTable, error) {
	resp, err := region.list(SERVICE_NAT_V2, "dnat_rules/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNatDTable{}
	err = resp.Unmarshal(ret, "dnat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetNatgateway(id string) (*SNatgateway, error) {
	resp, err := self.list(SERVICE_NAT_V2, "nat_gateways/"+id, nil)
	if err != nil {
		return nil, err
	}
	nat := &SNatgateway{region: self}
	err = resp.Unmarshal(nat, "nat_gateway")
	if err != nil {
		return nil, err
	}
	return nat, nil
}

func (self *SRegion) DeleteNatgateway(id string) error {
	_, err := self.delete(SERVICE_NAT_V2, "nat_gateways/"+id)
	return err
}
