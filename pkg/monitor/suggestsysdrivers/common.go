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
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
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

func DealAlertData(drvType monitor.SuggestDriverType, oldAlerts []models.SSuggestSysAlert, newAlerts []jsonutils.JSONObject) {
	rules, err := models.SuggestSysRuleManager.GetRules(drvType)
	if err != nil {
		log.Errorf("get suggest rule by type %q error: %v", drvType, err)
		return
	}
	if len(rules) == 0 {
		log.Errorf("not found suggest rule by type %q", drvType)
		return
	}

	rules[0].UpdateExecTime()
	adminCredential := auth.AdminCredential()

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
			_, err := db.DoCreate(models.SuggestSysAlertManager, context.Background(), adminCredential, nil, newAlert,
				adminCredential)
			if err != nil {
				log.Errorf("create new suggest alert %v error: %v", newAlert, err)
			}
		}
	}

	for _, oldAlert := range oldMap {
		err := oldAlert.RealDelete(context.Background(), adminCredential)
		if err != nil {
			log.Errorln("删除旧alert数据失败", err)
		}
	}
}

func doSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool, drv models.ISuggestSysRuleDriver) {
	var instance *monitor.SSuggestSysAlertSetting
	suggestSysSettingMap, err := models.SuggestSysRuleManager.FetchSuggestSysAlertSettings(drv.GetType())
	if err != nil {
		log.Errorf("DoSuggestSysRule error: %v", err)
		return
	}
	if details, ok := suggestSysSettingMap[drv.GetType()]; ok {
		instance = details.Setting
	}
	rule, err := models.SuggestSysRuleManager.GetRuleByType(drv.GetType())
	if err != nil {
		log.Errorf("Get rule by type %s: %v", drv.GetType(), err)
		return
	}
	drv.Run(rule, instance)
}

func getLastAlerts(rule models.ISuggestSysRuleDriver) ([]models.SSuggestSysAlert, error) {
	oldAlert, err := models.SuggestSysAlertManager.GetResources(rule.GetType())
	if err != nil {
		return oldAlert, errors.Wrapf(err, "get last alerts by type %s", rule.GetType())
	}
	return oldAlert, nil
}

type iRuleDriver interface {
	models.ISuggestSysRuleDriver
	GetLatestAlerts(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error)
}

func Run(drv iRuleDriver, rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	oldAlert, err := getLastAlerts(drv)
	if err != nil {
		log.Errorf("get %s old alert result: %v", drv.GetType(), err)
		return
	}

	newAlerts, err := drv.GetLatestAlerts(rule, setting)
	if err != nil {
		log.Errorf("get %s latest alert results: %v", drv.GetType(), err)
		return
	}
	DealAlertData(drv.GetType(), oldAlert, newAlerts)
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
	suggestSysAlert.Type = string(rule.GetType())
	suggestSysAlert.ResMeta = obj
	suggestSysAlert.Action = string(rule.GetAction())
	suggestSysAlert.Status = monitor.SUGGEST_ALERT_READY
	getResourceAmount(suggestSysAlert, time.Now().Add(-30*24*time.Hour))
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
		return []jsonutils.JSONObject{}
	}
	if list == nil || len(list.Data) == 0 {
		return []jsonutils.JSONObject{}
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

func getResourceAmount(alert *models.SSuggestSysAlert, lastUsedTime time.Time) {
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString("system"), "scope")
	param.Add(jsonutils.NewString("0"), "limit")
	filter := fmt.Sprintf("resource_id.equals(%s)", alert.ResId)
	param.Add(jsonutils.NewString(filter), "filter")

	start_day := timeutils.ShortDate(lastUsedTime)
	end_day := timeutils.ShortDate(time.Now())

	param.Add(jsonutils.NewString(start_day), "start_day")
	param.Add(jsonutils.NewString(end_day), "end_day")
	session := auth.GetAdminSession(context.Background(), "", "")
	billRtn, err := mod.DailyBills.List(session, param)
	if err != nil {
		log.Errorln(err)
		return
	}
	for _, bill := range billRtn.Data {
		amount, err := bill.Float("amount")
		if err != nil {
			log.Errorln(err)
			break
		}
		alert.Amount += amount
		currency, err := bill.GetString("currency")
		if err != nil {
			log.Errorln(err)
		}
		alert.Currency = currency
	}
}

func ListAllResources(manager modulebase.Manager, params *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.NewInt(0), "limit")
	var count int
	session := auth.GetAdminSession(context.Background(), "", "")
	objs := make([]jsonutils.JSONObject, 0)
	for {
		params.Set("offset", jsonutils.NewInt(int64(count)))
		result, err := manager.List(session, params)
		if err != nil {
			return nil, errors.Wrapf(err, "list %s resources with params %s", manager.KeyString(), params.String())
		}
		for _, data := range result.Data {
			objs = append(objs, data)
		}
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	return objs, nil
}
