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

package ucloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	EIP_CHARGE_TYPE_BY_TRAFFIC   = "traffic"
	EIP_CHARGE_TYPE_BY_BANDWIDTH = "bandwidth"
)

// https://docs.ucloud.cn/api/unet-api/describe_eip
type SEip struct {
	region *SRegion

	BandwidthMb       int               `json:"Bandwidth"`
	BandwidthType     int               `json:"BandwidthType"`
	ChargeType        string            `json:"ChargeType"`
	CreateTime        int64             `json:"CreateTime"`
	EIPAddr           []EIPAddr         `json:"EIPAddr"`
	EIPID             string            `json:"EIPId"`
	Expire            bool              `json:"Expire"`
	ExpireTime        int64             `json:"ExpireTime"`
	Name              string            `json:"Name"`
	PayMode           string            `json:"PayMode"`
	Remark            string            `json:"Remark"`
	Resource          Resource          `json:"Resource"`
	ShareBandwidthSet ShareBandwidthSet `json:"ShareBandwidthSet"`
	Status            string            `json:"Status"`
	Tag               string            `json:"Tag"`
	Weight            int               `json:"Weight"`
}

func (self *SEip) GetProjectId() string {
	return self.region.client.projectId
}

type EIPAddr struct {
	IP           string `json:"IP"`
	OperatorName string `json:"OperatorName"`
}

type Resource struct {
	ResourceID   string `json:"ResourceID"`
	ResourceName string `json:"ResourceName"`
	ResourceType string `json:"ResourceType"`
	Zone         string `json:"Zone"`
}

type ShareBandwidthSet struct {
	ShareBandwidth     int    `json:"ShareBandwidth"`
	ShareBandwidthID   string `json:"ShareBandwidthId"`
	ShareBandwidthName string `json:"ShareBandwidthName"`
}

func (self *SEip) GetId() string {
	return self.EIPID
}

func (self *SEip) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SEip) GetGlobalId() string {
	return self.GetId()
}

// 弹性IP的资源绑定状态, 枚举值为: used: 已绑定, free: 未绑定, freeze: 已冻结
func (self *SEip) GetStatus() string {
	switch self.Status {
	case "used":
		return api.EIP_STATUS_ASSOCIATE // ?
	case "free":
		return api.EIP_STATUS_READY
	case "freeze":
		return api.EIP_STATUS_UNKNOWN
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEip) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEipById(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEip) IsEmulated() bool {
	return false
}

func (self *SEip) GetMetadata() *jsonutils.JSONDict {
	return nil
}

// 付费方式, 枚举值为: Year, 按年付费; Month, 按月付费; Dynamic, 按小时付费; Trial, 试用. 按小时付费和试用这两种付费模式需要开通权限.
func (self *SEip) GetBillingType() string {
	switch self.ChargeType {
	case "Year", "Month":
		return billing_api.BILLING_TYPE_PREPAID
	default:
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SEip) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime, 0)
}

func (self *SEip) GetExpiredAt() time.Time {
	return time.Unix(self.ExpireTime, 0)
}

func (self *SEip) GetIpAddr() string {
	if len(self.EIPAddr) > 1 {
		log.Warningf("GetIpAddr %d eip addr found", len(self.EIPAddr))
	} else if len(self.EIPAddr) == 0 {
		return ""
	}

	return self.EIPAddr[0].IP
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetAssociationType() string {
	return "server"
}

// 已绑定的资源类型, 枚举值为: uhost, 云主机；natgw：NAT网关；ulb：负载均衡器；upm: 物理机; hadoophost: 大数据集群;fortresshost：堡垒机；udockhost：容器；udhost：私有专区主机；vpngw：IPSec VPN；ucdr：云灾备；dbaudit：数据库审计。
func (self *SEip) GetAssociationExternalId() string {
	if self.Resource.ResourceType == "uhost" {
		return self.Resource.ResourceID
	} else if self.Resource.ResourceType != "" {
		log.Warningf("GetAssociationExternalId bind with %s %s.expect uhost", self.Resource.ResourceType, self.Resource.ResourceID)
	}

	return ""
}

func (self *SEip) GetBandwidth() int {
	return self.BandwidthMb
}

func (self *SEip) GetINetworkId() string {
	return ""
}

// 弹性IP的计费模式, 枚举值为: "Bandwidth", 带宽计费; "Traffic", 流量计费; "ShareBandwidth",共享带宽模式. 默认为 "Bandwidth".
func (self *SEip) GetInternetChargeType() string {
	switch self.PayMode {
	case "Bandwidth":
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	case "Traffic":
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	default:
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	}
}

func (self *SEip) Delete() error {
	return self.region.DeallocateEIP(self.GetId())
}

func (self *SEip) Associate(instanceId string) error {
	return self.region.AssociateEip(self.GetId(), instanceId)
}

func (self *SEip) Dissociate() error {
	return self.region.DissociateEip(self.GetId(), self.Resource.ResourceID)
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.GetId(), bw)
}

// https://docs.ucloud.cn/api/unet-api/allocate_eip
// 增加共享带宽模式ShareBandwidth
func (self *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	if len(eip.BGPType) == 0 {
		if strings.HasPrefix(self.GetId(), "cn-") {
			eip.BGPType = "Bgp"
		} else {
			eip.BGPType = "International"
		}
	}

	params := NewUcloudParams()
	params.Set("OperatorName", eip.BGPType)
	params.Set("Bandwidth", eip.BandwidthMbps)
	params.Set("Name", eip.Name)
	var payMode string
	switch eip.ChargeType {
	case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
		payMode = "Traffic"
	case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		payMode = "Bandwidth"
	}
	params.Set("PayMode", payMode)
	params.Set("ChargeType", "Dynamic") // 按需付费

	eips := make([]SEip, 0)
	err := self.DoAction("AllocateEIP", params, &eips)
	if err != nil {
		return nil, err
	}

	if len(eips) == 1 {
		eip := eips[0]
		eip.region = self
		eip.Refresh()
		return &eip, nil
	} else {
		return nil, fmt.Errorf("CreateEIP %d eip created", len(eips))
	}
}

// https://docs.ucloud.cn/api/unet-api/release_eip
func (self *SRegion) DeallocateEIP(eipId string) error {
	params := NewUcloudParams()
	params.Set("EIPId", eipId)

	return self.DoAction("ReleaseEIP", params, nil)
}

// https://docs.ucloud.cn/api/unet-api/bind_eip
func (self *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := NewUcloudParams()
	params.Set("EIPId", eipId)
	params.Set("ResourceType", "uhost")
	params.Set("ResourceId", instanceId)

	return self.DoAction("BindEIP", params, nil)
}

// https://docs.ucloud.cn/api/unet-api/unbind_eip
func (self *SRegion) DissociateEip(eipId string, instanceId string) error {
	params := NewUcloudParams()
	params.Set("EIPId", eipId)
	params.Set("ResourceType", "uhost")
	params.Set("ResourceId", instanceId)

	return self.DoAction("UnBindEIP", params, nil)
}

// https://docs.ucloud.cn/api/unet-api/modify_eip_bandwidth
func (self *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	params := NewUcloudParams()
	params.Set("EIPId", eipId)
	params.Set("Bandwidth", bw)

	return self.DoAction("ModifyEIPBandwidth", params, nil)
}
