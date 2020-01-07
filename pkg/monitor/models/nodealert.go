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

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/options"
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
	recipients string) (*SAlertNotification, error) {
	userIds := strings.Split(recipients, ",")
	return AlertNotificationManager.CreateOneCloudNotification(ctx, userCred, alertName, channel, userIds)
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
		return nil, httperrors.NewInputParameterError("metric is missing")
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
		return nil, httperrors.NewInputParameterError("recipients is empty")
	}
	notification, err := man.CreateNotification(ctx, userCred, data.Metric, data.Channel, data.Recipients)
	if err != nil {
		return nil, errors.Wrap(err, "create notification")
	}
	if data.NodeId == "" {
		return nil, httperrors.NewInputParameterError("node_id is empty")
	}
	nodeName, resType, err := man.validateResourceId(ctx, data.Type, data.NodeId)
	if err != nil {
		return nil, err
	}
	data.NodeName = nodeName
	name, err := man.genName(ownerId, resType, nodeName, data.Metric)
	if err != nil {
		return nil, err
	}
	alertInput := data.ToAlertCreateInput(name, field, measurement, "telegraf", []string{notification.GetId()})
	alertInput, err = AlertManager.ValidateCreateData(ctx, userCred, ownerId, query, alertInput)
	if err != nil {
		return nil, err
	}
	data.AlertCreateInput = &alertInput
	return &data, nil
}

func (man *SNodeAlertManager) genName(ownerId mcclient.IIdentityProvider, resType string, nodeName string, metric string) (string, error) {
	nameHint := fmt.Sprintf("%s %s %s", resType, nodeName, metric)
	name, err := db.GenerateName(man, ownerId, nameHint)
	if err != nil {
		return "", err
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
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.NodeAlertListInput) (*sqlchemy.SQuery, error) {
	return AlertManager.ListItemFilter(ctx, q, userCred, query.ToAlertListInput())
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
			if err := obj.Settings.Unmarshal(settings, "settings"); err != nil {
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
	alert.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
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

func (alert *SV1Alert) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (monitor.AlertV1Details, error) {
	var err error
	out := monitor.AlertV1Details{}
	out.VirtualResourceDetails, err = alert.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
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
	out.Level = setting.Level
	out.Comparator = cmp
	out.Threshold = cond.Evaluator.Params[0]
	out.Period = cond.Query.From

	notification := alert.GetNotificationBySetting(setting)
	if notification != nil {
		out.Recipients = strings.Join(notification.UserIds, ",")
		out.Channel = notification.Channel
	}

	q := cond.Query
	measurement := q.Model.Measurement
	field := q.Model.Selects[0][0].Params[0]
	db := q.Model.Database
	out.Measurement = measurement
	out.Field = field
	out.DB = db
	noti, err := alert.GetNotification()
	if err != nil {
		return out, err
	}
	if noti != nil {
		out.NotifierId = noti.GetId()
	}
	return out, nil
}

func (alert *SNodeAlert) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (monitor.NodeAlertDetails, error) {
	var err error
	out := monitor.NodeAlertDetails{}
	commonDetails, err := alert.SV1Alert.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return out, err
	}
	out.AlertV1Details = commonDetails

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

func (alert *SV1Alert) GetNotification() (*SAlertNotification, error) {
	setting, err := alert.GetSettings()
	if err != nil {
		return nil, err
	}
	nIds := setting.Notifications
	if len(nIds) == 0 {
		return nil, nil
	}
	// only get first notification setting
	nId := nIds[0]
	obj, err := AlertNotificationManager.GetNotification(nId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get notificatoin %s", nId)
	}
	return obj, nil
}

func (alert *SV1Alert) UpdateNotification(channel, recipients *string) error {
	obj, err := alert.GetNotification()
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

func (alert *SV1Alert) GetNotificationBySetting(setting *monitor.AlertSetting) *monitor.NotificationSettingOneCloud {
	nIds := setting.Notifications
	if len(nIds) == 0 {
		return nil
	}
	// only get first notification setting
	nId := nIds[0]
	obj, err := AlertNotificationManager.GetNotification(nId)
	if err != nil {
		log.Errorf("Get notification by %s: %v", nId, err)
		return nil
	}
	if obj == nil {
		return nil
	}
	ocSetting := new(monitor.NotificationSettingOneCloud)
	if err := obj.Settings.Unmarshal(ocSetting); err != nil {
		log.Errorf("Unmarshal notification %s setting: %v", nId, err)
		return nil
	}
	return ocSetting
}

func (alert *SNodeAlert) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	notis, err := alert.GetNotifications()
	if err != nil {
		return err
	}
	for _, noti := range notis {
		if err := noti.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return err
		}
		if err := noti.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (alert *SNodeAlert) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input monitor.NodeAlertUpdateInput) (*jsonutils.JSONDict, error) {
	ret := monitor.AlertUpdateInput{}
	details, err := alert.GetExtraDetails(context.TODO(), nil, nil, false)
	if err != nil {
		return nil, err
	}

	nameChange := false
	if input.NodeId != nil && *input.NodeId != details.NodeId {
		nameChange = true
		ret.ResourceId = input.NodeId
		details.NodeId = *input.NodeId
		if err := alert.setNodeId(ctx, userCred, details.NodeId); err != nil {
			return nil, err
		}
	}
	if input.Type != nil && *input.Type != details.Type {
		nameChange = true
		ret.ResourceType = input.Type
		details.Type = *input.Type
		if err := alert.setType(ctx, userCred, details.Type); err != nil {
			return nil, err
		}
	}
	nodeName, resType, err := NodeAlertManager.validateResourceId(ctx, details.Type, details.NodeId)
	if err != nil {
		return nil, err
	}
	if details.NodeName != nodeName {
		nameChange = true
		if err := alert.setNodeName(ctx, userCred, nodeName); err != nil {
			return nil, err
		}
		details.NodeName = nodeName
	}
	if input.Level != nil && *input.Level != details.Level {
		details.Level = *input.Level
	}

	if input.Window != nil && *input.Window != details.Window {
		details.Window = *input.Window
		freq, err := time.ParseDuration(details.Window)
		if err != nil {
			return nil, err
		}
		freqSec := int64(freq / time.Second)
		ret.Frequency = &freqSec
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
			return nil, err
		}
		details.Measurement = measurement
		details.Field = field
	}

	name := alert.Name
	if nameChange {
		name, err = NodeAlertManager.genName(userCred, resType, details.NodeName, details.Metric)
		if err != nil {
			return nil, err
		}
		ret.Name = &name
	}

	ds, err := DataSourceManager.GetDefaultSource()
	if err != nil {
		return nil, errors.Wrap(err, "get default data source")
	}
	// hack: update notification here
	if err := alert.UpdateNotification(input.Channel, input.Recipients); err != nil {
		return nil, errors.Wrap(err, "update notification")
	}
	tmpS := alert.getUpdateSetting(name, details, ds.GetId())
	os, err := alert.GetSettings()
	if err != nil {
		return nil, errors.Wrap(err, "get origin setting")
	}
	tmpS.Notifications = os.Notifications
	ret.Settings = &tmpS
	return alert.SAlert.ValidateUpdateData(ctx, userCred, query, ret)
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
			Level:      details.Level,
			Channel:    details.Channel,
			Recipients: details.Recipients,
		},
		Metric: details.Metric,
		Type:   details.Type,
		NodeId: details.NodeId,
	}
	out := data.ToAlertCreateInput(name, details.Field, details.Measurement, details.DB, []string{details.NotifierId})
	out.Settings = *setAlertDefaultSetting(&out.Settings, dsId)
	return out.Settings
}
