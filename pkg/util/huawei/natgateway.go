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
	"time"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	region *SRegion

	ID          string
	Name        string
	Description string
	Spec        string
	Status      string
	CreatedTime string `json:"created_at"`
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

func (gateway *SNatGateway) GetNatSpec() string {
	switch gateway.Spec {
	case "1":
		return "small"
	case "2":
		return "middle"
	case "3":
		return "large"
	case "4":
		return "xlarge"
	}
	// can't arrive
	return ""
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
	// Get all IEips of nat's region, which IpAddr is in the ExternalIPs of nat's Nat Rules.
	IEips, err := gateway.region.GetIEips()
	if err != nil {
		return nil, errors.Wrapf(err, `get all Eips of region %q error`, gateway.region.GetId())
	}
	dNatTables, err := gateway.GetIDNatEntries()
	if err != nil {
		return nil, errors.Wrapf(err, `get all DNatTable of gateway %q error`, gateway.GetId())
	}
	sNatTables, err := gateway.GetISNatEntries()
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

func (gateway *SNatGateway) GetIDNatEntries() ([]cloudprovider.ICloudDNatEntry, error) {
	dNatTable, err := gateway.getDNatEntries()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, gateway.GetId())
	}
	ret := make([]cloudprovider.ICloudDNatEntry, len(dNatTable))
	for i := range dNatTable {
		ret[i] = &dNatTable[i]
	}
	return ret, nil
}

func (gateway *SNatGateway) GetISNatEntries() ([]cloudprovider.ICloudSNatEntry, error) {
	sNatTable, err := gateway.getSNatEntries()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, gateway.GetId())
	}
	ret := make([]cloudprovider.ICloudSNatEntry, len(sNatTable))
	for i := range sNatTable {
		ret[i] = &sNatTable[i]
	}
	return ret, nil
}

func (gateway *SNatGateway) CreateIDNatEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudDNatEntry, error) {
	dnat, err := gateway.region.CreateNatDTable(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	dnat.gateway = gateway
	return &dnat, nil
}

func (gateway *SNatGateway) CreateISNatEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudSNatEntry, error) {
	snat, err := gateway.region.CreateNatSTable(rule, gateway.GetId())
	if err != nil {
		return nil, err
	}
	snat.gateway = gateway
	return &snat, nil
}

func (gateway *SNatGateway) GetIDNatEntryByID(id string) (cloudprovider.ICloudDNatEntry, error) {
	dnat, err := gateway.region.GetNatDTableByID(id)
	if err != nil {
		return nil, err
	}
	dnat.gateway = gateway
	return &dnat, nil
}

func (gateway *SNatGateway) GetISNatEntryByID(id string) (cloudprovider.ICloudSNatEntry, error) {
	snat, err := gateway.region.GetNatSTableByID(id)
	if err != nil {
		return nil, err
	}
	snat.gateway = gateway
	return &snat, nil
}

func (region *SRegion) GetNatGateway(natGatewayID string) ([]SNatGateway, error) {
	queues := make(map[string]string)
	if natGatewayID != "" {
		queues["id"] = natGatewayID
	}
	natGateways := make([]SNatGateway, 0, 2)
	err := doListAllWithMarker(region.ecsClient.NatGateways.List, queues, &natGateways)
	if err != nil {
		return nil, errors.Wrapf(err, "get nat gateways error by natgatewayid")
	}
	for i := range natGateways {
		natGateways[i].region = region
	}
	return natGateways, nil
}

func (region *SRegion) GetNatGateways(vpcID string) ([]SNatGateway, error) {
	queues := map[string]string{
		"router_id": vpcID,
	}
	natGateways := make([]SNatGateway, 0, 2)
	err := doListAllWithMarker(region.ecsClient.NatGateways.List, queues, &natGateways)
	if err != nil {
		return nil, errors.Wrapf(err, "get nat gateways error by vpcid")
	}
	for i := range natGateways {
		natGateways[i].region = region
	}
	return natGateways, nil
}

func (region *SRegion) CreateNatDTable(rule cloudprovider.SNatDRule, gatewayID string) (SNatDTableEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayID
	params["private_ip"] = rule.InternalIP
	params["internal_service_port"] = rule.InternalPort
	params["floating_ip_id"] = rule.ExternalIPID
	params["external_service_port"] = rule.ExternalPort
	params["protocol"] = rule.Protocol

	packParams := map[string]map[string]interface{}{
		"dnat_rule": params,
	}

	ret := SNatDTableEntry{}
	err := DoCreate(region.ecsClient.DNatRules.Create, jsonutils.Marshal(packParams), &ret)
	if err != nil {
		return SNatDTableEntry{}, errors.Wrapf(err, `create dnat rule of nat gateway %q failed`, gatewayID)
	}
	return ret, nil
}

func (region *SRegion) CreateNatSTable(rule cloudprovider.SNatSRule, gatewayID string) (SNatSTableEntry, error) {
	params := make(map[string]interface{})
	params["nat_gateway_id"] = gatewayID
	params["cidr"] = rule.SourceCIDR
	params["floating_ip_id"] = rule.ExternalIPID

	packParams := map[string]map[string]interface{}{
		"snat_rule": params,
	}

	ret := SNatSTableEntry{}
	err := DoCreate(region.ecsClient.SNatRules.Create, jsonutils.Marshal(packParams), &ret)
	if err != nil {
		return SNatSTableEntry{}, errors.Wrapf(err, `create snat rule of nat gateway %q failed`, gatewayID)
	}
	return ret, nil
}

func (region *SRegion) GetNatDTableByID(id string) (SNatDTableEntry, error) {
	dnat := SNatDTableEntry{}
	err := DoGet(region.ecsClient.DNatRules.Get, id, map[string]string{}, &dnat)

	if err != nil {
		return SNatDTableEntry{}, cloudprovider.ErrNotFound
	}
	return dnat, nil
}

func (region *SRegion) GetNatSTableByID(id string) (SNatSTableEntry, error) {
	snat := SNatSTableEntry{}
	err := DoGet(region.ecsClient.SNatRules.Get, id, map[string]string{}, &snat)
	if err != nil {
		return SNatSTableEntry{}, cloudprovider.ErrNotFound
	}
	return snat, nil
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
