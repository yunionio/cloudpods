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
	"context"
	"time"

	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type usedAddressQueryArgs struct {
	network  *SNetwork
	userCred mcclient.TokenCredential
	owner    mcclient.IIdentityProvider
	scope    rbacscope.TRbacScope
	addrOnly bool
	addrType api.TAddressType
}

type usedAddressQueryProvider interface {
	KeywordPlural() string
	usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery
}

func getUsedAddressQueryProviders() []usedAddressQueryProvider {
	return []usedAddressQueryProvider{
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
}

func getUsedAddress6QueryProviders() []usedAddressQueryProvider {
	return []usedAddressQueryProvider{
		GuestnetworkManager,
		ReservedipManager,
	}
}

func (manager *SGuestnetworkManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = GuestnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	field := "ip_addr"
	if args.addrType == api.AddressTypeIPv6 {
		field = "ip6_addr"
	}
	if args.addrOnly {
		retq = baseq.Query(baseq.Field(field))
	} else {
		var fields []sqlchemy.IQueryField = []sqlchemy.IQueryField{
			baseq.Field(field),
		}
		ownerq := GuestManager.FilterByOwner(ctx, GuestManager.Query(), GuestManager, args.userCred, args.owner, args.scope).SubQuery()
		fields = append(fields,
			baseq.Field("mac_addr"),
			sqlchemy.NewStringField(GuestManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		)
		retq = baseq.Query(fields...).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("guest_id"),
			),
		)
	}
	retq = retq.IsNotEmpty(field)
	return retq
}

func (manager *SHostnetworkManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = HostnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := HostManager.FilterByOwner(ctx, HostManager.Query(), HostManager, args.userCred, args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			baseq.Field("mac_addr"),
			sqlchemy.NewStringField(HostManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
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

func (manager *SReservedipManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = ReservedipManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	field := "ip_addr"
	if args.addrType == api.AddressTypeIPv6 {
		field = "ip6_addr"
	}
	if args.addrOnly {
		retq = baseq.Query(baseq.Field(field))
	} else {
		var fields []sqlchemy.IQueryField = []sqlchemy.IQueryField{
			baseq.Field(field),
		}
		fields = append(fields,
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ReservedipManager.KeywordPlural()).Label("owner_type"),
			sqlchemy.CASTString(baseq.Field("id"), "owner_id"),
			baseq.Field("status").Label("owner_status"),
			baseq.Field("notes").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		)
		retq = baseq.Query(fields...)
	}
	retq = retq.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(baseq.Field("expired_at")),
		sqlchemy.GT(baseq.Field("expired_at"), time.Now()),
	))
	retq = retq.IsNotEmpty(field)
	return retq
}

func (manager *SGroupnetworkManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = GroupnetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		var fields []sqlchemy.IQueryField
		if args.addrType == api.AddressTypeIPv6 {
			fields = append(fields, baseq.Field("ip6_addr"))
		} else {
			fields = append(fields, baseq.Field("ip_addr"))
		}
		retq = baseq.Query(fields...)
	} else {
		var fields []sqlchemy.IQueryField
		if args.addrType == api.AddressTypeIPv6 {
			fields = append(fields, baseq.Field("ip6_addr"))
		} else {
			fields = append(fields, baseq.Field("ip_addr"))
		}
		ownerq := GroupManager.FilterByOwner(ctx, GroupManager.Query(), GroupManager, args.userCred, args.owner, args.scope).SubQuery()
		fields = append(fields,
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(GroupManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
			ownerq.Field("name").Label("owner"),
			sqlchemy.NewStringField("").Label("associate_id"),
			sqlchemy.NewStringField("").Label("associate_type"),
			baseq.Field("created_at"),
		)
		retq = baseq.Query(fields...).LeftJoin(
			ownerq,
			sqlchemy.Equals(
				ownerq.Field("id"),
				baseq.Field("group_id"),
			),
		)
	}
	return retq
}

func (manager *SLoadbalancernetworkManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = LoadbalancernetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := LoadbalancerManager.FilterByOwner(ctx, LoadbalancerManager.Query(), LoadbalancerManager, args.userCred, args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(LoadbalancerManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
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

func (manager *SElasticipManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = ElasticipManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := ElasticipManager.FilterByOwner(ctx, ElasticipManager.Query().Equals("network_id", args.network.Id), ElasticipManager, args.userCred, args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(ElasticipManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
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

func (manager *SNetworkinterfacenetworkManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = NetworkinterfacenetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := NetworkInterfaceManager.FilterByOwner(ctx, NetworkInterfaceManager.Query(), NetworkInterfaceManager, args.userCred, args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			ownerq.Field("mac").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkInterfaceManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
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

func (manager *SDBInstanceManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		baseq = DBInstanceNetworkManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq  *sqlchemy.SQuery
	)
	if args.addrOnly {
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		ownerq := DBInstanceManager.FilterByOwner(ctx, DBInstanceManager.Query(), DBInstanceManager, args.userCred, args.owner, args.scope).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			sqlchemy.NewStringField("").Label("mac_addr"),
			sqlchemy.NewStringField(DBInstanceManager.KeywordPlural()).Label("owner_type"),
			ownerq.Field("id").Label("owner_id"),
			ownerq.Field("status").Label("owner_status"),
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

func (manager *SNetworkAddressManager) usedAddressQuery(ctx context.Context, args *usedAddressQueryArgs) *sqlchemy.SQuery {
	var (
		retq *sqlchemy.SQuery
	)
	if args.addrOnly {
		baseq := NetworkAddressManager.Query().Equals("network_id", args.network.Id).SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
		)
	} else {
		baseq := NetworkAddressManager.Query().Equals("parent_type", api.NetworkAddressParentTypeGuestnetwork).Equals("network_id", args.network.Id).SubQuery()
		guestNetworks := GuestnetworkManager.Query().SubQuery()
		guests := GuestManager.Query().SubQuery()
		retq = baseq.Query(
			baseq.Field("ip_addr"),
			guestNetworks.Field("mac_addr").Label("mac_addr"),
			sqlchemy.NewStringField(NetworkAddressManager.KeywordPlural()).Label("owner_type"),
			baseq.Field("id").Label("owner_id"),
			sqlchemy.NewStringField("available").Label("owner_status"),
			guests.Field("name").Label("owner"),
			baseq.Field("parent_id").Label("associate_id"),
			baseq.Field("parent_type").Label("associate_type"),
			baseq.Field("created_at"),
		).Join(guestNetworks, sqlchemy.Equals(guestNetworks.Field("row_id"), baseq.Field("parent_id"))).Join(guests, sqlchemy.Equals(guests.Field("id"), guestNetworks.Field("guest_id")))
		retq = NetworkAddressManager.FilterByOwner(ctx, retq, NetworkAddressManager, args.userCred, args.owner, args.scope)
	}
	return retq
}
