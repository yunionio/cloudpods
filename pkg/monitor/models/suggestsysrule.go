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
	"database/sql"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysRuleManager *SSuggestSysRuleManager
)

func init() {
	SuggestSysRuleManager = &SSuggestSysRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SSuggestSysRule{},
			"suggestsysrule_tbl",
			"suggestsysrule",
			"suggestsysrules",
		),
	}
	SuggestSysRuleManager.SetVirtualObject(SuggestSysRuleManager)
}

type SSuggestSysRuleManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SSuggestSysRule struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	Type     string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Period   string               `width:"256" charset:"ascii" list:"user" update:"user"`
	TimeFrom string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Setting  jsonutils.JSONObject ` list:"user" update:"user"`
	ExecTime time.Time            `list:"user" update:"user"`
}

func (man *SSuggestSysRuleManager) FetchSuggestSysAlartSettings(ruleTypes ...string) (map[string]*monitor.SuggestSysRuleDetails, error) {
	suggestSysAlerSettingMap := make(map[string]*monitor.SuggestSysRuleDetails, 0)

	rules, err := man.GetRules(ruleTypes...)
	if err != nil {
		return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
	}
	for _, config := range rules {
		suggestSysRuleDetails := config.getMoreDetails(monitor.SuggestSysRuleDetails{})
		if err != nil {
			return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
		}
		suggestSysAlerSettingMap[config.Type] = &suggestSysRuleDetails
	}
	return suggestSysAlerSettingMap, nil
}

//根据数据库中查询得到的信息进行适配转换，同时更新drivers中的内容
func (dConfig *SSuggestSysRule) getSuggestSysAlertSetting() (*monitor.SSuggestSysAlertSetting, error) {
	setting := new(monitor.SSuggestSysAlertSetting)
	err := dConfig.Setting.Unmarshal(setting)
	if err != nil {
		return nil, errors.Wrap(err, "SSuggestSysRule getSuggestSysAlertSetting error")
	}
	return setting, nil
}

type DiskUnsed struct {
	Status string
}

func (manager *SSuggestSysRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.SuggestSysRuleListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (man *SSuggestSysRuleManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.SuggestSysRuleCreateInput) (monitor.SuggestSysRuleCreateInput, error) {
	if data.Period == "" {
		// default 30s
		data.Period = "30s"
	}
	if data.TimeFrom == "" {
		data.TimeFrom = "24h"
	}
	if data.Enabled == nil {
		enable := true
		data.Enabled = &enable
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if _, err := time.ParseDuration(data.TimeFrom); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.TimeFrom)
	}
	if dri, ok := suggestSysRuleDrivers[data.Type]; !ok {
		return data, httperrors.NewInputParameterError("not support type %q", data.Type)
	} else {
		//Type is uniq
		err := db.NewNameValidator(man, ownerId, data.Type, "")
		if err != nil {
			return data, err
		}
		if data.Type == monitor.SCALE_DOWN || data.Type == monitor.SCALE_UP {
			if data.Setting == nil {
				return data, httperrors.NewInputParameterError("no found rule setting")
			}
		}
		if data.Setting != nil {
			err = dri.ValidateSetting(data.Setting)
			if err != nil {
				return data, errors.Wrap(err, "validate setting error")
			}
		}
	}
	return data, nil
}

func (rule *SSuggestSysRule) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.SuggestSysRuleUpdateInput) (monitor.SuggestSysRuleUpdateInput, error) {
	if data.Period == "" {
		// default 30s
		data.Period = "30s"
	}
	if data.Enabled != nil {
		rule.SetEnabled(*data.Enabled)
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return data, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	if data.Setting != nil {
		err := suggestSysRuleDrivers[rule.Type].ValidateSetting(data.Setting)
		if err != nil {
			return data, errors.Wrap(err, "validate setting error")
		}
	}
	return data, nil
}

func (man *SSuggestSysRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysRuleDetails {
	rows := make([]monitor.SuggestSysRuleDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysRuleDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*SSuggestSysRule).getMoreDetails(rows[i])
	}
	return rows
}

func (self *SSuggestSysRule) getMoreDetails(out monitor.SuggestSysRuleDetails) monitor.SuggestSysRuleDetails {
	var err error
	out.Setting, err = self.getSuggestSysAlertSetting()
	if err != nil {
		log.Errorln("getMoreDetails err:", err)
	}
	out.ID = self.Id
	out.Name = self.Name
	out.Enabled = self.GetEnabled()
	return out
}

func (self *SSuggestSysRule) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.SuggestSysRuleDetails, error) {
	return monitor.SuggestSysRuleDetails{}, nil
}

//after create, update Cronjob's info
func (self *SSuggestSysRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.updateCronjob()
}

//after update, update Cronjob's info
func (self *SSuggestSysRule) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.updateCronjob()
}

func (self *SSuggestSysRule) updateCronjob() {
	cronman.GetCronJobManager().Remove(self.Type)
	if self.Enabled.Bool() {
		dur, _ := time.ParseDuration(self.Period)
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(self.Type, dur,
			suggestSysRuleDrivers[self.Type].DoSuggestSysRule, true)
	}
}

func (self *SSuggestSysRule) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "Enable")
}

func (self *SSuggestSysRule) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled.Bool() {
		db.Update(self, func() error {
			self.Enabled = tristate.True
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
		self.updateCronjob()
	}
	return nil, nil
}

func (self *SSuggestSysRule) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable")
}

func (self *SSuggestSysRule) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled.IsTrue() {
		db.Update(self, func() error {
			self.Enabled = tristate.False
			return nil
		})
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
		self.updateCronjob()
	}
	return nil, nil
}

func (self *SSuggestSysRuleManager) AllowGetPropertyRuleType(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSuggestSysRuleManager) GetPropertyRuleType(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ruleArr := jsonutils.NewArray()
	ret.Add(ruleArr, "rule-type")
	rules, err := self.GetRules()
	if err != nil {
		return ret, err
	}
	for _, rule := range rules {
		ruleArr.Add(jsonutils.NewString(rule.Type))
	}
	return ret, nil
}

func (self *SSuggestSysRuleManager) AllowGetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}
func (self *SSuggestSysRuleManager) GetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return DataSourceManager.GetDatabases()
}

func (self *SSuggestSysRuleManager) AllowGetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SSuggestSysRuleManager) GetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return DataSourceManager.GetMeasurements(query)
}

func (self *SSuggestSysRuleManager) AllowGetPropertyMetricMeasurement(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SSuggestSysRuleManager) GetPropertyMetricMeasurement(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return DataSourceManager.GetMetricMeasurement(query)
}

func (self *SSuggestSysRuleManager) GetRules(tp ...string) ([]SSuggestSysRule, error) {
	rules := make([]SSuggestSysRule, 0)
	query := self.Query()
	if len(tp) > 0 {
		query.In("type", tp)
	}
	err := db.FetchModelObjects(self, query, &rules)
	if err != nil && err != sql.ErrNoRows {
		log.Errorln(errors.Wrap(err, "db.FetchModelObjects"))
		return rules, err
	}
	return rules, nil
}

func (self *SSuggestSysRule) UpdateExecTime() {
	db.Update(self, func() error {
		self.ExecTime = time.Now()
		return nil
	})
}
