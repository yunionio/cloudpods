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
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/apimap"
)

type ModelSetsMaxUpdatedAt struct {
	Vpcs               time.Time
	Wires              time.Time
	Networks           time.Time
	Guests             time.Time
	Hosts              time.Time
	SecurityGroups     time.Time
	SecurityGroupRules time.Time
	Guestnetworks      time.Time
	Guestsecgroups     time.Time
	Elasticips         time.Time
	NetworkAddresses   time.Time

	DnsZones   time.Time
	DnsRecords time.Time

	RouteTables time.Time

	Groupguests   time.Time
	Groupnetworks time.Time

	LoadbalancerNetworks  time.Time
	LoadbalancerListeners time.Time
	LoadbalancerAcls      time.Time
}

func NewModelSetsMaxUpdatedAt() *ModelSetsMaxUpdatedAt {
	return &ModelSetsMaxUpdatedAt{
		Vpcs:               apihelper.PseudoZeroTime,
		Wires:              apihelper.PseudoZeroTime,
		Networks:           apihelper.PseudoZeroTime,
		Guests:             apihelper.PseudoZeroTime,
		Hosts:              apihelper.PseudoZeroTime,
		SecurityGroups:     apihelper.PseudoZeroTime,
		SecurityGroupRules: apihelper.PseudoZeroTime,
		Guestnetworks:      apihelper.PseudoZeroTime,
		Guestsecgroups:     apihelper.PseudoZeroTime,
		Elasticips:         apihelper.PseudoZeroTime,
		NetworkAddresses:   apihelper.PseudoZeroTime,

		DnsZones:   apihelper.PseudoZeroTime,
		DnsRecords: apihelper.PseudoZeroTime,

		RouteTables: apihelper.PseudoZeroTime,

		Groupguests:   apihelper.PseudoZeroTime,
		Groupnetworks: apihelper.PseudoZeroTime,

		LoadbalancerNetworks:  apihelper.PseudoZeroTime,
		LoadbalancerListeners: apihelper.PseudoZeroTime,
		LoadbalancerAcls:      apihelper.PseudoZeroTime,
	}
}

type ModelSets struct {
	Vpcs               Vpcs
	Wires              Wires
	Networks           Networks
	Guests             Guests
	Hosts              Hosts
	SecurityGroups     SecurityGroups
	SecurityGroupRules SecurityGroupRules
	Guestnetworks      Guestnetworks
	Guestsecgroups     Guestsecgroups
	Elasticips         Elasticips
	NetworkAddresses   NetworkAddresses

	DnsZones   DnsZones
	DnsRecords DnsRecords

	RouteTables RouteTables

	Groupguests   Groupguests
	Groupnetworks Groupnetworks
	Groups        Groups

	LoadbalancerNetworks  LoadbalancerNetworks
	LoadbalancerListeners LoadbalancerListeners
	LoadbalancerAcls      LoadbalancerAcls
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		Vpcs:               Vpcs{},
		Wires:              Wires{},
		Networks:           Networks{},
		Guests:             Guests{},
		Hosts:              Hosts{},
		SecurityGroups:     SecurityGroups{},
		SecurityGroupRules: SecurityGroupRules{},
		Guestnetworks:      Guestnetworks{},
		Guestsecgroups:     Guestsecgroups{},
		Elasticips:         Elasticips{},
		NetworkAddresses:   NetworkAddresses{},

		DnsZones:   DnsZones{},
		DnsRecords: DnsRecords{},

		RouteTables: RouteTables{},

		Groupguests:   Groupguests{},
		Groupnetworks: Groupnetworks{},
		Groups:        Groups{},

		LoadbalancerNetworks:  LoadbalancerNetworks{},
		LoadbalancerListeners: LoadbalancerListeners{},
		LoadbalancerAcls:      LoadbalancerAcls{},
	}
}

func (mss *ModelSets) ModelSetList() []apihelper.IModelSet {
	// it's ordered this way to favour creation, not deletion
	return []apihelper.IModelSet{
		mss.Vpcs,
		mss.Wires,
		mss.Networks,
		mss.Guests,
		mss.Hosts,
		mss.SecurityGroups,
		mss.SecurityGroupRules,
		mss.Guestnetworks,
		mss.Guestsecgroups,
		mss.Elasticips,
		mss.NetworkAddresses,

		mss.DnsZones,
		mss.DnsRecords,

		mss.RouteTables,

		mss.Groupguests,
		mss.Groupnetworks,
		mss.Groups,

		mss.LoadbalancerNetworks,
		mss.LoadbalancerListeners,
		mss.LoadbalancerAcls,
	}
}

func (mss *ModelSets) NewEmpty() apihelper.IModelSets {
	return NewModelSets()
}

func (mss *ModelSets) copy_() *ModelSets {
	mssCopy := &ModelSets{
		Vpcs:               mss.Vpcs.Copy().(Vpcs),
		Wires:              mss.Wires.Copy().(Wires),
		Networks:           mss.Networks.Copy().(Networks),
		Guests:             mss.Guests.Copy().(Guests),
		Hosts:              mss.Hosts.Copy().(Hosts),
		SecurityGroups:     mss.SecurityGroups.Copy().(SecurityGroups),
		SecurityGroupRules: mss.SecurityGroupRules.Copy().(SecurityGroupRules),
		Guestnetworks:      mss.Guestnetworks.Copy().(Guestnetworks),
		Guestsecgroups:     mss.Guestsecgroups.Copy().(Guestsecgroups),
		Elasticips:         mss.Elasticips.Copy().(Elasticips),
		NetworkAddresses:   mss.NetworkAddresses.Copy().(NetworkAddresses),

		DnsZones:   mss.DnsZones.Copy().(DnsZones),
		DnsRecords: mss.DnsRecords.Copy().(DnsRecords),

		RouteTables: mss.RouteTables.Copy().(RouteTables),

		Groupguests:   mss.Groupguests.Copy().(Groupguests),
		Groupnetworks: mss.Groupnetworks.Copy().(Groupnetworks),
		Groups:        mss.Groups.Copy().(Groups),

		LoadbalancerNetworks:  mss.LoadbalancerNetworks.Copy().(LoadbalancerNetworks),
		LoadbalancerListeners: mss.LoadbalancerListeners.Copy().(LoadbalancerListeners),
		LoadbalancerAcls:      mss.LoadbalancerAcls.Copy().(LoadbalancerAcls),
	}
	return mssCopy
}

func (mss *ModelSets) Copy() apihelper.IModelSets {
	return mss.copy_()
}

func (mss *ModelSets) CopyJoined() apihelper.IModelSets {
	mssCopy := mss.copy_()
	mssCopy.join()
	return mssCopy
}

func (mss *ModelSets) ApplyUpdates(mssNews apihelper.IModelSets) apihelper.ModelSetsUpdateResult {
	r := apihelper.ModelSetsUpdateResult{
		Changed: false,
		Correct: true,
	}
	mssList := mss.ModelSetList()
	mssNewsList := mssNews.ModelSetList()
	for i, mss := range mssList {
		mssNews := mssNewsList[i]
		msR := apihelper.ModelSetApplyUpdates(mss, mssNews)
		if !r.Changed && msR.Changed {
			r.Changed = true
		}
	}
	if r.Changed {
		r.Correct = mss.join()
	}
	return r
}

func (mss *ModelSets) FetchFromAPIMap(s *mcclient.ClientSession) (apihelper.IModelSets, error) {
	mssNews := mss.NewEmpty()
	ret, err := apimap.APIMap.GetVPCAgentTopo(s)
	if err != nil {
		return nil, errors.Wrap(err, "GetVPCAgentTopo")
	}
	if err := ret.Unmarshal(mssNews, "models"); err != nil {
		return nil, errors.Wrap(err, "Unmarshal topo")
	}
	return mssNews, nil
}

func (mss *ModelSets) join() bool {
	mss.Guests.initJoin()
	mss.Groups = Groups{}
	var p []bool
	var msg []string
	p = append(p, mss.Vpcs.joinWires(mss.Wires))
	msg = append(msg, "mss.Vpcs.joinWires(mss.Wires)")
	p = append(p, mss.Vpcs.joinRouteTables(mss.RouteTables))
	msg = append(msg, "mss.Vpcs.joinRouteTables(mss.RouteTables)")
	p = append(p, mss.Wires.joinNetworks(mss.Networks))
	msg = append(msg, "mss.Wires.joinNetworks(mss.Networks)")
	p = append(p, mss.Vpcs.joinNetworks(mss.Networks))
	msg = append(msg, "mss.Vpcs.joinNetworks(mss.Networks)")
	p = append(p, mss.Networks.joinGuestnetworks(mss.Guestnetworks))
	msg = append(msg, "mss.Networks.joinGuestnetworks(mss.Guestnetworks)")
	p = append(p, mss.Networks.joinNetworkAddresses(mss.NetworkAddresses))
	msg = append(msg, "mss.Networks.joinNetworkAddresses(mss.NetworkAddresses)")
	p = append(p, mss.Networks.joinLoadbalancerNetworks(mss.LoadbalancerNetworks))
	msg = append(msg, "mss.Networks.joinLoadbalancerNetworks(mss.LoadbalancerNetworks)")
	p = append(p, mss.Networks.joinElasticips(mss.Elasticips))
	msg = append(msg, "mss.Networks.joinElasticips(mss.Elasticips)")
	p = append(p, mss.Guests.joinHosts(mss.Hosts))
	msg = append(msg, "mss.Guests.joinHosts(mss.Hosts)")
	p = append(p, mss.Guests.joinSecurityGroups(mss.SecurityGroups))
	msg = append(msg, "mss.Guests.joinSecurityGroups(mss.SecurityGroups)")
	p = append(p, mss.Guests.joinGroupguests(mss.Groups, mss.Groupguests))
	msg = append(msg, "mss.Guests.joinGroupguests(mss.Groups, mss.Groupguests)")
	p = append(p, mss.SecurityGroups.joinSecurityGroupRules(mss.SecurityGroupRules))
	msg = append(msg, "mss.SecurityGroups.joinSecurityGroupRules(mss.SecurityGroupRules)")
	p = append(p, mss.Guestsecgroups.join(mss.SecurityGroups, mss.Guests))
	msg = append(msg, "mss.Guestsecgroups.join(mss.SecurityGroups, mss.Guests)")
	p = append(p, mss.Guestnetworks.joinGuests(mss.Guests))
	msg = append(msg, "mss.Guestnetworks.joinGuests(mss.Guests)")
	p = append(p, mss.Guestnetworks.joinElasticips(mss.Elasticips))
	msg = append(msg, "mss.Guestnetworks.joinElasticips(mss.Elasticips)")
	p = append(p, mss.Guestnetworks.joinNetworkAddresses(mss.NetworkAddresses))
	msg = append(msg, "mss.Guestnetworks.joinNetworkAddresses(mss.NetworkAddresses)")
	p = append(p, mss.Groups.joinGroupnetworks(mss.Groupnetworks, mss.Networks))
	msg = append(msg, "mss.Groups.joinGroupnetworks(mss.Groupnetworks, mss.Networks)")
	p = append(p, mss.Groupnetworks.joinElasticips(mss.Elasticips))
	msg = append(msg, "mss.Groupnetworks.joinElasticips(mss.Elasticips)")
	p = append(p, mss.LoadbalancerNetworks.joinElasticips(mss.Elasticips))
	msg = append(msg, "mss.LoadbalancerNetworks.joinElasticips(mss.Elasticips)")
	p = append(p, mss.LoadbalancerNetworks.joinLoadbalancerListeners(mss.LoadbalancerListeners))
	msg = append(msg, "mss.LoadbalancerNetworks.joinLoadbalancerListeners(mss.LoadbalancerListeners)")
	p = append(p, mss.LoadbalancerListeners.joinLoadbalancerAcls(mss.LoadbalancerAcls))
	msg = append(msg, "mss.LoadbalancerListeners.joinLoadbalancerAcls(mss.LoadbalancerAcls)")
	p = append(p, mss.DnsZones.joinRecords(mss.DnsRecords))
	msg = append(msg, "mss.Vpcs.joinRecords(mss.DnsRecords)")
	ret := true
	var failMsg []string
	for i, b := range p {
		if !b {
			ret = false
			failMsg = append(failMsg, msg[i])
		}
	}
	if !ret {
		log.Errorln(strings.Join(failMsg, ","))
	} else {
		for _, g := range mss.Guests {
			g.FixIsDefaults()
		}
	}
	return ret
}
