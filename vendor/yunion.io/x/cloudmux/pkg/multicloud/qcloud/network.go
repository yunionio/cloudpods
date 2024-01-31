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
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	QcloudTags
	wire *SWire

	CidrBlock               string
	Zone                    string
	SubnetId                string
	VpcId                   string
	SubnetName              string
	AvailableIpAddressCount int
	CreatedTime             time.Time
	EnableBroadcast         bool
	IsDefault               bool
	RouteTableId            string
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if len(self.SubnetName) > 0 {
		return self.SubnetName
	}
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SNetwork) Delete() error {
	return self.wire.zone.region.DeleteNetwork(self.SubnetId)
}

func (self *SRegion) DeleteNetwork(networkId string) error {
	params := make(map[string]string)
	params["SubnetId"] = networkId

	interfaces := []SNetworkInterface{}
	for {
		_interfaces, total, err := self.GetNetworkInterfaces([]string{}, "", networkId, len(interfaces), 50)
		if err != nil {
			return errors.Wrapf(err, "DeleteNetwork.GetNetworkInterfaces")
		}
		interfaces = append(interfaces, _interfaces...)
		if len(interfaces) >= total {
			break
		}
	}

	for _, nic := range interfaces {
		err := self.DeleteNetworkInterface(nic.NetworkInterfaceId)
		if err != nil {
			return errors.Wrapf(err, "DeleteNetwork.DeleteNetworkInterface")
		}
	}

	_, err := self.vpcRequest("DeleteSubnet", params)
	if err != nil {
		return errors.Wrapf(err, "vpcRequest.DeleteSubnet")
	}
	return nil
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) SetTags(tags map[string]string, replace bool) error {
	return self.wire.vpc.region.SetResourceTags("vpc", "subnet", []string{self.SubnetId}, tags, replace)
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

//https://cloud.tencent.com/document/product/215/20046
func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	return pref.MaskLen
}

func (self *SNetwork) GetIsPublic() bool {
	// return self.IsDefault
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	log.Debugf("network refresh %s", self.SubnetId)
	new, err := self.wire.zone.region.GetNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SRegion) CreateNetwork(zoneId string, vpcId string, name string, cidr string, desc string) (string, error) {
	params := make(map[string]string)
	params["Zone"] = zoneId
	params["VpcId"] = vpcId
	params["CidrBlock"] = cidr
	params["SubnetName"] = name
	body, err := self.vpcRequest("CreateSubnet", params)
	if err != nil {
		return "", err
	}
	return body.GetString("Subnet", "SubnetId")
}

func (self *SNetwork) GetProjectId() string {
	return ""
}
