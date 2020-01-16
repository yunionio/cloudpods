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

package ctyun

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SEip struct {
	region *SRegion

	IPVersion           int64   `json:"ip_version"`
	BandwidthShareType  string  `json:"bandwidth_share_type"`
	Type                string  `json:"type"`
	PrivateIPAddress    string  `json:"private_ip_address"`
	EnterpriseProjectID string  `json:"enterprise_project_id"`
	Status              string  `json:"status"`
	PublicIPAddress     string  `json:"public_ip_address"`
	ID                  string  `json:"id"`
	TenantID            string  `json:"tenant_id"`
	Profile             Profile `json:"profile"`
	BandwidthName       string  `json:"bandwidth_name"`
	BandwidthID         string  `json:"bandwidth_id"`
	PortID              string  `json:"port_id"`
	BandwidthSize       int     `json:"bandwidth_size"`
	CreateTime          int64   `json:"create_time"`
	MasterOrderID       string  `json:"masterOrderId"`
	WorkOrderResourceID string  `json:"workOrderResourceId"`
	ExpireTime          int64   `json:"expireTime"`
	IsFreeze            int64   `json:"isFreeze"`
}

func (self *SEip) GetBillingType() string {
	if len(self.MasterOrderID) > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SEip) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime/1000, 0)
}

func (self *SEip) GetExpiredAt() time.Time {
	if self.ExpireTime == 0 {
		return time.Time{}
	}

	return time.Unix(self.ExpireTime/1000, 0)
}

func (self *SEip) GetId() string {
	return self.ID
}

func (self *SEip) GetName() string {
	return self.BandwidthName
}

func (self *SEip) GetGlobalId() string {
	return self.GetId()
}

func (self *SEip) GetStatus() string {
	switch self.Status {
	case "ACTIVE", "DOWN":
		return api.EIP_STATUS_READY
	case "ERROR":
		return api.EIP_STATUS_ALLOCATE_FAIL
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEip) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.ID)
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

func (self *SEip) GetProjectId() string {
	return ""
}

func (self *SEip) GetIpAddr() string {
	return self.PublicIPAddress
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetINetworkId() string {
	return ""
}

func (self *SEip) GetAssociationType() string {
	orders, err := self.region.GetOrder(self.WorkOrderResourceID)
	if err != nil {
		log.Errorf("SEip.GetAssociationType %s", err)
		return ""
	}

	for i := range orders {
		order := orders[i]
		if strings.Contains(order.ResourceType, "LOADBALANCER") {
			return api.EIP_ASSOCIATE_TYPE_ELB
		} else {
			return api.EIP_ASSOCIATE_TYPE_SERVER
		}
	}

	return ""
}

// eip查询接口未返回 绑定的实例ID/网卡port id。导致不能正常找出关联的主机
func (self *SEip) GetAssociationExternalId() string {
	vms, err := self.region.GetVMs()
	if err != nil {
		log.Errorf("SEip.GetAssociationExternalId.GetVMs %s", err)
		return ""
	}

	for i := range vms {
		vm := vms[i]
		nics, err := self.region.GetNics(vm.GetId())
		if err != nil {
			log.Errorf("SEip.GetAssociationExternalId.GetNics %s", err)
			return ""
		}

		for _, nic := range nics {
			if nic.PortID == self.PortID {
				return vm.GetGlobalId()
			}
		}
	}

	return ""
}

// http://ctyun-api-url/apiproxy/v3/queryNetworkDetail
func (self *SEip) GetBandwidth() int {
	return self.BandwidthSize
}

func (self *SEip) GetInternetChargeType() string {
	// todo: fix me
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SEip) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) Associate(instanceId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) Dissociate() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return cloudprovider.ErrNotImplemented
}

type Profile struct {
	OrderID   string `json:"order_id"`
	RegionID  string `json:"region_id"`
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
}

func (self *SRegion) GetEips() ([]SEip, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	eips := make([]SEip, 0)
	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryIps", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetEips.DoGet")
	}

	err = resp.Unmarshal(&eips, "returnObj", "publicips")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetEips.Unmarshal")
	}

	for i := range eips {
		eips[i].region = self
	}

	return eips, nil
}

func (self *SRegion) GetEip(eipId string) (*SEip, error) {
	params := map[string]string{
		"regionId":   self.GetId(),
		"publicIpId": eipId,
	}

	eips := make([]SEip, 0)
	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryIps", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetEip.DoGet")
	}

	err = resp.Unmarshal(&eips, "returnObj", "publicips")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetEip.Unmarshal")
	}

	if len(eips) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetEip")
	} else if len(eips) == 1 {
		eips[0].region = self
		return &eips[0], nil
	} else {
		return nil, errors.Wrap(cloudprovider.ErrDuplicateId, "SRegion.GetEip")
	}
}

func (self *SRegion) CreateEip(zoneId, name, size, shareType string) (*SEip, error) {
	eipParams := jsonutils.NewDict()
	eipParams.Set("regionId", jsonutils.NewString(self.GetId()))
	eipParams.Set("zoneId", jsonutils.NewString(zoneId))
	eipParams.Set("name", jsonutils.NewString(name))
	eipParams.Set("type", jsonutils.NewString("5_telcom"))
	eipParams.Set("size", jsonutils.NewString(size))
	eipParams.Set("shareType", jsonutils.NewString(shareType))

	params := map[string]jsonutils.JSONObject{
		"createIpInfo": eipParams,
	}

	eip := &SEip{}
	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/createIp", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateEip.DoPost")
	}

	err = resp.Unmarshal(eip, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateEip.Unmarshal")
	}

	eip.region = self
	return eip, nil
}
