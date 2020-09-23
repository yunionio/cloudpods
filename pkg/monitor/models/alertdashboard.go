package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAlertDashBoardManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SAlertDashBoard struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	db.SScopedResourceBase

	Refresh  string               `nullable:"false" list:"user" create:"required" update:"user"`
	Settings jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
	Message  string               `charset:"utf8" list:"user" create:"optional" update:"user"`
}

var AlertDashBoardManager *SAlertDashBoardManager

func init() {
	AlertDashBoardManager = &SAlertDashBoardManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertDashBoard{},
			"alertdashboard_tbl",
			"alertdashboard",
			"alertdashboards",
		),
	}

	AlertDashBoardManager.SetVirtualObject(AlertDashBoardManager)
}

func (manager *SAlertDashBoardManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SAlertDashBoardManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (man *SAlertDashBoardManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertDashBoardListInput,
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

func (man *SAlertDashBoardManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.AlertDashBoardCreateInput) (monitor.AlertDashBoardCreateInput, error) {
	if len(data.Refresh) != 0 {
		if _, err := time.ParseDuration(data.Refresh); err != nil {
			return data, httperrors.NewInputParameterError("Invalid refresh format: %s", data.Refresh)
		}
	}
	if len(data.CommonMetricInputQuery.MetricQuery) == 0 {
		return data, merrors.NewArgIsEmptyErr("metric_query")
	} else {
		for _, query := range data.CommonMetricInputQuery.MetricQuery {
			if len(query.Comparator) != 0 {
				if !utils.IsInStringArray(getQueryEvalType(query.Comparator), validators.EvaluatorDefaultTypes) {
					return data, httperrors.NewInputParameterError("the Comparator is illegal:", query.Comparator)
				}
			}
			if len(query.Reduce) != 0 {
				if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
					return data, httperrors.NewInputParameterError("the reduce is illegal", query.Reduce)
				}
			}
		}
		err := CommonAlertManager.ValidateMetricQuery(&data.CommonMetricInputQuery, data.Scope, ownerId)
		if err != nil {
			return data, errors.Wrap(err, "metric query error")
		}
	}

	name, err := CommonAlertManager.genName(ownerId, data.Name)
	if err != nil {
		return data, err
	}
	data.Name = name

	alertCreateInput := man.toAlertCreateInput(data)
	//alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertCreateInput)
	//if err != nil {
	//	return data, err
	//}
	data.AlertCreateInput = alertCreateInput
	enable := true
	if data.Enabled == nil {
		data.Enabled = &enable
	}
	return data, nil
}

func (man *SAlertDashBoardManager) toAlertCreateInput(input monitor.AlertDashBoardCreateInput) monitor.AlertCreateInput {
	ret := new(monitor.AlertCreateInput)
	for _, metricquery := range input.CommonMetricInputQuery.MetricQuery {
		condition := monitor.AlertCondition{
			Type:      "query",
			Query:     *metricquery.AlertQuery,
			Reducer:   monitor.Condition{Type: metricquery.Reduce},
			Evaluator: monitor.Condition{Type: getQueryEvalType(metricquery.Comparator), Params: []float64{metricquery.Threshold}},
			Operator:  "and",
		}
		if metricquery.FieldOpt != "" {
			condition.Reducer.Operators = []string{metricquery.FieldOpt}
		}
		ret.Settings.Conditions = append(ret.Settings.Conditions, condition)
	}
	return *ret
}

func (dash *SAlertDashBoard) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	return dash.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (man *SAlertDashBoardManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertDashBoardListInput,
) (*sqlchemy.SQuery, error) {
	q, err := AlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SAlertDashBoardManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertDashBoardDetails {
	rows := make([]monitor.AlertDashBoardDetails, len(objs))
	alertRows := AlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].AlertDetails = alertRows[i]
		rows[i], _ = objs[i].(*SAlertDashBoard).GetMoreDetails(rows[i])
	}
	return rows
}

func (dash *SAlertDashBoard) GetMoreDetails(out monitor.AlertDashBoardDetails) (monitor.AlertDashBoardDetails, error) {
	setting, err := dash.GetSettings()
	if err != nil {
		return out, err
	}
	if len(setting.Conditions) == 0 {
		return out, nil
	}

	out.CommonAlertMetricDetails = make([]*monitor.CommonAlertMetricDetails, len(setting.Conditions))
	for i, cond := range setting.Conditions {
		metricDetails := dash.GetCommonAlertMetricDetailsFromAlertCondition(i, cond)
		out.CommonAlertMetricDetails[i] = metricDetails
	}
	return out, nil
}

func (dash *SAlertDashBoard) GetCommonAlertMetricDetailsFromAlertCondition(index int,
	cond monitor.AlertCondition) *monitor.
	CommonAlertMetricDetails {
	metricDetails := new(monitor.CommonAlertMetricDetails)
	getCommonAlertMetricDetailsFromCondition(&cond, metricDetails)
	return metricDetails
}

func (dash *SAlertDashBoard) GetSettings() (*monitor.AlertSetting, error) {
	setting := new(monitor.AlertSetting)
	if dash.Settings == nil {
		return setting, nil
	}
	if err := dash.Settings.Unmarshal(setting); err != nil {
		return nil, errors.Wrapf(err, "dashboard %s unmarshal", dash.GetId())
	}
	return setting, nil
}

func (dash *SAlertDashBoard) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	updataInput := new(monitor.AlertDashBoardUpdateInput)
	if refresh, _ := data.GetString("refresh"); len(refresh) > 0 {
		if _, err := time.ParseDuration(refresh); err != nil {
			return data, httperrors.NewInputParameterError("Invalid refresh format: %s", refresh)
		}
	}

	if metric_query, _ := data.GetArray("metric_query"); len(metric_query) > 0 {
		for i, _ := range metric_query {
			query := new(monitor.CommonAlertQuery)
			err := metric_query[i].Unmarshal(query)
			if err != nil {
				return data, errors.Wrap(err, "metric_query Unmarshal error")
			}
			if len(query.Comparator) != 0 {
				if !utils.IsInStringArray(getQueryEvalType(query.Comparator), validators.EvaluatorDefaultTypes) {
					return data, httperrors.NewInputParameterError("the Comparator is illegal:", query.Comparator)
				}
			}
			if len(query.Reduce) != 0 {
				if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
					return data, httperrors.NewInputParameterError("the reduce is illegal", query.Reduce)
				}
			}
		}
		metricQuery := new(monitor.CommonMetricInputQuery)
		err := data.Unmarshal(metricQuery)
		if err != nil {
			return data, errors.Wrap(err, "metric_query Unmarshal error")
		}
		ownerId, _ := AlertDashBoardManager.FetchOwnerId(ctx, data)
		if ownerId == nil {
			ownerId = userCred
		}
		scope, _ := data.GetString("scope")
		err = CommonAlertManager.ValidateMetricQuery(metricQuery, scope, ownerId)
		if err != nil {
			return data, errors.Wrap(err, "metric query error")
		}

		data.Update(jsonutils.Marshal(metricQuery))
		err = data.Unmarshal(updataInput)
		if err != nil {
			return data, errors.Wrap(err, "updataInput Unmarshal err")
		}
		alertCreateInput := dash.getUpdateAlertInput(*updataInput)
		//alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, nil, query, alertCreateInput)
		//if err != nil {
		//	return data, err
		//}
		data.Set("settings", jsonutils.Marshal(&alertCreateInput.Settings))
		updataInput.StandaloneResourceBaseUpdateInput, err = dash.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred,
			query, updataInput.StandaloneResourceBaseUpdateInput)
		if err != nil {
			return data, errors.Wrap(err, "SAlertDashBoard.ValidateUpdateData")
		}
		data.Update(jsonutils.Marshal(updataInput))
	}
	return data, nil
}

func (dash *SAlertDashBoard) getUpdateAlertInput(updateInput monitor.AlertDashBoardUpdateInput) monitor.AlertCreateInput {
	input := monitor.AlertDashBoardCreateInput{
		CommonMetricInputQuery: updateInput.CommonMetricInputQuery,
	}
	createInput := AlertDashBoardManager.toAlertCreateInput(input)
	return createInput
}
