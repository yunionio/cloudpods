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

package baidu

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

type SEip struct {
	multicloud.SEipBase
	SBaiduTag

	region *SRegion

	Name            string
	Eip             string
	EipId           string
	Status          string
	InstanceType    string
	InstanceId      string
	ShareGroupId    string
	EipInstanceType string
	BandwidthInMbps int
	PaymentTiming   string
	BilingMethod    string
	CreateTime      time.Time
	ExpireTime      time.Time
	Region          string
	RouteTypt       string
}

func (self *SEip) GetId() string {
	return self.Eip
}

func (self *SEip) GetName() string {
	return self.Name
}

func (self *SEip) GetGlobalId() string {
	return self.Eip
}

func (self *SEip) GetStatus() string {
	switch self.Status {
	case "creating":
		return api.EIP_STATUS_ALLOCATE
	case "available", "binded", "updating", "paused":
		return api.EIP_STATUS_READY
	case "binding":
		return api.EIP_STATUS_ASSOCIATE
	case "unbinding":
		return api.EIP_STATUS_DISSOCIATE
	case "unavailable":
		return api.EIP_STATUS_UNKNOWN
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEip) Refresh() error {
	eip, err := self.region.GetEip(self.Eip)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEip) GetIpAddr() string {
	return self.Eip
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetAssociationType() string {
	switch self.InstanceType {
	case "BCC", "BBC", "DCC", "ENI":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	case "NAT":
		return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
	case "BLB":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	default:
		return strings.ToLower(self.InstanceType)
	}
}

func (self *SEip) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEip) GetBillingType() string {
	if strings.EqualFold(self.PaymentTiming, "prepaid") {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEip) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SEip) GetExpiredAt() time.Time {
	return self.ExpireTime
}

func (self *SEip) Delete() error {
	return self.region.DeleteEip(self.Eip)
}

func (self *SEip) GetBandwidth() int {
	return self.BandwidthInMbps
}

func (self *SEip) GetInternetChargeType() string {
	if strings.EqualFold(self.BilingMethod, "ByTraffic") {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	return self.region.AssociateEip(self.Eip, conf.InstanceId)
}

func (region *SRegion) AssociateEip(id string, instanceId string) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("bind", "")
	body := map[string]interface{}{
		"instanceType": "BCC",
		"instanceId":   instanceId,
	}
	_, err := region.eipUpdate(fmt.Sprintf("v1/eip/%s", id), params, body)
	return err
}

func (self *SEip) Dissociate() error {
	return self.region.DissociateEip(self.Eip)
}

func (region *SRegion) DissociateEip(id string) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("unbind", "")
	_, err := region.eipUpdate(fmt.Sprintf("v1/eip/%s", id), params, nil)
	return err
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.Eip, bw)
}

func (self *SEip) GetProjectId() string {
	return ""
}

func (region *SRegion) GetEips(instanceId string) ([]SEip, error) {
	params := url.Values{}
	if len(instanceId) > 0 {
		params.Set("instanceId", instanceId)
		params.Set("instanceType", "BCC")
	}
	ret := []SEip{}
	for {
		resp, err := region.eipList("v1/eip", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker string
			EipList    []SEip
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.EipList...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) GetEip(id string) (*SEip, error) {
	resp, err := region.eipList(fmt.Sprintf("v1/eip/%s", id), nil)
	if err != nil {
		return nil, err
	}
	ret := &SEip{region: region}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) DeleteEip(id string) error {
	_, err := region.eipDelete(fmt.Sprintf("v1/eip/%s", id), nil)
	return err
}

func (region *SRegion) UpdateEipBandwidth(id string, bw int) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	params.Set("resize", "")
	body := map[string]interface{}{
		"newBandwidthInMbps": bw,
	}
	_, err := region.eipUpdate(fmt.Sprintf("v1/eip/%s", id), params, body)
	return err
}

func (region *SRegion) CreateEip(opts *cloudprovider.SEip) (*SEip, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	tags := []BaiduTag{}
	for k, v := range opts.Tags {
		tags = append(tags, BaiduTag{
			TagKey:   k,
			TagValue: v,
		})
	}
	billing := map[string]interface{}{
		"paymentTiming": "Postpaid",
		"billingMethod": "ByTraffic",
	}
	if opts.ChargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		billing["billingMethod"] = "ByBandwidth"
	}
	body := map[string]interface{}{
		"name":            opts.Name,
		"bandwidthInMbps": opts.BandwidthMbps,
		"tags":            tags,
		"billing":         billing,
	}
	if len(opts.BGPType) > 0 {
		body["routeType"] = opts.BGPType
	}
	resp, err := region.eipPost("v1/eip", params, body)
	if err != nil {
		return nil, err
	}
	eipId, err := resp.GetString("eip")
	if err != nil {
		return nil, err
	}
	return region.GetEip(eipId)
}
