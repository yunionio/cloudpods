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
	"time"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type usedAddressQueryArgs struct {
	network  *SNetwork
	owner    mcclient.IIdentityProvider
	scope    rbacutils.TRbacScope
	addrOnly bool
}

type usedAddressQueryProvider interface {
	usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery
}

var usedAddressQueryProviders = []usedAddressQueryProvider{
	GuestnetworkManager,
	HostnetworkManager,
	ReservedipManager,
	GroupnetworkManager,
	LoadbalancernetworkManager,
	ElasticipManager,
	NetworkinterfacenetworkManager,
	DBInstanceManager,
}

func (manager *SGuestnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	guestnetworks := GuestnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var guestNetQ *sqlchemy.SQuery
	if args.addrOnly {
		guestNetQ = guestnetworks.Query(
			guestnetworks.Field("ip_addr"),
		)
	} else {
		guests := GuestManager.FilterByOwner(GuestManager.Query(), args.owner, args.scope).SubQuery()
		guestNetQ = guestnetworks.Query(
			guestnetworks.Field("ip_addr"),
			guestnetworks.Field("mac_addr"),
			sqlchemy.NewStringField(GuestManager.KeywordPlural()).Label("owner_type"),
			guests.Field("id").Label("owner_id"),
			guests.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			guestnetworks.Field("created_at"),
		).LeftJoin(
			guests,
			sqlchemy.Equals(
				guests.Field("id"),
				guestnetworks.Field("guest_id"),
			),
		)
	}
	return guestNetQ
}

func (manager *SHostnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	hostnetworks := HostnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var hostNetQ *sqlchemy.SQuery
	if args.addrOnly {
		hostNetQ = hostnetworks.Query(
			hostnetworks.Field("ip_addr"),
		)
	} else {
		hosts := HostManager.FilterByOwner(HostManager.Query(), args.owner, args.scope).SubQuery()
		hostNetQ = hostnetworks.Query(
			hostnetworks.Field("ip_addr"),
			hostnetworks.Field("mac_addr"),
			sqlchemy.NewStringField(HostManager.KeywordPlural()).Label("owner_type"),
			hosts.Field("id").Label("owner_id"),
			hosts.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			hostnetworks.Field("created_at"),
		).LeftJoin(
			hosts,
			sqlchemy.Equals(
				hosts.Field("id"),
				hostnetworks.Field("baremetal_id"),
			),
		)
	}
	return hostNetQ
}

func (manager *SReservedipManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	reserved := ReservedipManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var reservedQ *sqlchemy.SQuery
	if args.addrOnly {
		reservedQ = reserved.Query(
			reserved.Field("ip_addr"),
		)
	} else {
		reservedQ = reserved.Query(
			reserved.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ReservedipManager.KeywordPlural()).Label("owner_type"),
			reserved.Field("id").Label("owner_id"),
			reserved.Field("notes").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			reserved.Field("created_at"),
		)
	}
	reservedQ = reservedQ.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(reserved.Field("expired_at")),
		sqlchemy.GT(reserved.Field("expired_at"), time.Now()),
	))
	return reservedQ
}

func (manager *SGroupnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	groupnetworks := GroupnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var groupNetQ *sqlchemy.SQuery
	if args.addrOnly {
		groupNetQ = groupnetworks.Query(
			groupnetworks.Field("ip_addr"),
		)
	} else {
		groups := GroupManager.FilterByOwner(GroupManager.Query(), args.owner, args.scope).SubQuery()
		groupNetQ = groupnetworks.Query(
			groupnetworks.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(GroupManager.KeywordPlural()).Label("owner_type"),
			groups.Field("id").Label("owner_id"),
			groups.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			groupnetworks.Field("created_at"),
		).LeftJoin(
			groups,
			sqlchemy.Equals(
				groups.Field("id"),
				groupnetworks.Field("group_id"),
			),
		)
	}
	return groupNetQ
}

func (manager *SLoadbalancernetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	lbnetworks := LoadbalancernetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var lbNetQ *sqlchemy.SQuery
	if args.addrOnly {
		lbNetQ = lbnetworks.Query(
			lbnetworks.Field("ip_addr"),
		)
	} else {
		loadbalancers := LoadbalancerManager.FilterByOwner(LoadbalancerManager.Query(), args.owner, args.scope).SubQuery()
		lbNetQ = lbnetworks.Query(
			lbnetworks.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(LoadbalancerManager.KeywordPlural()).Label("owner_type"),
			loadbalancers.Field("id").Label("owner_id"),
			loadbalancers.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			lbnetworks.Field("created_at"),
		).LeftJoin(
			loadbalancers,
			sqlchemy.Equals(
				loadbalancers.Field("id"),
				lbnetworks.Field("loadbalancer_id"),
			),
		)
	}
	return lbNetQ
}

func (manager *SElasticipManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	elasticips := ElasticipManager.Query().Equals("network_id", args.network.Id).SubQuery()
	ownerEips := ElasticipManager.FilterByOwner(ElasticipManager.Query().Equals("network_id", args.network.Id), args.owner, args.scope).SubQuery()
	var eipQ *sqlchemy.SQuery
	if args.addrOnly {
		eipQ = elasticips.Query(
			elasticips.Field("ip_addr"),
		)
	} else {
		eipQ = elasticips.Query(
			elasticips.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ElasticipManager.KeywordPlural()).Label("owner_type"),
			ownerEips.Field("id").Label("owner_id"),
			ownerEips.Field("name").Label("owner"),
			ownerEips.Field("associate_id"),
			ownerEips.Field("associate_type"),
			elasticips.Field("created_at"),
		).LeftJoin(
			ownerEips,
			sqlchemy.Equals(
				elasticips.Field("id"),
				ownerEips.Field("id"),
			),
		)
	}
	return eipQ
}

func (manager *SNetworkinterfacenetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	netifnetworks := NetworkinterfacenetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var netifsQ *sqlchemy.SQuery
	if args.addrOnly {
		netifsQ = netifnetworks.Query(
			netifnetworks.Field("ip_addr"),
		)
	} else {
		netifs := NetworkInterfaceManager.FilterByOwner(NetworkInterfaceManager.Query(), args.owner, args.scope).SubQuery()
		netifsQ = netifnetworks.Query(
			netifnetworks.Field("ip_addr"),
			netifs.Field("mac").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkInterfaceManager.KeywordPlural()).Label("owner_type"),
			netifs.Field("id").Label("owner_id"),
			netifs.Field("name").Label("owner"),
			netifs.Field("associate_id"),
			netifs.Field("associate_type"),
			netifnetworks.Field("created_at"),
		).LeftJoin(
			netifs,
			sqlchemy.Equals(
				netifnetworks.Field("networkinterface_id"),
				netifs.Field("id"),
			),
		)
	}
	return netifsQ
}

func (manager *SDBInstanceManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	dbnetworks := DBInstanceNetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
	var dbNetQ *sqlchemy.SQuery
	if args.addrOnly {
		dbNetQ = dbnetworks.Query(
			dbnetworks.Field("ip_addr"),
		)
	} else {
		dbinstances := DBInstanceManager.FilterByOwner(DBInstanceManager.Query(), args.owner, args.scope).SubQuery()
		dbNetQ = dbnetworks.Query(
			dbnetworks.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(DBInstanceManager.KeywordPlural()).Label("owner_type"),
			dbinstances.Field("id").Label("owner_id"),
			dbinstances.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			dbnetworks.Field("created_at"),
		).LeftJoin(
			dbinstances,
			sqlchemy.Equals(
				dbinstances.Field("id"),
				dbnetworks.Field("dbinstance_id"),
			),
		)
	}
	return dbNetQ
}
