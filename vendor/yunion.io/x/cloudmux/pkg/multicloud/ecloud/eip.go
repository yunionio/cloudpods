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

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEip struct {
	region *SRegion
	multicloud.SEipBase
	EcloudTags

	BandWidthMbSize int    `json:"bandwidthMbSize,omitempty"`
	BandWidthType   string `json:"bandwidthType,omitempty"`
	BindType        string `json:"bindType,omitempty"`
	// 使用，未使用
	Bound bool `json:"bound,omitempty"`
	// bandwidthCharge, trafficCharge
	ChargeModeEnum string    `json:"chargeModeEnum,omitempty"`
	CreateTime     time.Time `json:"-"`
	// OpenAPI 返回的创建时间字符串
	CreatedTimeStr string `json:"createdTime,omitempty"`
	Frozen         bool   `json:"frozen,omitempty"`
	// 备案状态
	IcpStatus     string `json:"icpStatus,omitempty"`
	Id            string `json:"id,omitempty"`
	IpType        string `json:"ipType,omitempty"`
	Ipv6          string `json:"ipv6,omitempty"`
	Name          string `json:"name,omitempty"` // 公网IPv4地址
	NicName       string `json:"nicName,omitempty"`
	PortNetworkId string `json:"portNetworkId,omitempty"`
	Region        string `json:"region,omitempty"`
	ResourceId    string `json:"resourceId,omitempty"`
	RouterId      string `json:"routerId,omitempty"`
	// BINDING, UNBOUND, FROZEN
	Status string `json:"status,omitempty"`
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
	if len(e.CreatedTimeStr) > 0 {
		if t, err := time.Parse("2006-01-02 15:04:05", e.CreatedTimeStr); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, e.CreatedTimeStr); err == nil {
			return t
		}
	}
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
	return ""
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
	// 使用 OpenAPI EIP 详情：GET /api/openapi-eip/acl/v3/floatingip/getRespWithBw/{ipId}
	req := NewOpenApiEbsRequest(r.RegionId, fmt.Sprintf("/api/openapi-eip/acl/v3/floatingip/getRespWithBw/%s", id), nil, nil)
	err := r.client.doGet(context.Background(), req.Base(), &eip)
	if err != nil {
		return nil, err
	}
	eip.region = r
	return &eip, nil
}

// GetEipByAddr 使用 OpenAPI EIP 详情查询（按地址）：
// GET /api/openapi-eip/acl/v3/floatingip/apiDetail?ipAddr={addr}
func (r *SRegion) GetEipByAddr(addr string) (*SEip, error) {
	var eip SEip
	params := map[string]string{
		"ipAddress": addr,
	}
	req := NewOpenApiEbsRequest(r.RegionId, "/api/openapi-eip/acl/v3/floatingip/apiDetail", params, nil)
	if err := r.client.doGet(context.Background(), req.Base(), &eip); err != nil {
		return nil, err
	}
	eip.region = r
	return &eip, nil
}
