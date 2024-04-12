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

package ksyun

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEipResp struct {
	RequestID    string `json:"RequestId"`
	NextToken    string `json:"NextToken"`
	AddressesSet []SEip `json:"AddressesSet"`
	TotalCount   int    `json:"TotalCount"`
}

type SEip struct {
	multicloud.SEipBase
	region *SRegion
	SKsTag

	PublicIP             string `json:"PublicIp"`
	AllocationID         string `json:"AllocationId"`
	State                string `json:"State"`
	IPState              string `json:"IpState"`
	LineID               string `json:"LineId"`
	BandWidth            int    `json:"BandWidth"`
	InstanceType         string `json:"InstanceType"`
	InstanceID           string `json:"InstanceId"`
	ChargeType           string `json:"ChargeType"`
	IPVersion            string `json:"IpVersion"`
	ProjectID            string `json:"ProjectId"`
	CreateTime           string `json:"CreateTime"`
	Mode                 string `json:"Mode"`
	NetworkInterfaceID   string `json:"NetworkInterfaceId,omitempty"`
	NetworkInterfaceType string `json:"NetworkInterfaceType,omitempty"`
	PrivateIPAddress     string `json:"PrivateIpAddress,omitempty"`
	InternetGatewayID    string `json:"InternetGatewayId,omitempty"`
	HostType             string `json:"HostType,omitempty"`
}

func (region *SRegion) GetEips(eipIds []string) ([]SEip, error) {
	params := map[string]string{
		"MaxResults": "1000",
	}
	for i, eipId := range eipIds {
		params[fmt.Sprintf("AllocationId.%d", i+1)] = eipId
	}
	res := []SEip{}
	for {
		resp, err := region.eipRequest("DescribeAddresses", params)
		if err != nil {
			return nil, errors.Wrap(err, "get eips")
		}
		part := SEipResp{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal eip")
		}
		res = append(res, part.AddressesSet...)
		if len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return res, nil
}

func (region *SRegion) GetEip(eipId string) (*SEip, error) {
	eips, err := region.GetEips([]string{eipId})
	if err != nil {
		return nil, errors.Wrap(err, "GetEips")
	}
	for _, eip := range eips {
		if eip.GetId() == eipId {
			return &eip, nil
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "eip id:%s", eipId)
}

func (eip *SEip) GetId() string {
	return eip.AllocationID
}

func (eip *SEip) GetName() string {
	return eip.AllocationID
}

func (eip *SEip) GetGlobalId() string {
	return eip.AllocationID
}

func (eip *SEip) GetTags() (map[string]string, error) {
	tags, err := eip.region.ListTags("eip", eip.AllocationID)
	if err != nil {
		return nil, err
	}
	return tags.GetTags(), nil
}

func (eip *SEip) GetStatus() string {
	switch eip.State {
	case "associate":
		return api.EIP_STATUS_READY
	case "disassociate":
		return api.EIP_STATUS_READY
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (eip *SEip) Refresh() error {
	extEip, err := eip.region.GetEip(eip.AllocationID)
	if err != nil {
		return errors.Wrap(err, "region.GetEip")
	}
	return jsonutils.Update(eip, &extEip)
}

func (eip *SEip) GetIpAddr() string {
	return eip.PublicIP
}

func (eip *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (eip *SEip) GetAssociationType() string {
	switch eip.InstanceType {
	case "Ipfwd":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	case "Slb":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	default:
		return eip.InstanceType
	}
}

func (eip *SEip) GetAssociationExternalId() string {
	return eip.InstanceID
}

func (eip *SEip) GetBandwidth() int {
	return int(eip.BandWidth) // Mb
}

func (eip *SEip) GetINetworkId() string {
	return ""
}

func (eip *SEip) GetInternetChargeType() string {
	return ""
}

func (eip *SEip) GetBillingType() string {
	if eip.ChargeType == "Monthly" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (eip *SEip) GetCreatedAt() time.Time {
	createdAt, _ := time.Parse("2006-01-02 15:04:05", eip.CreateTime)
	return createdAt
}

func (eip *SEip) GetExpiredAt() time.Time {
	return time.Time{}
}

func (eip *SEip) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEip) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetInstancePortId(instanceId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
func (region *SRegion) AllocateEIP(opts *cloudprovider.SEip) (*SEip, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) AssociateEipWithPortId(eipId string, portId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DissociateEip(eipId string) error {
	return region.AssociateEipWithPortId(eipId, "")
}

func (region *SRegion) UpdateEipBandwidth(bandwidthId string, bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (eip *SEip) GetProjectId() string {
	return ""
}
