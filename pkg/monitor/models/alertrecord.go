package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AlertRecordManager *SAlertRecordManager
)

type SAlertRecordManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
}

type SAlertRecord struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource

	AlertId   string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Level     string               `charset:"ascii" width:"36" nullable:"false" default:"normal" list:"user" update:"user"`
	State     string               `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" update:"user"`
	EvalData  jsonutils.JSONObject `list:"user" update:"user"`
	AlertRule jsonutils.JSONObject `list:"user" update:"user"`
}

func init() {
	AlertRecordManager = &SAlertRecordManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertRecord{},
			"alertrecord_tbl",
			"alertrecord",
			"alertrecords",
		),
	}

	AlertRecordManager.SetVirtualObject(AlertRecordManager)
}

func (manager *SAlertRecordManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SAlertRecordManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (manager *SAlertRecordManager) ListItemFilter(
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
	if len(query.Level) != 0 {
		q = q.Equals("level", query.Level)
	}
	if len(query.State) != 0 {
		q = q.Equals("state", query.State)
	}
	if len(query.AlertId) != 0 {
		q = q.Equals("alert_id", query.AlertId)
	}
	return q, nil
}

func (man *SAlertRecordManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertRecordListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SAlertRecordManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertRecordDetails {
	rows := make([]monitor.AlertRecordDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertRecordDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SAlertRecord).GetMoreDetails(rows[i])
	}
	return rows
}

func (record *SAlertRecord) GetMoreDetails(out monitor.AlertRecordDetails) (monitor.AlertRecordDetails, error) {
	evalMatchs := make([]monitor.EvalMatch, 0)
	if record.EvalData != nil {
		err := record.EvalData.Unmarshal(&evalMatchs)
		if err != nil {
			return out, errors.Wrap(err, "record Unmarshal evalMatchs error")
		}
		out.ResNum = int64(len(evalMatchs))
	}

	return out, nil
}

func (man *SAlertRecordManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, data monitor.AlertRecordCreateInput) (monitor.AlertRecordCreateInput, error) {
	return data, nil
}

func (record *SAlertRecord) GetEvalData() ([]monitor.EvalMatch, error) {
	ret := make([]monitor.EvalMatch, 0)
	if err := record.EvalData.Unmarshal(&ret); err != nil {
		return nil, errors.Wrap(err, "unmarshal evalMatchs error")
	}
	return ret, nil
}

func (record *SAlertRecord) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	record.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := GetAlertResourceManager().ReconcileFromRecord(ctx, userCred, ownerId, record); err != nil {
		log.Errorf("Reconcile from alert record error: %v", err)
		return
	}
	err := GetAlertResourceManager().NotifyAlertResourceCount(ctx)
	if err != nil {
		log.Errorf("NotifyAlertResourceCount error: %v", err)
		return
	}
}

func (record *SAlertRecord) GetState() monitor.AlertStateType {
	return monitor.AlertStateType(record.State)
}
