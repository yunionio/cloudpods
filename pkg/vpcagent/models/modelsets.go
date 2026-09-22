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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/apimap"
)

type ModelSetsMaxUpdatedAt struct {
	Vpcs                  time.Time
	Wires                 time.Time
	Networks              time.Time
	Guests                time.Time
	Hosts                 time.Time
	SecurityGroups        time.Time
	SecurityGroupRules    time.Time
	Guestnetworks         time.Time
	Guestsecgroups        time.Time
	Elasticips            time.Time
	NetworkAddresses      time.Time
	Guestnetworksecgroups time.Time

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
		Vpcs:                  apihelper.PseudoZeroTime,
		Wires:                 apihelper.PseudoZeroTime,
		Networks:              apihelper.PseudoZeroTime,
		Guests:                apihelper.PseudoZeroTime,
		Hosts:                 apihelper.PseudoZeroTime,
		SecurityGroups:        apihelper.PseudoZeroTime,
		SecurityGroupRules:    apihelper.PseudoZeroTime,
		Guestnetworks:         apihelper.PseudoZeroTime,
		Guestsecgroups:        apihelper.PseudoZeroTime,
		Elasticips:            apihelper.PseudoZeroTime,
		NetworkAddresses:      apihelper.PseudoZeroTime,
		Guestnetworksecgroups: apihelper.PseudoZeroTime,

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

// The json tags below pin the wire format.  ModelSets is the payload of the
// apimap /modelsets API, and its keys are derived from the field names when no
// tag is set.  A rename would silently change the key, and Unmarshal drops an
// unmatched key without an error, so a whole model set would go missing
// unnoticed.  Keep these tags in sync with expectedModelSetKeys in
// modelsets_wire_test.go.
type ModelSetsStats struct {
	Vpcs                  int `json:"vpcs"`
	Wires                 int `json:"wires"`
	Networks              int `json:"networks"`
	Guests                int `json:"guests"`
	Hosts                 int `json:"hosts"`
	SecurityGroups        int `json:"security_groups"`
	SecurityGroupRules    int `json:"security_group_rules"`
	Guestnetworks         int `json:"guestnetworks"`
	Guestsecgroups        int `json:"guestsecgroups"`
	Elasticips            int `json:"elasticips"`
	NetworkAddresses      int `json:"network_addresses"`
	Guestnetworksecgroups int `json:"guestnetworksecgroups"`

	DnsZones   int `json:"dns_zones"`
	DnsRecords int `json:"dns_records"`

	RouteTables int `json:"route_tables"`

	Groupguests   int `json:"groupguests"`
	Groupnetworks int `json:"groupnetworks"`
	Groups        int `json:"groups"`

	LoadbalancerNetworks  int `json:"loadbalancer_networks"`
	LoadbalancerListeners int `json:"loadbalancer_listeners"`
	LoadbalancerAcls      int `json:"loadbalancer_acls"`
}

type ModelSets struct {
	Vpcs                  Vpcs                  `json:"vpcs"`
	Wires                 Wires                 `json:"wires"`
	Networks              Networks              `json:"networks"`
	Guests                Guests                `json:"guests"`
	Hosts                 Hosts                 `json:"hosts"`
	SecurityGroups        SecurityGroups        `json:"security_groups"`
	SecurityGroupRules    SecurityGroupRules    `json:"security_group_rules"`
	Guestnetworks         Guestnetworks         `json:"guestnetworks"`
	Guestsecgroups        Guestsecgroups        `json:"guestsecgroups"`
	Elasticips            Elasticips            `json:"elasticips"`
	NetworkAddresses      NetworkAddresses      `json:"network_addresses"`
	Guestnetworksecgroups Guestnetworksecgroups `json:"guestnetworksecgroups"`

	DnsZones   DnsZones   `json:"dns_zones"`
	DnsRecords DnsRecords `json:"dns_records"`

	RouteTables RouteTables `json:"route_tables"`

	Groupguests   Groupguests   `json:"groupguests"`
	Groupnetworks Groupnetworks `json:"groupnetworks"`
	Groups        Groups        `json:"groups"`

	LoadbalancerNetworks  LoadbalancerNetworks  `json:"loadbalancer_networks"`
	LoadbalancerListeners LoadbalancerListeners `json:"loadbalancer_listeners"`
	LoadbalancerAcls      LoadbalancerAcls      `json:"loadbalancer_acls"`
}

func NewModelSets() *ModelSets {
	return &ModelSets{
		Vpcs:                  Vpcs{},
		Wires:                 Wires{},
		Networks:              Networks{},
		Guests:                Guests{},
		Hosts:                 Hosts{},
		SecurityGroups:        SecurityGroups{},
		SecurityGroupRules:    SecurityGroupRules{},
		Guestnetworks:         Guestnetworks{},
		Guestsecgroups:        Guestsecgroups{},
		Elasticips:            Elasticips{},
		NetworkAddresses:      NetworkAddresses{},
		Guestnetworksecgroups: Guestnetworksecgroups{},

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
		mss.Guestnetworksecgroups,

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
		Vpcs:                  mss.Vpcs.Copy().(Vpcs),
		Wires:                 mss.Wires.Copy().(Wires),
		Networks:              mss.Networks.Copy().(Networks),
		Guests:                mss.Guests.Copy().(Guests),
		Hosts:                 mss.Hosts.Copy().(Hosts),
		SecurityGroups:        mss.SecurityGroups.Copy().(SecurityGroups),
		SecurityGroupRules:    mss.SecurityGroupRules.Copy().(SecurityGroupRules),
		Guestnetworks:         mss.Guestnetworks.Copy().(Guestnetworks),
		Guestsecgroups:        mss.Guestsecgroups.Copy().(Guestsecgroups),
		Elasticips:            mss.Elasticips.Copy().(Elasticips),
		NetworkAddresses:      mss.NetworkAddresses.Copy().(NetworkAddresses),
		Guestnetworksecgroups: mss.Guestnetworksecgroups.Copy().(Guestnetworksecgroups),

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

var (
	_ apihelper.IModelSets       = (*ModelSets)(nil)
	_ apihelper.IAPIMapModelSets = (*ModelSets)(nil)
)

// APIMapTimestamp returns the version of the model sets currently served by
// the apimap service.
func (mss *ModelSets) APIMapTimestamp(s *mcclient.ClientSession) (int64, error) {
	return apimap.APIMap.GetModelSetsTimestamp(s)
}

// FetchFromAPIMap fetches the model sets from the apimap service, along with
// the version of the payload.  The payload is flat: the join links between
// models are all tagged json:"-", so they are rebuilt by ApplyUpdates and
// join() rather than carried over the wire.
func (mss *ModelSets) FetchFromAPIMap(s *mcclient.ClientSession) (apihelper.IModelSets, int64, error) {
	ret, ts, err := apimap.APIMap.GetModelSets(s)
	if err != nil {
		return nil, 0, errors.Wrap(err, "GetModelSets")
	}
	mssNews := mss.NewEmpty()
	if err := ret.Unmarshal(mssNews, "models"); err != nil {
		return nil, 0, errors.Wrap(err, "unmarshal models")
	}
	return mssNews, ts, nil
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
	p = append(p, mss.Guestnetworksecgroups.joinSecurityGroups(mss.SecurityGroups))
	msg = append(msg, "mss.Guestnetworksecgroups.joinSecurityGroups(mss.SecurityGroups)")
	p = append(p, mss.Guestnetworks.joinGuests(mss.Guests))
	msg = append(msg, "mss.Guestnetworks.joinGuests(mss.Guests)")
	p = append(p, mss.Guestnetworks.joinElasticips(mss.Elasticips))
	msg = append(msg, "mss.Guestnetworks.joinElasticips(mss.Elasticips)")
	p = append(p, mss.Guestnetworks.joinNetworkAddresses(mss.NetworkAddresses))
	msg = append(msg, "mss.Guestnetworks.joinNetworkAddresses(mss.NetworkAddresses)")
	p = append(p, mss.Guestnetworks.joinGuestnetworksecgroups(mss.Guestnetworksecgroups))
	msg = append(msg, "mss.Guestnetworks.joinGuestnetworksecgroups(mss.Guestnetworksecgroups)")
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

func (mss ModelSets) Stats() ModelSetsStats {
	return ModelSetsStats{
		Vpcs:                  len(mss.Vpcs),
		Wires:                 len(mss.Wires),
		Networks:              len(mss.Networks),
		Guests:                len(mss.Guests),
		Hosts:                 len(mss.Hosts),
		SecurityGroups:        len(mss.SecurityGroups),
		SecurityGroupRules:    len(mss.SecurityGroupRules),
		Guestnetworks:         len(mss.Guestnetworks),
		Guestsecgroups:        len(mss.Guestsecgroups),
		Elasticips:            len(mss.Elasticips),
		NetworkAddresses:      len(mss.NetworkAddresses),
		Guestnetworksecgroups: len(mss.Guestnetworksecgroups),

		DnsZones:   len(mss.DnsZones),
		DnsRecords: len(mss.DnsRecords),

		RouteTables: len(mss.RouteTables),

		Groupguests:   len(mss.Groupguests),
		Groupnetworks: len(mss.Groupnetworks),
		Groups:        len(mss.Groups),

		LoadbalancerNetworks:  len(mss.LoadbalancerNetworks),
		LoadbalancerListeners: len(mss.LoadbalancerListeners),
		LoadbalancerAcls:      len(mss.LoadbalancerAcls),
	}
}

// Total is the number of models in all the model sets, which is what a worker
// that pushes them out compares batches by.
func (msss ModelSetsStats) Total() int {
	return msss.Vpcs + msss.Wires + msss.Networks + msss.Guests + msss.Hosts +
		msss.SecurityGroups + msss.SecurityGroupRules + msss.Guestnetworks +
		msss.Guestsecgroups + msss.Elasticips + msss.NetworkAddresses +
		msss.Guestnetworksecgroups + msss.DnsZones + msss.DnsRecords +
		msss.RouteTables + msss.Groupguests + msss.Groupnetworks + msss.Groups +
		msss.LoadbalancerNetworks + msss.LoadbalancerListeners + msss.LoadbalancerAcls
}

func (msss ModelSetsStats) Dump() string {
	return jsonutils.Marshal(msss).String()
}
