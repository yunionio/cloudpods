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

package huawei

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("traffic")
	InternetChargeByBandwidth = TInternetChargeType("bandwidth")
)

type Port struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	AdminStateUp    string `json:"admin_state_up"`
	DNSName         string `json:"dns_name"`
	MACAddress      string `json:"mac_address"`
	NetworkID       string `json:"network_id"`
	TenantID        string `json:"tenant_id"`
	DeviceID        string `json:"device_id"`
	DeviceOwner     string `json:"device_owner"`
	BindingVnicType string `json:"binding:vnic_type"`
}

type Bandwidth struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Size                int64          `json:"size"`
	ShareType           string         `json:"share_type"`
	PublicipInfo        []PublicipInfo `json:"publicip_info"`
	TenantID            string         `json:"tenant_id"`
	BandwidthType       string         `json:"bandwidth_type"`
	ChargeMode          string         `json:"charge_mode"`
	BillingInfo         string         `json:"billing_info"`
	EnterpriseProjectID string         `json:"enterprise_project_id"`
}

type PublicipInfo struct {
	PublicipID      string `json:"publicip_id"`
	PublicipAddress string `json:"publicip_address"`
	PublicipType    string `json:"publicip_type"`
	IPVersion       int64  `json:"ip_version"`
}

type SProfile struct {
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
	RegionID  string `json:"region_id"`
	OrderID   string `json:"order_id"`
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090598.html
type SEipAddress struct {
	region *SRegion

	ID                  string    `json:"id"`
	Status              string    `json:"status"`
	Profile             *SProfile `json:"profile,omitempty"`
	Type                string    `json:"type"`
	PublicIPAddress     string    `json:"public_ip_address"`
	PrivateIPAddress    string    `json:"private_ip_address"`
	TenantID            string    `json:"tenant_id"`
	CreateTime          time.Time `json:"create_time"`
	BandwidthID         string    `json:"bandwidth_id"`
	BandwidthShareType  string    `json:"bandwidth_share_type"`
	BandwidthSize       int64     `json:"bandwidth_size"`
	BandwidthName       string    `json:"bandwidth_name"`
	EnterpriseProjectID string    `json:"enterprise_project_id"`
	IPVersion           int64     `json:"ip_version"`
	PortId              string    `json:"port_id"`
}

func (self *SEipAddress) GetId() string {
	return self.ID
}

func (self *SEipAddress) GetName() string {
	if len(self.BandwidthName) == 0 {
		return self.BandwidthName
	}

	return self.PublicIPAddress
}

func (self *SEipAddress) GetGlobalId() string {
	return self.ID
}

func (self *SEipAddress) GetStatus() string {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090598.html
	switch self.Status {
	case "ACTIVE", "DOWN", "ELB":
		return api.EIP_STATUS_READY
	case "PENDING_CREATE", "NOTIFYING":
		return api.EIP_STATUS_ALLOCATE
	case "BINDING":
		return api.EIP_STATUS_ALLOCATE
	case "BIND_ERROR":
		return api.EIP_STATUS_ALLOCATE_FAIL
	case "PENDING_DELETE", "NOTIFY_DELETE":
		return api.EIP_STATUS_DEALLOCATE
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEipAddress) IsEmulated() bool {
	return false
}

func (self *SEipAddress) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SEipAddress) GetIpAddr() string {
	return self.PublicIPAddress
}

func (self *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetPort() *Port {
	if len(self.PortId) == 0 {
		return nil
	}

	if self.port != nil {
		return self.port
	}

	port, err := self.region.GetPort(self.PortId)
	if err != nil {
		return nil
	} else {
		self.port = &port
	}

	return self.port
}

func (self *SEipAddress) GetAssociationType() string {
	return "server"
}

func (self *SEipAddress) GetAssociationExternalId() string {
	// network/0273a359d61847fc83405926c958c746/ext-floatingips?tenantId=0273a359d61847fc83405926c958c746&limit=2000
	// 只能通过 port id 反查device id.
	if len(self.PortId) > 0 {
		port, err := self.region.GetPort(self.PortId)
		if err != nil {
			return ""
		}

		return port.DeviceID
	}

	return ""
}

func (self *SEipAddress) GetBandwidth() int {
	return int(self.BandwidthSize) // Mb
}

func (self *SEipAddress) GetINetworkId() string {
	return ""
}

func (self *SEipAddress) GetInternetChargeType() string {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090603.html
	bandwidth, err := self.region.GetEipBandwidth(self.BandwidthID)
	if err != nil {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}

	if bandwidth.ChargeMode != "traffic" {
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	} else {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
}

func (self *SEipAddress) GetBillingType() string {
	if self.Profile == nil {
		return billing_api.BILLING_TYPE_POSTPAID
	} else {
		return billing_api.BILLING_TYPE_PREPAID
	}
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.ID)
}

func (self *SEipAddress) Associate(instanceId string) error {
	portId, err := self.region.GetInstancePortId(instanceId)
	if err != nil {
		return err
	}

	if len(self.PortId) > 0 {
		if self.PortId == portId {
			return nil
		}

		return fmt.Errorf("eip %s aready associate with port %s", self.GetId(), self.PortId)
	}

	err = self.region.AssociateEipWithPortId(self.ID, portId)
	if err != nil {
		return err
	}

	err = cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	port, err := self.region.GetPort(self.PortId)
	if err != nil {
		return err
	}

	err = self.region.DissociateEip(self.ID, port.DeviceID)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.BandwidthID, bw)
}

func (self *SRegion) GetInstancePortId(instanceId string) (string, error) {
	// 目前只绑定一个网卡
	// todo: 还需要按照ports状态进行过滤
	ports, err := self.GetPorts(instanceId)
	if err != nil {
		return "", err
	}

	if len(ports) == 0 {
		return "", fmt.Errorf("AssociateEip instance %s port is empty", instanceId)
	}

	return ports[0].ID, nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090596.html
func (self *SRegion) AllocateEIP(name string, bwMbps int, chargeType TInternetChargeType, bgpType string) (*SEipAddress, error) {
	paramsStr := `
{
    "publicip": {
        "type": "%s",
   "ip_version": 4
    },
    "bandwidth": {
        "name": "%s",
        "size": %d,
        "share_type": "PER",
        "charge_mode": "%s"
    }
}
`
	if len(bgpType) == 0 {
		return nil, fmt.Errorf("AllocateEIP bgp type should not be empty")
	}
	paramsStr = fmt.Sprintf(paramsStr, bgpType, name, bwMbps, chargeType)
	params, _ := jsonutils.ParseString(paramsStr)
	eip := SEipAddress{}
	err := DoCreate(self.ecsClient.Eips.Create, params, &eip)
	return &eip, err
}

func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	var eip SEipAddress
	err := DoGet(self.ecsClient.Eips.Get, eipId, nil, &eip)
	eip.region = self
	return &eip, err
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	_, err := self.ecsClient.Eips.Delete(eipId, nil)
	return err
}

func (self *SRegion) AssociateEip(eipId string, instanceId string) error {
	portId, err := self.GetInstancePortId(instanceId)
	if err != nil {
		return err
	}
	return self.AssociateEipWithPortId(eipId, portId)
}

func (self *SRegion) AssociateEipWithPortId(eipId string, portId string) error {
	params := jsonutils.NewDict()
	publicIPObj := jsonutils.NewDict()
	publicIPObj.Add(jsonutils.NewString(portId), "port_id")
	params.Add(publicIPObj, "publicip")

	_, err := self.ecsClient.Eips.Update(eipId, params)
	return err
}

func (self *SRegion) DissociateEip(eipId string, instanceId string) error {
	eip, err := self.GetEip(eipId)
	if err != nil {
		return err
	}

	// 已经是解绑状态
	if eip.Status == "DOWN" {
		return nil
	}

	remoteInstanceId := eip.GetAssociationExternalId()
	if remoteInstanceId != instanceId {
		return fmt.Errorf("eip %s associate with another instance %s", eipId, remoteInstanceId)
	}

	paramsStr := `{"publicip":{"port_id":null}}`
	params, _ := jsonutils.ParseString(paramsStr)
	_, err = self.ecsClient.Eips.Update(eipId, params)
	return err
}

func (self *SRegion) UpdateEipBandwidth(bandwidthId string, bw int) error {
	paramStr := `{
		"bandwidth":
		{
           "size": %d
		}
	}`

	paramStr = fmt.Sprintf(paramStr, bw)
	params, _ := jsonutils.ParseString(paramStr)
	_, err := self.ecsClient.Bandwidths.Update(bandwidthId, params)
	return err
}

func (self *SRegion) GetEipBandwidth(bandwidthId string) (Bandwidth, error) {
	bandwidth := Bandwidth{}
	err := DoGet(self.ecsClient.Bandwidths.Get, bandwidthId, nil, &bandwidth)
	return bandwidth, err
}

func (self *SRegion) GetPort(portId string) (Port, error) {
	port := Port{}
	err := DoGet(self.ecsClient.Port.Get, portId, nil, &port)
	return port, err
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0030591299.html
func (self *SRegion) GetPorts(instanceId string) ([]Port, error) {
	ports := make([]Port, 0)
	querys := map[string]string{}
	if len(instanceId) > 0 {
		querys["device_id"] = instanceId
	}

	err := doListAllWithMarker(self.ecsClient.Port.List, querys, &ports)
	return ports, err
}

func (self *SEipAddress) GetProjectId() string {
	return ""
}
