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

package suggestsysdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type EIPUnused struct {
	*baseDriver
}

func NewEIPUsedDriver() models.ISuggestSysRuleDriver {
	return &EIPUnused{
		baseDriver: newBaseDriver(
			monitor.EIP_UNUSED,
			monitor.EIP_MONITOR_RES_TYPE,
			monitor.DELETE_DRIVER_ACTION,
			monitor.EIP_MONITOR_SUGGEST,
		),
	}
}

func (dri *EIPUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	obj := new(monitor.EIPUnused)
	input.EIPUnused = obj
	return nil
}

func (rule *EIPUnused) Run(instance *monitor.SSuggestSysAlertSetting) {
	oldAlert, err := getLastAlerts(rule)
	if err != nil {
		log.Errorln(err)
		return
	}
	newAlert, err := rule.getEIPUnused(instance)
	if err != nil {
		log.Errorln(errors.Wrap(err, "getEIPUnused error"))
		return
	}

	na, _ := newAlert.GetArray()
	DealAlertData(rule.GetType(), oldAlert, na)
}

func (rule *EIPUnused) getEIPUnused(instance *monitor.SSuggestSysAlertSetting) (*jsonutils.JSONArray, error) {
	rules, _ := models.SuggestSysRuleManager.GetRules(rule.GetType())
	if len(rules) == 0 {
		log.Println("Rule may have been deleted")
		return jsonutils.NewArray(), nil
	}
	duration, _ := time.ParseDuration(rules[0].TimeFrom)
	//处理逻辑
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), "limit")
	query.Add(jsonutils.NewString("system"), "scope")
	rtn, err := modules.Elasticips.List(session, query)
	if err != nil {
		return nil, err
	}
	EIPUnsedArr := jsonutils.NewArray()
	for _, row := range rtn.Data {
		//Determine whether EIP is used
		if row.ContainsIgnoreCases("associate_type") || row.ContainsIgnoreCases("associate_id") {
			continue
		}
		id, _ := row.GetString("id")
		logInput := logInput{
			ObjId:   id,
			ObjType: "eip",
			Limit:   "0",
			Scope:   "system",
			Action:  db.ACT_DETACH,
		}

		latestTime, err := getResourceObjLatestUsedTime(row, logInput)
		if err != nil {
			continue
		}
		//Judge that the unused time is beyond the duration time
		if time.Now().Add(-duration).Sub(latestTime) < 0 {
			continue
		}
		suggestSysAlert, err := getSuggestSysAlertFromJson(row, rule)
		if err != nil {
			return EIPUnsedArr, errors.Wrap(err, "getEIPUnused's alertData Unmarshal error")
		}

		input := &monitor.SSuggestSysAlertSetting{
			EIPUnused: &monitor.EIPUnused{},
		}
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(input)
		if instance != nil {
			suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		}

		problem := jsonutils.NewDict()
		rtnTime := fmt.Sprintf("%.1fm", time.Now().Sub(latestTime).Minutes())
		problem.Add(jsonutils.NewString(rtnTime), "eipUnused time")
		suggestSysAlert.Problem = problem

		getResourceAmount(suggestSysAlert, latestTime)
		EIPUnsedArr.Add(jsonutils.Marshal(suggestSysAlert))
	}
	return EIPUnsedArr, nil
}

func (rule *EIPUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	var instance *monitor.SSuggestSysAlertSetting
	suggestSysSettingMap, err := models.SuggestSysRuleManager.FetchSuggestSysAlertSettings(rule.GetType())
	if err != nil {
		log.Errorln("DoSuggestSysRule error :", err)
		return
	}
	if details, ok := suggestSysSettingMap[rule.GetType()]; ok {
		instance = details.Setting
	}
	rule.Run(instance)
}

func (rule *EIPUnused) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	_, err := modules.Elasticips.Delete(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		log.Errorln("delete unused eip error", err)
		return err
	}
	return nil
}

func (rule *EIPUnused) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert, params *jsonutils.JSONDict) error {
	suggestSysAlert.SetStatus(userCred, monitor.SUGGEST_ALERT_START_DELETE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ResolveUnusedTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
