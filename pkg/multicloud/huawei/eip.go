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
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("traffic")
	InternetChargeByBandwidth = TInternetChargeType("bandwidth")
)

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
	port   *Port
	multicloud.SEipBase
	multicloud.HuaweiTags

	Alias               string
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
	EnterpriseProjectId string
}

func (self *SEipAddress) GetId() string {
	return self.ID
}

func (self *SEipAddress) GetName() string {
	if len(self.Alias) > 0 {
		return self.Alias
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
	if len(self.PortId) == 0 {
		return ""
	}
	port, err := self.region.GetPort(self.PortId)
	if err != nil {
		log.Errorf("Get eip %s port %s error: %v", self.ID, self.PortId, err)
		return ""
	}

	switch port.DeviceOwner {
	case "neutron:LOADBALANCER", "neutron:LOADBALANCERV2":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	case "network:nat_gateway":
		return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
	default:
		return port.DeviceOwner
	}
}

func (self *SEipAddress) GetAssociationExternalId() string {
	// network/0273a359d61847fc83405926c958c746/ext-floatingips?tenantId=0273a359d61847fc83405926c958c746&limit=2000
	// 只能通过 port id 反查device id.
	if len(self.PortId) > 0 {
		port, _ := self.region.GetPort(self.PortId)
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
	if bandwidth.ChargeMode == "traffic" {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SEipAddress) GetBillingType() string {
	if self.Profile == nil {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
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

func (self *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	portId, err := self.region.GetInstancePortId(conf.InstanceId)
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

	err = cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 10*time.Second, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.ID)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
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
func (self *SRegion) AllocateEIP(name string, bwMbps int, chargeType TInternetChargeType, bgpType string, projectId string) (*SEipAddress, error) {
	params := map[string]interface{}{
		"bandwidth": map[string]interface{}{
			"name":        name,
			"size":        bwMbps,
			"share_type":  "PER",
			"charge_mode": chargeType,
		},
		"publicip": map[string]interface{}{
			"type":       bgpType,
			"ip_version": 4,
			"alias":      name,
		},
	}
	if len(projectId) > 0 {
		params["enterprise_project_id"] = projectId
	}
	resp, err := self.vpcCreate("publicips", params)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: self}
	return eip, resp.Unmarshal(eip, "publicip")
}

func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	resp, err := self.vpcGet("publicips/" + eipId)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: self}
	return eip, resp.Unmarshal(eip, "publicip")
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	_, err := self.vpcDelete("publicips/" + eipId)
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
	params := map[string]interface{}{
		"publicip": map[string]interface{}{
			"port_id": portId,
		},
	}
	_, err := self.vpcUpdate("publicips/"+eipId, params)
	return err
}

func (self *SRegion) DissociateEip(eipId string) error {
	return self.AssociateEipWithPortId(eipId, "")
}

func (self *SRegion) UpdateEipBandwidth(bandwidthId string, bw int) error {
	params := map[string]interface{}{
		"bandwidth": map[string]interface{}{
			"size": bw,
		},
	}
	_, err := self.vpcUpdate("bandwidths/"+bandwidthId, params)
	return err
}

func (self *SRegion) GetEipBandwidth(id string) (*Bandwidth, error) {
	resp, err := self.vpcGet("bandwidths/" + id)
	if err != nil {
		return nil, err
	}
	ret := &Bandwidth{}
	return ret, resp.Unmarshal(ret, "bandwidth")
}

func (self *SEipAddress) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SRegion) GetEips(portId string, addrs []string) ([]SEipAddress, error) {
	query := url.Values{}
	for _, addr := range addrs {
		query.Add("public_ip_address", addr)
	}
	if len(portId) > 0 {
		query.Set("port_id", portId)
	}
	resp, err := self.vpcList("publicips", query)
	if err != nil {
		return nil, err
	}
	eips := []SEipAddress{}
	err = resp.Unmarshal(&eips, "publicips")
	if err != nil {
		return nil, err
	}
	for i := range eips {
		eips[i].region = self
	}
	return eips, nil
}
