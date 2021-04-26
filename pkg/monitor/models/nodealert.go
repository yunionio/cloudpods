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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	NodeAlertMetadataType     = "type"
	NodeAlertMetadataNodeId   = "node_id"
	NodeAlertMetadataNodeName = "node_name"
)

var NodeAlertManager *SNodeAlertManager

func init() {
	NodeAlertManager = NewNodeAlertManager()
}

type SV1AlertManager struct {
	SAlertManager
}

type SNodeAlertManager struct {
	SV1AlertManager
}

func NewNodeAlertManager() *SNodeAlertManager {
	man := &SNodeAlertManager{
		SV1AlertManager: SV1AlertManager{
			*NewAlertManager(SNodeAlert{}, "nodealert", "nodealerts"),
		},
	}
	man.SetVirtualObject(man)
	return man
}

type SV1Alert struct {
	SAlert
}

type SNodeAlert struct {
	SV1Alert
}

func (v1man *SV1AlertManager) CreateNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	alertName string,
	channel string,
	recipients string) (*SNotification, error) {
	userIds := strings.Split(recipients, ",")
	return NotificationManager.CreateOneCloudNotification(ctx, userCred, alertName, channel, userIds, "")
}

func (man *SNodeAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.NodeAlertCreateInput) (*monitor.NodeAlertCreateInput, error) {
	if data.Period == "" {
		data.Period = "5m"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if data.Metric == "" {
		return nil, merrors.NewArgIsEmptyErr("metric")
	}
	parts := strings.Split(data.Metric, ".")
	if len(parts) != 2 {
		return nil, httperrors.NewInputParameterError("metric %s is invalid format, usage <measurement>.<field>", data.Metric)
	}
	measurement, field, err := GetMeasurementField(data.Metric)
	if err != nil {
		return nil, err
	}
	if data.Recipients == "" {
		return nil, merrors.NewArgIsEmptyErr("recipients")
	}
	if data.NodeId == "" {
		return nil, merrors.NewArgIsEmptyErr("node_id")
	}
	nodeName, resType, err := man.validateResourceId(ctx, data.Type, data.NodeId)
	if err != nil {
		return nil, err
	}
	data.NodeName = nodeName
	name, err := man.genName(ctx, ownerId, resType, nodeName, data.Metric)
	if err != nil {
		return nil, err
	}
	alertInput := data.ToAlertCreateInput(name, field, measurement, "telegraf")
	alertInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertInput)
	if err != nil {
		return nil, err
	}
	data.AlertCreateInput = alertInput
	return &data, nil
}

func (man *SNodeAlertManager) genName(ctx context.Context, ownerId mcclient.IIdentityProvider, resType string, nodeName string, metric string) (string, error) {
	nameHint := fmt.Sprintf("%s %s %s", resType, nodeName, metric)
	name, err := db.GenerateName(ctx, man, ownerId, nameHint)
	if err != nil {
		return "", err
	}
	if name != nameHint {
		return "", httperrors.NewDuplicateNameError(man.Keyword(), metric)
	}
	return name, nil
}

func (man *SNodeAlertManager) validateResourceId(ctx context.Context, nodeType, nodeId string) (string, string, error) {
	var (
		retType  string
		nodeName string
		err      error
	)
	switch nodeType {
	case monitor.NodeAlertTypeHost:
		retType = "宿主机"
		nodeName, err = man.validateHostResource(ctx, nodeId)
	case monitor.NodeAlertTypeGuest:
		retType = "虚拟机"
		nodeName, err = man.validateGuestResource(ctx, nodeId)
	default:
		return "", "", httperrors.NewInputParameterError("unsupported resource type %s", nodeType)
	}
	return nodeName, retType, err
}

func (man *SNodeAlertManager) validateGuestResource(ctx context.Context, id string) (string, error) {
	return man.validateResourceByMod(ctx, &modules.Servers, id)
}

func (man *SNodeAlertManager) validateHostResource(ctx context.Context, id string) (string, error) {
	return man.validateResourceByMod(ctx, &modules.Hosts, id)
}

func (man *SNodeAlertManager) validateResourceByMod(ctx context.Context, mod modulebase.Manager, id string) (string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	ret, err := mod.Get(s, id, nil)
	if err != nil {
		return "", err
	}
	name, err := ret.GetString("name")
	if err != nil {
		return "", err
	}
	return name, nil
}

func (man *SNodeAlertManager) ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// hack: always use details in query to get more details
	query.Set("details", jsonutils.JSONTrue)
	return query, nil
}

func (man *SV1AlertManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.V1AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SAlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SAlertManager.ListItemFilter")
	}
	return q, nil
}

func (man *SV1AlertManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.V1AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SAlertManager.OrderByExtraFields(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SAlertManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SV1AlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SAlertManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (alertV1 *SV1Alert) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.V1AlertUpdateInput,
) (monitor.V1AlertUpdateInput, error) {
	var err error
	input.AlertUpdateInput, err = alertV1.SAlert.ValidateUpdateData(ctx, userCred, query, input.AlertUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SAlert.ValidateUpdateData")
	}
	return input, nil
}

func (man *SNodeAlertManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.NodeAlertListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SV1AlertManager.ListItemFilter(ctx, q, userCred, query.V1AlertListInput)
	if err != nil {
		return nil, err
	}
	if len(query.Metric) > 0 {

	}
	if len(query.Type) > 0 {

	}
	if len(query.NodeId) > 0 {

	}
	if len(query.NodeName) > 0 {

	}
	q = q.Equals("used_by", AlertNotificationUsedByNodeAlert)
	return q, nil
}

func (man *SNodeAlertManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.NodeAlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SV1AlertManager.OrderByExtraFields(ctx, q, userCred, query.V1AlertListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SV1AlertManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SNodeAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SV1AlertManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SNodeAlertManager) GetAlert(id string) (*SNodeAlert, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SNodeAlert), nil
}

func (man *SNodeAlertManager) CustomizeFilterList(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject) (
	*db.CustomizeListFilters, error) {
	filters, err := man.SV1AlertManager.CustomizeFilterList(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	input := new(monitor.NodeAlertListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	wrapF := func(f func(obj *SNodeAlert) (bool, error)) func(object jsonutils.JSONObject) (bool, error) {
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
		mF := func(obj *SNodeAlert) (bool, error) {
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

	if input.NodeName != "" {
		nf := func(obj *SNodeAlert) (bool, error) {
			return obj.getNodeName() == input.NodeName, nil
		}
		filters.Append(wrapF(nf))
	}

	if input.NodeId != "" {
		filters.Append(wrapF(func(obj *SNodeAlert) (bool, error) {
			return obj.getNodeId() == input.NodeId, nil
		}))
	}

	if input.Type != "" {
		filters.Append(wrapF(func(obj *SNodeAlert) (bool, error) {
			return obj.getType() == input.Type, nil
		}))
	}

	return filters, nil
}

func (alert *SV1Alert) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	notiName, channel, recipients, usedBy string) error {
	noti, err := NodeAlertManager.CreateNotification(ctx, userCred, notiName, channel, recipients)
	if err != nil {
		return errors.Wrap(err, "create notification")
	}
	if alert.Id == "" {
		alert.Id = db.DefaultUUIDGenerator()
	}
	alert.UsedBy = usedBy
	_, err = alert.AttachNotification(
		ctx, userCred, noti,
		monitor.AlertNotificationStateUnknown,
		usedBy)
	return err
}

func (alert *SV1Alert) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	if data.(*jsonutils.JSONDict).Contains("status") {
		status, _ := data.(*jsonutils.JSONDict).GetString("status")
		alert.UpdateIsEnabledStatus(ctx, userCred, &status)
	}
}

func (alert *SNodeAlert) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	if err := alert.SAlert.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(monitor.NodeAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}
	return alert.SV1Alert.CustomizeCreate(ctx, userCred, input.Metric, input.Channel, input.Recipients, AlertNotificationUsedByNodeAlert)
}

func (alert *SNodeAlert) getNodeId() string {
	return alert.GetMetadata(NodeAlertMetadataNodeId, nil)
}

func (alert *SNodeAlert) setNodeId(ctx context.Context, userCred mcclient.TokenCredential, id string) error {
	return alert.SetMetadata(ctx, NodeAlertMetadataNodeId, id, userCred)
}

func (alert *SNodeAlert) getNodeName() string {
	return alert.GetMetadata(NodeAlertMetadataNodeName, nil)
}

func (alert *SNodeAlert) setNodeName(ctx context.Context, userCred mcclient.TokenCredential, name string) error {
	return alert.SetMetadata(ctx, NodeAlertMetadataNodeName, name, userCred)
}

func (alert *SNodeAlert) getType() string {
	return alert.GetMetadata(NodeAlertMetadataType, nil)
}

func (alert *SNodeAlert) setType(ctx context.Context, userCred mcclient.TokenCredential, typ string) error {
	return alert.SetMetadata(ctx, NodeAlertMetadataType, typ, userCred)
}

func (alert *SNodeAlert) PostCreate(ctx context.Context,
	userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	alert.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := new(monitor.NodeAlertCreateInput)
	if err := data.Unmarshal(input); err != nil {
		log.Errorf("post create unmarshal input: %v", err)
		return
	}
	if err := alert.setNodeId(ctx, userCred, input.NodeId); err != nil {
		log.Errorf("set node id: %v", err)
		return
	}
	if err := alert.setNodeName(ctx, userCred, input.NodeName); err != nil {
		log.Errorf("set node name: %v", err)
		return
	}
	if err := alert.setType(ctx, userCred, input.Type); err != nil {
		log.Errorf("set type: %v", err)
		return
	}
}

func (man *SV1AlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertV1Details {
	rows := make([]monitor.AlertV1Details, len(objs))

	alertRows := man.SAlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertV1Details{
			AlertDetails: alertRows[i],
		}
	}

	return rows
}

func (alert *SV1Alert) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.AlertV1Details, error) {
	return monitor.AlertV1Details{}, nil
}

const (
	V1AlertDisabledStatus = "Disabled"
	V1AlertEnabledStatus  = "Enabled"
)

func (alert *SV1Alert) GetStatus() string {
	if alert.IsEnable() {
		return V1AlertEnabledStatus
	}
	return V1AlertDisabledStatus
}

func (alert *SV1Alert) getMoreDetails(out monitor.AlertV1Details, usedBy string) (monitor.AlertV1Details, error) {
	out.Name = alert.GetName()
	if alert.Frequency < 60 {
		out.Window = fmt.Sprintf("%ds", alert.Frequency)
	} else {
		out.Window = fmt.Sprintf("%dm", alert.Frequency/60)
	}

	setting, err := alert.GetSettings()
	if err != nil {
		return out, err
	}
	if len(setting.Conditions) == 0 {
		return out, nil
	}
	cond := setting.Conditions[0]
	cmp := ""
	switch cond.Evaluator.Type {
	case "gt":
		cmp = ">="
	case "lt":
		cmp = "<="
	}
	out.Level = alert.Level
	out.Comparator = cmp
	out.Threshold = cond.Evaluator.Params[0]
	out.Period = cond.Query.From

	noti, err := alert.GetNotification(usedBy)
	if err != nil {
		return out, err
	}
	if noti != nil {
		out.NotifierId = noti.GetId()
		settings := new(monitor.NotificationSettingOneCloud)
		if err := noti.Settings.Unmarshal(settings); err != nil {
			return out, err
		}
		out.Recipients = strings.Join(settings.UserIds, ",")
		out.Channel = settings.Channel
	}

	q := cond.Query
	measurement := q.Model.Measurement
	field := q.Model.Selects[0][0].Params[0]
	db := q.Model.Database
	out.Measurement = measurement
	out.Field = field
	out.DB = db
	out.Status = alert.GetStatus()
	return out, nil
}

func (man *SNodeAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.NodeAlertDetails {
	rows := make([]monitor.NodeAlertDetails, len(objs))

	v1Rows := man.SV1AlertManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = monitor.NodeAlertDetails{
			AlertV1Details: v1Rows[i],
		}
		rows[i], _ = objs[i].(*SNodeAlert).getMoreDetails(rows[i])
	}

	return rows
}

func (alert *SNodeAlert) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.NodeAlertDetails, error) {
	return monitor.NodeAlertDetails{}, nil
}

func (alert *SNodeAlert) getMoreDetails(out monitor.NodeAlertDetails) (monitor.NodeAlertDetails, error) {
	var err error
	out.AlertV1Details, err = alert.SV1Alert.getMoreDetails(out.AlertV1Details, AlertNotificationUsedByNodeAlert)
	if err != nil {
		return out, errors.Wrap(err, "SV1Alert.getMoreDetails")
	}

	out.Type = alert.getType()
	out.NodeId = alert.getNodeId()
	out.NodeName = alert.getNodeName()

	setting, err := alert.GetSettings()
	if err != nil {
		return out, err
	}
	if len(setting.Conditions) == 0 {
		return out, nil
	}
	out.Metric = fmt.Sprintf("%s.%s", out.Measurement, out.Field)

	return out, nil
}

func (alert *SV1Alert) GetNotification(usedby string) (*SNotification, error) {
	alertNotis, err := alert.GetNotifications()
	if err != nil {
		return nil, err
	}
	if len(alertNotis) == 0 {
		return nil, nil
	}
	for _, an := range alertNotis {
		if an.GetUsedBy() == usedby {
			return an.GetNotification()
		}
	}
	return nil, httperrors.NewNotFoundError("not found alert notification used by %s", usedby)
}

func (alert *SV1Alert) UpdateNotification(usedBy string, channel, recipients *string) error {
	obj, err := alert.GetNotification(usedBy)
	if err != nil {
		return errors.Wrap(err, "Get notification when update")
	}
	if obj == nil {
		return nil
	}
	setting := new(monitor.NotificationSettingOneCloud)
	if err := obj.Settings.Unmarshal(setting); err != nil {
		return errors.Wrap(err, "unmarshal onecloud notification setting")
	}
	if channel != nil {
		setting.Channel = *channel
	}
	if recipients != nil {
		setting.UserIds = strings.Split(*recipients, ",")
	}
	_, err = db.Update(obj, func() error {
		obj.Settings = jsonutils.Marshal(setting)
		return nil
	})
	return err
}

func (alert *SV1Alert) UpdateIsEnabledStatus(ctx context.Context, userCred mcclient.TokenCredential, status *string) error {
	if status == nil {
		return nil
	}
	if status != nil {
		s := *status
		if s == V1AlertDisabledStatus {
			if _, err := alert.PerformDisable(ctx, userCred, nil, apis.PerformDisableInput{}); err != nil {
				return err
			}
			db.Update(&alert.SAlert, func() error {
				alert.SetStatus(userCred, V1AlertDisabledStatus, "")
				return nil
			})
		} else {
			if _, err := alert.PerformEnable(ctx, userCred, nil, apis.PerformEnableInput{}); err != nil {
				return err
			}
			db.Update(&alert.SAlert, func() error {
				alert.SetStatus(userCred, V1AlertEnabledStatus, "")
				return nil
			})
		}
	}
	return nil
}

func (alert *SV1Alert) CustomizeDelete(
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
		if !conf.IsDefault {
			if err := conf.CustomizeDelete(ctx, userCred, query, data); err != nil {
				return err
			}
			if err := conf.Delete(ctx, userCred); err != nil {
				return err
			}
		}
		if err := noti.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (alert *SNodeAlert) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.NodeAlertUpdateInput,
) (monitor.NodeAlertUpdateInput, error) {
	// ret := monitor.AlertUpdateInput{}
	detailsList := NodeAlertManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{alert}, nil, false)
	if len(detailsList) == 0 {
		panic("inconsistent return results of FetchCustomizeColumns")
	}
	details := detailsList[0]

	nameChange := false
	if input.NodeId != nil && *input.NodeId != details.NodeId {
		nameChange = true
		details.NodeId = *input.NodeId
		if err := alert.setNodeId(ctx, userCred, details.NodeId); err != nil {
			return input, err
		}
	}
	if input.Type != nil && *input.Type != details.Type {
		nameChange = true
		details.Type = *input.Type
		if err := alert.setType(ctx, userCred, details.Type); err != nil {
			return input, err
		}
	}
	nodeName, resType, err := NodeAlertManager.validateResourceId(ctx, details.Type, details.NodeId)
	if err != nil {
		return input, err
	}
	if details.NodeName != nodeName {
		nameChange = true
		if err := alert.setNodeName(ctx, userCred, nodeName); err != nil {
			return input, err
		}
		details.NodeName = nodeName
	}
	if input.Level != nil && *input.Level != details.Level {
		details.Level = *input.Level
		// ret.Level = input.Level
	}

	if input.Window != nil && *input.Window != details.Window {
		details.Window = *input.Window
		freq, err := time.ParseDuration(details.Window)
		if err != nil {
			return input, err
		}
		freqSec := int64(freq / time.Second)
		input.Frequency = &freqSec
	}

	if input.Threshold != nil && *input.Threshold != details.Threshold {
		details.Threshold = *input.Threshold
	}

	if input.Comparator != nil && *input.Comparator != details.Comparator {
		details.Comparator = *input.Comparator
	}

	if input.Period != nil && *input.Period != details.Period {
		details.Period = *input.Period
	}

	if input.Metric != nil && *input.Metric != details.Metric {
		details.Metric = *input.Metric
		measurement, field, err := GetMeasurementField(*input.Metric)
		if err != nil {
			return input, err
		}
		details.Measurement = measurement
		details.Field = field
	}

	name := alert.Name
	if nameChange {
		name, err = NodeAlertManager.genName(ctx, userCred, resType, details.NodeName, details.Metric)
		if err != nil {
			return input, err
		}
		input.Name = name
	}

	ds, err := DataSourceManager.GetDefaultSource()
	if err != nil {
		return input, errors.Wrap(err, "get default data source")
	}
	// hack: update notification here
	if err := alert.UpdateNotification(AlertNotificationUsedByNodeAlert, input.Channel, input.Recipients); err != nil {
		return input, errors.Wrap(err, "update notification")
	}
	tmpS := alert.getUpdateSetting(name, details, ds.GetId())
	input.Settings = &tmpS

	input.V1AlertUpdateInput, err = alert.SV1Alert.ValidateUpdateData(ctx, userCred, query, input.V1AlertUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SV1Alert.ValidateUpdateData")
	}

	return input, nil
}

func (alert *SNodeAlert) getUpdateSetting(
	name string,
	details monitor.NodeAlertDetails,
	dsId string,
) monitor.AlertSetting {
	data := monitor.NodeAlertCreateInput{
		ResourceAlertV1CreateInput: monitor.ResourceAlertV1CreateInput{
			Period:     details.Period,
			Window:     details.Window,
			Comparator: details.Comparator,
			Threshold:  details.Threshold,
			Channel:    details.Channel,
			Recipients: details.Recipients,
		},
		Metric: details.Metric,
		Type:   details.Type,
		NodeId: details.NodeId,
	}
	data.Level = details.Level
	out := data.ToAlertCreateInput(name, details.Field, details.Measurement, details.DB)
	out.Settings = *setAlertDefaultSetting(&out.Settings, dsId)
	return out.Settings
}

func (alert *SNodeAlert) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	alert.SV1Alert.PostUpdate(ctx, userCred, query, data)
}

func (alert *SNodeAlert) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return alert.SV1Alert.CustomizeDelete(ctx, userCred, query, data)
}

func (m *SNodeAlertManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {

	return q
}
