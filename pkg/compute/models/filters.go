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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func RangeObjectsFilter(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, regionField sqlchemy.IQueryField, zoneField sqlchemy.IQueryField, managerField sqlchemy.IQueryField, hostField sqlchemy.IQueryField, storageField sqlchemy.IQueryField) *sqlchemy.SQuery {
	for _, rangeObj := range rangeObjs {
		q = rangeObjFilter(q, rangeObj, regionField, zoneField, managerField, hostField, storageField)
	}
	return q
}

func rangeObjFilter(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, regionField sqlchemy.IQueryField, zoneField sqlchemy.IQueryField, managerField sqlchemy.IQueryField, hostField sqlchemy.IQueryField, storageField sqlchemy.IQueryField) *sqlchemy.SQuery {
	if rangeObj == nil {
		return q
	}
	kw := rangeObj.Keyword()
	switch kw {
	case "zone":
		zone := rangeObj.(*SZone)
		if hostField != nil {
			hosts := HostManager.Query("id", "zone_id").SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostField))
			q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.Id))
		} else if storageField != nil {
			storages := StorageManager.Query("id", "zone_id").SubQuery()
			q = q.Join(storages, sqlchemy.Equals(storages.Field("id"), storageField))
			q = q.Filter(sqlchemy.Equals(storages.Field("zone_id"), zone.Id))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, zone.Id))
		} else if regionField != nil {
			q = q.Filter(sqlchemy.Equals(regionField, zone.CloudregionId))
		}
	case "wire":
		wire := rangeObj.(*SWire)
		if hostField != nil {
			netifs := NetInterfaceManager.Query("baremetal_id", "wire_id").SubQuery()
			q = q.Join(netifs, sqlchemy.Equals(netifs.Field("baremetal_id"), hostField))
			q = q.Filter(sqlchemy.Equals(netifs.Field("wire_id"), wire.Id))
		} else if storageField != nil {
			netifs := NetInterfaceManager.Query("baremetal_id", "wire_id").SubQuery()
			hoststorages := HoststorageManager.Query("host_id", "storage_id").SubQuery()
			q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("storage_id"), storageField))
			q = q.Join(netifs, sqlchemy.Equals(hoststorages.Field("host_id"), netifs.Field("baremetal_id")))
			q = q.Filter(sqlchemy.Equals(netifs.Field("wire_id"), wire.Id))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, wire.ZoneId))
		} else if regionField != nil {
			vpc, _ := wire.GetVpc()
			q = q.Filter(sqlchemy.Equals(regionField, vpc.CloudregionId))
		}
	case "host":
		host := rangeObj.(*SHost)
		if hostField != nil {
			q = q.Filter(sqlchemy.Equals(hostField, host.Id))
		} else if storageField != nil {
			hoststorages := HoststorageManager.Query("host_id", "storage_id").SubQuery()
			q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("storage_id"), storageField))
			q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), host.Id))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, host.ZoneId))
		} else if regionField != nil {
			zone, _ := host.GetZone()
			q = q.Filter(sqlchemy.Equals(regionField, zone.CloudregionId))
		}
	case "storage":
		storage := rangeObj.(*SStorage)
		if hostField != nil {
			hoststorages := HoststorageManager.Query("host_id", "storage_id").SubQuery()
			q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hostField))
			q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), storage.Id))
		} else if storageField != nil {
			q = q.Filter(sqlchemy.Equals(storageField, storage.Id))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, storage.ZoneId))
		} else if regionField != nil {
			zone, _ := storage.GetZone()
			q = q.Filter(sqlchemy.Equals(regionField, zone.CloudregionId))
		}
	case "cloudprovider":
		if hostField != nil {
			hosts := HostManager.Query("id", "manager_id").SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostField))
			q = q.Filter(sqlchemy.Equals(hosts.Field("manager_id"), rangeObj.GetId()))
		} else if storageField != nil {
			storages := StorageManager.Query("id", "manager_id").SubQuery()
			q = q.Join(storages, sqlchemy.Equals(storages.Field("id"), storageField))
			q = q.Filter(sqlchemy.Equals(storages.Field("manager_id"), rangeObj.GetId()))
		} else if managerField != nil {
			q = q.Filter(sqlchemy.Equals(managerField, rangeObj.GetId()))
		}
	case "cloudaccount":
		if hostField != nil {
			hosts := HostManager.Query("id", "manager_id").SubQuery()
			providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostField))
			q = q.Join(providers, sqlchemy.Equals(hosts.Field("manager_id"), providers.Field("id")))
			q = q.Filter(sqlchemy.Equals(providers.Field("cloudaccount_id"), rangeObj.GetId()))
		} else if storageField != nil {
			storages := StorageManager.Query("id", "manager_id").SubQuery()
			providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
			q = q.Join(storages, sqlchemy.Equals(storages.Field("id"), storageField))
			q = q.Join(providers, sqlchemy.Equals(storages.Field("manager_id"), providers.Field("id")))
			q = q.Filter(sqlchemy.Equals(providers.Field("cloudaccount_id"), rangeObj.GetId()))
		} else if managerField != nil {
			providers := CloudproviderManager.Query("id", "cloudaccount_id").SubQuery()
			q = q.Join(providers, sqlchemy.Equals(providers.Field("id"), managerField))
			q = q.Filter(sqlchemy.Equals(providers.Field("cloudaccount_id"), rangeObj.GetId()))
		}
	case "cloudregion":
		if hostField != nil {
			hosts := HostManager.Query("id", "zone_id").SubQuery()
			zones := ZoneManager.Query("id", "cloudregion_id").SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostField))
			q = q.Join(zones, sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")))
			q = q.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), rangeObj.GetId()))
		} else if storageField != nil {
			storages := StorageManager.Query("id", "zone_id").SubQuery()
			zones := ZoneManager.Query("id", "cloudregion_id").SubQuery()
			q = q.Join(storages, sqlchemy.Equals(storages.Field("id"), storageField))
			q = q.Join(zones, sqlchemy.Equals(zones.Field("id"), storages.Field("zone_id")))
			q = q.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), rangeObj.GetId()))
		} else if zoneField != nil {
			zones := ZoneManager.Query("id", "cloudregion_id").SubQuery()
			q = q.Join(zones, sqlchemy.Equals(zones.Field("id"), zoneField))
			q = q.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), rangeObj.GetId()))
		} else if regionField != nil {
			q = q.Filter(sqlchemy.Equals(regionField, rangeObj.GetId()))
		}
	case "schedtag":
		if hostField != nil {
			hostschedtags := HostschedtagManager.Query("host_id", "schedtag_id").SubQuery()
			q = q.Join(hostschedtags, sqlchemy.Equals(hostschedtags.Field("host_id"), hostField))
			q = q.Filter(sqlchemy.Equals(hostschedtags.Field("schedtag_id"), rangeObj.GetId()))
		} else if storageField != nil {
			storageschedtags := StorageschedtagManager.Query("storage_id", "schedtag_id").SubQuery()
			q = q.Join(storageschedtags, sqlchemy.Equals(storageschedtags.Field("storage_id"), storageField))
			q = q.Filter(sqlchemy.Equals(storageschedtags.Field("schedtag_id"), rangeObj.GetId()))
		}
	}
	return q
}

func scopeOwnerIdFilter(q *sqlchemy.SQuery, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SQuery {
	switch scope {
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
	return q
}
