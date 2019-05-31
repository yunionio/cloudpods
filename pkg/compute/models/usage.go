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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func AttachUsageQuery(
	q *sqlchemy.SQuery,
	hosts *sqlchemy.SSubQuery,
	hostTypes []string,
	resourceTypes []string,
	providers []string, cloudEnv string,
	rangeObj db.IStandaloneModel,
) *sqlchemy.SQuery {
	if len(hostTypes) > 0 {
		q = q.Filter(sqlchemy.In(hosts.Field("host_type"), hostTypes))
	}
	if len(resourceTypes) > 0 {
		if utils.IsInStringArray(api.HostResourceTypeShared, resourceTypes) {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(hosts.Field("resource_type")),
				sqlchemy.In(hosts.Field("resource_type"), resourceTypes),
			))
		} else {
			q = q.Filter(sqlchemy.In(hosts.Field("resource_type"), resourceTypes))
		}
	}
	if len(providers) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id")).In("provider", providers).SubQuery()
		q = q.Filter(sqlchemy.In(hosts.Field("manager_id"), subq))
	}
	if len(cloudEnv) > 0 {
		switch cloudEnv {
		case api.CLOUD_ENV_PUBLIC_CLOUD:
			q = q.Filter(sqlchemy.In(hosts.Field("manager_id"), CloudproviderManager.GetPublicProviderIdsQuery()))
		case api.CLOUD_ENV_PRIVATE_CLOUD:
			q = q.Filter(sqlchemy.In(hosts.Field("manager_id"), CloudproviderManager.GetPrivateProviderIdsQuery()))
		case api.CLOUD_ENV_ON_PREMISE:
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.In(hosts.Field("manager_id"), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
					sqlchemy.IsNullOrEmpty(hosts.Field("manager_id")),
				),
			)
		}
	}
	if rangeObj == nil {
		return q
	}
	//rangeObjId := rangeObj.GetId()
	kw := rangeObj.Keyword()
	// log.Debugf("rangeObj keyword: %s", kw)
	switch kw {
	case "zone":
		zone := rangeObj.(*SZone)
		q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.Id))
	case "wire":
		wire := rangeObj.(*SWire)
		hostWires := HostwireManager.Query().SubQuery()
		subq := hostWires.Query(hostWires.Field("host_id")).Equals("wire_id", wire.Id).SubQuery()
		q = q.Filter(sqlchemy.In(hosts.Field("id"), subq))
	case "host":
		q = q.Filter(sqlchemy.Equals(hosts.Field("id"), rangeObj.GetId()))
	case "cloudprovider":
		q = q.Filter(sqlchemy.Equals(hosts.Field("manager_id"), rangeObj.GetId()))
	case "cloudaccount":
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id")).Equals("cloudaccount_id", rangeObj.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(hosts.Field("manager_id"), subq))
	case "schedtag":
		aggHosts := HostschedtagManager.Query().SubQuery()
		q = q.Join(aggHosts, sqlchemy.AND(
			sqlchemy.Equals(hosts.Field("id"), aggHosts.Field("host_id")),
			sqlchemy.IsFalse(aggHosts.Field("deleted")))).
			Filter(sqlchemy.Equals(aggHosts.Field("schedtag_id"), rangeObj.GetId()))
	case "cloudregion":
		zones := ZoneManager.Query().SubQuery()
		q = q.Join(zones, sqlchemy.Equals(hosts.Field("zone_id"), zones.Field("id")))
		q = q.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), rangeObj.GetId()))
	}
	return q
}
