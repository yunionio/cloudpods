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
	"yunion.io/x/pkg/errors"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	region *SRegion

	natDTable []cloudprovider.ICloudNatDTable
	natSTable []cloudprovider.ICloudNatSTable

	ID          string
	Name        string
	Description string
	Spec        string
	Status      string
	CreatedTime string `json:"created_at"`
}

func (nat *SNatGateway) GetId() string {
	return nat.ID
}

func (nat *SNatGateway) GetName() string {
	return nat.Name
}

func (nat *SNatGateway) GetGlobalId() string {
	return nat.GetId()
}

func (nat *SNatGateway) GetStatus() string {
	return NatResouceStatusTransfer(nat.Status)
}

func (nat *SNatGateway) GetNatSpec() string {
	switch nat.Spec {
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

func (nat *SNatGateway) GetDescription() string {
	return nat.Description
}

func (nat *SNatGateway) GetBillingType() string {
	// Up to 2019.07.17, only support post pay
	return billing_api.BILLING_TYPE_POSTPAID
}

func (nat *SNatGateway) GetCreatedAt() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05.000000", nat.CreatedTime)
	return t
}

func (nat *SNatGateway) GetExpiredAt() time.Time {
	// no support for expired time
	return time.Time{}
}

func (nat *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	// Get all IEips of nat's region, which IpAddr is in the ExternalIPs of nat's Nat Rules.
	IEips, err := nat.region.GetIEips()
	if err != nil {
		return nil, errors.Wrapf(err, `get all Eips of region %q error`, nat.region.GetId())
	}
	dNatTables, err := nat.GetINatDTables()
	if err != nil {
		return nil, errors.Wrapf(err, `get all DNatTable of gateway %q error`, nat.GetId())
	}
	sNatTables, err := nat.GetINatSTables()
	if err != nil {
		return nil, errors.Wrapf(err, `get all SNatTable of gateway %q error`, nat.GetId())
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

func (nat *SNatGateway) GetINatDTables() ([]cloudprovider.ICloudNatDTable, error) {
	if nat.natDTable != nil {
		return nat.natDTable, nil
	}
	dNatTable, err := nat.getDNatEntries()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, nat.GetId())
	}
	ret := make([]cloudprovider.ICloudNatDTable, len(dNatTable))
	for i := range dNatTable {
		ret[i] = &dNatTable[i]
	}
	return ret, nil
}

func (nat *SNatGateway) GetINatSTables() ([]cloudprovider.ICloudNatSTable, error) {
	if nat.natSTable != nil {
		return nat.natSTable, nil
	}
	sNatTable, err := nat.getSNatEntries()
	if err != nil {
		return nil, errors.Wrapf(err, `get dnat table of nat gateway %q`, nat.GetId())
	}
	ret := make([]cloudprovider.ICloudNatSTable, len(sNatTable))
	for i := range sNatTable {
		ret[i] = &sNatTable[i]
	}
	return ret, nil
}

func (region *SRegion) GetNatGateway(natGatewayID string) ([]SNatGateway, error) {
	queues := make(map[string]string)
	if natGatewayID != "" {
		queues["id"] = natGatewayID
	}
	natGateways := make([]SNatGateway, 0, 2)
	err := doListAllWithMarker(region.ecsClient.NatGateways.List, queues, &natGateways)
	if err != nil {
		return nil, errors.Wrapf(err, "get nat gateways error")
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
