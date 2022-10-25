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

package hcs

import (
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	multicloud.HcsTags
	region *SRegion

	Id                string
	Name              string
	Description       string
	Spec              string
	Status            string
	InternalNetworkId string
	CreatedTime       string `json:"created_at"`
}

func (gateway *SNatGateway) GetId() string {
	return gateway.Id
}

func (gateway *SNatGateway) GetName() string {
	return gateway.Name
}

func (gateway *SNatGateway) GetGlobalId() string {
	return gateway.GetId()
}

func (gateway *SNatGateway) GetStatus() string {
	return NatResouceStatusTransfer(gateway.Status)
}

func (self *SNatGateway) Delete() error {
	return self.region.DeleteNatGateway(self.Id)
}

func (self *SNatGateway) Refresh() error {
	nat, err := self.region.GetNatGateway(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetNatGateway(%s)", self.Id)
	}
	return jsonutils.Update(self, nat)
}

func (self *SNatGateway) GetINetworkId() string {
	return self.InternalNetworkId
}

func (gateway *SNatGateway) GetNatSpec() string {
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

func (gateway *SNatGateway) GetDescription() string {
	return gateway.Description
}

func (gateway *SNatGateway) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (gateway *SNatGateway) GetCreatedAt() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05.000000", gateway.CreatedTime)
	return t
}

func (gateway *SNatGateway) GetExpiredAt() time.Time {
	return time.Time{}
}

func (gateway *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	IEips, err := gateway.region.GetIEips()
	if err != nil {
		return nil, errors.Wrapf(err, `get all Eips of region %q error`, gateway.region.GetId())
	}
	dNatTables, err := gateway.GetINatDTable()
	if err != nil {
		return nil, errors.Wrapf(err, `get all DNatTable of gateway %q error`, gateway.GetId())
	}
	sNatTables, err := gateway.GetINatSTable()
	if err != nil {
		return nil, errors.Wrapf(err, `get all SNatTable of gateway %q error`, gateway.GetId())
	}

	// Get natIPSet of nat rules
	natIPSet := make(map[string]struct{})
	for _, snat := range sNatTables {
		natIPSet[snat.GetIP()] = struct{}{}
	}
	for _, dnat := range dNatTables {
		natIPSet[dnat.GetExternalIp()] = struct{}{}
	}

	// Add Eip whose GetIpAddr() in natIPSet to ret
	ret := make([]cloudprovider.ICloudEIP, 0, 2)
	for i := range IEips {
		if _, ok := natIPSet[IEips[i].GetIpAddr()]; ok {
			ret = append(ret, IEips[i])
		}
	}
	return ret, nil
}

func (gateway *SNatGateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	dNatTable, err := gateway.getNatDTable()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, gateway.GetId())
	}
	ret := make([]cloudprovider.ICloudNatDEntry, len(dNatTable))
	for i := range dNatTable {
		ret[i] = &dNatTable[i]
	}
	return ret, nil
}

func (gateway *SNatGateway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
	sNatTable, err := gateway.getNatSTable()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, gateway.GetId())
	}
	ret := make([]cloudprovider.ICloudNatSEntry, len(sNatTable))
	for i := range sNatTable {
		ret[i] = &sNatTable[i]
	}
	return ret, nil
}

func (gateway *SNatGateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	dnat, err := gateway.region.CreateNatDEntry(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	dnat.gateway = gateway
	return dnat, nil
}

func (gateway *SNatGateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	snat, err := gateway.region.CreateNatSEntry(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	snat.gateway = gateway
	return snat, nil
}

func (gateway *SNatGateway) GetINatDEntryByID(id string) (cloudprovider.ICloudNatDEntry, error) {
	dnat, err := gateway.region.GetNatDEntry(id)
	if err != nil {
		return nil, err
	}
	dnat.gateway = gateway
	return dnat, nil
}

func (gateway *SNatGateway) GetINatSEntryByID(id string) (cloudprovider.ICloudNatSEntry, error) {
	snat, err := gateway.region.GetNatSEntry(id)
	if err != nil {
		return nil, err
	}
	snat.gateway = gateway
	return snat, nil
}

func (self *SRegion) GetNatGateways(vpcId string) ([]SNatGateway, error) {
	query := url.Values{}
	if len(vpcId) > 0 {
		query.Set("router_id", vpcId)
	}
	ret := []SNatGateway{}
	return ret, self.list("nat", "v2.0", "nat_gateways", query, &ret)
}

func NatResouceStatusTransfer(status string) string {
	// In Huawei Cloud, there are isx resource status of Nat, "ACTIVE", "PENDING_CREATE",
	// "PENDING_UPDATE", "PENDING_DELETE", "EIP_FREEZED", "INACTIVE".
	switch status {
	case "ACTIVE":
		return api.NAT_STAUTS_AVAILABLE
	case "PENDING_CREATE":
		return api.NAT_STATUS_ALLOCATE
	case "PENDING_UPDATE", "PENDING_DELETE":
		return api.NAT_STATUS_DEPLOYING
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (self *SRegion) GetNatGateway(id string) (*SNatGateway, error) {
	ret := &SNatGateway{region: self}
	return ret, self.get("nat", "v2.0", "nat_gateways/"+id, ret)
}

func (self *SVpc) CreateINatGateway(opts *cloudprovider.NatGatewayCreateOptions) (cloudprovider.ICloudNatGateway, error) {
	nat, err := self.region.CreateNatGateway(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNatGateway")
	}
	return nat, nil
}

func (self *SRegion) CreateNatGateway(opts *cloudprovider.NatGatewayCreateOptions) (*SNatGateway, error) {
	spec := ""
	switch strings.ToLower(opts.NatSpec) {
	case api.NAT_SPEC_SMALL:
		spec = "1"
	case api.NAT_SPEC_MIDDLE:
		spec = "2"
	case api.NAT_SPEC_LARGE:
		spec = "3"
	case api.NAT_SPEC_XLARGE:
		spec = "4"
	}
	params := map[string]interface{}{
		"nat_gateway": map[string]interface{}{
			"name":                opts.Name,
			"description":         opts.Desc,
			"router_id":           opts.VpcId,
			"internal_network_id": opts.NetworkId,
			"spec":                spec,
		},
	}
	ret := SNatGateway{region: self}
	return &ret, self.create("nat", "v2.0", "nat_gateways", params, &ret)
}

func (self *SRegion) DeleteNatGateway(id string) error {
	return self.delete("nat", "v2.0", "nat_gateways/"+id)
}
