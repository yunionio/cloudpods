package suggestsysdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

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
	return suggestSysAlert, nil
}
