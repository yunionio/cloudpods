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

package apsara

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBandwidthPackageIds struct {
	BandwidthPackageId []string
}

type SForwardTableIds struct {
	ForwardTableId []string
}

type SSnatTableIds struct {
	SnatTableId []string
}

type SNatGetway struct {
	multicloud.SNatGatewayBase
	ApsaraTags

	vpc *SVpc

	BandwidthPackageIds SBandwidthPackageIds
	BusinessStatus      string
	CreationTime        time.Time
	ExpiredTime         time.Time
	NetworkType         string
	Description         string
	ForwardTableIds     SForwardTableIds
	SnatTableIds        SSnatTableIds
	InstanceChargeType  TChargeType
	Name                string
	NatGatewayId        string
	RegionId            string
	Spec                string
	Status              string
	VpcId               string
}

func (nat *SNatGetway) GetId() string {
	return nat.NatGatewayId
}

func (nat *SNatGetway) GetGlobalId() string {
	return nat.NatGatewayId
}

func (nat *SNatGetway) GetName() string {
	if len(nat.Name) > 0 {
		return nat.Name
	}
	return nat.NatGatewayId
}

func (self *SNatGetway) GetINetworkId() string {
	return ""
}

func (nat *SNatGetway) GetStatus() string {
	switch nat.Status {
	case "Initiating":
		return api.NAT_STATUS_ALLOCATE
	case "Available":
		return api.NAT_STAUTS_AVAILABLE
	case "Pending":
		return api.NAT_STATUS_DEPLOYING
	default:
		return api.NAT_STATUS_UNKNOWN
	}

}

func (nat *SNatGetway) GetBillingType() string {
	return convertChargeType(nat.InstanceChargeType)
}

func (nat *SNatGetway) GetNatSpec() string {
	return nat.Spec
}

func (nat *SNatGetway) GetNetworkType() string {
	return nat.NetworkType
}

func (nat *SNatGetway) GetCreatedAt() time.Time {
	return nat.CreationTime
}

func (nat *SNatGetway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips := []SEipAddress{}
	for {
		parts, total, err := nat.vpc.region.GetEips("", nat.NatGatewayId, len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, parts...)
		if len(eips) >= total {
			break
		}
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		eips[i].region = nat.vpc.region
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (nat *SNatGetway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	itables := []cloudprovider.ICloudNatDEntry{}
	for _, dtableId := range nat.ForwardTableIds.ForwardTableId {
		dtables, err := nat.vpc.region.GetAllDTables(dtableId)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(dtables); i++ {
			dtables[i].nat = nat
			itables = append(itables, &dtables[i])
		}
	}
	return itables, nil
}

func (nat *SNatGetway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
	stables, err := nat.getSnatEntries()
	if err != nil {
		return nil, err
	}
	itables := []cloudprovider.ICloudNatSEntry{}
	for i := 0; i < len(stables); i++ {
		stables[i].nat = nat
		itables = append(itables, &stables[i])
	}
	return itables, nil
}

func (nat *SNatGetway) GetINatDEntryById(id string) (cloudprovider.ICloudNatDEntry, error) {
	dNATEntry, err := nat.vpc.region.GetForwardTableEntry(nat.ForwardTableIds.ForwardTableId[0], id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	dNATEntry.nat = nat
	return &dNATEntry, nil
}

func (nat *SNatGetway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	sNATEntry, err := nat.vpc.region.GetSNATEntry(nat.SnatTableIds.SnatTableId[0], id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	sNATEntry.nat = nat
	return &sNATEntry, nil
}

func (nat *SNatGetway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	entryID, err := nat.vpc.region.CreateForwardTableEntry(rule, nat.ForwardTableIds.ForwardTableId[0])
	if err != nil {
		return nil, errors.Wrapf(err, `create dnat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatDEntryById(entryID)
}

func (nat *SNatGetway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	entryID, err := nat.vpc.region.CreateSNATTableEntry(rule, nat.SnatTableIds.SnatTableId[0])
	if err != nil {
		return nil, errors.Wrapf(err, `create snat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatSEntryById(entryID)
}

func (self *SRegion) GetNatGateways(vpcId string, natGwId string, offset, limit int) ([]SNatGetway, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(natGwId) > 0 {
		params["NatGatewayId"] = natGwId
	}

	body, err := self.vpcRequest("DescribeNatGateways", params)
	if err != nil {
		log.Errorf("GetVSwitches fail %s", err)
		return nil, 0, err
	}

	if self.client.debug {
		log.Debugf("%s", body.PrettyString())
	}

	gateways := make([]SNatGetway, 0)
	err = body.Unmarshal(&gateways, "NatGateways", "NatGateway")
	if err != nil {
		log.Errorf("Unmarshal gateways fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return gateways, int(total), nil
}
