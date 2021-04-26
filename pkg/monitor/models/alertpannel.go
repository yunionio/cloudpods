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

var (
	AlertPanelManager *SAlertPanelManager
)

func init() {
	AlertPanelManager = &SAlertPanelManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertPanel{},
			"alertpanel_tbl",
			"alertpanel",
			"alertpanels",
		),
	}
	AlertPanelManager.SetVirtualObject(AlertPanelManager)
}

type SAlertPanelManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SAlertPanel struct {
	db.SStatusStandaloneResourceBase
	db.SScopedResourceBase

	Settings jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
	Message  string               `charset:"utf8" list:"user" create:"optional" update:"user"`
}

func (manager *SAlertPanelManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SAlertPanelManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (man *SAlertPanelManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertPanelListInput,
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

func (man *SAlertPanelManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.AlertPanelCreateInput) (monitor.AlertPanelCreateInput, error) {
	if len(data.DashboardId) == 0 {
		return data, httperrors.NewInputParameterError("dashboard_id is empty")
	} else {
		_, err := AlertDashBoardManager.getDashboardByid(data.DashboardId)
		if err != nil {
			return data, httperrors.NewInputParameterError("can not find dashboard:%s", data.DashboardId)
		}
	}
	if len(data.CommonMetricInputQuery.MetricQuery) == 0 {
		return data, merrors.NewArgIsEmptyErr("metric_query")
	} else {
		for _, query := range data.CommonMetricInputQuery.MetricQuery {
			if len(query.Comparator) != 0 {
				if !utils.IsInStringArray(getQueryEvalType(query.Comparator), validators.EvaluatorDefaultTypes) {
					return data, httperrors.NewInputParameterError("the Comparator is illegal: %s", query.Comparator)
				}
			}
			if len(query.Reduce) != 0 {
				if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
					return data, httperrors.NewInputParameterError("the reduce is illegal %s", query.Reduce)
				}
			}
		}
		err := CommonAlertManager.ValidateMetricQuery(&data.CommonMetricInputQuery, data.Scope, ownerId)
		if err != nil {
			return data, errors.Wrap(err, "metric query error")
		}
	}

	//name, err := db.GenerateName(man, ownerId, data.Name)
	//if err != nil {
	//	return data, err
	//}

	alertCreateInput := man.toAlertCreateInput(data)
	data.AlertCreateInput = alertCreateInput
	enable := true
	if data.Enabled == nil {
		data.Enabled = &enable
	}

	//data.Name = name
	return data, nil
}

func (man *SAlertPanelManager) HasName() bool {
	return false
}

func (man *SAlertPanelManager) toAlertCreateInput(input monitor.AlertPanelCreateInput) monitor.AlertCreateInput {
	ret := new(monitor.AlertCreateInput)
	for _, metricquery := range input.CommonMetricInputQuery.MetricQuery {
		condition := monitor.AlertCondition{
			Type:  "query",
			Query: *metricquery.AlertQuery,
			//Reducer:   monitor.Condition{Type: metricquery.Reduce},
			//Evaluator: monitor.Condition{Type: getQueryEvalType(metricquery.Comparator), Params: []float64{metricquery.Threshold}},
			Operator: "and",
		}
		ret.Settings.Conditions = append(ret.Settings.Conditions, condition)
	}
	return *ret
}

func (panel *SAlertPanel) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	err := panel.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return errors.Wrap(err, "alertPanel SScopedResourceBase CustomizeCreate error")
	}
	dashboardId, err := data.GetString("dashboard_id")
	if len(dashboardId) == 0 {
		return errors.Wrap(err, "panel CustomizeCreate can not get dashboard_id")
	}
	return panel.attachDashboard(ctx, dashboardId)
}

func (panel *SAlertPanel) attachDashboard(ctx context.Context, dashboardId string) error {
	joint := new(SAlertDashboardPanel)
	joint.DashboardId = dashboardId
	if panel.Id == "" {
		panel.Id = db.DefaultUUIDGenerator()
	}
	joint.PanelId = panel.Id
	err := AlertDashBoardPanelManager.TableSpec().Insert(ctx, joint)
	if err != nil {
		return errors.Wrap(err, "panel attach dashboard error")
	}
	return nil
}

func (man *SAlertPanelManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertPanelListInput,
) (*sqlchemy.SQuery, error) {
	q, err := AlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	if len(query.DashboardId) != 0 {
		joinq := AlertDashBoardPanelManager.Query(AlertDashBoardPanelManager.GetSlaveFieldName()).Equals(
			AlertDashBoardPanelManager.GetMasterFieldName(), query.DashboardId).SubQuery()
		q = q.In("id", joinq)
	}
	return q, nil
}

func (man *SAlertPanelManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.PanelDetails {
	rows := make([]monitor.PanelDetails, len(objs))
	alertRows := AlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].AlertDetails = alertRows[i]
		rows[i], _ = objs[i].(*SAlertPanel).GetMoreDetails(rows[i])
	}
	return rows
}

func (panel *SAlertPanel) GetMoreDetails(out monitor.PanelDetails) (monitor.PanelDetails, error) {
	setting, err := panel.GetSettings()
	if err != nil {
		return out, err
	}
	if len(setting.Conditions) == 0 {
		return out, nil
	}

	out.CommonAlertMetricDetails = make([]*monitor.CommonAlertMetricDetails, len(setting.Conditions))
	for i, cond := range setting.Conditions {
		metricDetails := panel.GetCommonAlertMetricDetailsFromAlertCondition(i, &cond)
		out.CommonAlertMetricDetails[i] = metricDetails
		setting.Conditions[i] = cond
	}
	panel.Settings = jsonutils.Marshal(setting)
	return out, nil
}

func (dash *SAlertPanel) GetCommonAlertMetricDetailsFromAlertCondition(index int,
	cond *monitor.AlertCondition) *monitor.
	CommonAlertMetricDetails {
	metricDetails := new(monitor.CommonAlertMetricDetails)
	getCommonAlertMetricDetailsFromCondition(cond, metricDetails)
	return metricDetails
}

func (dash *SAlertPanel) GetSettings() (*monitor.AlertSetting, error) {
	setting := new(monitor.AlertSetting)
	if dash.Settings == nil {
		return setting, nil
	}
	if err := dash.Settings.Unmarshal(setting); err != nil {
		return nil, errors.Wrapf(err, "dashboard %s unmarshal", dash.GetId())
	}
	return setting, nil
}

func (dash *SAlertPanel) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	updataInput := new(monitor.AlertPanelUpdateInput)
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
					return data, httperrors.NewInputParameterError("the Comparator is illegal: %s", query.Comparator)
				}
			}
			if len(query.Reduce) != 0 {
				if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
					return data, httperrors.NewInputParameterError("the reduce is illegal: %s", query.Reduce)
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
		data.Set("settings", jsonutils.Marshal(&alertCreateInput.Settings))
		updataInput.StandaloneResourceBaseUpdateInput, err = dash.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred,
			query, updataInput.StandaloneResourceBaseUpdateInput)
		if err != nil {
			return data, errors.Wrap(err, "SAlertPanel.ValidateUpdateData")
		}
		data.Update(jsonutils.Marshal(updataInput))
	}
	return data, nil
}

func (panel *SAlertPanel) getUpdateAlertInput(updateInput monitor.AlertPanelUpdateInput) monitor.AlertCreateInput {
	input := monitor.AlertPanelCreateInput{
		CommonMetricInputQuery: updateInput.CommonMetricInputQuery,
	}
	createInput := AlertPanelManager.toAlertCreateInput(input)
	return createInput
}

func (panel *SAlertPanel) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	dashboardPanel, err := panel.getPanelJoint()
	if err != nil {
		return errors.Wrap(err, "panel getPanelJoint error")
	}
	err = dashboardPanel.Detach(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "dashboardPanel do detach error")
	}
	return nil
}

func (panel *SAlertPanel) getPanelJoint() (*SAlertDashboardPanel, error) {
	iModel, err := db.NewModelObject(AlertDashBoardPanelManager)
	if err != nil {
		return nil, errors.Wrap(err, "panel NewModelObject error")
	}
	query := AlertDashBoardPanelManager.Query().Equals(AlertDashBoardPanelManager.GetSlaveFieldName(), panel.Id)
	err = query.First(iModel)
	if err != nil {
		return nil, errors.Wrapf(err, "alertdashboardpanelmanager exc first query by panel:%s error", panel.Id)
	}
	return iModel.(*SAlertDashboardPanel), nil
}
