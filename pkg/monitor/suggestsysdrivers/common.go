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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type logInput struct {
	ObjId   string `json:"obj_id"`
	ObjType string `json:"obj_type"`
	Action  string `json:"action"`
	Scope   string `json:"scope"`
	Limit   string `json:"limit"`
}

func DealAlertData(typ string, oldAlerts []models.SSuggestSysAlert, newAlerts []jsonutils.JSONObject) {
	rules, _ := models.SuggestSysRuleManager.GetRules(typ)
	rules[0].UpdateExecTime()

	oldMap := make(map[string]models.SSuggestSysAlert, 0)
	for _, alert := range oldAlerts {
		oldMap[alert.ResId] = alert
	}
	for _, newAlert := range newAlerts {
		res_id, _ := newAlert.GetString("res_id")
		if oldAlert, ok := oldMap[res_id]; ok {
			//更新的alert
			_, err := db.Update(&oldAlert, func() error {
				err := newAlert.Unmarshal(&oldAlert)
				if err != nil {
					errMsg := fmt.Sprintf("unmarshal fail: %s", err)
					log.Errorf(errMsg)
				}
				return nil
			})
			if err != nil {
				log.Errorln("更新alert失败", err)
			}
			delete(oldMap, res_id)
		} else {
			//新增的alert
			adminCredential := auth.AdminCredential()
			_, err := db.DoCreate(models.SuggestSysAlertManager, context.Background(), adminCredential, nil, newAlert,
				adminCredential)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	for _, oldAlert := range oldMap {
		err := oldAlert.RealDelete(context.Background(), auth.AdminCredential())
		if err != nil {
			log.Errorln("删除旧alert数据失败", err)
		}
	}
}

func doSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool, rule models.ISuggestSysRuleDriver) {
	var instance *monitor.SSuggestSysAlertSetting
	suggestSysSettingMap, err := models.SuggestSysRuleManager.FetchSuggestSysAlartSettings(rule.GetType())
	if err != nil {
		log.Errorln("DoSuggestSysRule error :", err)
		return
	}
	if details, ok := suggestSysSettingMap[rule.GetType()]; ok {
		instance = details.Setting
	}
	rule.Run(instance)
}

func getLastAlerts(rule models.ISuggestSysRuleDriver) ([]models.SSuggestSysAlert, error) {
	oldAlert, err := models.SuggestSysAlertManager.GetResources(rule.GetType())
	if err != nil {
		log.Errorln(errors.Wrap(err, "db.FetchModelObjects"))
		return oldAlert, err
	}
	return oldAlert, nil
}

func getSuggestSysAlertFromJson(obj jsonutils.JSONObject, rule models.ISuggestSysRuleDriver) (*models.SSuggestSysAlert, error) {
	suggestSysAlert := new(models.SSuggestSysAlert)
	alertData := jsonutils.DeepCopy(obj).(*jsonutils.JSONDict)
	id, _ := alertData.GetString("id")
	alertData.Add(jsonutils.NewString(id), "res_id")
	alertData.Remove("id")

	err := alertData.Unmarshal(suggestSysAlert)
	if err != nil {
		return nil, errors.Wrap(err, "getSuggestSysAlertFromJson's alertData Unmarshal error")
	}
	if val, err := alertData.GetString("account"); err == nil {
		suggestSysAlert.Cloudaccount = val
	}
	suggestSysAlert.Type = rule.GetType()
	suggestSysAlert.ResMeta = obj
	suggestSysAlert.Action = monitor.DRIVER_ACTION
	suggestSysAlert.Status = monitor.SUGGEST_ALERT_READY
	return suggestSysAlert, nil
}

func getResourceObjLatestUsedTime(resObj jsonutils.JSONObject, param logInput) (time.Time, error) {
	logActions := getResourceObjLogOfAction(param)
	latestTime, err := getLatestActionTimeFromLogs(logActions)
	if err != nil {
		return time.Time{}, err
	}

	if latestTime == nil {
		createdAt, _ := resObj.GetTime("created_at")
		latestTime = &createdAt
	}
	return *latestTime, nil
}

func getResourceObjLogOfAction(param logInput) []jsonutils.JSONObject {
	session := auth.GetAdminSession(context.Background(), "", "")
	list, err := mod.Logs.List(session, jsonutils.Marshal(&param))
	if err != nil {
		log.Errorln("get Logs err", err)
		return jsonutils.NewArray().Value()
	}
	if list == nil || len(list.Data) == 0 {
		return jsonutils.NewArray().Value()
	}
	return list.Data
}

func getLatestActionTimeFromLogs(logActions []jsonutils.JSONObject) (*time.Time, error) {
	var latestTime *time.Time = nil
	for _, aLog := range logActions {
		ops_time, err := aLog.GetTime("ops_time")
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		if latestTime == nil {

			latestTime = &ops_time
		}
		if ops_time.Sub(*latestTime) > 0 {
			latestTime = &ops_time
		}
	}
	return latestTime, nil
}
