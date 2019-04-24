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

package db

import (
	"time"

	u "yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/cache"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

const (
	CacheKind = "DBCache"

	StorageDBCache = "Storages"
	WireDBCache    = "Wires"

	GroupDBCache      = "Groups"
	GroupGuestDBCache = "Groupguests"
	GroupHostDBCache  = "Grouphosts"
	HostDBCache       = "Hosts"

	ClusterDBCache     = "Clusters"
	ClusterHostDBCache = "Clusterhosts"
	HostWireDBCache    = "Hostwires"

	AggregateDBCache          = "Aggregates"
	AggregateHostDBCache      = "AggregateHosts"
	AggregateBaremetalDBCache = "AggregateBaremetals"

	BaremetalAgentDBCache = "BaremetalAgents"

	NetworksDBCache      = "Networks"
	NetInterfacesDBCache = "NetInterfaces"
	WiresDBCache         = "Wires"
)

func getUpdate(d []interface{}) ([]string, error) {
	return nil, nil
}

func DefaultCachedItems() []cache.CachedItem {
	if !models.DBValid() {
		panic("DB not init before cache items")
	}
	return []cache.CachedItem{
		newClusterDBCache(),
		newBaremetalAgentDBCache(),
		newAggregateDBCache(),
		newAggregateHostDBCache(),
	}
}

func NewCachedItems(items []string) (cachedItems []cache.CachedItem) {
	if !models.DBValid() {
		panic("DB not init before cache items")
	}
	for _, item := range items {
		switch item {
		case NetworksDBCache:
			cachedItems = append(cachedItems, newNetworksDBCache())
		case NetInterfacesDBCache:
			cachedItems = append(cachedItems, newNetInterfacesDBCache())
		case WiresDBCache:
			cachedItems = append(cachedItems, newWiresDBCache())
		default:
			return nil
		}
	}
	return
}

func uuidKey(obj interface{}) (string, error) {
	return obj.(models.Modeler).UUID(), nil
}

func newDBCache(name string, r models.Resourcer, ttl, period time.Duration) cache.CachedItem {
	update := func(ids []string) ([]interface{}, error) {
		return models.FetchByIDs(r, ids)
	}

	load := func() ([]interface{}, error) {
		return models.All(r)
	}

	item := new(dbItem)
	item.CachedItem = cache.NewCacheItem(
		name, ttl, period, uuidKey, update, load, getUpdate,
	)
	return item
}

func newClusterDBCache() cache.CachedItem {
	return newDBCache(ClusterDBCache, models.Clusters,
		u.ToDuration(o.GetOptions().ClusterDBCacheTTL),
		u.ToDuration(o.GetOptions().ClusterDBCachePeriod))
}

func newBaremetalAgentDBCache() cache.CachedItem {
	return newDBCache(BaremetalAgentDBCache, models.BaremetalAgents,
		u.ToDuration(o.GetOptions().BaremetalAgentDBCacheTTL),
		u.ToDuration(o.GetOptions().BaremetalAgentDBCachePeriod))
}

func newAggregateDBCache() cache.CachedItem {
	return newDBCache(AggregateDBCache, models.Aggregates,
		u.ToDuration(o.GetOptions().AggregateDBCacheTTL),
		u.ToDuration(o.GetOptions().AggregateDBCachePeriod))
}

func newAggregateHostDBCache() cache.CachedItem {
	return newDBCache(AggregateHostDBCache, models.AggregateHosts,
		u.ToDuration(o.GetOptions().AggregateHostDBCacheTTL),
		u.ToDuration(o.GetOptions().AggregateHostDBCachePeriod))
}

func newNetworksDBCache() cache.CachedItem {
	return newDBCache(NetworksDBCache, models.Networks,
		u.ToDuration(o.GetOptions().NetworksDBCacheTTL),
		u.ToDuration(o.GetOptions().NetworksDBCachePeriod))
}

func newNetInterfacesDBCache() cache.CachedItem {
	return newDBCache(AggregateHostDBCache, models.NetInterfaces,
		u.ToDuration(o.GetOptions().NetinterfaceDBCacheTTL),
		u.ToDuration(o.GetOptions().NetinterfaceDBCachePeriod))
}

func newWiresDBCache() cache.CachedItem {
	return newDBCache(AggregateHostDBCache, models.Wires,
		u.ToDuration(o.GetOptions().WireDBCacheTTL),
		u.ToDuration(o.GetOptions().WireDBCachePeriod))
}
