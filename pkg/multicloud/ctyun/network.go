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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SNetwork struct {
	vpc  *SVpc
	wire *SWire

	CIDR            string `json:"cidr"`
	FirstDcn        string `json:"firstDcn"`
	Gateway         string `json:"gateway"`
	Name            string `json:"name"`
	NeutronSubnetID string `json:"neutronSubnetId"`
	RegionID        string `json:"regionId"`
	ResVLANID       string `json:"resVlanId"`
	SecondDcn       string `json:"secondDcn"`
	VLANStatus      string `json:"vlanStatus"`
	VpcID           string `json:"vpcId"`
	ZoneID          string `json:"zoneId"`
	ZoneName        string `json:"zoneName"`
}

func (self *SNetwork) GetId() string {
	return self.ResVLANID
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SNetwork) GetStatus() string {
	switch self.VLANStatus {
	case "ACTIVE", "UNKNOWN":
		return api.NETWORK_STATUS_AVAILABLE
	case "ERROR":
		return api.NETWORK_STATUS_UNKNOWN
	default:
		return api.NETWORK_STATUS_UNKNOWN
	}
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.GetId())
	new, err := self.wire.region.GetNetwork(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120
}

func (self *SRegion) GetNetwroks(vpcId string) ([]SNetwork, error) {
	querys := map[string]string{
		"regionId": self.GetId(),
	}
	if len(vpcId) > 0 {
		querys["vpcId"] = vpcId
	}

	networks := make([]SNetwork, 0)
	resp, err := self.client.DoGet("/apiproxy/v3/getSubnets", querys)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNetwroks.DoGet")
	}

	err = resp.Unmarshal(&networks, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNetwroks.Unmarshal")
	}

	for i := range networks {
		vpc, err := self.GetVpc(networks[i].VpcID)
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.GetNetwork.GetVpc")
		}
		networks[i].vpc = vpc

		networks[i].wire = &SWire{
			region: self,
			vpc:    vpc,
		}

		networks[i].wire.addNetwork(&networks[i])
	}

	return networks, err
}

func (self *SRegion) getNetwork(subnetId string) (*SNetwork, error) {
	querys := map[string]string{
		"subnetId": subnetId,
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/querySubnetDetail", querys)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.getNetwork.DoGet")
	}

	network := &SNetwork{}
	err = resp.Unmarshal(network, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.getNetwork.Unmarshal")
	}

	return network, nil
}

func (self *SRegion) GetNetwork(subnetId string) (*SNetwork, error) {
	network, err := self.getNetwork(subnetId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNetwork.getNetwork")
	}

	vpc, err := self.GetVpc(network.VpcID)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNetwork.GetVpc")
	}
	network.vpc = vpc

	network.wire = &SWire{
		region: self,
		vpc:    vpc,
	}

	network.wire.addNetwork(network)
	return network, err
}

func (self *SRegion) CreateNetwork(vpcId, zoneId, name, cidr, gatewayIp, dhcpEnable string) (*SNetwork, error) {
	networkParams := jsonutils.NewDict()
	networkParams.Set("regionId", jsonutils.NewString(self.GetId()))
	networkParams.Set("zoneId", jsonutils.NewString(zoneId))
	networkParams.Set("name", jsonutils.NewString(name))
	networkParams.Set("cidr", jsonutils.NewString(cidr))
	networkParams.Set("gatewayIp", jsonutils.NewString(gatewayIp))
	networkParams.Set("dhcpEnable", jsonutils.NewString(dhcpEnable))
	networkParams.Set("vpcId", jsonutils.NewString(vpcId))
	// DNS地址，如果主机需要访问公网就需要填写该值，不填写就不能使用DNS解析
	// networkParams.Set("primaryDns", jsonutils.NewString(primaryDns))
	// networkParams.Set("secondaryDns", jsonutils.NewString(secondaryDns))

	params := map[string]jsonutils.JSONObject{
		"jsonStr": networkParams,
	}

	network := &SNetwork{}
	resp, err := self.client.DoPost("/apiproxy/v3/createSubnet", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateNetwork.DoPost")
	}

	err = resp.Unmarshal(network, "returnObj")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.CreateNetwork.Unmarshal")
	}

	return network, err
}
