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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	QcloudTags

	region *SRegion

	iwires []cloudprovider.ICloudWire

	CidrBlock       string
	Ipv6CidrBlock   string
	CreatedTime     time.Time
	DhcpOptionsId   string
	DnsServerSet    []string
	DomainName      string
	EnableMulticast bool
	IsDefault       bool
	VpcId           string
	VpcName         string
}

func (self *SVpc) GetId() string {
	return self.VpcId
}

func (self *SVpc) GetName() string {
	if len(self.VpcName) > 0 {
		return self.VpcName
	}
	return self.VpcId
}

func (self *SVpc) GetGlobalId() string {
	return self.VpcId
}

func (self *SVpc) GetIsDefault() bool {
	return self.IsDefault
}

func (self *SVpc) GetCidrBlock() string {
	return self.CidrBlock
}

func (self *SVpc) GetCidrBlock6() string {
	return self.Ipv6CidrBlock
}

func (self *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (self *SVpc) Delete() error {
	return self.region.DeleteVpc(self.VpcId)
}

func (self *SVpc) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("vpc", "vpc", []string{self.VpcId}, tags, replace)
}

func (self *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (self *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	rts := []cloudprovider.ICloudRouteTable{}
	routetables, err := self.region.GetAllRouteTables(self.GetId(), []string{})
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetAllRouteTables(%s, []string{})", self.GetId())
	}
	for i := range routetables {
		routetables[i].vpc = self
		rts = append(rts, &routetables[i])
	}
	return rts, nil
}

func (self *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	routetables, err := self.region.GetAllRouteTables(self.GetId(), []string{routeTableId})
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.GetAllRouteTables(%s, []string{})", self.GetId())
	}
	if len(routetables) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(routetables) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	routetables[0].vpc = self
	return &routetables[0], nil
}

func (self *SVpc) getWireByZoneId(zoneId string) *SWire {
	for i := 0; i <= len(self.iwires); i++ {
		wire := self.iwires[i].(*SWire)
		if wire.zone.Zone == zoneId {
			return wire
		}
	}
	return nil
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	nats := []SNatGateway{}
	for {
		part, total, err := self.region.GetNatGateways(self.VpcId, len(nats), 50)
		if err != nil {
			return nil, err
		}
		nats = append(nats, part...)
		if len(nats) >= total {
			break
		}
	}
	inats := []cloudprovider.ICloudNatGateway{}
	for i := 0; i < len(nats); i++ {
		nats[i].vpc = self
		inats = append(inats, &nats[i])
	}
	return inats, nil
}

func (self *SVpc) fetchNetworks() error {
	networks, total, err := self.region.GetNetworks(nil, self.VpcId, 0, 50)
	if err != nil {
		return err
	}
	if total > len(networks) {
		networks, _, err = self.region.GetNetworks(nil, self.VpcId, 0, total)
		if err != nil {
			return err
		}
	}
	for i := 0; i < len(networks); i += 1 {
		wire := self.getWireByZoneId(networks[i].Zone)
		networks[i].wire = wire
		wire.addNetwork(&networks[i])
	}
	return nil
}

func (self *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(self.iwires); i += 1 {
		if self.iwires[i].GetGlobalId() == wireId {
			return self.iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if self.iwires == nil {
		err := self.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.iwires, nil
}

func (self *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SVpc) Refresh() error {
	new, err := self.region.getVpc(self.VpcId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SVpc) addWire(wire *SWire) {
	if self.iwires == nil {
		self.iwires = make([]cloudprovider.ICloudWire, 0)
	}
	self.iwires = append(self.iwires, wire)
}

func (self *SVpc) GetAuthorityOwnerId() string {
	return self.region.client.ownerName
}

func (self *SVpc) ProposeJoinICloudInterVpcNetwork(opts *cloudprovider.SVpcJointInterVpcNetworkOption) error {
	instance := SCcnAttachInstanceInput{
		InstanceType:   "VPC",
		InstanceId:     self.GetId(),
		InstanceRegion: self.region.GetId(),
	}
	err := self.region.AttachCcnInstances(opts.InterVpcNetworkId, opts.NetworkAuthorityOwnerId, []SCcnAttachInstanceInput{instance})
	if err != nil {
		return errors.Wrapf(err, "self.region.AttachCcnInstance(%s,%s,%s)", jsonutils.Marshal(opts).String(), self.GetId(), self.region.GetId())
	}
	return nil
}
