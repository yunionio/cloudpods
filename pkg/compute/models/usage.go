package models

import (
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func AttachUsageQuery(
	q *sqlchemy.SQuery,
	hosts *sqlchemy.SSubQuery,
	hostIdField sqlchemy.IQueryField,
	hostTypes []string,
	rangeObj db.IStandaloneModel,
) *sqlchemy.SQuery {
	if len(hostTypes) != 0 {
		q = q.Filter(sqlchemy.In(hosts.Field("host_type"), hostTypes))
	}
	if rangeObj == nil {
		return q
	}
	rangeObjId := rangeObj.GetId()
	kw := rangeObj.Keyword()
	log.Debugf("rangeObj keyword: %s", kw)
	switch kw {
	case "zone":
		q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), rangeObjId))
	case "wire":
		hostWires := HostwireManager.Query().SubQuery()
		q = q.Join(hostWires, sqlchemy.AND(
			sqlchemy.Equals(hosts.Field("id"), hostWires.Field("host_id")),
			sqlchemy.IsFalse(hostWires.Field("deleted")))).
			Filter(sqlchemy.Equals(hostWires.Field("wire_id"), rangeObjId))
	case "host":
		q = q.Filter(sqlchemy.Equals(hostIdField, rangeObjId))
	case "vcenter":
		q = q.Filter(sqlchemy.Equals(hosts.Field("manager_id"), rangeObjId))
	case "schedtag":
		aggHosts := SchedtagManager.Query().SubQuery()
		q = q.Join(aggHosts, sqlchemy.AND(
			sqlchemy.Equals(hosts.Field("id"), aggHosts.Field("host_id")),
			sqlchemy.IsFalse(aggHosts.Field("deleted")))).
			Filter(sqlchemy.Equals(aggHosts.Field("schedtag_id"), rangeObjId))
	}
	return q
}
