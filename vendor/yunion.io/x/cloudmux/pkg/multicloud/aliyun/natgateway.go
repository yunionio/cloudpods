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

package aliyun

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

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

type NatGatewayPrivateInfo struct {
	EniInstanceId    string
	IzNo             string
	MaxBandwidth     int
	PrivateIpAddress string
	VswitchId        string
}

type SNatGateway struct {
	multicloud.SNatGatewayBase
	AliyunTags

	vpc *SVpc

	BandwidthPackageIds   SBandwidthPackageIds
	BusinessStatus        string
	CreationTime          time.Time
	ExpiredTime           time.Time
	Description           string
	ForwardTableIds       SForwardTableIds
	NetworkType           string
	SnatTableIds          SSnatTableIds
	InstanceChargeType    TChargeType
	Name                  string
	NatGatewayId          string
	RegionId              string
	Spec                  string
	Status                string
	VpcId                 string
	NatGatewayPrivateInfo NatGatewayPrivateInfo
}

func (nat *SNatGateway) GetId() string {
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetGlobalId() string {
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetName() string {
	if len(nat.Name) > 0 {
		return nat.Name
	}
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetStatus() string {
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

func (self *SNatGateway) GetINetworkId() string {
	return self.NatGatewayPrivateInfo.VswitchId
}

func (self *SNatGateway) GetNetworkType() string {
	return self.NetworkType
}

func (self *SNatGateway) GetIpAddr() string {
	return self.NatGatewayPrivateInfo.PrivateIpAddress
}

func (self *SNatGateway) GetBandwidthMb() int {
	return self.NatGatewayPrivateInfo.MaxBandwidth
}

func (self *SNatGateway) Delete() error {
	return self.vpc.region.DeleteNatGateway(self.NatGatewayId, false)
}

func (nat *SNatGateway) GetBillingType() string {
	return convertChargeType(nat.InstanceChargeType)
}

func (nat *SNatGateway) GetNatSpec() string {
	if len(nat.Spec) == 0 {
		return api.ALIYUN_NAT_SKU_DEFAULT
	}
	return nat.Spec
}

func (self *SNatGateway) Refresh() error {
	nat, total, err := self.vpc.region.GetNatGateways("", self.NatGatewayId, 0, 1)
	if err != nil {
		return errors.Wrapf(err, "GetNatGateways")
	}
	if total > 1 {
		return errors.Wrapf(cloudprovider.ErrDuplicateId, "get %d natgateways by id %s", total, self.NatGatewayId)
	}
	if total == 0 {
		return errors.Wrapf(cloudprovider.ErrNotFound, self.NatGatewayId)
	}
	return jsonutils.Update(self, nat[0])
}

func (nat *SNatGateway) GetCreatedAt() time.Time {
	return nat.CreationTime
}

func (nat *SNatGateway) GetExpiredAt() time.Time {
	return nat.ExpiredTime
}

func (nat *SNatGateway) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips := []SEipAddress{}
	for {
		parts, total, err := nat.vpc.region.GetEips("", nat.NatGatewayId, "", len(eips), 50)
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

func (nat *SNatGateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
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

func (nat *SNatGateway) GetINatSTable() ([]cloudprovider.ICloudNatSEntry, error) {
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

func (nat *SNatGateway) GetINatDEntryById(id string) (cloudprovider.ICloudNatDEntry, error) {
	dNATEntry, err := nat.vpc.region.GetForwardTableEntry(nat.ForwardTableIds.ForwardTableId[0], id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	dNATEntry.nat = nat
	return &dNATEntry, nil
}

func (nat *SNatGateway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	sNATEntry, err := nat.vpc.region.GetSNATEntry(nat.SnatTableIds.SnatTableId[0], id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	sNATEntry.nat = nat
	return &sNATEntry, nil
}

func (nat *SNatGateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	entryID, err := nat.vpc.region.CreateForwardTableEntry(rule, nat.ForwardTableIds.ForwardTableId[0])
	if err != nil {
		return nil, errors.Wrapf(err, `create dnat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatDEntryById(entryID)
}

func (nat *SNatGateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	entryID, err := nat.vpc.region.CreateSNATTableEntry(rule, nat.SnatTableIds.SnatTableId[0])
	if err != nil {
		return nil, errors.Wrapf(err, `create snat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatSEntryById(entryID)
}

func (self *SRegion) GetNatGateways(vpcId string, natGwId string, offset, limit int) ([]SNatGateway, int, error) {
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
		return nil, 0, errors.Wrapf(err, "DescribeNatGateways")
	}

	if self.client.debug {
		log.Debugf("%s", body.PrettyString())
	}

	gateways := make([]SNatGateway, 0)
	err = body.Unmarshal(&gateways, "NatGateways", "NatGateway")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "body.Unmarshal")
	}
	total, _ := body.Int("TotalCount")
	return gateways, int(total), nil
}

func (self *SVpc) CreateINatGateway(opts *cloudprovider.NatGatewayCreateOptions) (cloudprovider.ICloudNatGateway, error) {
	nat, err := self.region.CreateNatGateway(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNatGateway")
	}
	nat.vpc = self
	return nat, nil
}

func (self *SRegion) CreateNatGateway(opts *cloudprovider.NatGatewayCreateOptions) (*SNatGateway, error) {
	params := map[string]string{
		"RegionId":           self.RegionId,
		"VpcId":              opts.VpcId,
		"VSwitchId":          opts.NetworkId,
		"NatType":            "Enhanced",
		"Name":               opts.Name,
		"Description":        opts.Desc,
		"ClientToken":        utils.GenRequestId(20),
		"InstanceChargeType": "PostPaid",
		"InternetChargeType": "PayBySpec",
	}
	if len(opts.NatSpec) == 0 || opts.NatSpec == api.ALIYUN_NAT_SKU_DEFAULT {
		params["InternetChargeType"] = "PayByLcu"
	} else {
		params["Spec"] = opts.NatSpec
	}

	if opts.BillingCycle != nil {
		params["InstanceChargeType"] = "PrePaid"
		params["PricingCycle"] = "Month"
		params["AutoPay"] = "false"
		if opts.BillingCycle.GetYears() > 0 {
			params["PricingCycle"] = "Year"
			params["Duration"] = fmt.Sprintf("%d", opts.BillingCycle.GetYears())
		} else if opts.BillingCycle.GetMonths() > 0 {
			params["PricingCycle"] = "Year"
			params["Duration"] = fmt.Sprintf("%d", opts.BillingCycle.GetMonths())
		}
		if opts.BillingCycle.AutoRenew {
			params["AutoPay"] = "true"
		}
	}
	resp, err := self.vpcRequest("CreateNatGateway", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNatGateway")
	}
	natId, err := resp.GetString("NatGatewayId")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Get(NatGatewayId)")
	}
	if len(natId) == 0 {
		return nil, errors.Errorf("empty NatGatewayId after created")
	}

	var nat *SNatGateway = nil
	err = cloudprovider.Wait(time.Second*5, time.Minute*15, func() (bool, error) {
		nats, total, err := self.GetNatGateways("", natId, 0, 1)
		if err != nil {
			return false, errors.Wrapf(err, "GetNatGateways(%s)", natId)
		}
		if total > 1 {
			return false, errors.Wrapf(cloudprovider.ErrDuplicateId, "get %d nats", total)
		}
		if total == 0 {
			return false, errors.Wrapf(cloudprovider.ErrNotFound, "search %s after %s created", opts.Name, natId)
		}
		nat = &nats[0]
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cloudprovider.Wait")
	}

	return nat, nil
}

func (self *SRegion) DeleteNatGateway(natId string, isForce bool) error {
	params := map[string]string{
		"RegionId":     self.RegionId,
		"NatGatewayId": natId,
	}
	if isForce {
		params["Force"] = "true"
	}
	_, err := self.vpcRequest("DeleteNatGateway", params)
	return errors.Wrapf(err, "DeleteNatGateway")
}

func (self *SNatGateway) GetTags() (map[string]string, error) {
	_, tags, err := self.vpc.region.ListSysAndUserTags(ALIYUN_SERVICE_VPC, "NATGATEWAY", self.NatGatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, "ListTags")
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps, nil
}

func (self *SNatGateway) GetSysTags() map[string]string {
	tags, _, err := self.vpc.region.ListSysAndUserTags(ALIYUN_SERVICE_VPC, "NATGATEWAY", self.NatGatewayId)
	if err != nil {
		return nil
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps
}

func (self *SNatGateway) SetTags(tags map[string]string, replace bool) error {
	return self.vpc.region.SetResourceTags(ALIYUN_SERVICE_VPC, "NATGATEWAY", self.GetId(), tags, replace)
}
