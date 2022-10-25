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

package hcs

import (
	"fmt"
	"net/url"
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

type TInternetChargeType string

const (
	InternetChargeByTraffic   = TInternetChargeType("traffic")
	InternetChargeByBandwidth = TInternetChargeType("bandwidth")
)

type Bandwidth struct {
	Id                  string         `json:"id"`
	Name                string         `json:"name"`
	Size                int64          `json:"size"`
	ShareType           string         `json:"share_type"`
	PublicipInfo        []PublicipInfo `json:"publicip_info"`
	TenantId            string         `json:"tenant_id"`
	BandwidthType       string         `json:"bandwidth_type"`
	ChargeMode          string         `json:"charge_mode"`
	BillingInfo         string         `json:"billing_info"`
	EnterpriseProjectId string         `json:"enterprise_project_id"`
}

type PublicipInfo struct {
	PublicipId      string `json:"publicip_id"`
	PublicipAddress string `json:"publicip_address"`
	PublicipType    string `json:"publicip_type"`
	IPVersion       int64  `json:"ip_version"`
}

type SProfile struct {
	UserId    string `json:"user_id"`
	ProductId string `json:"product_id"`
	RegionId  string `json:"region_id"`
	OrderId   string `json:"order_id"`
}

type SEip struct {
	multicloud.SEipBase
	multicloud.HcsTags

	region *SRegion
	port   *Port

	Alias               string
	Id                  string    `json:"id"`
	Status              string    `json:"status"`
	Profile             *SProfile `json:"profile,omitempty"`
	Type                string    `json:"type"`
	PublicIPAddress     string    `json:"public_ip_address"`
	PrivateIPAddress    string    `json:"private_ip_address"`
	TenantId            string    `json:"tenant_id"`
	CreateTime          time.Time `json:"create_time"`
	BandwidthId         string    `json:"bandwidth_id"`
	BandwidthShareType  string    `json:"bandwidth_share_type"`
	BandwidthSize       int64     `json:"bandwidth_size"`
	BandwidthName       string    `json:"bandwidth_name"`
	EnterpriseProjectId string    `json:"enterprise_project_id"`
	IPVersion           int64     `json:"ip_version"`
	PortId              string    `json:"port_id"`
}

func (self *SEip) GetId() string {
	return self.Id
}

func (self *SEip) GetName() string {
	if len(self.Alias) > 0 {
		return self.Alias
	}
	return self.PublicIPAddress
}

func (self *SEip) GetGlobalId() string {
	return self.Id
}

func (self *SEip) GetStatus() string {
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

func (self *SEip) Refresh() error {
	if self.IsEmulated() {
		return nil
	}
	new, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SEip) IsEmulated() bool {
	return false
}

func (self *SEip) GetIpAddr() string {
	return self.PublicIPAddress
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetPort() *Port {
	if len(self.PortId) == 0 {
		return nil
	}

	if self.port != nil {
		return self.port
	}

	port, err := self.region.GetPort(self.PortId)
	if err != nil {
		return nil
	}
	self.port = port
	return self.port
}

func (self *SEip) GetAssociationType() string {
	if len(self.PortId) == 0 {
		return ""
	}
	port, err := self.region.GetPort(self.PortId)
	if err != nil {
		log.Errorf("Get eip %s port %s error: %v", self.Id, self.PortId, err)
		return ""
	}

	if strings.HasPrefix(port.DeviceOwner, "compute") {
		return api.EIP_ASSOCIATE_TYPE_SERVER
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

func (self *SEip) GetAssociationExternalId() string {
	// network/0273a359d61847fc83405926c958c746/ext-floatingips?tenantId=0273a359d61847fc83405926c958c746&limit=2000
	// 只能通过 port id 反查device id.
	if len(self.PortId) > 0 {
		port, _ := self.region.GetPort(self.PortId)
		return port.DeviceId
	}
	return ""
}

func (self *SEip) GetBandwidth() int {
	return int(self.BandwidthSize) // Mb
}

func (self *SEip) GetINetworkId() string {
	return ""
}

func (self *SEip) GetInternetChargeType() string {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090603.html
	bandwidth, err := self.region.GetEipBandwidth(self.BandwidthId)
	if err != nil {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	if bandwidth.ChargeMode == "traffic" {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SEip) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEip) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SEip) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEip) Delete() error {
	return self.region.DeallocateEIP(self.Id)
}

func (self *SEip) Associate(conf *cloudprovider.AssociateConfig) error {
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

	err = self.region.AssociateEipWithPortId(self.Id, portId)
	if err != nil {
		return err
	}

	err = cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 10*time.Second, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEip) Dissociate() error {
	err := self.region.DissociateEip(self.Id)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.BandwidthId, bw)
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

	return ports[0].Id, nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090596.html
func (self *SRegion) AllocateEIP(name string, bwMbps int, chargeType TInternetChargeType, bgpType string, projectId string) (*SEip, error) {
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
	eip := &SEip{region: self}
	return eip, self.vpcCreate("publicips", params, eip)
}

func (self *SRegion) GetEip(eipId string) (*SEip, error) {
	eip := &SEip{region: self}
	return eip, self.vpcGet("publicips/"+eipId, eip)
}

func (self *SRegion) DeallocateEIP(eipId string) error {
	return self.vpcDelete("publicips/" + eipId)
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
	return self.vpcUpdate("publicips/"+eipId, params)
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
	return self.vpcUpdate("bandwidths/"+bandwidthId, params)
}

func (self *SRegion) GetEipBandwidth(id string) (*Bandwidth, error) {
	ret := &Bandwidth{}
	return ret, self.vpcGet("bandwidths/"+id, ret)
}

func (self *SEip) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SRegion) GetEips(portId string, addrs []string) ([]SEip, error) {
	query := url.Values{}
	for _, addr := range addrs {
		query.Add("public_ip_address", addr)
	}
	if len(portId) > 0 {
		query.Set("port_id", portId)
	}
	eips := []SEip{}
	return eips, self.vpcList("publicips", query, &eips)
}

type SEipType struct {
	Id    string
	Type  string
	Name  string
	Group string
}

func (self *SRegion) GetEipTypes() ([]SEipType, error) {
	query := url.Values{}
	ret := []SEipType{}
	return ret, self.vpcList("publicip_types", query, &ret)
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips("", nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i += 1 {
		eips[i].region = self
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	var ctype TInternetChargeType
	switch eip.ChargeType {
	case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
		ctype = InternetChargeByTraffic
	case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		ctype = InternetChargeByBandwidth
	}

	// todo: 如何避免hardcode。集成到cloudmeta服务中？
	if len(eip.BGPType) == 0 {
		types, err := self.GetEipTypes()
		if err != nil {
			return nil, errors.Wrapf(err, "GetEipTypes")
		}
		if len(types) > 0 {
			eip.BGPType = types[0].Type
		}
	}

	// 华为云EIP名字最大长度64
	if len(eip.Name) > 64 {
		eip.Name = eip.Name[:64]
	}

	ieip, err := self.AllocateEIP(eip.Name, eip.BandwidthMbps, ctype, eip.BGPType, eip.ProjectId)
	ieip.region = self
	if err != nil {
		return nil, err
	}

	err = cloudprovider.WaitStatus(ieip, api.EIP_STATUS_READY, 5*time.Second, 60*time.Second)
	return ieip, err
}
