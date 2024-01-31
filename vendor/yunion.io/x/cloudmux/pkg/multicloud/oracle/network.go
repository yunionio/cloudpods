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

package oracle

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	SOracleTag
	wire *SWire

	AvailabilityDomain string
	Id                 string
	CidrBlock          string
	DisplayName        string
	DnsLabel           string
	LifecycleState     string
	SecurityListIds    []string
	SubnetDomainName   string
	TimeCreated        time.Time
	VcnId              string
	VirtualRouterIp    string
	VirtualRouterMac   string
}

func (self *SNetwork) GetId() string {
	return self.Id
}

func (self *SNetwork) GetName() string {
	return self.DisplayName
}

func (self *SNetwork) GetGlobalId() string {
	return self.Id
}

func (self *SNetwork) GetStatus() string {
	// AVAILABLE, PROVISIONING, TERMINATED, TERMINATING, UPDATING
	switch self.LifecycleState {
	case "AVAILABLE", "UPDATING":
		return api.NETWORK_STATUS_AVAILABLE
	case "PROVISIONING":
		return apis.STATUS_CREATING
	case "TERMINATED", "TERMINATING":
		return apis.STATUS_DELETING
	default:
		return api.NETWORK_STATUS_UNKNOWN
	}
}

func (self *SNetwork) Delete() error {
	return self.wire.vpc.region.DeleteNetwork(self.Id)
}

func (self *SRegion) DeleteNetwork(networkId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CidrBlock)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

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
	return true
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	subnet, err := self.wire.vpc.region.GetNetwork(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, subnet)
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SRegion) GetNetworks(vpcId string) ([]SNetwork, error) {
	params := map[string]interface{}{}
	if len(vpcId) > 0 {
		params["vcnId"] = vpcId
	}
	resp, err := self.list(SERVICE_IAAS, "subnets", params)
	if err != nil {
		return nil, err
	}
	ret := []SNetwork{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	resp, err := self.get(SERVICE_IAAS, "subnets", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNetwork{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
