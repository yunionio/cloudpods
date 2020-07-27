package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	CommonAlertMetadataAlertType = "alert_type"
	CommonAlertMetadataFieldOpt  = "field_opt"
)

var (
	CommonAlertManager *SCommonAlertManager
)

func init() {
	CommonAlertManager = &SCommonAlertManager{
		SAlertManager: *NewAlertManager(SCommonAlert{}, "commonalert", "commonalerts"),
	}
	CommonAlertManager.SetVirtualObject(CommonAlertManager)
}

type ISubscriptionManager interface {
	SetAlert(alert *SCommonAlert)
	DeleteAlert(alert *SCommonAlert)
}

type SCommonAlertManager struct {
	SAlertManager
	subscriptionManager ISubscriptionManager
}

type SCommonAlert struct {
	SAlert
}

func (man *SCommonAlertManager) SetSubscriptionManager(sman ISubscriptionManager) {
	man.subscriptionManager = sman
}

func (man *SCommonAlertManager) SetSubscriptionAlert(alert *SCommonAlert) {
	man.subscriptionManager.SetAlert(alert)
}

func (man *SCommonAlertManager) DeleteSubscriptionAlert(alert *SCommonAlert) {
	man.subscriptionManager.DeleteAlert(alert)
}

func (man *SCommonAlertManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (man *SCommonAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.CommonAlertCreateInput) (monitor.CommonAlertCreateInput, error) {
	if data.Period == "" {
		data.Period = "5m"
	}
	if data.Name == "" {
		return data, httperrors.NewInputParameterError("name is empty")
	}
	if data.Level == "" {
		return data, httperrors.NewInputParameterError("level is empty")
	}
	if !utils.IsInStringArray(data.Level, monitor.CommonAlertLevels) {
		return data, httperrors.NewInputParameterError("Invalid level format: %s", data.Level)
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	// 默认的系统配置Recipients=commonalert-default
	if data.AlertType != monitor.CommonAlertSystemAlertType && len(data.Recipients) == 0 {
		return data, httperrors.NewInputParameterError("recipients is empty")
	}

	if len(data.CommonMetricInputQuery.MetricQuery) == 0 {
		return data, httperrors.NewInputParameterError("metric_query is empty")
	} else {
		for _, query := range data.CommonMetricInputQuery.MetricQuery {
			if !utils.IsInStringArray(getQueryEvalType(query.Comparator), validators.EvaluatorDefaultTypes) {
				return data, httperrors.NewInputParameterError("the Comparator is illegal:", query.Comparator)
			}
			if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
				return data, httperrors.NewInputParameterError("the reduce is illegal", query.Reduce)
			}
			if query.Threshold == 0 {
				return data, httperrors.NewInputParameterError("threshold is meaningless")
			}
		}
	}

	if data.AlertType != "" {
		if !utils.IsInStringArray(data.AlertType, validators.CommonAlertType) {
			return data, httperrors.NewInputParameterError("the AlertType is illegal:%s", data.AlertType)
		}
	}
	err := man.ValidateMetricQuery(&data.CommonMetricInputQuery)
	if err != nil {
		return data, errors.Wrap(err, "metric query error")
	}

	name, err := man.genName(ownerId, data.Name)
	if err != nil {
		return data, err
	}
	data.Name = name

	alertCreateInput := man.toAlertCreatInput(data)
	alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertCreateInput)
	if err != nil {
		return data, err
	}
	data.AlertCreateInput = alertCreateInput
	return data, nil

}

func (man *SCommonAlertManager) genName(ownerId mcclient.IIdentityProvider, name string) (string,
	error) {
	name, err := db.GenerateName(man, ownerId, name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (man *SCommonAlertManager) ValidateMetricQuery(metricRequest *monitor.CommonMetricInputQuery) error {
	for _, q := range metricRequest.MetricQuery {
		metriInputQuery := monitor.MetricInputQuery{
			From:     metricRequest.From,
			To:       metricRequest.To,
			Interval: metricRequest.Interval,
		}
		setDefaultValue(q.AlertQuery, &metriInputQuery)
		err := UnifiedMonitorManager.ValidateInputQuery(q.AlertQuery)
		if err != nil {
			return err
		}
	}
	return nil
}

func (alert *SCommonAlert) setAlertType(ctx context.Context, userCred mcclient.TokenCredential, alertType string) error {
	return alert.SetMetadata(ctx, CommonAlertMetadataAlertType, alertType, userCred)
}

func (alert *SCommonAlert) getAlertType() string {
	return alert.GetMetadata(CommonAlertMetadataAlertType, nil)
}

func (alert *SCommonAlert) setFieldOpt(ctx context.Context, userCred mcclient.TokenCredential, fieldOpt string) error {
	return alert.SetMetadata(ctx, CommonAlertMetadataFieldOpt, fieldOpt, userCred)
}

func (alert *SCommonAlert) getFieldOpt() string {
	return alert.GetMetadata(CommonAlertMetadataFieldOpt, nil)
}

func (alert *SCommonAlert) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	if err := alert.SAlert.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(monitor.CommonAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}

	return alert.customizeCreateNotis(ctx, userCred, query, data)
}

func (alert *SCommonAlert) customizeCreateNotis(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) error {
	input := new(monitor.CommonAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}
	//user_by 弃用
	if input.AlertType == monitor.CommonAlertSystemAlertType {
		return alert.createAlertNoti(ctx, userCred, input.Name, "webconsole", []string{}, true)
	}
	for _, channel := range input.Channel {
		err := alert.createAlertNoti(ctx, userCred, input.Name, channel, input.Recipients, false)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("create notify[channel is %s]error", channel))
		}
	}
	return nil
}

func (alert *SCommonAlert) createAlertNoti(ctx context.Context, userCred mcclient.TokenCredential,
	notiName, channel string, userIds []string, isSysNoti bool) error {
	noti, err := NotificationManager.CreateOneCloudNotification(ctx, userCred, notiName, channel, userIds)
	if err != nil {
		return errors.Wrap(err, "create notification")
	}
	if isSysNoti {
		_, err = db.Update(noti, func() error {
			//对于默认系统报警，isDefault=true
			noti.IsDefault = true
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "create notification")
		}
	}
	if alert.Id == "" {
		alert.Id = db.DefaultUUIDGenerator()
	}
	//alert.UsedBy = usedBy
	_, err = alert.AttachNotification(
		ctx, userCred, noti,
		monitor.AlertNotificationStateUnknown,
		"")
	return err
}

func (alert *SCommonAlert) PostCreate(ctx context.Context,
	userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {

	input := new(monitor.CommonAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		log.Errorf("post create unmarshal input: %v", err)
		return
	}

	alert.SetStatus(userCred, monitor.ALERT_STATUS_READY, "")

	if input.AlertType != "" {
		alert.setAlertType(ctx, userCred, input.AlertType)
	}
	fieldOpt := ""
	for i, metricQ := range input.CommonMetricInputQuery.MetricQuery {
		if metricQ.FieldOpt != "" {
			if i == 0 {
				fieldOpt = metricQ.FieldOpt
				continue
			}
			fieldOpt = fmt.Sprintf("%s+%s", fieldOpt, metricQ.FieldOpt)
		}
	}
	if fieldOpt != "" {
		alert.setFieldOpt(ctx, userCred, fieldOpt)
	}
	_, err := alert.PerformSetScope(ctx, userCred, query, data)
	if err != nil {
		log.Errorln(errors.Wrap(err, "Alert PerformSetScope"))
	}
	CommonAlertManager.SetSubscriptionAlert(alert)
}

func (man *SCommonAlertManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.CommonAlertListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SAlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	q.Filter(sqlchemy.IsNull(q.Field("used_by")))

	if len(query.Level) > 0 {
		q.Equals("level", query.Level)
	}

	return q, nil
}

func (man *SCommonAlertManager) CustomizeFilterList(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject) (
	*db.CustomizeListFilters, error) {
	filters, err := man.SAlertManager.CustomizeFilterList(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	input := new(monitor.CommonAlertListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	wrapF := func(f func(obj *SCommonAlert) (bool, error)) func(object jsonutils.JSONObject) (bool, error) {
		return func(data jsonutils.JSONObject) (bool, error) {
			id, err := data.GetString("id")
			if err != nil {
				return false, err
			}
			obj, err := man.GetAlert(id)
			if err != nil {
				return false, err
			}
			return f(obj)
		}
	}

	if input.Metric != "" {
		metric := input.Metric
		meaurement, field, err := GetMeasurementField(metric)
		if err != nil {
			return nil, err
		}
		mF := func(obj *SCommonAlert) (bool, error) {
			settings := new(monitor.AlertSetting)
			if err := obj.Settings.Unmarshal(settings); err != nil {
				return false, errors.Wrapf(err, "alert %s unmarshal", obj.GetId())
			}
			for _, s := range settings.Conditions {
				if s.Query.Model.Measurement == meaurement && len(s.Query.Model.Selects) == 1 {
					if IsQuerySelectHasField(s.Query.Model.Selects[0], field) {
						return true, nil
					}
				}
			}
			return false, nil
		}
		filters.Append(wrapF(mF))
	}

	if input.AlertType != "" {
		filters.Append(wrapF(func(obj *SCommonAlert) (bool, error) {
			return obj.getAlertType() == input.AlertType, nil
		}))
	}
	return filters, nil
}

func (man *SCommonAlertManager) GetAlert(id string) (*SCommonAlert, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCommonAlert), nil
}

func (man *SCommonAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.CommonAlertDetails {
	rows := make([]monitor.CommonAlertDetails, len(objs))
	alertRows := man.SAlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].AlertDetails = alertRows[i]
		rows[i], _ = objs[i].(*SCommonAlert).GetMoreDetails(rows[i])
	}
	return rows
}

func (alert *SCommonAlert) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	isForce, _ := data.Bool("force")
	if isForce {
		return isForce
	}
	alert_type := alert.getAlertType()
	switch alert_type {
	case monitor.CommonAlertSystemAlertType:
		return false
	default:
		return true
	}
}

func (alert *SCommonAlert) GetMoreDetails(out monitor.CommonAlertDetails) (monitor.CommonAlertDetails, error) {
	var err error
	alertNotis, err := alert.GetNotifications()
	if err != nil {
		return out, err
	}
	for i, alertNoti := range alertNotis {
		noti, err := alertNoti.GetNotification()
		if err != nil {
			return out, errors.Wrap(err, "get notify err:")
		}
		settings := new(monitor.NotificationSettingOneCloud)
		if err := noti.Settings.Unmarshal(settings); err != nil {
			return out, err
		}
		if i == 0 {
			out.Recipients = settings.UserIds
		}
		out.Channel = append(out.Channel, settings.Channel)
	}
	out.Status = alert.GetStatus()
	out.AlertType = alert.getAlertType()
	if alert.Frequency < 60 {
		out.Period = fmt.Sprintf("%ds", alert.Frequency)
	} else {
		out.Period = fmt.Sprintf("%dm", alert.Frequency/60)
	}

	err = alert.getCommonAlertMetricDetails(&out)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (alert *SCommonAlert) getCommonAlertMetricDetails(out *monitor.CommonAlertDetails) error {
	setting, err := alert.GetSettings()
	if err != nil {
		return err
	}
	if len(setting.Conditions) == 0 {
		return nil
	}
	fieldOpt := alert.getFieldOpt()
	out.CommonAlertMetricDetails = make([]*monitor.CommonAlertMetricDetails, len(setting.Conditions))
	for i, cond := range setting.Conditions {
		metricDetails := new(monitor.CommonAlertMetricDetails)
		cmp := ""
		switch cond.Evaluator.Type {
		case "gt":
			cmp = ">="
		case "lt":
			cmp = "<="
		}
		metricDetails.Comparator = cmp
		metricDetails.Threshold = cond.Evaluator.Params[0]
		metricDetails.Reduce = cond.Reducer.Type

		if fieldOpt != "" {
			metricDetails.FieldOpt = strings.Split(fieldOpt, "+")[i]
		}
		q := cond.Query
		measurement := q.Model.Measurement
		field := ""
		for i, sel := range q.Model.Selects {
			if i == 0 {
				field = sel[0].Params[0]
				continue
			}
			if metricDetails.FieldOpt != "" {
				field = fmt.Sprintf("%s%s%s", field, metricDetails.FieldOpt, sel[0].Params[0])
			}
		}
		//field := q.Model.Selects[0][0].Params[0]
		db := q.Model.Database
		var groupby string
		for _, grb := range q.Model.GroupBy {
			if grb.Type == "tag" {
				groupby = grb.Params[0]
				break
			}
		}

		metricDetails.Measurement = measurement
		metricDetails.Field = field
		metricDetails.DB = db
		metricDetails.Groupby = groupby
		out.CommonAlertMetricDetails[i] = metricDetails
	}
	return nil
}

func getQueryEvalType(evalType string) string {
	typ := ""
	switch evalType {
	case ">=", ">":
		typ = "gt"
	case "<=", "<":
		typ = "lt"
	}
	return typ
}

func (man *SCommonAlertManager) toAlertCreatInput(input monitor.CommonAlertCreateInput) monitor.AlertCreateInput {
	freq, _ := time.ParseDuration(input.Period)
	ret := new(monitor.AlertCreateInput)
	ret.Name = input.Name
	ret.Frequency = int64(freq / time.Second)
	ret.Level = input.Level
	//ret.Settings =monitor.AlertSetting{}
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

func (alert *SCommonAlert) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.CommonAlertUpdateInput,
) (monitor.CommonAlertUpdateInput, error) {

	if data.Period == "" {
		data.Period = "5m"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if len(data.Recipients) == 0 {
		return data, httperrors.NewInputParameterError("recipients is empty")
	}
	if len(data.CommonMetricInputQuery.MetricQuery) == 0 {
		return data, httperrors.NewInputParameterError("metric_query is empty")
	} else {
		for _, query := range data.CommonMetricInputQuery.MetricQuery {
			if !utils.IsInStringArray(getQueryEvalType(query.Comparator), validators.EvaluatorDefaultTypes) {
				return data, httperrors.NewInputParameterError("the Comparator is illegal:", query.Comparator)
			}
			if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
				return data, httperrors.NewInputParameterError("the reduce is illegal", query.Reduce)
			}
			if query.Threshold == 0 {
				return data, httperrors.NewInputParameterError("threshold is meaningless")
			}
		}
	}

	err := CommonAlertManager.ValidateMetricQuery(&data.CommonMetricInputQuery)
	if err != nil {
		return data, errors.Wrap(err, "metric query error")
	}

	if data.Period != "" {
		frep, _ := time.ParseDuration(data.Period)
		freqSpec := int64(frep / time.Second)
		data.Frequency = &freqSpec
	}

	alertCreateInput := alert.getUpdateAlertInput(data)
	alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, nil, query, alertCreateInput)
	if err != nil {
		return data, err
	}
	data.Settings = &alertCreateInput.Settings
	data.AlertUpdateInput, err = alert.SAlert.ValidateUpdateData(ctx, userCred, query, data.AlertUpdateInput)
	if err != nil {
		return data, errors.Wrap(err, "SAlert.ValidateUpdateData")
	}
	if alert.getAlertType() == monitor.CommonAlertSystemAlertType {
		tmp := data
		tmp.Settings = nil
		return tmp, nil
	}
	return data, nil
}

func (alert *SCommonAlert) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	updateInput := new(monitor.CommonAlertUpdateInput)
	data.Unmarshal(updateInput)
	if err := alert.UpdateNotification(ctx, userCred, query, data); err != nil {
		log.Errorln("update notification", err)
	}
	_, err := alert.PerformSetScope(ctx, userCred, query, data)
	if err != nil {
		log.Errorln(errors.Wrap(err, "Alert PerformSetScope"))
	}
	CommonAlertManager.SetSubscriptionAlert(alert)
}

func (alert *SCommonAlert) UpdateNotification(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := alert.customizeDeleteNotis(ctx, userCred, query, data)
	if err != nil {
		return errors.Wrap(err, "update notification err")
	}
	name, _ := data.GetString("name")
	if len(name) == 0 {
		data.(*jsonutils.JSONDict).Add(jsonutils.NewString(alert.Name), "name")
	}
	err = alert.customizeCreateNotis(ctx, userCred, query, data)
	if err != nil {
		log.Errorln(err)
	}
	return err
}

func (alert *SCommonAlert) getUpdateAlertInput(updateInput monitor.CommonAlertUpdateInput) monitor.AlertCreateInput {
	input := monitor.CommonAlertCreateInput{
		CommonMetricInputQuery: updateInput.CommonMetricInputQuery,
		Period:                 updateInput.Period,
	}
	alertCreateInput := CommonAlertManager.toAlertCreatInput(input)
	return alertCreateInput
}

func (alert *SCommonAlert) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	alert.SetStatus(userCred, monitor.ALERT_STATUS_DELETING, "")
	err := alert.customizeDeleteNotis(ctx, userCred, query, data)
	if err != nil {
		alert.SetStatus(userCred, monitor.ALERT_STATUS_DELETE_FAIL, "")

	}
	return err
}

func (alert *SCommonAlert) customizeDeleteNotis(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	notis, err := alert.GetNotifications()
	if err != nil {
		return err
	}
	for _, noti := range notis {
		conf, err := noti.GetNotification()
		if err != nil {
			return err
		}
		if err := conf.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return err
		}
		if err := conf.Delete(ctx, userCred); err != nil {
			return err
		}
		if err := noti.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (alert *SCommonAlert) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	CommonAlertManager.DeleteSubscriptionAlert(alert)

	return alert.SAlert.Delete(ctx, userCred)
}

func (self *SCommonAlertManager) GetSystemAlerts() ([]SCommonAlert, error) {
	objs := make([]SCommonAlert, 0)
	q := CommonAlertManager.Query()
	metaData := db.Metadata.Query().SubQuery()
	q.Join(metaData, sqlchemy.Equals(
		metaData.Field("obj_id"), q.Field("id")))
	q.Filter(sqlchemy.AND(sqlchemy.Equals(metaData.Field("key"), CommonAlertMetadataAlertType),
		sqlchemy.Equals(metaData.Field("value"), monitor.CommonAlertSystemAlertType)))
	err := db.FetchModelObjects(self, q, &objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (alert *SCommonAlert) AllowPerformSetScope(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (alert *SCommonAlert) PerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return CommonAlertManager.PerformSetScope(ctx, alert, userCred, data)
}

func (manager *SCommonAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SScopedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "status":
		q.AppendField(sqlchemy.DISTINCT(field, q.Field("status"))).Distinct()
		return q, nil
	}
	return q, httperrors.ErrNotFound
}
