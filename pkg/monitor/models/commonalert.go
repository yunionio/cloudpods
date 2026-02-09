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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	notiapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	CommonAlertMetadataAlertType = "alert_type"
	CommonAlertMetadataFieldOpt  = "field_opt"
	CommonAlertMetadataPointStr  = "point_str"
	CommonAlertMetadataName      = "meta_name"

	COMPANY_COPYRIGHT_ONECLOUD = "云联"
	BRAND_ONECLOUD_NAME_CN     = "云联壹云"
	BRAND_ONECLOUD_NAME_EN     = "YunionCloud"
)

var (
	CommonAlertManager *SCommonAlertManager
)

func init() {
	GetCommonAlertManager()
	//registry.RegisterService(CommonAlertManager)
}

func GetCommonAlertManager() *SCommonAlertManager {
	if CommonAlertManager != nil {
		return CommonAlertManager
	}
	CommonAlertManager = &SCommonAlertManager{
		SAlertManager: *NewAlertManager(SCommonAlert{}, "commonalert", "commonalerts"),
	}
	CommonAlertManager.SetVirtualObject(CommonAlertManager)
	return CommonAlertManager
}

type ISubscriptionManager interface {
	SetAlert(alert *SCommonAlert)
	DeleteAlert(alert *SCommonAlert)
}

type SCommonAlertManager struct {
	SAlertManager
	subscriptionManager ISubscriptionManager
}

type ICommonAlert interface {
	db.IModel

	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	DeleteAttachAlertRecords(ctx context.Context, userCred mcclient.TokenCredential) []error
	DetachAlertResourceOnDisable(ctx context.Context, userCred mcclient.TokenCredential) []error
	DetachMonitorResourceJoint(ctx context.Context, userCred mcclient.TokenCredential) error
	UpdateMonitorResourceJoint(ctx context.Context, userCred mcclient.TokenCredential) error
	GetMoreDetails(ctx context.Context, out monitor.CommonAlertDetails) (monitor.CommonAlertDetails, error)
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

func (man *SCommonAlertManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SCommonAlertManager) Init() error {
	return nil
}

func (man *SCommonAlertManager) Run(ctx context.Context) error {
	alerts, err := man.GetAlerts(monitor.CommonAlertListInput{})
	if err != nil {
		return err
	}
	errs := make([]error, 0)
	for _, alert := range alerts {
		err := alert.UpdateMonitorResourceJoint(ctx, auth.AdminCredential())
		if err != nil {
			errs = append(errs, err)
		}

	}
	return errors.NewAggregate(errs)
}

func (man *SCommonAlertManager) UpdateAlertsResType(ctx context.Context, userCred mcclient.TokenCredential) error {
	alerts, err := man.GetAlerts(monitor.CommonAlertListInput{})
	if err != nil {
		return err
	}
	errs := make([]error, 0)
	for _, alert := range alerts {
		err := alert.UpdateResType()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (man *SCommonAlertManager) validateRoles(ctx context.Context, roles []string) ([]string, error) {
	ids := []string{}
	for _, role := range roles {
		roleCache, err := db.RoleCacheManager.FetchRoleByIdOrName(ctx, role)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch role by id or name: %q", role)
		}
		ids = append(ids, roleCache.GetId())
	}
	return ids, nil
}

func (man *SCommonAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.CommonAlertCreateInput) (monitor.CommonAlertCreateInput, error) {
	if data.Period == "" {
		data.Period = "5m"
	}
	if data.AlertDuration == 0 {
		data.AlertDuration = 1
	}
	if data.Name == "" {
		return data, merrors.NewArgIsEmptyErr("name")
	}
	if data.Level == "" {
		data.Level = monitor.CommonAlertLevelNormal
	}
	if len(data.Channel) == 0 {
		data.Channel = []string{monitor.DEFAULT_SEND_NOTIFY_CHANNEL}
	}
	//else {
	//	data.Channel = append(data.Channel, monitor.DEFAULT_SEND_NOTIFY_CHANNEL)
	//}

	if !utils.IsInStringArray(data.Level, monitor.CommonAlertLevels) {
		return data, httperrors.NewInputParameterError("Invalid level format: %s", data.Level)
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if data.SilentPeriod != "" {
		if _, err := time.ParseDuration(data.SilentPeriod); err != nil {
			return data, httperrors.NewInputParameterError("Invalid silent_period format: %s", data.SilentPeriod)
		}
	}
	// 默认的系统配置Recipients=commonalert-default
	if !utils.IsInStringArray(data.AlertType, []string{
		monitor.CommonAlertSystemAlertType,
		monitor.CommonAlertServiceAlertType,
	}) {
		if len(data.Recipients) == 0 && len(data.RobotIds) == 0 && len(data.Roles) == 0 {
			return data, merrors.NewArgIsEmptyErr("recipients, robot_ids or roles")
		}
	}

	if len(data.CommonMetricInputQuery.MetricQuery) == 0 {
		return data, merrors.NewArgIsEmptyErr("metric_query")
	} else {
		for _, query := range data.CommonMetricInputQuery.MetricQuery {
			if err := validateCommonAlertQuery(query); err != nil {
				return data, err
			}
			if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
				return data, httperrors.NewInputParameterError("the reduce is illegal: %s", query.Reduce)
			}
			/*if query.Threshold == 0 {
				return data, httperrors.NewInputParameterError("threshold is meaningless")
			}*/
			if strings.Contains(query.From, "now-") || strings.Contains(query.To, "now") {
				query.To = "now"
				query.From = "1h"
			}
		}
	}

	if data.AlertType != "" {
		if !utils.IsInStringArray(data.AlertType, validators.CommonAlertType) {
			return data, httperrors.NewInputParameterError("Invalid AlertType: %s", data.AlertType)
		}
	}
	var err = man.ValidateMetricQuery(&data.CommonMetricInputQuery, data.Scope, ownerId, true)
	if err != nil {
		return data, errors.Wrap(err, "metric query error")
	}

	// validate role
	if len(data.Roles) != 0 {
		roleIds, err := man.validateRoles(ctx, data.Roles)
		if err != nil {
			return data, errors.Wrap(err, "validateRole")
		}
		data.Roles = roleIds
		if !utils.IsInStringArray(data.Scope, []string{
			notiapi.SUBSCRIBER_SCOPE_SYSTEM,
			notiapi.SUBSCRIBER_SCOPE_DOMAIN,
			notiapi.SUBSCRIBER_SCOPE_PROJECT,
		}) {
			return data, httperrors.NewInputParameterError("unsupport scope %s", data.Scope)
		}
	}

	name, err := man.genName(ctx, ownerId, data.Name)
	if err != nil {
		return data, err
	}
	data.Name = name

	alertCreateInput, err := man.toAlertCreatInput(data)
	if err != nil {
		return data, errors.Wrap(err, "to alert creation input")
	}
	alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertCreateInput)
	if err != nil {
		return data, err
	}
	data.AlertCreateInput = alertCreateInput
	return data, nil
}

func (man *SCommonAlertManager) genName(ctx context.Context, ownerId mcclient.IIdentityProvider, name string) (string,
	error) {
	lockman.LockRawObject(ctx, man.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

	name, err := db.GenerateName(ctx, man, ownerId, name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (man *SCommonAlertManager) ValidateMetricQuery(metricRequest *monitor.CommonMetricInputQuery, scope string, ownerId mcclient.IIdentityProvider, isAlert bool) error {
	for _, q := range metricRequest.MetricQuery {
		metriInputQuery := monitor.MetricQueryInput{
			From:     metricRequest.From,
			To:       metricRequest.To,
			Interval: metricRequest.Interval,
		}
		setDefaultValue(q.AlertQuery, &metriInputQuery, scope, ownerId, isAlert)
		err := UnifiedMonitorManager.ValidateInputQuery(q.AlertQuery, &metriInputQuery)
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
	ret := alert.GetMetadata(context.Background(), CommonAlertMetadataAlertType, nil)
	return ret
}

func (alert *SCommonAlert) setFieldOpt(ctx context.Context, userCred mcclient.TokenCredential, fieldOpt string) error {
	return alert.SetMetadata(ctx, CommonAlertMetadataFieldOpt, fieldOpt, userCred)
}

func (alert *SCommonAlert) getFieldOpt() string {
	return alert.GetMetadata(context.Background(), CommonAlertMetadataFieldOpt, nil)
}

func (alert *SCommonAlert) setPointStr(ctx context.Context, userCred mcclient.TokenCredential, fieldOpt string) error {
	return alert.SetMetadata(ctx, CommonAlertMetadataPointStr, fieldOpt, userCred)
}

func (alert *SCommonAlert) getPointStr() string {
	return alert.GetMetadata(context.Background(), CommonAlertMetadataPointStr, nil)
}

func (alert *SCommonAlert) setMetaName(ctx context.Context, userCred mcclient.TokenCredential, metaName string) error {
	return alert.SetMetadata(ctx, CommonAlertMetadataName, metaName, userCred)
}

func (alert *SCommonAlert) getMetaName() string {
	return alert.GetMetadata(context.Background(), CommonAlertMetadataName, nil)
}

func (alert *SCommonAlert) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	err := alert.SMonitorScopedResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	alert.State = string(monitor.AlertStateUnknown)
	alert.LastStateChange = time.Now()
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

	if input.AlertType == monitor.CommonAlertSystemAlertType {
		s := &monitor.NotificationSettingOneCloud{
			Channel: "webconsole",
		}
		return alert.createAlertNoti(ctx, userCred, input.Name, s, input.SilentPeriod, true)
	}

	alert.SetChannel(ctx, input.Channel)
	for _, channel := range input.Channel {
		s := &monitor.NotificationSettingOneCloud{
			Channel: channel,
			UserIds: input.Recipients,
		}
		if err := alert.createAlertNoti(ctx, userCred, input.Name, s, input.SilentPeriod, false); err != nil {
			return errors.Wrap(err, fmt.Sprintf("create notify[channel is %s] error", channel))
		}
	}

	if len(input.RobotIds) != 0 {
		s := &monitor.NotificationSettingOneCloud{
			Channel:  string(notify.NotifyByRobot),
			RobotIds: input.RobotIds,
		}
		if err := alert.createAlertNoti(ctx, userCred, input.Name, s, input.SilentPeriod, false); err != nil {
			return errors.Wrapf(err, "create alert notification by robot %v", input.RobotIds)
		}
	}
	if len(input.Roles) != 0 {
		s := &monitor.NotificationSettingOneCloud{
			RoleIds: input.Roles,
		}
		if err := alert.createAlertNoti(ctx, userCred, input.Name, s, input.SilentPeriod, false); err != nil {
			return errors.Wrapf(err, "create alert notify by role %v, scope %q", input.Roles, input.Scope)
		}
	}
	return nil
}

func (alert *SCommonAlert) createAlertNoti(ctx context.Context, userCred mcclient.TokenCredential, notiName string, settings *monitor.NotificationSettingOneCloud, silentPeriod string, isSysNoti bool) error {
	noti, err := NotificationManager.CreateOneCloudNotification(ctx, userCred, notiName, settings, silentPeriod)
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

	alert.SetStatus(ctx, userCred, monitor.ALERT_STATUS_READY, "")

	if input.AlertType != "" {
		if err := alert.setAlertType(ctx, userCred, input.AlertType); err != nil {
			log.Errorf("etAlertType error for %q: %v", alert.GetId(), err)
		}
	}
	fieldOpt := ""
	for i, metricQ := range input.CommonMetricInputQuery.MetricQuery {
		if metricQ.FieldOpt != "" {
			if i == 0 {
				fieldOpt = string(metricQ.FieldOpt)
				continue
			}
			fieldOpt = fmt.Sprintf("%s+%s", fieldOpt, metricQ.FieldOpt)
		}
	}
	if fieldOpt != "" {
		alert.setFieldOpt(ctx, userCred, fieldOpt)
	}
	if input.GetPointStr {
		alert.setPointStr(ctx, userCred, strconv.FormatBool(input.GetPointStr))
	}
	if len(input.MetaName) != 0 {
		alert.setMetaName(ctx, userCred, input.MetaName)
	}
	_, err := alert.PerformSetScope(ctx, userCred, query, data)
	if err != nil {
		log.Errorln(errors.Wrap(err, "Alert PerformSetScope"))
	}
	CommonAlertManager.SetSubscriptionAlert(alert)
	//alert.StartUpdateMonitorAlertJointTask(ctx, userCred)
}

func (man *SCommonAlertManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.CommonAlertListInput,
) (*sqlchemy.SQuery, error) {
	// 如果指定了时间段和 top 参数，执行特殊的 top 查询
	if query.Top != nil {
		return man.getTopAlertsByResourceCount(ctx, q, userCred, query)
	}

	q, err := man.SAlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	man.FieldListFilter(q, query)

	return q, nil
}

func (man *SCommonAlertManager) FieldListFilter(q *sqlchemy.SQuery, input monitor.CommonAlertListInput) {
	// if len(input.UsedBy) == 0 {
	// 	q.Filter(sqlchemy.IsNull(q.Field("used_by")))
	// } else {
	// 	q.Equals("used_by", input.UsedBy)
	// }
	if len(input.UsedBy) != 0 {
		q.Equals("used_by", input.UsedBy)
	}

	if len(input.Level) > 0 {
		q.Equals("level", input.Level)
	}
	if len(input.Name) != 0 {
		q.Contains("name", input.Name)
	}
}

// getTopAlertsByResourceCount 查询指定时间段内报警资源最多的 top N 监控策略
func (man *SCommonAlertManager) getTopAlertsByResourceCount(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.CommonAlertListInput,
) (*sqlchemy.SQuery, error) {
	// 验证时间段和 top 参数
	startTime, endTime, top, err := validateTopQueryInput(query.TopQueryInput)
	if err != nil {
		return nil, err
	}

	// 查询指定时间段内的 AlertRecord
	recordQuery := AlertRecordManager.Query("alert_id", "res_ids")
	recordQuery = recordQuery.GE("created_at", startTime).LE("created_at", endTime)
	recordQuery = recordQuery.IsNotNull("res_type").IsNotEmpty("res_type")
	recordQuery = recordQuery.IsNotEmpty("res_ids")

	// 应用权限过滤
	recordQuery, err = AlertRecordManager.SScopedResourceBaseManager.ListItemFilter(
		ctx, recordQuery, userCred, query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "AlertRecordManager.ListItemFilter")
	}

	// 执行查询获取所有记录
	type RecordRow struct {
		AlertId string
		ResIds  string
	}
	rows := make([]RecordRow, 0)
	err = recordQuery.All(&rows)
	if err != nil {
		return nil, errors.Wrap(err, "query alert records")
	}

	// 统计每个 alert_id 的唯一资源数量
	alertResourceCount := make(map[string]sets.String)
	for _, row := range rows {
		if len(row.ResIds) == 0 {
			continue
		}
		// 解析 res_ids（逗号分隔）
		resIds := strings.Split(row.ResIds, ",")
		if alertResourceCount[row.AlertId] == nil {
			alertResourceCount[row.AlertId] = sets.NewString()
		}
		for _, resId := range resIds {
			resId = strings.TrimSpace(resId)
			if len(resId) > 0 {
				alertResourceCount[row.AlertId].Insert(resId)
			}
		}
	}

	// 转换为切片并按资源数量排序
	type AlertCount struct {
		AlertId string
		Count   int
	}
	alertCounts := make([]AlertCount, 0, len(alertResourceCount))
	for alertId, resSet := range alertResourceCount {
		alertCounts = append(alertCounts, AlertCount{
			AlertId: alertId,
			Count:   resSet.Len(),
		})
	}

	// 按资源数量降序排序
	for i := 0; i < len(alertCounts)-1; i++ {
		for j := i + 1; j < len(alertCounts); j++ {
			if alertCounts[i].Count < alertCounts[j].Count {
				alertCounts[i], alertCounts[j] = alertCounts[j], alertCounts[i]
			}
		}
	}

	// 获取 top N 的 alert_id
	topAlertIds := make([]string, 0, top)
	for i := 0; i < top && i < len(alertCounts); i++ {
		topAlertIds = append(topAlertIds, alertCounts[i].AlertId)
	}

	if len(topAlertIds) == 0 {
		// 如果没有找到任何记录，返回空查询
		return q.FilterByFalse(), nil
	}

	// 用 top alert_id 过滤 CommonAlert 查询
	q, err = man.SAlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	man.FieldListFilter(q, query)
	q = q.In("id", topAlertIds)

	return q, nil
}

func (manager *SCommonAlertManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)
	if keys.Contains("tenant") {
		if projectId, ok := rowMap["tenant_id"]; ok && projectId != "" {
			tenant, err := db.TenantCacheManager.FetchTenantById(ctx, projectId)
			if err == nil {
				res.Set("tenant", jsonutils.NewString(fmt.Sprintf("%s/%s", tenant.GetName(),
					tenant.GetProjectDomain())))
			}
		} else {
			tenant, err := db.TenantCacheManager.FetchDomainById(ctx, rowMap["domain_id"])
			if err == nil {
				dictionaryVal := GetGlobalSettingsDictionary(ctx, "domain")
				res.Set("tenant", jsonutils.NewString(fmt.Sprintf("%s%s", tenant.GetProjectDomain(), dictionaryVal)))
			}
		}
	}
	return res
}

func GetGlobalSettingsDictionary(ctx context.Context, param string) (val string) {
	s := auth.GetAdminSession(ctx, "")
	globalSettings, err := yunionconf.Parameters.GetGlobalSettings(s, jsonutils.NewDict())
	if err != nil {
		log.Errorf("GetGlobalSettings err:%v", err)
		return
	}
	dictionary, err := globalSettings.Get("value", "dictionary")
	if err != nil {
		log.Errorf("can not get dictionary:%s", globalSettings.String())
		return
	}
	lang := appctx.Lang(ctx)
	switch lang {
	case language.English:
		val, _ = dictionary.GetString("en", param)
	default:
		val, _ = dictionary.GetString("zh", param)
	}
	return
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
				return true, nil
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
			settings := obj.Settings
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

	if len(input.ResType) != 0 {
		mF := func(obj *SCommonAlert) (bool, error) {
			settings := obj.Settings
			for _, s := range settings.Conditions {
				if mesurement, contain := MetricMeasurementManager.measurementsCache.Get(s.Query.Model.
					Measurement); contain {
					if utils.IsInStringArray(mesurement.ResType, input.ResType) {
						return true, nil
					}
				}
			}
			return false, nil
		}
		filters.Append(wrapF(mF))
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

func (manager *SCommonAlertManager) GetAlerts(input monitor.CommonAlertListInput) ([]SCommonAlert, error) {
	alerts := make([]SCommonAlert, 0)
	query := manager.Query()
	manager.FieldListFilter(query, input)
	err := db.FetchModelObjects(manager, query, &alerts)
	if err != nil {
		return nil, errors.Wrap(err, "SCommonAlertManagerGetAlerts err")
	}
	return alerts, nil
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
	alertIds := make([]string, len(objs))
	for i := range rows {
		rows[i].AlertDetails = alertRows[i]
		alert := objs[i].(*SCommonAlert)
		alertIds[i] = alert.Id
	}
	ans := []SAlertnotification{}
	err := AlertNotificationManager.Query().In("alert_id", alertIds).Desc("index").All(&ans)
	if err != nil {
		log.Errorf("q.All")
		return rows
	}
	alertNotificationMap := make(map[string][]SAlertnotification)
	notificateIds := []string{}
	for i := range ans {
		_, ok := alertNotificationMap[ans[i].AlertId]
		if !ok {
			alertNotificationMap[ans[i].AlertId] = []SAlertnotification{}
		}
		notificateIds = append(notificateIds, ans[i].NotificationId)
		alertNotificationMap[ans[i].AlertId] = append(alertNotificationMap[ans[i].AlertId], ans[i])
	}
	notis := make(map[string]SNotification)
	err = db.FetchModelObjectsByIds(NotificationManager, "id", notificateIds, notis)
	if err != nil {
		log.Errorf("db.FetchModelObjectsByIds err:%v", err)
		return rows
	}
	metas := []db.SMetadata{}
	err = db.Metadata.Query().In("obj_id", alertIds).Equals("obj_type", man.Keyword()).Equals("key", CommonAlertMetadataAlertType).All(&metas)
	if err != nil {
		log.Errorf("db.MetadataManager.Query err:%v", err)
		return rows
	}
	metaMap := make(map[string]string)
	for i := range metas {
		metaMap[metas[i].ObjId] = metas[i].Value
	}
	for i := range rows {
		alert := objs[i].(*SCommonAlert)
		if alertNotis, ok := alertNotificationMap[alertIds[i]]; ok {
			channel := sets.String{}
			for j, alertNoti := range alertNotis {
				noti, ok := notis[alertNoti.NotificationId]
				if !ok {
					continue
				}
				settings := new(monitor.NotificationSettingOneCloud)
				if err := noti.Settings.Unmarshal(settings); err != nil {
					log.Errorf("Unmarshal to NotificationSettingOneCloud err:%v", err)
					return rows
				}
				if j == 0 {
					rows[i].Recipients = settings.UserIds
				}
				if !utils.IsInStringArray(settings.Channel,
					[]string{monitor.DEFAULT_SEND_NOTIFY_CHANNEL, string(notify.NotifyByRobot)}) {
					channel.Insert(settings.Channel)
				}
				if noti.Frequency != 0 {
					rows[i].SilentPeriod = fmt.Sprintf("%dm", noti.Frequency/60)
				}
				if len(settings.RobotIds) != 0 {
					rows[i].RobotIds = settings.RobotIds
				}
				if len(settings.RoleIds) != 0 {
					rows[i].RoleIds = settings.RoleIds
				}
			}
			rows[i].Channel = channel.List()
		}
		rows[i].AlertType = metaMap[alertIds[i]]
		if alert.Frequency < 60 {
			rows[i].Period = fmt.Sprintf("%ds", alert.Frequency)
		} else {
			rows[i].Period = fmt.Sprintf("%dm", alert.Frequency/60)
		}
		rows[i].AlertDuration = alert.For / alert.Frequency
		if rows[i].AlertDuration == 0 {
			rows[i].AlertDuration = 1
		}

		err := alert.getCommonAlertMetricDetails(&rows[i])
		if err != nil {
			log.Errorf("getCommonAlertMetricDetails err:%v", err)
			return rows
		}
	}
	return rows
}

func (alert *SCommonAlert) ValidateDeleteCondition(ctx context.Context, info *monitor.CommonAlertDetails) error {
	if gotypes.IsNil(info) {
		info = &monitor.CommonAlertDetails{}
	}
	if info.AlertType == monitor.CommonAlertSystemAlertType {
		return httperrors.NewInputParameterError("Cannot delete system alert")
	}
	return nil
}

func (alert *SCommonAlert) GetMoreDetails(ctx context.Context, out monitor.CommonAlertDetails) (monitor.CommonAlertDetails, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	token := s.GetToken()
	ret := CommonAlertManager.FetchCustomizeColumns(ctx, token, jsonutils.NewDict(), []interface{}{alert}, stringutils2.SSortedStrings{}, false)
	if len(ret) != 1 {
		return out, errors.Wrapf(cloudprovider.ErrNotFound, "FetchCustomizeColumns")
	}
	return ret[0], nil
}

func (alert *SCommonAlert) getCommonAlertMetricDetails(out *monitor.CommonAlertDetails) error {
	metricDetails, err := alert.GetCommonAlertMetricDetails()
	if err != nil {
		return err
	}
	out.CommonAlertMetricDetails = metricDetails
	return nil
}

func (alert *SCommonAlert) GetCommonAlertMetricDetails() ([]*monitor.CommonAlertMetricDetails, error) {
	setting, err := alert.GetSettings()
	if err != nil {
		return nil, errors.Wrap(err, "get alert settings")
	}
	if len(setting.Conditions) == 0 {
		return nil, nil
	}
	ret := make([]*monitor.CommonAlertMetricDetails, len(setting.Conditions))
	for i, cond := range setting.Conditions {
		metricDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(i, &cond)
		ret[i] = metricDetails
		setting.Conditions[i] = cond
	}
	// side effect, update setting cause of setting.Conditions has changed by GetCommonAlertMetricDetailsFromAlertCondition
	alert.Settings = setting
	return ret, nil
}

func (alert *SCommonAlert) GetCommonAlertMetricDetailsFromAlertCondition(index int, cond *monitor.AlertCondition) *monitor.CommonAlertMetricDetails {
	fieldOpt := alert.getFieldOpt()
	metricDetails := new(monitor.CommonAlertMetricDetails)
	if fieldOpt != "" {
		metricDetails.FieldOpt = strings.Split(fieldOpt, "+")[index]
	}
	poinStr := alert.getPointStr()
	if len(poinStr) != 0 {
		bl, _ := strconv.ParseBool(poinStr)
		metricDetails.GetPointStr = bl
	}
	getCommonAlertMetricDetailsFromCondition(cond, metricDetails)
	return metricDetails
}

func getCommonAlertMetricDetailsFromCondition(
	cond *monitor.AlertCondition,
	metricDetails *monitor.CommonAlertMetricDetails,
) {
	cmp := ""
	switch cond.Evaluator.Type {
	case "gt":
		cmp = ">"
	case "eq":
		cmp = "=="
	case "lt":
		cmp = "<"
	case "within_range":
		cmp = "within_range"
	case "outside_range":
		cmp = "outside_range"
	}
	metricDetails.Comparator = cmp

	// 处理 ranged types
	if utils.IsInStringArray(cond.Evaluator.Type, validators.EvaluatorRangedTypes) {
		if len(cond.Evaluator.Params) >= 2 {
			metricDetails.ThresholdRange = []float64{cond.Evaluator.Params[0], cond.Evaluator.Params[1]}
		}
	} else {
		// 处理默认 types
		if len(cond.Evaluator.Params) != 0 {
			metricDetails.Threshold = cond.Evaluator.Params[0]
		}
	}
	metricDetails.Reduce = cond.Reducer.Type

	metricDetails.ConditionType = cond.Type
	if metricDetails.ConditionType == monitor.METRIC_QUERY_TYPE_NO_DATA {
		metricDetails.ThresholdStr = monitor.METRIC_QUERY_NO_DATA_THESHOLD
		metricDetails.Comparator = monitor.METRIC_QUERY_NO_DATA_THESHOLD
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
	cond.Query.Model.Tags = filterDefaultTags(q.Model.Tags)
	metricDetails.Measurement = measurement
	metricDetails.Field = field
	metricDetails.DB = db
	metricDetails.Groupby = groupby
	metricDetails.Filters = cond.Query.Model.Tags
	metricDetails.Operator = cond.Operator

	//fill measurement\field desciption info
	getMetricDescriptionDetails(metricDetails)
}

func filterDefaultTags(queryTag []monitor.MetricQueryTag) []monitor.MetricQueryTag {
	newQueryTags := make([]monitor.MetricQueryTag, 0)
	for i, tagFilter := range queryTag {
		if tagFilter.Key == "tenant_id" {
			continue
		}
		if tagFilter.Key == "domain_id" {
			continue
		}
		if tagFilter.Key == hostconsts.TELEGRAF_TAG_KEY_RES_TYPE {
			continue
		}
		newQueryTags = append(newQueryTags, queryTag[i])
	}
	return newQueryTags
}

func getMetricDescriptionDetails(metricDetails *monitor.CommonAlertMetricDetails) {
	influxdbMeasurements := DataSourceManager.getMetricDescriptions([]monitor.InfluxMeasurement{
		{Measurement: metricDetails.Measurement},
	})
	if len(influxdbMeasurements) == 0 {
		return
	}
	if len(influxdbMeasurements[0].MeasurementDisplayName) != 0 {
		metricDetails.MeasurementDisplayName = influxdbMeasurements[0].MeasurementDisplayName
	}
	if len(influxdbMeasurements[0].ResType) != 0 {
		metricDetails.ResType = influxdbMeasurements[0].ResType
	}
	fields := make([]string, 0)
	if len(metricDetails.FieldOpt) != 0 {
		fields = append(fields, strings.Split(metricDetails.Field, metricDetails.FieldOpt)...)
	} else {
		fields = append(fields, metricDetails.Field)
	}
	for _, field := range fields {
		if influxdbMeasurements[0].FieldDescriptions == nil {
			return
		}
		if fieldDes, ok := influxdbMeasurements[0].FieldDescriptions[field]; ok {
			metricDetails.FieldDescription = fieldDes
			if utils.IsInStringArray(metricDetails.FieldDescription.Unit, []string{
				monitor.METRIC_UNIT_COUNT,
				monitor.METRIC_UNIT_NULL,
			}) {
				metricDetails.FieldDescription.Unit = ""
			}
			if len(metricDetails.FieldOpt) != 0 {
				metricDetails.FieldDescription.Name = metricDetails.Field
				metricDetails.FieldDescription.DisplayName = metricDetails.Field
				getExtraFieldDetails(metricDetails)
				break
			}
		}
	}
}

func getExtraFieldDetails(metricDetails *monitor.CommonAlertMetricDetails) {
	if metricDetails.FieldOpt == string(monitor.CommonAlertFieldOptDivision) && metricDetails.Threshold < float64(1) {
		metricDetails.Threshold = metricDetails.Threshold * float64(100)
		metricDetails.FieldDescription.Unit = "%"
	}
}

func getQueryEvalType(evalType string) monitor.EvaluatorType {
	var typ monitor.EvaluatorType
	switch evalType {
	case ">=", ">":
		typ = monitor.EvaluatorTypeGT
	case "<=", "<":
		typ = monitor.EvaluatorTypeLT
	case "==":
		typ = monitor.EvaluatorTypeEQ
	case "within_range":
		typ = monitor.EvaluatorTypeWithinRange
	case "outside_range":
		typ = monitor.EvaluatorTypeOutsideRange
	}
	return typ
}

// validateCommonAlertQuery 校验 CommonAlertQuery 的 comparator 和 threshold_range
func validateCommonAlertQuery(query *monitor.CommonAlertQuery) error {
	if query.ConditionType == monitor.METRIC_QUERY_TYPE_NO_DATA {
		query.Comparator = "=="
	}
	evalType := getQueryEvalType(query.Comparator)
	if !sets.NewString(append(
		validators.EvaluatorDefaultTypes,
		validators.EvaluatorRangedTypes...)...).Has(string(evalType)) {
		return httperrors.NewInputParameterError("the Comparator is illegal: %s", query.Comparator)
	}
	// 验证 ranged types 的参数
	if utils.IsInStringArray(string(evalType), validators.EvaluatorRangedTypes) {
		if len(query.ThresholdRange) < 2 {
			return httperrors.NewInputParameterError("threshold_range or outside_range requires 2 parameters, got %d", len(query.ThresholdRange))
		}
		// 确保第一项小于等于第二项
		if query.ThresholdRange[0] > query.ThresholdRange[1] {
			return httperrors.NewInputParameterError("threshold_range first value (%v) must be less than or equal to second value (%v)", query.ThresholdRange[0], query.ThresholdRange[1])
		}
	}
	return nil
}

// validateComparatorAndThreshold 校验字符串形式的 comparator, threshold 和 threshold_range
func validateComparatorAndThreshold(comparator string, threshold string, thresholdRange []jsonutils.JSONObject) error {
	var evalType monitor.EvaluatorType
	if len(comparator) != 0 {
		evalType = getQueryEvalType(comparator)
		if !utils.IsInStringArray(string(evalType), append(validators.EvaluatorDefaultTypes, validators.EvaluatorRangedTypes...)) {
			return httperrors.NewInputParameterError("the Comparator is illegal: %s", comparator)
		}
		// 验证 ranged types 的参数
		if utils.IsInStringArray(string(evalType), validators.EvaluatorRangedTypes) {
			if len(thresholdRange) < 2 {
				return httperrors.NewInputParameterError("threshold_range or outside_range requires 2 parameters, got %d", len(thresholdRange))
			}
		}
	}
	if len(threshold) != 0 {
		_, err := strconv.ParseFloat(threshold, 64)
		if err != nil {
			return httperrors.NewInputParameterError("threshold:%s should be number type", threshold)
		}
	}
	if len(thresholdRange) > 0 {
		if len(thresholdRange) < 2 {
			return httperrors.NewInputParameterError("threshold_range requires 2 parameters, got %d", len(thresholdRange))
		}
		vals := make([]float64, len(thresholdRange))
		for i, val := range thresholdRange {
			parsedVal, err := strconv.ParseFloat(val.String(), 64)
			if err != nil {
				return httperrors.NewInputParameterError("threshold_range[%d]: %s should be number type", i, val.String())
			}
			vals[i] = parsedVal
		}
		// 确保第一项小于等于第二项
		if vals[0] > vals[1] {
			return httperrors.NewInputParameterError("threshold_range first value (%v) must be less than or equal to second value (%v)", vals[0], vals[1])
		}
	}
	return nil
}

func (man *SCommonAlertManager) toAlertCreatInput(input monitor.CommonAlertCreateInput) (monitor.AlertCreateInput, error) {
	freq, _ := time.ParseDuration(input.Period)
	ret := new(monitor.AlertCreateInput)
	ret.Name = input.Name
	ret.Frequency = int64(freq / time.Second)
	if input.AlertDuration != 1 {
		ret.For = ret.Frequency * input.AlertDuration
	}
	ret.Level = input.Level
	//ret.Settings =monitor.AlertSetting{}
	for _, metricquery := range input.CommonMetricInputQuery.MetricQuery {
		conditionType := "query"
		if len(metricquery.ConditionType) != 0 {
			conditionType = metricquery.ConditionType
		}
		evalType := getQueryEvalType(metricquery.Comparator)
		var evaluatorParams []float64
		// 处理 ranged types (within_range, outside_range)
		if utils.IsInStringArray(string(evalType), validators.EvaluatorRangedTypes) {
			if len(metricquery.ThresholdRange) < 2 {
				return *ret, httperrors.NewInputParameterError("threshold_range or outside_range requires 2 parameters, got %d", len(metricquery.ThresholdRange))
			}
			fieldOpt := monitor.CommonAlertFieldOpt(metricquery.FieldOpt)
			evaluatorParams = []float64{
				fieldOperatorThreshold(fieldOpt, metricquery.ThresholdRange[0]),
				fieldOperatorThreshold(fieldOpt, metricquery.ThresholdRange[1]),
			}
		} else {
			// 处理默认 types (gt, lt, eq)
			fieldOpt := monitor.CommonAlertFieldOpt(metricquery.FieldOpt)
			evaluatorParams = []float64{fieldOperatorThreshold(fieldOpt, metricquery.Threshold)}
		}
		condition := monitor.AlertCondition{
			Type:      conditionType,
			Query:     *metricquery.AlertQuery,
			Reducer:   monitor.Condition{Type: metricquery.Reduce},
			Evaluator: monitor.Condition{Type: string(evalType), Params: evaluatorParams},
			Operator:  "and",
		}
		if metricquery.Operator != "" {
			if !sets.NewString("and", "or").Has(metricquery.Operator) {
				return *ret, httperrors.NewInputParameterError("invalid operator %s", metricquery.Operator)
			}
			condition.Operator = metricquery.Operator
		}
		if metricquery.FieldOpt != "" {
			condition.Reducer.Operators = []string{string(metricquery.FieldOpt)}
		}
		ret.Settings.Conditions = append(ret.Settings.Conditions, condition)
	}
	return *ret, nil
}

func fieldOperatorThreshold(opt monitor.CommonAlertFieldOpt, threshold float64) float64 {
	if opt == monitor.CommonAlertFieldOptDivision && threshold > 1 {
		return threshold / float64(100)
	}
	return threshold
}

func (alert *SCommonAlert) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	generateName, _ := data.GetString("generate_name")
	if len(generateName) != 0 && alert.Name != generateName {
		name, err := db.GenerateName(ctx, CommonAlertManager, userCred, generateName)
		if err != nil {
			return data, err
		}
		data.Set("name", jsonutils.NewString(name))
	}
	statusUpdate := apis.StatusStandaloneResourceBaseUpdateInput{}
	data.Unmarshal(&statusUpdate)
	_, err := alert.SAlert.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, statusUpdate)
	if err != nil {
		return data, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	updataInput := new(monitor.CommonAlertUpdateInput)
	if period, _ := data.GetString("period"); len(period) > 0 {
		if _, err := time.ParseDuration(period); err != nil {
			return data, httperrors.NewInputParameterError("Invalid period format: %s", period)
		}
		if period != "" {
			frep, _ := time.ParseDuration(period)
			freqSpec := int64(frep / time.Second)
			dur, _ := data.Int("alert_duration")
			if dur > 1 {
				alertFor := freqSpec * dur
				data.Set("for", jsonutils.NewInt(alertFor))
			}
			data.Set("frequency", jsonutils.NewInt(freqSpec))
		}
	}
	if silentPeriod, _ := data.GetString("silent_period"); len(silentPeriod) > 0 {
		if _, err := time.ParseDuration(silentPeriod); err != nil {
			return data, httperrors.NewInputParameterError("Invalid silent_period format: %s", silentPeriod)
		}
	}

	tmp := jsonutils.NewArray()
	if metric_query, _ := data.GetArray("metric_query"); len(metric_query) > 0 {
		for i := range metric_query {
			query := new(monitor.CommonAlertQuery)
			err := metric_query[i].Unmarshal(query)
			if err != nil {
				return data, errors.Wrap(err, "metric_query Unmarshal error")
			}
			if err := validateCommonAlertQuery(query); err != nil {
				return data, err
			}
			if _, ok := monitor.AlertReduceFunc[query.Reduce]; !ok {
				return data, httperrors.NewInputParameterError("the reduce is illegal: %s", query.Reduce)
			}
			/*if query.Threshold == 0 {
				return data, httperrors.NewInputParameterError("threshold is meaningless")
			}*/
			if strings.Contains(query.From, "now-") {
				query.To = "now"
				query.From = "1h"
			}
			tmp.Add(jsonutils.Marshal(query))
		}
		data.Add(tmp, "metric_query")
		metricQuery := new(monitor.CommonMetricInputQuery)
		err := data.Unmarshal(metricQuery)
		if err != nil {
			return data, errors.Wrap(err, "metric_query Unmarshal error")
		}
		scope, _ := data.GetString("scope")
		ownerId := CommonAlertManager.GetOwnerId(ctx, userCred, data)
		err = CommonAlertManager.ValidateMetricQuery(metricQuery, scope, ownerId, true)
		if err != nil {
			return data, errors.Wrap(err, "metric query error")
		}
		if alert.getAlertType() == monitor.CommonAlertSystemAlertType {
			forceUpdate, _ := data.Bool("force_update")
			if !forceUpdate {
				return data, nil
			}
		}
		data.Update(jsonutils.Marshal(metricQuery))
		err = data.Unmarshal(updataInput)
		if err != nil {
			return data, errors.Wrap(err, "updataInput Unmarshal err")
		}
		alertCreateInput, err := alert.getUpdateAlertInput(*updataInput)
		if err != nil {
			return data, errors.Wrap(err, "getUpdateAlertInput")
		}
		alertCreateInput, err = AlertManager.ValidateCreateData(ctx, userCred, nil, query, alertCreateInput)
		if err != nil {
			return data, err
		}
		data.Set("settings", jsonutils.Marshal(&alertCreateInput.Settings))
		updataInput.AlertUpdateInput, err = alert.SAlert.ValidateUpdateData(ctx, userCred, query, updataInput.AlertUpdateInput)
		if err != nil {
			return data, errors.Wrap(err, "SAlert.ValidateUpdateData")
		}
		updataInput.For = alertCreateInput.For
		data.Update(jsonutils.Marshal(updataInput))
	}
	return data, nil
}

func (manager *SCommonAlertManager) GetOwnerId(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) mcclient.IIdentityProvider {
	ownId, _ := CommonAlertManager.FetchOwnerId(ctx, data)
	if ownId == nil {
		ownId = userCred
	}
	return ownId
}

func (alert *SCommonAlert) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	updateInput := new(monitor.CommonAlertUpdateInput)
	data.Unmarshal(updateInput)
	if len(updateInput.Channel) != 0 {
		if err := alert.UpdateNotification(ctx, userCred, query, data); err != nil {
			log.Errorf("update notification error: %v", err)
		}
		alert.SetChannel(ctx, updateInput.Channel)
	}
	if _, err := data.GetString("scope"); err == nil {
		_, err = alert.PerformSetScope(ctx, userCred, query, data)
		if err != nil {
			log.Errorf("Alert PerformSetScope: %v", err)
		}
	}
	if updateInput.GetPointStr {
		alert.setPointStr(ctx, userCred, strconv.FormatBool(updateInput.GetPointStr))
	}
	if len(updateInput.MetaName) != 0 {
		alert.setMetaName(ctx, userCred, updateInput.MetaName)
	}
	CommonAlertManager.SetSubscriptionAlert(alert)
	//alert.StartUpdateMonitorAlertJointTask(ctx, userCred)
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
		log.Errorf("customizeCreateNotis for %s(%s): %v", alert.GetName(), alert.GetId(), err)
	}
	return err
}

func (alert *SCommonAlert) getUpdateAlertInput(updateInput monitor.CommonAlertUpdateInput) (monitor.AlertCreateInput, error) {
	input := monitor.CommonAlertCreateInput{
		CommonMetricInputQuery: updateInput.CommonMetricInputQuery,
		Period:                 updateInput.Period,
	}
	input.AlertDuration = updateInput.AlertDuration
	return CommonAlertManager.toAlertCreatInput(input)
}

func (alert *SCommonAlert) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	alert.SetStatus(ctx, userCred, monitor.ALERT_STATUS_DELETING, "")
	err := alert.customizeDeleteNotis(ctx, userCred, query, data)
	if err != nil {
		alert.SetStatus(ctx, userCred, monitor.ALERT_STATUS_DELETE_FAIL, "")
		return errors.Wrap(err, "customizeDeleteNotis")
	}
	alert.StartDeleteTask(ctx, userCred)
	alert.StartDetachMonitorAlertJointTask(ctx, userCred)
	return alert.SAlert.CustomizeDelete(ctx, userCred, query, data)
}

func (alert *SCommonAlert) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return alert.SStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCommonAlert) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) {
	RunModelTask("DeleteAlertRecordTask", self, func() error {
		onErr := func(err error) {
			msg := jsonutils.NewString(err.Error())
			// db.OpsLog.LogEvent(self, db.ACT_DELETE_FAIL, msg, userCred)
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_DELETE, msg, userCred, false)
		}

		errs := self.DeleteAttachAlertRecords(ctx, userCred)
		if len(errs) != 0 {
			err := errors.Wrapf(errors.NewAggregate(errs), "DeleteAttachAlertRecords of %s", self.GetName())
			onErr(err)
			return err
		}
		if err := self.RealDelete(ctx, userCred); err != nil {
			err = errors.Wrapf(err, "RealDelete CommonAlert %s", self.GetName())
			onErr(err)
			return err
		}

		// db.OpsLog.LogEvent(self, db.ACT_DELETE, nil, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_DELETE, nil, userCred, true)
		return nil
	})
}

func (self *SCommonAlert) DeleteAttachAlertRecords(ctx context.Context, userCred mcclient.TokenCredential) (errs []error) {
	records, err := AlertRecordManager.GetAlertRecordsByAlertId(self.GetId())
	if err != nil {
		errs = append(errs, errors.Wrap(err, "GetAlertRecordsByAlertId error"))
		return
	}
	for i, _ := range records {
		err := records[i].Delete(ctx, userCred)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "delete attach record:%s error", records[i].GetId()))
		}
	}
	return
}

func (alert *SCommonAlert) customizeDeleteNotis(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return alert.deleteNotifications(ctx, userCred, query, data)
}

func (alert *SCommonAlert) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	CommonAlertManager.DeleteSubscriptionAlert(alert)

	return nil
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

func (alert *SCommonAlert) PerformSetScope(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	domainId := jsonutils.GetAnyString(data, []string{"domain_id", "domain", "project_domain_id", "project_domain"})
	projectId := jsonutils.GetAnyString(data, []string{"project_id", "project"})
	if len(domainId) == 0 && len(projectId) == 0 {
		scope, _ := data.GetString("scope")
		if len(scope) != 0 {
			switch rbacscope.TRbacScope(scope) {
			case rbacscope.ScopeSystem:

			case rbacscope.ScopeDomain:
				domainId = userCred.GetProjectDomainId()
				data.(*jsonutils.JSONDict).Set("domain_id", jsonutils.NewString(domainId))
			case rbacscope.ScopeProject:
				projectId = userCred.GetProjectId()
				data.(*jsonutils.JSONDict).Set("project_id", jsonutils.NewString(projectId))
			}
		}
	}
	return db.PerformSetScope(ctx, alert, userCred, data)
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
	case "res_type":
		resTypeQuery := MetricMeasurementManager.Query("res_type").Distinct()
		return resTypeQuery, nil
	}
	return q, httperrors.ErrNotFound
}

func (alert *SCommonAlert) PerformConfig(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	period, _ := data.GetString("period")
	comparator, _ := data.GetString("comparator")
	threshold, _ := data.GetString("threshold")
	thresholdRange, _ := data.GetArray("threshold_range")
	if len(period) != 0 {
		if _, err := time.ParseDuration(period); err != nil {
			return data, httperrors.NewInputParameterError("Invalid period format: %s", period)
		}
	}
	reason, _ := data.GetString("reason")
	if err := validateComparatorAndThreshold(comparator, threshold, thresholdRange); err != nil {
		return data, err
	}
	_, err := db.Update(alert, func() error {
		if len(period) != 0 {
			freq, _ := time.ParseDuration(period)
			alert.Frequency = int64(freq / time.Second)
		}
		setting, _ := alert.GetSettings()
		if len(comparator) != 0 {
			evalType := getQueryEvalType(comparator)
			setting.Conditions[0].Evaluator.Type = string(evalType)
		}
		// 处理 ranged types
		if len(thresholdRange) >= 2 {
			vals := make([]float64, 2)
			for i := 0; i < 2 && i < len(thresholdRange); i++ {
				val, _ := strconv.ParseFloat(thresholdRange[i].String(), 64)
				vals[i] = fieldOperatorThreshold("", val)
			}
			setting.Conditions[0].Evaluator.Params = vals
		} else if len(threshold) != 0 {
			val, _ := strconv.ParseFloat(threshold, 64)
			setting.Conditions[0].Evaluator.Params = []float64{fieldOperatorThreshold("", val)}
		}
		alert.Settings = setting
		if len(reason) != 0 {
			alert.Reason = reason
		}
		return nil
	})
	PerformConfigLog(alert, userCred)
	return jsonutils.Marshal(alert), err
}

func PerformConfigLog(model db.IModel, userCred mcclient.TokenCredential) {
	// db.OpsLog.LogEvent(model, db.ACT_UPDATE_RULE, "", userCred)
	logclient.AddSimpleActionLog(model, logclient.ACT_UPDATE_RULE, nil, userCred, true)
}

func (alert *SCommonAlert) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(alert, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	//alert.StartUpdateMonitorAlertJointTask(ctx, userCred)
	return nil, nil
}

func (alert *SCommonAlert) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(alert, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	err = alert.StartDetachTask(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "alert StartDetachTask error")
	}
	return nil, nil
}

func (alert *SCommonAlert) StartDetachTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	RunModelTask("DetachAlertResourceTask", alert, func() error {
		onErr := func(err error) {
			msg := jsonutils.NewString(err.Error())
			// db.OpsLog.LogEvent(alert, db.ACT_DETACH, msg, userCred)
			logclient.AddActionLogWithContext(ctx, alert, logclient.ACT_DETACH_ALERTRESOURCE, msg, userCred, false)
		}
		errs := alert.DetachAlertResourceOnDisable(ctx, userCred)
		if len(errs) != 0 {
			err := errors.Wrapf(errors.NewAggregate(errs), "DetachAlertResourceOnDisable of alert %s", alert.GetName())
			onErr(err)
			return err
		}
		if err := MonitorResourceAlertManager.DetachJoint(ctx, userCred,
			monitor.MonitorResourceJointListInput{AlertId: alert.GetId()}); err != nil {
			log.Errorf("DetachJoint when alert(%s) disable: %v", alert.GetName(), err)
		}
		logclient.AddActionLogWithContext(ctx, alert, logclient.ACT_DETACH_ALERTRESOURCE, nil, userCred, true)
		return nil
	})
	return nil
}

func (alert *SCommonAlert) DetachAlertResourceOnDisable(ctx context.Context,
	userCred mcclient.TokenCredential) (errs []error) {
	return CommonAlertManager.DetachAlertResourceByAlertId(ctx, userCred, alert.Id)
}

func (manager *SCommonAlertManager) DetachAlertResourceByAlertId(ctx context.Context,
	userCred mcclient.TokenCredential, alertId string) (errs []error) {
	resources, err := GetAlertResourceManager().getResourceFromAlertId(alertId)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "getResourceFromAlert error"))
		return
	}
	for _, resource := range resources {
		err := resource.DetachAlert(ctx, userCred, alertId)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "resource:%s DetachAlert:%s err", resource.Id, alertId))
		}
	}
	return
}

func (alert *SCommonAlert) StartUpdateMonitorAlertJointTask(ctx context.Context, userCred mcclient.TokenCredential) {
	RunModelTask("UpdateMonitorResourceJointTask", alert, func() error {
		if err := alert.UpdateMonitorResourceJoint(ctx, userCred); err != nil {
			return errors.Wrapf(err, "UpdateMonitorResourceJoint of alert %s", alert.GetName())
		}
		return nil
	})
}

func (alert *SCommonAlert) UpdateMonitorResourceJoint(ctx context.Context, userCred mcclient.TokenCredential) error {
	var resType string
	setting, _ := alert.GetSettings()
	for _, con := range setting.Conditions {
		measurement, _ := MetricMeasurementManager.GetCache().Get(con.Query.Model.Measurement)
		if measurement == nil {
			resType = monitor.METRIC_RES_TYPE_HOST
		} else {
			resType = measurement.ResType
		}
	}
	ret, err := alert.TestRunAlert(userCred, monitor.AlertTestRunInput{})
	if err != nil {
		return errors.Wrapf(err, "TestRunAlert %s", alert.GetName())
	}
	if len(ret.AlertOKEvalMatches) > 0 {
		matches := make([]monitor.EvalMatch, len(ret.AlertOKEvalMatches))
		for i := range ret.AlertOKEvalMatches {
			matches[i] = *ret.AlertOKEvalMatches[i]
		}
		input := &UpdateMonitorResourceAlertInput{
			AlertId:       alert.GetId(),
			Matches:       matches,
			ResType:       resType,
			AlertState:    string(monitor.AlertStateOK),
			SendState:     monitor.SEND_STATE_SILENT,
			TriggerTime:   time.Now(),
			AlertRecordId: "",
		}
		if err := MonitorResourceManager.UpdateMonitorResourceAttachJoint(ctx, userCred, input); err != nil {
			return errors.Wrap(err, "UpdateMonitorResourceAttachJoint")
		}
		return nil
	}
	return nil
}

func (alert *SCommonAlert) StartDetachMonitorAlertJointTask(ctx context.Context, userCred mcclient.TokenCredential) {
	RunModelTask("DetachMonitorResourceJointTask", alert, func() error {
		if err := alert.DetachMonitorResourceJoint(ctx, userCred); err != nil {
			err = errors.Wrapf(err, "DetachMonitorResourceJoint of alert %s", alert.GetName())
			msg := jsonutils.NewString(err.Error())
			// db.OpsLog.LogEvent(alert, db.ACT_DETACH_MONITOR_RESOURCE_JOINT, msg, userCred)
			logclient.AddActionLogWithContext(ctx, alert, logclient.ACT_DETACH_MONITOR_RESOURCE_JOINT, msg, userCred, false)
			return err
		}
		logclient.AddActionLogWithContext(ctx, alert, logclient.ACT_DETACH_MONITOR_RESOURCE_JOINT, nil, userCred, true)
		return nil
	})
}

func (alert *SCommonAlert) DetachMonitorResourceJoint(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := MonitorResourceAlertManager.DetachJoint(ctx, userCred, monitor.MonitorResourceJointListInput{AlertId: alert.GetId()})
	if err != nil {
		return errors.Wrap(err, "SCommonAlert DetachJoint err")
	}
	return nil
}

func (alert *SCommonAlert) UpdateResType() error {
	setting, _ := alert.GetSettings()
	if len(setting.Conditions) == 0 {
		return nil
	}
	measurement, _ := MetricMeasurementManager.GetCache().Get(setting.Conditions[0].Query.Model.Measurement)
	if measurement == nil {
		return nil
	}
	_, err := db.Update(alert, func() error {
		alert.ResType = measurement.ResType
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "alert:%s UpdateResType err", alert.Name)
	}
	return nil
}

func (alert *SCommonAlert) GetSilentPeriod() (int64, error) {
	notis, err := alert.GetNotifications()
	if err != nil {
		return 0, errors.Wrap(err, "GetNotifications")
	}
	for _, n := range notis {
		noti, _ := n.GetNotification()
		if noti != nil && noti.Frequency != 0 {
			return noti.Frequency, nil
		}
	}
	return 0, nil
}

func (alert *SCommonAlert) GetAlertRules(silentPeriod int64) ([]*monitor.AlertRecordRule, error) {
	rules := make([]*monitor.AlertRecordRule, 0)
	settings, err := alert.GetSettings()
	if err != nil {
		return nil, errors.Wrapf(err, "get alert %s settings", alert.GetId())
	}
	for index := range settings.Conditions {
		rule := alert.GetAlertRule(settings, index, silentPeriod)
		rules = append(rules, rule)
	}
	return rules, nil
}

func (alert *SCommonAlert) GetAlertRule(settings *monitor.AlertSetting, index int, silentPeriod int64) *monitor.AlertRecordRule {
	alertDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(index, &settings.Conditions[index])
	rule := &monitor.AlertRecordRule{
		ResType:         alertDetails.ResType,
		Metric:          fmt.Sprintf("%s.%s", alertDetails.Measurement, alertDetails.Field),
		Measurement:     alertDetails.Measurement,
		Database:        alertDetails.DB,
		MeasurementDesc: alertDetails.MeasurementDisplayName,
		Field:           alertDetails.Field,
		FieldDesc:       alertDetails.FieldDescription.DisplayName,
		Comparator:      alertDetails.Comparator,
		Unit:            alertDetails.FieldDescription.Unit,
		Threshold:       RationalizeValueFromUnit(alertDetails.Threshold, alertDetails.FieldDescription.Unit, ""),
		ThresholdRange:  alertDetails.ThresholdRange,
		ConditionType:   alertDetails.ConditionType,
		Reducer:         alertDetails.Reduce,
	}
	if len(rule.ResType) == 0 {
		if alertDetails.DB == monitor.METRIC_DATABASE_TELE {
			rule.ResType = monitor.METRIC_RES_TYPE_HOST
		}
	}
	if alert.Frequency < 60 {
		rule.Period = fmt.Sprintf("%ds", alert.Frequency)
	} else {
		rule.Period = fmt.Sprintf("%dm", alert.Frequency/60)
	}
	rule.AlertDuration = alert.For / alert.Frequency
	if rule.AlertDuration == 0 {
		rule.AlertDuration = 1
	}
	if silentPeriod > 0 {
		rule.SilentPeriod = fmt.Sprintf("%dm", silentPeriod/60)
	}
	return rule
}

func (alert *SCommonAlert) GetResourceAlert(resourceId string, metric string) (*SMonitorResourceAlert, error) {
	return MonitorResourceAlertManager.GetResourceAlert(alert.GetId(), resourceId, metric)
}

func (alert *SCommonAlert) IsResourceMetricAlerting(resourceId string, metric string) (bool, error) {
	ra, err := alert.GetResourceAlert(resourceId, metric)
	if err != nil {
		return false, errors.Wrapf(err, "GetResourceAlert")
	}
	if ra.AlertState == string(monitor.AlertStateAlerting) {
		return true, nil
	}
	return false, nil
}

var fileSize = []string{"bps", "Bps", "byte"}

func RationalizeValueFromUnit(value float64, unit string, opt string) string {
	if utils.IsInStringArray(unit, fileSize) {
		if unit == "byte" {
			return (FormatFileSize(value, unit, float64(1024)))
		}
		return FormatFileSize(value, unit, float64(1000))
	}
	if unit == "%" && monitor.CommonAlertFieldOptDivision == monitor.CommonAlertFieldOpt(opt) {
		return fmt.Sprintf("%0.2f%s", value*100, unit)
	}
	return fmt.Sprintf("%0.2f%s", value, unit)
}

// 单位转换 保留2位小数
func FormatFileSize(fileSize float64, unit string, unitsize float64) (size string) {
	if fileSize < unitsize {
		return fmt.Sprintf("%.2f%s", fileSize, unit)
	} else if fileSize < (unitsize * unitsize) {
		return fmt.Sprintf("%.2fK%s", float64(fileSize)/float64(unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.2fM%s", float64(fileSize)/float64(unitsize*unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.2fG%s", float64(fileSize)/float64(unitsize*unitsize*unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.2fT%s", float64(fileSize)/float64(unitsize*unitsize*unitsize*unitsize), unit)
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fE%s", float64(fileSize)/float64(unitsize*unitsize*unitsize*unitsize*unitsize), unit)
	}
}

type SCompanyInfo struct {
	Copyright string `json:"copyright"`
	Name      string `json:"name"`
}

func GetCompanyInfo(ctx context.Context) (SCompanyInfo, error) {
	/* info, err := getBrandFromCopyrightApi(ctx)
	if err == nil && len(info.Name) != 0 {
		return *info, nil
	}
	if err != nil {
		log.Errorf("getBrandFromCopyrightApi err:%v", err)
	}
	return getBrandFromInfoApi(ctx)
	*/
	return SCompanyInfo{
		Name: options.Options.GetPlatformName(appctx.Lang(ctx)),
	}, nil
}

/*
func getBrandFromCopyrightApi(ctx context.Context) (*SCompanyInfo, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	obj, err := modules.Copyright.Update(session, "copyright", jsonutils.NewDict())
	if err != nil {
		return nil, err
	}
	var info SCompanyInfo
	lang := i18n.Lang(ctx)
	switch lang {
	case language.English:
		info.Name, _ = obj.GetString("brand_en")
	default:
		info.Name, _ = obj.GetString("brand_cn")
	}
	return &info, nil
}

func getBrandFromInfoApi(ctx context.Context) (SCompanyInfo, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	obj, err := modules.Info.Get(session, "info", jsonutils.NewDict())
	if err != nil {
		return SCompanyInfo{}, err
	}
	var info SCompanyInfo
	err = obj.Unmarshal(&info)
	if err != nil {
		return SCompanyInfo{}, err
	}
	if strings.Contains(info.Copyright, COMPANY_COPYRIGHT_ONECLOUD) {
		lang := i18n.Lang(ctx)
		switch lang {
		case language.English:
			info.Name = BRAND_ONECLOUD_NAME_EN
		default:
			info.Name = BRAND_ONECLOUD_NAME_CN
		}
	}
	return info, nil
}
*/
