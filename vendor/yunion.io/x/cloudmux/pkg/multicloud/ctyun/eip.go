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
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEip struct {
	region *SRegion
	multicloud.SEipBase
	CtyunTags

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
		// bugfix: 由于没有接口可以返向查询出关联的device，这里默认关联的是server？
		if len(self.PortID) > 0 || len(self.PrivateIPAddress) > 0 {
			return api.EIP_ASSOCIATE_TYPE_SERVER
		}

		log.Errorf("SEip.GetAssociationType %s", err)
		return ""
	}

	for i := range orders {
		order := orders[i]
		if strings.Contains(order.ResourceType, "LOADBALANCER") {
			return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
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
	return ""
}

func (self *SEip) Delete() error {
	return self.region.DeleteEip(self.GetId())
}

func (self *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
	nics, err := self.region.GetNics(conf.InstanceId)
	if err != nil {
		return errors.Wrap(err, "Eip.Associate.GetNics")
	}

	if len(nics) == 0 {
		return errors.Wrap(fmt.Errorf("no network card found"), "Eip.Associate.GetINics")
	}

	return self.region.AssociateEip(self.GetId(), nics[0].PortID)
}

func (self *SEip) Dissociate() error {
	err := self.region.DissociateEip(self.GetId())
	if err != nil {
		return err
	}

	err = cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 5*time.Second, 10*time.Second, 180*time.Second)
	return errors.Wrap(err, "SEip.Dissociate")
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return self.region.ChangeBandwidthEip(self.GetName(), self.GetId(), strconv.Itoa(bw))
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

func (self *SRegion) CreateEip(zoneId, name, size, shareType, chargeType string) (*SEip, error) {
	eipParams := jsonutils.NewDict()
	eipParams.Set("regionId", jsonutils.NewString(self.GetId()))
	eipParams.Set("zoneId", jsonutils.NewString(zoneId))
	eipParams.Set("name", jsonutils.NewString(name))
	eipParams.Set("type", jsonutils.NewString("5_telcom"))
	eipParams.Set("size", jsonutils.NewString(size))
	eipParams.Set("shareType", jsonutils.NewString(shareType))
	eipParams.Set("chargeMode", jsonutils.NewString(chargeType))

	params := map[string]jsonutils.JSONObject{
		"createIpInfo": eipParams,
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/createIp", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateEip.DoPost")
	}

	eipId, err := resp.GetString("returnObj", "id")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateEip.GetEipId")
	}

	return self.GetEip(eipId)
}

func (self *SRegion) DeleteEip(publicIpId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId":   jsonutils.NewString(self.GetId()),
		"publicIpId": jsonutils.NewString(publicIpId),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/deleteIp", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DeleteEip.DoPost")
	}

	var ok bool
	err = resp.Unmarshal(&ok, "returnObj")
	if !ok {
		msg, _ := resp.GetString("message")
		return errors.Wrap(fmt.Errorf(msg), "SRegion.DeleteEip.JobFailed")
	}

	return nil
}

// 这里networkCardId 实际指的是port id
func (self *SRegion) AssociateEip(publicIpId, networkCardId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId":      jsonutils.NewString(self.GetId()),
		"publicIpId":    jsonutils.NewString(publicIpId),
		"networkCardId": jsonutils.NewString(networkCardId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/bindIp", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.AssociateEip.DoPost")
	}

	return nil
}

func (self *SRegion) DissociateEip(publicIpId string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId":   jsonutils.NewString(self.GetId()),
		"publicIpId": jsonutils.NewString(publicIpId),
	}

	_, err := self.client.DoPost("/apiproxy/v3/ondemand/unbindIp", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.DissociateEip.DoPost")
	}

	return nil
}

type SBandwidth struct {
	PublicipInfo        []PublicipInfo `json:"publicip_info"`
	EnterpriseProjectID string         `json:"enterprise_project_id"`
	Name                string         `json:"name"`
	ID                  string         `json:"id"`
	ShareType           string         `json:"share_type"`
	Size                int64          `json:"size"`
	TenantID            string         `json:"tenant_id"`
	ChargeMode          string         `json:"charge_mode"`
	BandwidthType       string         `json:"bandwidth_type"`
}

type PublicipInfo struct {
	PublicipType    string `json:"publicip_type"`
	PublicipAddress string `json:"publicip_address"`
	IPVersion       int64  `json:"ip_version"`
	PublicipID      string `json:"publicip_id"`
}

func (self *SRegion) ChangeBandwidthEip(name, publicIpId, sizeMb string) error {
	params := map[string]jsonutils.JSONObject{
		"regionId":   jsonutils.NewString(self.GetId()),
		"publicIpId": jsonutils.NewString(publicIpId),
		"name":       jsonutils.NewString(name),
		"size":       jsonutils.NewString(sizeMb),
	}

	resp, err := self.client.DoPost("/apiproxy/v3/ondemand/upgradeNetwork", params)
	if err != nil {
		return errors.Wrap(err, "SRegion.ChangeBandwidthEip.DoPost")
	}

	bandwidth := SBandwidth{}
	err = resp.Unmarshal(&bandwidth, "returnObj")
	if err != nil {
		return errors.Wrap(err, "SRegion.ChangeBandwidthEip.Unmarshal")
	}

	return nil
}
