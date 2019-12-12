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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func rangeObjectsFilter(q *sqlchemy.SQuery, rangeObjs []db.IStandaloneModel, regionField sqlchemy.IQueryField, zoneField sqlchemy.IQueryField, managerField sqlchemy.IQueryField) *sqlchemy.SQuery {
	for _, rangeObj := range rangeObjs {
		q = rangeObjFilter(q, rangeObj, regionField, zoneField, managerField)
	}
	return q
}

func rangeObjFilter(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, regionField sqlchemy.IQueryField, zoneField sqlchemy.IQueryField, managerField sqlchemy.IQueryField) *sqlchemy.SQuery {
	if rangeObj == nil {
		return q
	}
	kw := rangeObj.Keyword()
	switch kw {
	case "zone":
		zone := rangeObj.(*SZone)
		if regionField != nil {
			q = q.Filter(sqlchemy.Equals(regionField, zone.CloudregionId))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, zone.Id))
		}
	case "wire":
		wire := rangeObj.(*SWire)
		if regionField != nil {
			vpc := wire.getVpc()
			q = q.Filter(sqlchemy.Equals(regionField, vpc.CloudregionId))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, wire.ZoneId))
		}
	case "host":
		host := rangeObj.(*SHost)
		if regionField != nil {
			zone := host.GetZone()
			q = q.Filter(sqlchemy.Equals(regionField, zone.CloudregionId))
		} else if zoneField != nil {
			q = q.Filter(sqlchemy.Equals(zoneField, host.ZoneId))
		}
	case "cloudprovider":
		q = q.Filter(sqlchemy.Equals(managerField, rangeObj.GetId()))
	case "cloudaccount":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id")).Equals("cloudaccount_id", rangeObj.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(managerField, subq))
	case "cloudregion":
		if regionField != nil {
			q = q.Filter(sqlchemy.Equals(regionField, rangeObj.GetId()))
		} else if zoneField != nil {
			zones := ZoneManager.Query().SubQuery()
			subq := zones.Query(zones.Field("id")).Equals("cloudregion_id", rangeObj.GetId()).SubQuery()
			q = q.Filter(sqlchemy.In(zoneField, subq))
		}
	}
	return q
}

func scopeOwnerIdFilter(q *sqlchemy.SQuery, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) *sqlchemy.SQuery {
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacutils.ScopeProject:
		q = q.Equals("tenant_id", ownerId.GetProjectId())
	}
	return q
}
