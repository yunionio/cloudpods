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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	HuaweiTags
	region *SRegion

	ID                string
	Name              string
	Description       string
	Spec              string
	Status            string
	InternalNetworkId string
	CreatedTime       string `json:"created_at"`
}

func (gateway *SNatGateway) GetId() string {
	return gateway.ID
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
	return self.region.DeleteNatGateway(self.ID)
}

func (self *SNatGateway) Refresh() error {
	nat, err := self.region.GetNatGateway(self.ID)
	if err != nil {
		return errors.Wrapf(err, "GetNatGateway(%s)", self.ID)
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
	// Up to 2019.07.17, only support post pay
	return billing_api.BILLING_TYPE_POSTPAID
}

func (gateway *SNatGateway) GetCreatedAt() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05.000000", gateway.CreatedTime)
	return t
}

func (gateway *SNatGateway) GetExpiredAt() time.Time {
	// no support for expired time
	return time.Time{}
}

func (gateway *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	ports, err := gateway.region.GetPorts(gateway.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPorts(%s)", gateway.ID)
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
	dnat, err := gateway.region.GetNatDEntryByID(id)
	if err != nil {
		return nil, err
	}
	dnat.gateway = gateway
	return dnat, nil
}

func (gateway *SNatGateway) GetINatSEntryByID(id string) (cloudprovider.ICloudNatSEntry, error) {
	snat, err := gateway.region.GetNatSEntryByID(id)
	if err != nil {
		return nil, err
	}
	snat.gateway = gateway
	return snat, nil
}

func (region *SRegion) GetNatGateways(vpcId, id string) ([]SNatGateway, error) {
	query := url.Values{}
	if len(id) != 0 {
		query.Set("id", id)
	}
	if len(vpcId) != 0 {
		query.Set("vpc_id", vpcId)
	}
	ret := []SNatGateway{}
	for {
		resp, err := region.list(SERVICE_NAT, "private-nat/gateways", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Gateways []SNatGateway
			PageInfo sPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Gateways...)
		if len(part.PageInfo.NextMarker) == 0 || len(part.Gateways) == 0 {
			break
		}
		query.Set("marker", part.PageInfo.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) CreateNatDEntry(rule cloudprovider.SNatDRule, gatewayID string) (*SNatDEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayID
	params["private_ip"] = rule.InternalIP
	params["internal_service_port"] = rule.InternalPort
	params["floating_ip_id"] = rule.ExternalIPID
	params["external_service_port"] = rule.ExternalPort
	params["protocol"] = rule.Protocol

	resp, err := region.post(SERVICE_NAT, "private-nat/dnat-rules", map[string]interface{}{"dnat_rule": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create dnat")
	}
	ret := &SNatDEntry{}
	err = resp.Unmarshal(ret, "dnat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) CreateNatSEntry(rule cloudprovider.SNatSRule, gatewayId string) (*SNatSEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayId
	if len(rule.NetworkID) != 0 {
		params["network_id"] = rule.NetworkID
	}
	if len(rule.SourceCIDR) != 0 {
		params["cidr"] = rule.SourceCIDR
	}
	params["floating_ip_id"] = rule.ExternalIPID
	resp, err := region.post(SERVICE_NAT, "private-nat/snat-rules", map[string]interface{}{"snat_rule": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create snat")
	}
	ret := &SNatSEntry{}
	err = resp.Unmarshal(ret, "snat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetNatDEntryByID(id string) (*SNatDEntry, error) {
	resp, err := region.list(SERVICE_NAT, "private-nat/dnat-rules/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNatDEntry{}
	err = resp.Unmarshal(ret, "dnat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetNatSEntryByID(id string) (*SNatSEntry, error) {
	resp, err := region.list(SERVICE_NAT, "private-nat/snat-rules/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNatSEntry{}
	err = resp.Unmarshal(ret, "snat_rule")
	if err != nil {
		return nil, err
	}
	return ret, nil
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
	resp, err := self.list(SERVICE_NAT, "private-nat/gateways/"+id, nil)
	if err != nil {
		return nil, err
	}
	nat := &SNatGateway{region: self}
	err = resp.Unmarshal(nat, "gateway")
	if err != nil {
		return nil, err
	}
	return nat, nil
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
	resp, err := self.post(SERVICE_NAT, "private-nat/gateways", params)
	if err != nil {
		return nil, errors.Wrap(err, "create nat")
	}
	nat := &SNatGateway{region: self}
	err = resp.Unmarshal(nat, "gateway")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return nat, nil
}

func (self *SRegion) DeleteNatGateway(id string) error {
	_, err := self.delete(SERVICE_NAT, "private-nat/gateways/"+id)
	return err
}
