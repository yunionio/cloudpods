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

package jdcloud

import (
	"fmt"
	"time"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/models"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SEip struct {
	region *SRegion
	multicloud.SEipBase
	multicloud.JdcloudTags

	models.ElasticIp
}

func (e *SEip) GetId() string {
	return e.ElasticIpId
}

func (e *SEip) GetName() string {
	return e.ElasticIpAddress
}

func (e *SEip) GetGlobalId() string {
	return e.GetId()
}

func (e *SEip) GetStatus() string {
	return api.EIP_STATUS_READY
}

func (e *SEip) Refresh() error {
	return nil
}

func (e *SEip) IsEmulated() bool {
	return false
}

func (e *SEip) GetIpAddr() string {
	return e.ElasticIpAddress
}

func (e *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (e *SEip) GetAssociationType() string {
	switch e.InstanceType {
	case "compute":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	case "lb":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	default:
		return e.InstanceType
	}
}

func (e *SEip) GetAssociationExternalId() string {
	return e.InstanceId
}

func (e *SEip) GetBillingType() string {
	return billingType(&e.Charge)
}

func (e *SEip) GetCreatedAt() time.Time {
	return parseTime(e.CreatedTime)
}

func (e *SEip) GetExpiredAt() time.Time {
	return expireAt(&e.Charge)
}

func (e *SEip) Delete() error {
	return nil
}

func (e *SEip) GetBandwidth() int {
	return e.BandwidthMbps
}

func (e *SEip) GetINetworkId() string {
	return ""
}

func (e *SEip) GetInternetChargeType() string {
	switch e.Charge.ChargeMode {
	case "postpaid_by_usage":
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	case "postpaid_by_duration":
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	default:
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
}

func (e *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	return nil
}

func (e *SEip) Dissociate() error {
	return nil
}

func (e *SEip) ChangeBandwidth(bw int) error {
	return nil
}

func (e *SEip) GetProjectId() string {
	return ""
}

func (r *SRegion) GetEIPById(id string) (*SEip, error) {
	req := apis.NewDescribeElasticIpRequest(r.ID, id)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeElasticIp(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		return nil, fmt.Errorf(resp.Error.Message)
	}
	return &SEip{
		region:    r,
		ElasticIp: resp.Result.ElasticIp,
	}, nil
}
