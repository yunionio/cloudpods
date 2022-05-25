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
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	multicloud.SEipBase
	multicloud.QcloudTags

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
	Bandwidth             int
	InternetChargeType    string
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
		if info := strings.Split(self.InstanceId, "-"); len(info) > 0 {
			return info[0]
		}
		return self.InstanceId
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
	if self.Bandwidth > 0 {
		return self.Bandwidth
	}
	if len(self.InstanceId) > 0 {
		if strings.HasPrefix(self.InstanceId, "ins-") {
			if instance, err := self.region.GetInstance(self.InstanceId); err == nil {
				return instance.InternetAccessible.InternetMaxBandwidthOut
			}
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
	switch self.InternetChargeType {
	case "TRAFFIC_POSTPAID_BY_HOUR":
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	case "BANDWIDTH_PACKAGE", "BANDWIDTH_POSTPAID_BY_HOUR", "BANDWIDTH_PREPAID_BY_MONTH":
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	}
	if len(self.InstanceId) > 0 {
		if strings.HasPrefix(self.InstanceId, "ins-") {
			if instance, err := self.region.GetInstance(self.InstanceId); err == nil {
				switch instance.InternetAccessible.InternetChargeType {
				case InternetChargeTypeTrafficPostpaidByHour:
					return api.EIP_CHARGE_TYPE_BY_TRAFFIC
				default:
					return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
				}
			}
		}
	}
	return ""
}

func (self *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	err := self.region.AssociateEip(self.AddressId, conf.InstanceId)
	if err != nil {
		return err
	}
	if conf.Bandwidth > 0 && self.Bandwidth == 0 {
		err = self.region.UpdateInstanceBandwidth(conf.InstanceId, conf.Bandwidth, conf.ChargeType)
		if err != nil {
			log.Warningf("failed to change instance %s bandwidth -> %d error: %v", conf.InstanceId, conf.Bandwidth, err)
		}
	}
	return cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 5*time.Second, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.AddressId)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 5*time.Second, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	if len(self.InstanceId) > 0 && self.Bandwidth == 0 && self.GetAssociationType() == api.EIP_ASSOCIATE_TYPE_SERVER {
		return self.region.UpdateInstanceBandwidth(self.InstanceId, bw, "")
	}
	if len(self.InternetChargeType) > 0 {
		return self.region.ChangeEipBindWidth(self.AddressId, bw, self.InternetChargeType)
	}
	return nil
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

func (region *SRegion) AllocateEIP(name string, bwMbps int, chargeType string) (*SEipAddress, error) {
	params := make(map[string]string)
	params["AddressName"] = name
	if len(name) > 20 {
		params["AddressName"] = name[:20]
	}
	if bwMbps > 0 {
		params["InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bwMbps)
	}

	_, totalCount, err := region.GetBandwidthPackages([]string{}, 0, 50)
	if err != nil {
		return nil, errors.Wrapf(err, "GetBandwidthPackages")
	}
	if totalCount == 0 {
		switch chargeType {
		case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
			params["InternetChargeType"] = "TRAFFIC_POSTPAID_BY_HOUR"
		case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
			params["InternetChargeType"] = "BANDWIDTH_POSTPAID_BY_HOUR"
		}
	}

	addRessSet := []string{}
	body, err := region.vpcRequest("AllocateAddresses", params)
	if err != nil {
		return nil, errors.Wrapf(err, "AllocateAddresses")
	}
	err = body.Unmarshal(&addRessSet, "AddressSet")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return region.GetEip(addRessSet[0])
}

// https://cloud.tencent.com/document/api/215/16699
// 腾讯云eip不支持指定项目
func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return region.AllocateEIP(eip.Name, eip.BandwidthMbps, eip.ChargeType)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressIds.0"] = eipId

	_, err := region.vpcRequest("ReleaseAddresses", params)
	return errors.Wrapf(err, "ReleaseAddresses")
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := make(map[string]string)
	params["AddressId"] = eipId
	params["InstanceId"] = instanceId

	_, err := region.vpcRequest("AssociateAddress", params)
	return errors.Wrapf(err, "AssociateAddress")
}

func (region *SRegion) DissociateEip(eipId string) error {
	params := make(map[string]string)
	params["Region"] = region.Region
	params["AddressId"] = eipId

	_, err := region.vpcRequest("DisassociateAddress", params)
	return errors.Wrapf(err, "DisassociateAddress")
}

func (self *SRegion) UpdateInstanceBandwidth(instanceId string, bw int, chargeType string) error {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["InternetAccessible.InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bw)

	_, totalCount, err := self.GetBandwidthPackages([]string{}, 0, 50)
	if err != nil {
		return errors.Wrapf(err, "GetBandwidthPackages")
	}
	if totalCount > 0 {
		params["InstanceIds.0"] = instanceId
		_, err = self.cvmRequest("ResetInstancesInternetMaxBandwidth", params, true)
		return errors.Wrapf(err, "ResetInstancesInternetMaxBandwidth")
	}
	internetChargeType := "TRAFFIC_POSTPAID_BY_HOUR"
	if chargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		internetChargeType = "BANDWIDTH_POSTPAID_BY_HOUR"
	}
	params["InternetAccessible.InternetChargeType"] = internetChargeType
	action := "ResetInstancesInternetMaxBandwidth"

	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", instanceId)
	}
	if instance.InternetAccessible.InternetChargeType != internetChargeType { //避免 Code=InvalidParameterValue, Message=参数`InternetChargeType`中`TRAFFIC_POSTPAID_BY_HOUR`没有更改, RequestId=6be2f9bc-a967-41db-9f0d-aff789c703ca
		params["InstanceId"] = instanceId
		action = "ModifyInstanceInternetChargeType"
	} else {
		params["InstanceIds.0"] = instanceId
	}
	err = cloudprovider.Wait(time.Second*5, time.Minute*3, func() (bool, error) {
		instance, err := self.GetInstance(instanceId)
		if err != nil {
			return false, errors.Wrapf(err, "GetInstance(%s)", instanceId)
		}
		_bw := instance.InternetAccessible.InternetMaxBandwidthOut
		log.Infof("%s bandwidth from %d -> %d expect %d", action, _bw, bw, bw)
		if _bw == bw {
			return true, nil
		}
		if _, err := self.cvmRequest(action, params, true); err != nil {
			log.Errorf("%s %v", action, err)
			return false, nil
		}
		return false, nil
	})
	return errors.Wrapf(err, "cloudprovider.Wait bandwidth changed")
}

func (self *SRegion) ChangeEipBindWidth(eipId string, bw int, chargeType string) error {
	params := map[string]string{
		"Region":                  self.Region,
		"InternetMaxBandwidthOut": fmt.Sprintf("%d", bw),
		"InternetChargeType":      chargeType,
		"AddressId":               eipId,
	}
	_, err := self.vpcRequest("ModifyAddressInternetChargeType", params)
	return errors.Wrapf(err, "ModifyAddressInternetChargeType")
}

func (self *SEipAddress) GetProjectId() string {
	return ""
}
