// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"time"

	billing "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SNatGateway struct {
	multicloud.SNatGatewayBase
	VolcEngineTags

	vpc *SVpc

	NatGatewayId       string
	NatGatewayName     string
	Description        string
	Spec               string
	BillingType        int
	VpcId              string
	SubnetId           string
	ZoneId             string
	NetworkInterfaceId string
	ProjectName        string
	Status             string
	BusinessStatus     string
	LockReason         string
	CreationTime       time.Time
	UpdatedAt          time.Time
	ExpiredTime        time.Time
	OverdueTime        time.Time
	DeletedTime        time.Time
	PrivateIP          string
}

func (nat *SNatGateway) GetId() string {
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetGlobalId() string {
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetName() string {
	if len(nat.NatGatewayName) > 0 {
		return nat.NatGatewayName
	}
	return nat.NatGatewayId
}

func (nat *SNatGateway) GetStatus() string {
	switch nat.Status {
	case "Creating":
		return api.NAT_STATUS_ALLOCATE
	case "Available":
		return api.NAT_STAUTS_AVAILABLE
	case "Pending":
		return api.NAT_STATUS_DEPLOYING
	case "Deleteting":
		return api.NAT_STATUS_DELETING
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (nat *SNatGateway) GetINetworkId() string {
	return nat.SubnetId
}

func (nat *SNatGateway) GetNetworkType() string {
	return api.NAT_NETWORK_TYPE_INTERNET
}

func (nat *SNatGateway) GetIpAddr() string {
	if len(nat.PrivateIP) > 0 {
		return nat.PrivateIP
	}
	return ""
}

func (nat *SNatGateway) GetBandwidthMb() int {
	return 0
}

func (nat *SNatGateway) Delete() error {
	return nat.vpc.region.DeleteNatGateway(nat.NatGatewayId, false)
}

func (nat *SNatGateway) GetBillingType() string {
	switch nat.BillingType {
	case 1:
		return billing.BILLING_TYPE_PREPAID
	case 2:
		return billing.BILLING_TYPE_POSTPAID
	default:
		return ""
	}
}

func (nat *SNatGateway) GetNatSpec() string {
	if len(nat.Spec) == 0 {
		return "Small"
	}
	return nat.Spec
}

func (nat *SNatGateway) Refresh() error {
	newNat, _, err := nat.vpc.region.GetNatGateways("", nat.NatGatewayId, 1, 1)
	if err != nil {
		return errors.Wrapf(err, "GetNatGateways")
	}
	for _, nt := range newNat {
		if nt.NatGatewayId == nat.NatGatewayId {
			return jsonutils.Update(nat, nt)
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, "%s not found", nat.NatGatewayId)
}

func (nat *SNatGateway) GetCreatedAt() time.Time {
	return nat.CreationTime
}

func (nat *SNatGateway) GetExpiredAt() time.Time {
	return nat.ExpiredTime
}

func (nat *SNatGateway) GetINatDTable() ([]cloudprovider.ICloudNatDEntry, error) {
	stables, err := nat.getDnatEntries()
	if err != nil {
		return nil, err
	}
	itables := []cloudprovider.ICloudNatDEntry{}
	for i := 0; i < len(stables); i++ {
		stables[i].nat = nat
		itables = append(itables, &stables[i])
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
	dNATEntry, err := nat.vpc.region.GetDnatEntry(nat.NatGatewayId, id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	dNATEntry.nat = nat
	return &dNATEntry, nil
}

func (nat *SNatGateway) GetINatSEntryById(id string) (cloudprovider.ICloudNatSEntry, error) {
	sNATEntry, err := nat.vpc.region.GetSnatEntry(nat.NatGatewayId, id)
	if err != nil {
		return nil, cloudprovider.ErrNotFound
	}
	sNATEntry.nat = nat
	return &sNATEntry, nil
}

func (nat *SNatGateway) CreateINatDEntry(rule cloudprovider.SNatDRule) (cloudprovider.ICloudNatDEntry, error) {
	entryID, err := nat.vpc.region.CreateDnatEntry(rule, nat.NatGatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, `create dnat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatDEntryById(entryID)
}

func (nat *SNatGateway) CreateINatSEntry(rule cloudprovider.SNatSRule) (cloudprovider.ICloudNatSEntry, error) {
	entryID, err := nat.vpc.region.CreateSnatEntry(rule, nat.NatGatewayId)
	if err != nil {
		return nil, errors.Wrapf(err, `create snat rule for nat gateway %q`, nat.GetId())
	}
	return nat.GetINatSEntryById(entryID)
}

func (region *SRegion) GetNatGateways(vpcId string, natGatewayId string, pageNumber int, pageSize int) ([]SNatGateway, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(natGatewayId) > 0 {
		params["NatGatewayId.1"] = natGatewayId
	}
	body, err := region.natRequest("DescribeNatGateways", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeNatGateways")
	}
	gateways := make([]SNatGateway, 0)
	err = body.Unmarshal(&gateways, "NatGateways")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "body.Unmarshal")
	}
	total, _ := body.Int("TotalCount")
	return gateways, int(total), nil
}

func (region *SRegion) CreateNatGateway(opts *cloudprovider.NatGatewayCreateOptions) (*SNatGateway, error) {
	params := map[string]string{
		"VpcId":          opts.VpcId,
		"SubnetId":       opts.NetworkId,
		"NatGatewayName": opts.Name,
		"Description":    opts.Desc,
		"ClientToken":    utils.GenRequestId(20),
		"BillingType":    fmt.Sprintf("%d", 2),
	}
	if len(opts.NatSpec) != 0 {
		params["Spec"] = opts.NatSpec
	}

	if opts.BillingCycle != nil {
		params["BillingType"] = fmt.Sprintf("%d", 1)
		params["Period"] = fmt.Sprintf("%d", 1)
		params["PeriodUnit"] = "Month"
		if opts.BillingCycle.GetYears() > 0 {
			params["PeriodUnit"] = "Year"
			params["Period"] = fmt.Sprintf("%d", opts.BillingCycle.GetYears())
		} else if opts.BillingCycle.GetMonths() > 0 {
			params["PeriodUnit"] = "Month"
			params["Period"] = fmt.Sprintf("%d", opts.BillingCycle.GetMonths())
		}
	}
	resp, err := region.natRequest("CreateNatGateway", params)
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

	err = cloudprovider.Wait(time.Second*5, time.Minute*15, func() (bool, error) {
		_, _, err := region.GetNatGateways("", natId, 1, 1)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return true, err
		}
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find nat gateway after create")
	}
	nats, _, err := region.GetNatGateways("", natId, 1, 1)
	for _, nat := range nats {
		if nat.NatGatewayId == natId {
			return &nat, nil
		}
	}
	return nil, errors.Wrapf(err, "%s not found", natId)
}

func (region *SRegion) DeleteNatGateway(natId string, isForce bool) error {
	params := make(map[string]string)
	params["NatGatewayId"] = natId
	_, err := region.natRequest("DeleteNatGateway", params)
	return errors.Wrapf(err, "DeleteNatGateway")
}
