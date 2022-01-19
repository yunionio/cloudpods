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

package models

import (
	"fmt"

	"yunion.io/x/log"

	compute_models "yunion.io/x/onecloud/pkg/compute/models"
)

type Vpc struct {
	compute_models.SVpc

	RouteTable *RouteTable `json:"-"`

	Wire     *Wire    `json:"-"`
	Networks Networks `json:"-"`
}

func (el *Vpc) Copy() *Vpc {
	return &Vpc{
		SVpc: el.SVpc,
	}
}

type RouteTable struct {
	compute_models.SRouteTable

	Vpc *Vpc
}

func (el *RouteTable) Copy() *RouteTable {
	return &RouteTable{
		SRouteTable: el.SRouteTable,
	}
}

type Wire struct {
	compute_models.SWire

	Vpc *Vpc
}

func (el *Wire) Copy() *Wire {
	return &Wire{
		SWire: el.SWire,
	}
}

type Network struct {
	compute_models.SNetwork

	Vpc           *Vpc          `json:"-"`
	Wire          *Wire         `json:"-"`
	Guestnetworks Guestnetworks `json:"-"`
	Groupnetworks Groupnetworks `json:"-"`
	Elasticips    Elasticips    `json:"-"`
}

func (el *Network) Copy() *Network {
	return &Network{
		SNetwork: el.SNetwork,
	}
}

type Guestnetwork struct {
	compute_models.SGuestnetwork

	// Guest could be nil for when the guest is pending_deleted
	Guest     *Guest           `json:"-"`
	Network   *Network         `json:"-"`
	Elasticip *Elasticip       `json:"-"`
	SubIPs    NetworkAddresses `json:"-"`
}

func (el *Guestnetwork) Copy() *Guestnetwork {
	return &Guestnetwork{
		SGuestnetwork: el.SGuestnetwork,
	}
}

type NetworkAddress struct {
	compute_models.SNetworkAddress

	Guestnetwork *Guestnetwork `json:"-"`
	Network      *Network      `json:"-"`
}

func (el *NetworkAddress) Copy() *NetworkAddress {
	return &NetworkAddress{
		SNetworkAddress: el.SNetworkAddress,
	}
}

type Guest struct {
	compute_models.SGuest

	Host               *Host          `json:"-"`
	AdminSecurityGroup *SecurityGroup `json:"-"`
	SecurityGroups     SecurityGroups `json:"-"`
	Guestnetworks      Guestnetworks  `json:"-"`

	Groups Groups `json:"-"`
}

func (el *Guest) Copy() *Guest {
	return &Guest{
		SGuest: el.SGuest,
	}
}

func (el *Guest) GetVips() []string {
	ret := make([]string, 0)
	for _, g := range el.Groups {
		for _, gn := range g.Groupnetworks {
			ret = append(ret, fmt.Sprintf("%s/%d", gn.IpAddr, gn.Network.GuestIpMask))
		}
	}
	return ret
}

type Host struct {
	compute_models.SHost
}

func (el *Host) Copy() *Host {
	return &Host{
		SHost: el.SHost,
	}
}

type Guestsecgroup struct {
	compute_models.SGuestsecgroup

	Guest         *Guest         `json:"-"`
	SecurityGroup *SecurityGroup `json:"-"`
}

func (el *Guestsecgroup) ModelSetKey() string {
	return el.GuestId + "/" + el.SecgroupId
}

func (el *Guestsecgroup) Copy() *Guestsecgroup {
	return &Guestsecgroup{
		SGuestsecgroup: el.SGuestsecgroup,
	}
}

type SecurityGroup struct {
	compute_models.SSecurityGroup

	SecurityGroupRules SecurityGroupRules `json:"-"`
}

func (el *SecurityGroup) Copy() *SecurityGroup {
	return &SecurityGroup{
		SSecurityGroup: el.SSecurityGroup,
	}
}

type SecurityGroupRule struct {
	compute_models.SSecurityGroupRule

	SecurityGroup *SecurityGroup `json:"-"`
}

func (el *SecurityGroupRule) Copy() *SecurityGroupRule {
	return &SecurityGroupRule{
		SSecurityGroupRule: el.SSecurityGroupRule,
	}
}

type Elasticip struct {
	compute_models.SElasticip

	Network      *Network      `json:"-"`
	Guestnetwork *Guestnetwork `json:"-"`
	Groupnetwork *Groupnetwork `json:"-"`
}

func (el *Elasticip) Copy() *Elasticip {
	return &Elasticip{
		SElasticip: el.SElasticip,
	}
}

type DnsRecord struct {
	compute_models.SDnsRecord
}

func (el *DnsRecord) Copy() *DnsRecord {
	return &DnsRecord{
		SDnsRecord: el.SDnsRecord,
	}
}

type Groupguest struct {
	compute_models.SGroupguest

	Group *Group `json:"-"`
	Guest *Guest `json:"-"`
}

func (el *Groupguest) Copy() *Groupguest {
	return &Groupguest{
		SGroupguest: el.SGroupguest,
	}
}

type Groupnetwork struct {
	compute_models.SGroupnetwork

	Network *Network `json:"-"`
	Group   *Group   `json:"-"`

	Elasticip *Elasticip `json:"-"`
}

func (el *Groupnetwork) Copy() *Groupnetwork {
	return &Groupnetwork{
		SGroupnetwork: el.SGroupnetwork,
	}
}

func (el *Groupnetwork) GetGuestNetworks() []*Guestnetwork {
	ret := make([]*Guestnetwork, 0)
	if el.Group == nil {
		log.Errorf("Nil group for groupnetwork %s %s", el.GroupId, el.NetworkId)
	} else if el.Group.Groupguests == nil {
		log.Errorf("Nil groupguests for group %s", el.GroupId)
	} else {
		for _, gg := range el.Group.Groupguests {
			if gg.Guest == nil {
				log.Errorf("Nil guest for groupguest %s %s", gg.GroupId, gg.GuestId)
			} else if gg.Guest.Guestnetworks == nil {
				log.Errorf("Nil guestnetworks for guest %s", gg.GuestId)
			} else {
				for _, gn := range gg.Guest.Guestnetworks {
					ret = append(ret, gn)
				}
			}
		}
	}
	return ret
}

type Group struct {
	compute_models.SGroup

	Groupguests   Groupguests   `json:"-"`
	Groupnetworks Groupnetworks `json:"-"`
}

func (el *Group) Copy() *Group {
	return &Group{
		SGroup: el.SGroup,
	}
}
