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
	NetworkAddressManager,
}

func (manager *SGuestnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = GuestnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := GuestManager.FilterByOwner(GuestManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			baseq.Field("mac_addr"),
			sqlchemy.NewStringField(GuestManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("guest_id"),
			),
		)
	}
	return retq
}

func (manager *SHostnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = HostnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := HostManager.FilterByOwner(HostManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			baseq.Field("mac_addr"),
			sqlchemy.NewStringField(HostManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("baremetal_id"),
			),
		)
	}
	return retq
}

func (manager *SReservedipManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = ReservedipManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ReservedipManager.KeywordPlural()).Label("owner_type"),
			baseq.Field("id").Label("owner_id"),
			baseq.Field("notes").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		)
	}
	retq = retq.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(baseq.Field("expired_at")),
		sqlchemy.GT(baseq.Field("expired_at"), time.Now()),
	))
	return retq
}

func (manager *SGroupnetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = GroupnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := GroupManager.FilterByOwner(GroupManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(GroupManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("group_id"),
			),
		)
	}
	return retq
}

func (manager *SLoadbalancernetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = LoadbalancernetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := LoadbalancerManager.FilterByOwner(LoadbalancerManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(LoadbalancerManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("loadbalancer_id"),
			),
		)
	}
	return retq
}

func (manager *SElasticipManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = ElasticipManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := ElasticipManager.FilterByOwner(ElasticipManager.Query().Equals("network_id", args.network.Id), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ElasticipManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			ownerq.Field("associate_id"),
			ownerq.Field("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				baseq.Field("id"),
				ownerq.Field("id"),
			),
		)
	}
	return retq
}

func (manager *SNetworkinterfacenetworkManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = NetworkinterfacenetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := NetworkInterfaceManager.FilterByOwner(NetworkInterfaceManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			ownerq.Field("mac").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkInterfaceManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			ownerq.Field("associate_id"),
			ownerq.Field("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				baseq.Field("networkinterface_id"),
				ownerq.Field("id"),
			),
		)
	}
	return retq
}

func (manager *SDBInstanceManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = DBInstanceNetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := DBInstanceManager.FilterByOwner(DBInstanceManager.Query(), args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(DBInstanceManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("dbinstance_id"),
			),
		)
	}
	return retq
}

func (manager *SNetworkAddressManager) usedAddressQuery(args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = NetworkAddressManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkAddressManager.KeywordPlural()).Label("owner_type"),
			baseq.Field("id").Label("owner_id"),
			baseq.Field("id").Label("owner"),
			baseq.Field("parent_id").Label("associate_id"),
			baseq.Field("parent_type").Label("associate_type"),
			baseq.Field("created_at"),
		)
		retq = NetworkAddressManager.FilterByOwner(retq, args.owner, args.scope)
	}
	return retq
}
