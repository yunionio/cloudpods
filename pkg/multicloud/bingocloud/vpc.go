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

package bingocloud

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type DhcpOptions struct {
}

type SPool struct {
	Range string `json:"range"`
}

type IPConfig struct {
	NetworkAddress    string      `json:"network_address"`
	PrefixLength      int         `json:"prefix_length"`
	DefaultGateway    string      `json:"default_gateway"`
	DhcpOptions       DhcpOptions `json:"dhcp_options"`
	Pool              []SPool     `json:"pool"`
	DhcpServerAddress string      `json:"dhcp_server_address"`
}

type SVpc struct {
	multicloud.SVpc
	multicloud.STagBase

	region *SRegion

	LogicalTimestamp int      `json:"logical_timestamp"`
	VlanID           int      `json:"vlan_id"`
	UUID             string   `json:"uuid"`
	Name             string   `json:"name"`
	IPConfig         IPConfig `json:"ip_config,omitempty"`
}

func (self *SVpc) GetName() string {
	return self.Name
}

func (self *SVpc) GetId() string {
	return self.UUID
}

func (self *SVpc) GetGlobalId() string {
	return self.UUID
}

// func (self *SVpc) Delete() error {
// 	return self.region.DeleteVpc(self.UUID)
// }

func (self *SVpc) GetCidrBlock() string {
	if len(self.IPConfig.NetworkAddress) > 0 {
		return fmt.Sprintf("%s/%d", self.IPConfig.NetworkAddress, self.IPConfig.PrefixLength)
	}
	return ""
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return []cloudprovider.ICloudRouteTable{}, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SRegion) GetVpcs() ([]SVpc, error) {
	vpcs := []SVpc{}
	_, err := self.list("networks", url.Values{}, &vpcs)
	return vpcs, err
}

func (self *SRegion) GetVpc(id string) (*SVpc, error) {
	vpc := &SVpc{region: self}
	return vpc, self.get("networks", id, url.Values{}, vpc)
}

func (self *SRegion) CreateVpc(opts *cloudprovider.VpcCreateOptions) (*SVpc, error) {
	ipConfig := map[string]interface{}{}
	if len(opts.CIDR) > 0 {
		if addrs := strings.Split(opts.CIDR, "/"); len(addrs) == 2 {
			ipConfig["network_address"] = addrs[0]
			ipConfig["prefix_length"], _ = strconv.Atoi(addrs[1])
		}

	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	vlanId, vlanIds := -1, []int{}
	for i := range vpcs {
		vlanIds = append(vlanIds, vpcs[i].VlanID)
	}
	sort.Ints(vlanIds)
	for _, vlan := range vlanIds {
		if vlan == vlanId+1 {
			vlanId = vlan
		}
	}
	params := map[string]interface{}{
		"name":       opts.NAME,
		"annotation": opts.Desc,
		"ip_config":  ipConfig,
		"vlan_id":    vlanId + 1,
	}
	ret := struct {
		NetworkUUID string
	}{}
	err = self.post("networks", jsonutils.Marshal(params), &ret)
	if err != nil {
		return nil, err
	}
	return self.GetVpc(ret.NetworkUUID)
}

func (self *SRegion) DeleteVpc(id string) error {
	return self.delete("networks", id)
}
