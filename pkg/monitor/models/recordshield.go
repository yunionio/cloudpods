package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AlertRecordShieldManager *SAlertRecordShieldManager
)

type SAlertRecordShieldManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
}

func init() {
	AlertRecordShieldManager = &SAlertRecordShieldManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertRecordShield{},
			"alertrecordshield_tbl",
			"alertrecordshield",
			"alertrecordshields",
		),
	}

	AlertRecordShieldManager.SetVirtualObject(AlertRecordShieldManager)
}

type SAlertRecordShield struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource

	AlertId string `width:"36" charset:"ascii" nullable:"false" list:"user" create :"required" json:"alert_id"`
	ResName string `width:"36" nullable:"false"  create:"optional" list:"user" update:"user" json:"res_name"`
	ResType string `width:"36" nullable:"false"  create:"optional" list:"user" update:"user" json:"res_type"`

	StartTime time.Time `required:"optional" list:"user" update:"user" json:"start_time"`
	EndTime   time.Time `required:"optional" list:"user" update:"user" json:"end_time"`
}

func (manager *SAlertRecordShieldManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (manager *SAlertRecordShieldManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertRecordShieldListInput,
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

	q = manager.shieldListByDetailsFeild(q, query)
	return q, nil
}

func (manager *SAlertRecordShieldManager) shieldListByDetailsFeild(query *sqlchemy.SQuery,
	input monitor.AlertRecordShieldListInput) *sqlchemy.SQuery {
	if len(input.ResName) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("res_name"), input.ResName))
	}
	if len(input.ResType) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("res_type"), input.ResType))
	}
	if len(input.AlertId) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("alert_id"), input.AlertId))
	}

	alertQuery := CommonAlertManager.Query().SubQuery()
	if len(input.AlertName) != 0 {
		query.Join(alertQuery, sqlchemy.Equals(query.Field("alert_id"),
			alertQuery.Field("id"))).Filter(sqlchemy.Equals(alertQuery.Field("name"), input.AlertName))
	}
	return query
}

func (man *SAlertRecordShieldManager) OrderByExtraFields(
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

func (man *SAlertRecordShieldManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertRecordShieldDetails {
	rows := make([]monitor.AlertRecordShieldDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertRecordShieldDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SAlertRecordShield).GetMoreDetails(ctx, rows[i])
	}
	return rows
}

func (shield *SAlertRecordShield) GetMoreDetails(ctx context.Context, out monitor.AlertRecordShieldDetails) (monitor.
	AlertRecordShieldDetails, error) {
	// Alert May delete By someone
	out.AlertName = shield.AlertId
	commonAlert, err := CommonAlertManager.GetAlert(shield.AlertId)
	if err != nil {
		log.Errorf("GetAlert byId:%s err:%v", shield.AlertId, err)
		return out, nil
	}
	alertDetails, err := commonAlert.GetMoreDetails(ctx, monitor.CommonAlertDetails{})
	out.CommonAlertDetails = alertDetails
	return out, nil
}

func (man *SAlertRecordShieldManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject,
	data monitor.AlertRecordShieldCreateInput) (monitor.AlertRecordShieldCreateInput, error) {
	if len(data.AlertId) == 0 {
		return data, httperrors.NewInputParameterError("alert_id  is empty")
	}
	alert, err := CommonAlertManager.GetAlert(data.AlertId)
	if err != nil {
		return data, httperrors.NewInputParameterError("get resourceRecord err by:%s,err:%v", data.AlertId, err)
	}
	if len(data.ResName) == 0 {
		return data, httperrors.NewInputParameterError("shield res_name is empty")
	}
	if data.EndTime.Before(data.StartTime) {
		return data, httperrors.NewInputParameterError("end_time is before start_time")
	}
	if data.EndTime.Before(time.Now()) {
		return data, httperrors.NewInputParameterError("end_time is before now")
	}
	hint := fmt.Sprintf("%s-%s", alert.Name, data.ResName)
	name, err := db.GenerateName(ctx, man, ownerId, hint)
	if err != nil {
		return data, errors.Wrap(err, "get GenerateName err")
	}
	data.Name = name
	return data, nil
}

func (manager *SAlertRecordShieldManager) GetRecordShields(input monitor.AlertRecordShieldListInput) (
	[]SAlertRecordShield, error) {
	shields := make([]SAlertRecordShield, 0)
	shieldQuery := manager.Query()
	shieldQuery = manager.shieldListByDetailsFeild(shieldQuery, input)
	err := db.FetchModelObjects(manager, shieldQuery, &shields)
	if err != nil {
		return nil, err
	}
	return shields, nil
}
