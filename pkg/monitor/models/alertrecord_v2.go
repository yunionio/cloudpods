package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AlertRecordV2Manager *SAlertRecordV2Manager
)

type SAlertRecordV2Manager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
}

func init() {
	AlertRecordV2Manager = &SAlertRecordV2Manager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertResourceRecord{},
			"alertresourcerecord_tbl",
			"alertresourcerecord",
			"alertresourcerecords",
		),
	}

	AlertRecordV2Manager.SetVirtualObject(AlertRecordV2Manager)
}

type SAlertResourceRecord struct {
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource

	EvalData      jsonutils.JSONObject
	AlertTime     time.Time `list:"user" update:"user"`
	ResName       string    `width:"36" list:"user" update:"user"`
	Brand         string    `width:"36" list:"user" update:"user"`
	ResType       string    `width:"36" list:"user" update:"user"`
	TriggerVal    string    `width:"36" list:"user" update:"user"`
	AlertRecordId string    `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	AlertId       string    `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	SendState     string    `width:"36" charset:"ascii" default:"ok" list:"user" update:"user"`
}

func (man *SAlertRecordV2Manager) HasName() bool {
	return false
}

func (manager *SAlertRecordV2Manager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SAlertRecordV2Manager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertRecordListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	if query.Alerting {
		alertingQuery := AlertRecordManager.getAlertingRecordQuery().SubQuery()

		q.Join(alertingQuery, sqlchemy.Equals(q.Field("alert_id"), alertingQuery.Field("alert_id"))).Filter(
			sqlchemy.Equals(q.Field("alert_time"), alertingQuery.Field("max_created_at")))
	}
	if len(query.Level) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("level"), query.Level))
	}
	if len(query.State) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("state"), query.State))
	}
	if len(query.AlertId) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("alert_id"), query.AlertId))
	} else {
		q = q.IsNotEmpty("res_type").IsNotNull("res_type")
	}
	if len(query.ResType) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("res_type"), query.ResType))
	}
	if len(query.ResName) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("res_name"), query.ResName))
	}
	return q, nil
}

func (man *SAlertRecordV2Manager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertResourceRecordDetails {
	rows := make([]monitor.AlertResourceRecordDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertResourceRecordDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SAlertResourceRecord).GetMoreDetails(rows[i])
	}
	return rows
}

func (resRecord *SAlertResourceRecord) GetMoreDetails(out monitor.AlertResourceRecordDetails) (monitor.
	AlertResourceRecordDetails, error) {
	alertRecord, err := AlertRecordManager.GetAlertRecord(resRecord.AlertRecordId)
	if err != nil {
		return out, errors.Errorf("GetAlertRecord by id:%s err:%v", resRecord.AlertRecordId, err)
	}
	recordDetails, err := alertRecord.GetMoreDetails(monitor.AlertRecordDetails{})
	if err != nil {
		return out, err
	}
	out.AlertRule = alertRecord.AlertRule
	out.ResType = alertRecord.ResType
	out.Level = alertRecord.Level
	out.State = alertRecord.State

	out.AlertName = recordDetails.AlertName
	return out, nil
}

func CreateAlertResourceDetailsByAlertRecord(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	record *SAlertRecord) error {
	evalMatches, err := record.GetEvalData()
	if err != nil {
		return errors.Errorf("CreateAlertResourceDetailsByAlertRecord err. recordId:%s,AlertId:%s,err:%v", record.AlertId, record.GetId(), err)
	}

	for i, _ := range evalMatches {
		input := &monitor.AlertResourceRecordCreateInput{
			SStandaloneResourceBase: apis.SStandaloneResourceBase{
				Name: record.Name,
			},
			EvalData:      evalMatches[i],
			AlertTime:     record.CreatedAt,
			ResName:       evalMatches[i].Tags["name"],
			ResType:       record.ResType,
			Brand:         evalMatches[i].Tags["brand"],
			TriggerVal:    evalMatches[i].ValueStr,
			AlertRecordId: record.GetId(),
			AlertId:       record.AlertId,
			SendState:     record.SendState,
		}
		if _, ok := evalMatches[i].Tags[monitor.ALERT_RESOURCE_RECORD_SHIELD_KEY]; ok {
			input.SendState = monitor.ALERT_RESOURCE_RECORD_SHIELD_VALUE
		}
		resourceRecord, err := db.DoCreate(AlertRecordV2Manager, ctx, userCred, nil, input.JSON(input), ownerId)
		if err != nil {
			return errors.Wrapf(err, "create alert resource by data %s", input.JSON(input))
		}
		scopeParam := jsonutils.NewDict()
		scopeParam.Add(jsonutils.NewString(record.DomainId), "domain_id")
		scopeParam.Add(jsonutils.NewString(record.ProjectId), "project_id")
		_, err = db.PerformSetScope(ctx, resourceRecord.(*SAlertResourceRecord), userCred, scopeParam)
		if err != nil {
			return errors.Wrap(err, "resourceRecord PerformSetScope err")
		}
	}
	return nil
}

func (manager *SAlertRecordV2Manager) GetAlertResourceRecordById(id string) (*SAlertResourceRecord, error) {
	obj, err := manager.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SAlertResourceRecord), nil

}
