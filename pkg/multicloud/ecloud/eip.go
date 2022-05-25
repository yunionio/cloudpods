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

package ecloud

import (
	"context"
	"fmt"
	"time"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SEip struct {
	region *SRegion
	multicloud.SEipBase
	multicloud.EcloudTags

	BandWidthMbSize int
	BandWidthType   string
	BindType        string
	// 使用，未使用
	Bound bool
	// bandwidthCharge, trafficCharge
	ChargeModeEnum string
	CreateTime     time.Time
	Frozen         bool
	// 备案状态
	IcpStatus     string
	Id            string
	IpType        string
	Ipv6          string
	Name          string //公网IPv4地址
	NicName       string
	PortNetworkId string
	Region        string
	ResourceId    string
	RouterId      string
	// BINDING, UNBOUND, FROZEN
	Status string
}

func (e *SEip) GetId() string {
	return e.Id
}

func (e *SEip) GetName() string {
	return e.Name
}

func (e *SEip) GetGlobalId() string {
	return e.Id
}

func (e *SEip) GetStatus() string {
	switch e.Status {
	case "BINDING", "UNBOUND":
		return api.EIP_STATUS_READY
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (e *SEip) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (e *SEip) IsEmulated() bool {
	return false
}

func (e *SEip) GetIpAddr() string {
	return e.Name
}

func (e *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (e *SEip) GetAssociationType() string {
	switch e.BindType {
	case "vm":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	case "snat", "dnat":
		return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
	case "elb":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	default:
		return e.BindType
	}
}

func (e *SEip) GetAssociationExternalId() string {
	return e.ResourceId
}

func (e *SEip) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (e *SEip) GetCreatedAt() time.Time {
	return e.CreateTime
}

func (e *SEip) GetExpiredAt() time.Time {
	return time.Time{}
}

func (e *SEip) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (e *SEip) GetBandwidth() int {
	return e.BandWidthMbSize
}

func (e *SEip) GetINetworkId() string {
	return e.PortNetworkId
}

func (e *SEip) GetInternetChargeType() string {
	switch e.ChargeModeEnum {
	// bandwidthCharge, trafficCharge
	case "trafficCharge":
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	case "bandwidthCharge":
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	default:
		return "unknown"
	}
}

func (e *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (e *SEip) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (e *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

func (e *SEip) GetProjectId() string {
	return ""
}

func (r *SRegion) GetEipById(id string) (*SEip, error) {
	var eip SEip
	request := NewConsoleRequest(r.ID, fmt.Sprintf("/api/v2/floatingIp/getRespWithBw/%s", id), nil, nil)
	err := r.client.doGet(context.Background(), request, &eip)
	if err != nil {
		return nil, err
	}
	return &eip, nil
}
