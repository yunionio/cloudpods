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
	"fmt"
	"strconv"
	"strings"
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
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysRuleManager *SSuggestSysRuleManager
)

func init() {
	SuggestSysRuleManager = &SSuggestSysRuleManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			&SSuggestSysRule{},
			"suggestsysrule_tbl",
			"suggestsysrule",
			"suggestsysrules",
		),
	}
	SuggestSysRuleManager.SetVirtualObject(SuggestSysRuleManager)
	registry.RegisterService(SuggestSysRuleManager)
}

// +onecloud:swagger-gen-model-singular=suggestsysrule
// +onecloud:swagger-gen-model-plural=suggestsysrules
type SSuggestSysRuleManager struct {
	db.SStandaloneResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SSuggestSysRule struct {
	db.SStandaloneResourceBase
	db.SEnabledResourceBase

	Type     string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Period   string               `width:"256" charset:"ascii" list:"user" update:"user"`
	TimeFrom string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Setting  jsonutils.JSONObject `list:"user" update:"user"`
	ExecTime time.Time            `list:"user" update:"user"`
}

func (man *SSuggestSysRuleManager) FetchSuggestSysAlertSettings(ruleTypes ...monitor.SuggestDriverType) (map[monitor.SuggestDriverType]*monitor.SuggestSysRuleDetails, error) {
	suggestSysAlerSettingMap := make(map[monitor.SuggestDriverType]*monitor.SuggestSysRuleDetails, 0)

	rules, err := man.GetRules(ruleTypes...)
	if err != nil {
		return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
	}
	for _, config := range rules {
		suggestSysRuleDetails := config.getMoreDetails(monitor.SuggestSysRuleDetails{})
		if err != nil {
			return suggestSysAlerSettingMap, errors.Wrap(err, "FetchSuggestSysAlartSettings")
		}
		suggestSysAlerSettingMap[config.GetType()] = &suggestSysRuleDetails
	}
	return suggestSysAlerSettingMap, nil
}

func (rule *SSuggestSysRule) GetType() monitor.SuggestDriverType {
	return monitor.SuggestDriverType(rule.Type)
}

//根据数据库中查询得到的信息进行适配转换，同时更新drivers中的内容
func (rule *SSuggestSysRule) getSuggestSysAlertSetting() (*monitor.SSuggestSysAlertSetting, error) {
	setting := new(monitor.SSuggestSysAlertSetting)
	if rule.Setting == nil {
		rule.Setting = jsonutils.NewDict()
	}
	err := rule.Setting.Unmarshal(setting)
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
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
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
		data.Period = "12h"
	} else {
		data.Period = parseDuration(data.Period)
	}
	if data.TimeFrom == "" {
		data.TimeFrom = "24h"
	} else {
		data.TimeFrom = parseDuration(data.TimeFrom)
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
	if dri, ok := suggestSysRuleDrivers[monitor.SuggestDriverType(data.Type)]; !ok {
		return data, httperrors.NewInputParameterError("not support type %q", data.Type)
	} else {
		// Type is uniq
		if err := db.NewNameValidator(man, ownerId, data.Type, nil); err != nil {
			return data, err
		}
		if rule, err := man.GetRuleByType(monitor.SuggestDriverType(data.Type)); err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return data, err
			}
		} else if rule != nil {
			return data, httperrors.NewDuplicateResourceError("type %s rule already exists")
		}

		drvType := monitor.SuggestDriverType(data.Type)
		if drvType == monitor.SCALE_DOWN || drvType == monitor.SCALE_UP {
			if data.Setting == nil {
				return data, httperrors.NewInputParameterError("no found rule setting")
			}
		}
		if data.Setting != nil {
			if err := dri.ValidateSetting(data.Setting); err != nil {
				return data, errors.Wrap(err, "validate setting error")
			}
		} else {
			data.Setting = new(monitor.SSuggestSysAlertSetting)
		}
	}
	return data, nil
}

func (rule *SSuggestSysRule) GetDriver() ISuggestSysRuleDriver {
	return GetSuggestSysRuleDrivers()[rule.GetType()]
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
		err := rule.GetDriver().ValidateSetting(data.Setting)
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
	virtRows := man.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysRuleDetails{
			StandaloneResourceDetails: virtRows[i],
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
	self.Period = showDuration(self.Period)
	self.TimeFrom = showDuration(self.TimeFrom)
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

// after create, update Cronjob's info
func (self *SSuggestSysRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.updateCronjob()
}

// after update, update Cronjob's info
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
			self.GetDriver().DoSuggestSysRule, true)
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
	drivers := GetSuggestSysRuleDrivers()
	dbTypes := make(map[monitor.SuggestDriverType]string, 0)
	for _, rule := range rules {
		if _, ok := drivers[rule.GetType()]; !ok {
			return nil, fmt.Errorf("have invalid rule type :%s", string(rule.GetType()))
		}
		dbTypes[rule.GetType()] = ""
	}
	if len(dbTypes) == len(drivers) {
		return ret, nil
	}
	for typ, driver := range drivers {
		if _, ok := dbTypes[typ]; ok {
			continue
		}
		ruleArr.Add(jsonutils.NewString(string(driver.GetType())))
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
	ruleType, err := query.GetString("type")
	if err != nil {
		return nil, err
	}
	if _, ok := monitor.FilterSuggestRuleMeasureMentMap[monitor.SuggestDriverType(ruleType)]; !ok {
		return nil, fmt.Errorf("param type: %s is invalid", ruleType)
	}
	measurementFilter := getMeasurementFilter(monitor.FilterSuggestRuleMeasureMentMap[monitor.SuggestDriverType(ruleType)])
	return DataSourceManager.GetMeasurements(query, measurementFilter, "")
}

func getMeasurementFilter(filter string) string {
	return fmt.Sprintf(" MEASUREMENT =~ /%s.*/", filter)
}

func (self *SSuggestSysRuleManager) AllowGetPropertyMetricMeasurement(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SSuggestSysRuleManager) GetPropertyMetricMeasurement(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.GetMetricMeasurement(query)
}

func (self *SSuggestSysRuleManager) GetMetricMeasurement(query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("database")
	}
	measurement, _ := query.GetString("measurement")
	if measurement == "" {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("measurement")
	}
	dataSource, err := DataSourceManager.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	output := new(monitor.InfluxMeasurement)
	output.Measurement = measurement
	output.Database = database
	err = getAttributesOnMeasurement(database, monitor.METRIC_FIELD, output, db)
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "getAttributesOnMeasurement error")
	}
	return jsonutils.Marshal(output), nil
}

func (man *SSuggestSysRuleManager) GetRuleByType(tp monitor.SuggestDriverType) (*SSuggestSysRule, error) {
	query := man.Query().Equals("type", tp)
	rules := make([]SSuggestSysRule, 0)
	if err := db.FetchModelObjects(man, query, &rules); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}
	if len(rules) != 1 {
		return nil, errors.Wrapf(sqlchemy.ErrDuplicateEntry, "found %d type %s rules", len(rules), tp)
	}
	return &rules[0], nil
}

func (self *SSuggestSysRuleManager) GetRules(tp ...monitor.SuggestDriverType) ([]SSuggestSysRule, error) {
	rules := make([]SSuggestSysRule, 0)
	query := self.Query()
	if len(tp) > 0 {
		query.In("type", tp)
	}
	err := db.FetchModelObjects(self, query, &rules)
	if err != nil && err != sql.ErrNoRows {
		return rules, errors.Wrap(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (self *SSuggestSysRule) UpdateExecTime() {
	db.Update(self, func() error {
		self.ExecTime = time.Now()
		return nil
	})
}

func (manager *SSuggestSysRuleManager) Init() error {
	return nil
}

type ruleInfo struct {
	insertRuleTypes []string
	updateRules     map[string]*SSuggestSysRule
	deleteRules     map[string]*SSuggestSysRule
}

func (man *SSuggestSysRuleManager) Run(ctx context.Context) (err error) {
	sysRules, err := man.GetRules()
	if err != nil {
		return err
	}
	ruleOptor := getRuleInfo(sysRules)
	err = man.initDeleteDefaultRule(ruleOptor)
	if err != nil {
		return
	}
	log.Errorln("ruleTypes", ruleOptor.insertRuleTypes)
	err = man.initCreateDefaultRule(ruleOptor)
	if err != nil {
		return
	}
	err = man.initUpdateDefaultRule(ruleOptor)
	if err != nil {
		return
	}
	return nil
}

func (man *SSuggestSysRuleManager) initUpdateDefaultRule(rule ruleInfo) error {
	for ruleType, suggestRule := range rule.updateRules {
		ruleDris := GetSuggestSysRuleDrivers()
		if dri, ok := ruleDris[monitor.SuggestDriverType(ruleType)]; ok {
			ruleCreateInput := dri.GetDefaultRule()
			_, err := db.Update(suggestRule, func() error {
				suggestRule.Name = ruleCreateInput.Name
				suggestRule.Setting = jsonutils.Marshal(ruleCreateInput.Setting)
				suggestRule.TimeFrom = ruleCreateInput.TimeFrom
				suggestRule.Period = ruleCreateInput.Period
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "initUpdateDefaultRule error")
			}
		}
	}
	return nil
}

func (man *SSuggestSysRuleManager) initCreateDefaultRule(rule ruleInfo) error {
	adminCredential := auth.AdminCredential()
	for _, ruleType := range rule.insertRuleTypes {
		ruleDris := GetSuggestSysRuleDrivers()
		if dri, ok := ruleDris[monitor.SuggestDriverType(ruleType)]; ok {
			createInput := dri.GetDefaultRule()
			_, err := db.DoCreate(man, context.Background(), adminCredential, nil, jsonutils.Marshal(&createInput),
				adminCredential)
			if err != nil {
				return errors.Wrap(err, "initCreateDefaultRule error")
			}
		}
	}
	return nil
}

func (man *SSuggestSysRuleManager) initDeleteDefaultRule(rule ruleInfo) error {
	ctx := context.Background()
	adminCredential := auth.AdminCredential()
	for key, _ := range rule.deleteRules {
		err := db.DeleteModel(ctx, adminCredential, rule.deleteRules[key])
		if err != nil {
			return errors.Wrap(err, "initDeleteDefaultRule")
		}
	}
	return nil
}

func getRuleInfo(rules []SSuggestSysRule) (rul ruleInfo) {
	dbRules := make(map[string]*SSuggestSysRule)
	for i, _ := range rules {
		dbRules[rules[i].Type] = &rules[i]
	}
	defaultRules := GetSuggestSysRuleDriverTypes()
	for ruleType, _ := range dbRules {
		if defaultRules.Has(ruleType) {
			if rul.updateRules == nil {
				rul.updateRules = make(map[string]*SSuggestSysRule)
			}
			rul.updateRules[ruleType] = dbRules[ruleType]
			defaultRules.Delete(ruleType)
			delete(dbRules, ruleType)
		}
	}
	rul.insertRuleTypes = defaultRules.List()
	rul.deleteRules = dbRules
	return
}

func parseDuration(dur string) string {
	var hourInt int64
	if strings.Contains(dur, "d") {
		durDay := strings.Split(dur, "d")[0]
		dur = strings.Split(dur, "d")[1]
		durDayInt, _ := strconv.ParseInt(durDay, 10, 64)
		hourInt = durDayInt * 24

	}
	if strings.Contains(dur, "h") {
		durHour := strings.Split(dur, "h")[0]
		dur = strings.Split(dur, "h")[1]
		durHourInt, _ := strconv.ParseInt(durHour, 10, 64)
		hourInt += durHourInt
	}
	if hourInt != 0 {
		dur = fmt.Sprintf("%dh%s", hourInt, dur)
	}
	return dur
}

func showDuration(dur string) string {
	var durStr string
	if strings.Contains(dur, "h") {
		durInt, _ := strconv.ParseInt(strings.Split(dur, "h")[0], 10, 64)
		durStr = showDuration_(durInt, "h")
	}
	if strings.Contains(dur, "m") {
		durInt, _ := strconv.ParseInt(strings.Split(dur, "m")[0], 10, 64)
		durStr = showDuration_(durInt, "m")
	}
	if strings.Contains(dur, "s") {
		durInt, _ := strconv.ParseInt(strings.Split(dur, "s")[0], 10, 64)
		durStr = showDuration_(durInt, "s")
	}
	return durStr
}

func showDuration_(dur int64, sign string) string {
	var durUp, durSign int64
	var upSign, durStr string
	if sign == "s" && dur > 60 {
		upSign = "m"
		durUp = dur / 60
		durSign = dur % 60
		durStr = showDuration_(durUp, upSign)
	}
	if sign == "m" && dur > 60 {
		upSign = "h"
		durUp = dur / 60
		durSign = dur % 60
		durStr = showDuration_(durUp, upSign)
	}
	if sign == "h" && dur > 24 {
		upSign = "d"
		durUp = dur / 24
		durSign = dur % 24
	}

	if len(upSign) != 0 {
		durBuf := strings.Builder{}
		durBuf.WriteString(fmt.Sprintf("%s%d%s", durStr, durUp, upSign))
		if durSign != 0 {
			durBuf.WriteString(fmt.Sprintf("%d%s", durSign, sign))
		}
		return durBuf.String()
	}
	return fmt.Sprintf("%d%s", dur, sign)
}
