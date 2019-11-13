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

package qcloud

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

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("PayByTraffic")
	InternetChargeByBandwidth = TInternetChargeType("PayByBandwidth")
)

const (
	EIP_STATUS_CREATING      = "CREATING"
	EIP_STATUS_BINDING       = "BINDING"
	EIP_STATUS_BIND          = "BIND"
	EIP_STATUS_UNBINDING     = "UNBINDING"
	EIP_STATUS_UNBIND        = "UNBIND"
	EIP_STATUS_OFFLINING     = "OFFLINING"
	EIP_STATUS_BIND_ENI      = "BIND_ENI"
	EIP_STATUS_CREATE_FAILED = "CREATE_FAILED"

	EIP_TYPE_CALCIP     = "CalcIP"     //表示设备ip
	EIP_TYPE_WANIP      = "WanIP"      //普通公网ip
	EIP_TYPE_EIP        = "EIP"        //弹性公网ip
	EIP_TYPE_ANYCASTEIP = "AnycastEIP" //加速EIP
)

type SEipAddress struct {
	region *SRegion

	AddressId             string    //	EIP的ID，是EIP的唯一标识。
	AddressName           string    //	EIP名称。
	AddressStatus         string    //	EIP状态。
	AddressIp             string    //	外网IP地址
	InstanceId            string    //	绑定的资源实例ID。可能是一个CVM，NAT。
	CreatedTime           time.Time //	创建时间。按照ISO8601标准表示，并且使用UTC时间。格式为：YYYY-MM-DDThh:mm:ssZ。
	NetworkInterfaceId    string    //	绑定的弹性网卡ID
	PrivateAddressIp      string    //	绑定的资源内网ip
	IsArrears             bool      //	资源隔离状态。true表示eip处于隔离状态，false表示资源处于未隔离装填
	IsBlocked             bool      //	资源封堵状态。true表示eip处于封堵状态，false表示eip处于未封堵状态
	IsEipDirectConnection bool      //	eip是否支持直通模式。true表示eip支持直通模式，false表示资源不支持直通模式
	AddressType           string    //	eip资源类型，包括"CalcIP","WanIP","EIP","AnycastEIP"。其中"CalcIP"表示设备ip，“WanIP”表示普通公网ip，“EIP”表示弹性公网ip，“AnycastEip”表示加速EIP
	CascadeRelease        bool      //	eip是否在解绑后自动释放。true表示eip将会在解绑后自动释放，false表示eip在解绑后不会自动释放
}

func (self *SEipAddress) GetId() string {
	return self.AddressId
}

func (self *SEipAddress) GetName() string {
	if len(self.AddressName) > 0 && self.AddressName != "未命名" {
		return self.AddressName
	}
	return self.AddressId
}

func (self *SEipAddress) GetGlobalId() string {
	return self.AddressId
}

func (self *SEipAddress) GetStatus() string {
	switch self.AddressStatus {
	case EIP_STATUS_CREATING:
		return api.EIP_STATUS_ALLOCATE
	case EIP_STATUS_BINDING:
		return api.EIP_STATUS_ASSOCIATE
	case EIP_STATUS_UNBINDING:
		return api.EIP_STATUS_DISSOCIATE
	case EIP_STATUS_UNBIND, EIP_STATUS_BIND, EIP_STATUS_OFFLINING, EIP_STATUS_BIND_ENI:
		return api.EIP_STATUS_READY
	case EIP_STATUS_CREATE_FAILED:
		return api.EIP_STATUS_ALLOCATE_FAIL
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.AddressId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	if self.AddressId == self.InstanceId {
		// fixed Public IP
		return true
	} else {
		return false
	}
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.AddressIp
}

func (self *SEipAddress) GetMode() string {
	if self.InstanceId == self.AddressId {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	}
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	if len(self.InstanceId) > 0 {
		for prefix, instanceType := range map[string]string{
			"nat-": api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY,
			"ins-": api.EIP_ASSOCIATE_TYPE_SERVER,
			"lb-":  api.EIP_ASSOCIATE_TYPE_LOADBALANCER,
			"lbl-": api.EIP_ASSOCIATE_TYPE_LOADBALANCER,
		} {
			if strings.HasPrefix(self.InstanceId, prefix) {
				return instanceType
			}
		}
		return api.EIP_ASSOCIATE_TYPE_UNKNOWN
	}
	return ""
}

func (self *SEipAddress) GetAssociationExternalId() string {
	return self.InstanceId
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.AddressId)
}

func (self *SEipAddress) GetBandwidth() int {
	if len(self.InstanceId) > 0 {
		if instance, err := self.region.GetInstance(self.InstanceId); err == nil {
			return instance.InternetAccessible.InternetMaxBandwidthOut
		}
	}
	return 0
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return self.CreatedTime
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) GetInternetChargeType() string {
	if len(self.InstanceId) > 0 {
		if instance, err := self.region.GetInstance(self.InstanceId); err == nil {
			switch instance.InternetAccessible.InternetChargeType {
			case InternetChargeTypeTrafficPostpaidByHour:
				return api.EIP_CHARGE_TYPE_BY_TRAFFIC
			default:
				return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
			}
		}
	}
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEipAddress) Associate(instanceId string) error {
	err := self.region.AssociateEip(self.AddressId, instanceId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AddressId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	if self.GetInternetChargeType() == api.EIP_CHARGE_TYPE_BY_TRAFFIC {
		if len(self.InstanceId) > 0 {
			return self.region.UpdateInstanceBandwidth(self.InstanceId, bw)
		}
	}
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) GetEips(eipId string, instanceId string, offset int, limit int) ([]SEipAddress, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}

	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	if len(eipId) > 0 {
		params["AddressIds.0"] = eipId
	}

	if len(instanceId) > 0 {
		params["Filters.0.Name"] = "instance-id"
		params["Filters.0.Values.0"] = instanceId
	}

	body, err := region.vpcRequest("DescribeAddresses", params)
	if err != nil {
		log.Errorf("DescribeEipAddresses fail %s", err)
		return nil, 0, err
	}

	eips := make([]SEipAddress, 0)
	err = body.Unmarshal(&eips, "AddressSet")
	if err != nil {
		log.Errorf("Unmarshal EipAddress details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	for i := 0; i < len(eips); i++ {
		eips[i].region = region
	}
	return eips, int(total), nil
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eips, total, err := region.GetEips(eipId, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &eips[0], nil
}

func (region *SRegion) AllocateEIP(name string, bwMbps int, chargeType TInternetChargeType) (*SEipAddress, error) {
	params := make(map[string]string)
	params["Region"] = region.Region
	addRessSet := []string{}
	body, err := region.vpcRequest("AllocateAddresses", params)
	if err != nil {
		return nil, err
	}
	err = body.Unmarshal(&addRessSet, "AddressSet")
	if err != nil {
		return nil, err
	}
	if len(name) > 20 {
		name = name[:20]
	}
	if len(addRessSet) > 0 {
		params["AddressId"] = addRessSet[0]
		params["AddressName"] = name
		_, err = region.vpcRequest("ModifyAddressAttribute", params)
		if err != nil {
			return nil, err
		}

		eip, err := region.GetEip(addRessSet[0])
		if err != nil {
			return nil, err
		}
		return eip, cloudprovider.WaitStatus(eip, api.EIP_STATUS_READY, time.Second*5, time.Second*300)
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	var ctype TInternetChargeType
	switch eip.ChargeType {
	case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
		ctype = InternetChargeByTraffic
	case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		ctype = InternetChargeByBandwidth
	}
	return region.AllocateEIP(eip.Name, eip.BandwidthMbps, ctype)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressIds.0"] = eipId

	_, err := region.vpcRequest("ReleaseAddresses", params)
	if err != nil {
		log.Errorf("ReleaseAddresses fail %s", err)
	}
	return err
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := make(map[string]string)
	params["AddressId"] = eipId
	params["InstanceId"] = instanceId

	_, err := region.vpcRequest("AssociateAddress", params)
	if err != nil {
		log.Errorf("AssociateAddress fail %s", err)
	}
	return err
}

func (region *SRegion) DissociateEip(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressId"] = eipId

	_, err := region.vpcRequest("DisassociateAddress", params)
	if err != nil {
		log.Errorf("UnassociateEipAddress fail %s", err)
	}
	return err
}

func (region *SRegion) UpdateInstanceBandwidth(instanceId string, bw int) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["InstanceIds.0"] = instanceId
	params["InternetAccessible.InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bw)

	_, err := region.cvmRequest("ResetInstancesInternetMaxBandwidth", params, true)
	return err
}

func (self *SEipAddress) GetProjectId() string {
	return ""
}
