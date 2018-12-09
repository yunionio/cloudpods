package models

import (
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/utils"
)

func AttachUsageQuery(
	q *sqlchemy.SQuery,
	hosts *sqlchemy.SSubQuery,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	rangeObj db.IStandaloneModel,
) *sqlchemy.SQuery {
	if len(hostTypes) > 0 {
		q = q.Filter(sqlchemy.In(hosts.Field("host_type"), hostTypes))
	}
	if len(resourceTypes) > 0 {
		if utils.IsInStringArray(HostResourceTypeShared, resourceTypes) {
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
