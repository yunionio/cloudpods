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
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

func init() {
	models.RegisterSuggestSysRuleDrivers(NewEIPUsedDriver(), NewDiskUnusedDriver())
}

func InitSuggestSysRuleCronjob() {
	objs := make([]models.SSuggestSysRule, 0)
	q := models.SuggestSysRuleManager.Query()
	if q == nil {
		fmt.Println(" query is nil")
	}
	err := db.FetchModelObjects(models.SuggestSysRuleManager, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		log.Errorln("InitSuggestSysRuleCronjob db.FetchModelObjects error")
	}
	for _, driver := range models.GetSuggestSysRuleDrivers() {
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(driver.GetType(), time.Duration(30)*time.Second,
			driver.DoSuggestSysRule, true)
	}
	for _, suggestSysRuleConfig := range objs {
		cronman.GetCronJobManager().Remove(suggestSysRuleConfig.Type)
		if suggestSysRuleConfig.Enabled.Bool() {
			dur, _ := time.ParseDuration(suggestSysRuleConfig.Period)
			cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(suggestSysRuleConfig.Type, dur,
				models.GetSuggestSysRuleDrivers()[suggestSysRuleConfig.Type].DoSuggestSysRule, true)
		}
	}
}

func DealAlertData(oldAlerts []models.SSuggestSysAlert, newAlerts []jsonutils.JSONObject) {
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
		err := oldAlert.Delete(context.Background(), auth.AdminCredential())
		if err != nil {
			log.Errorln("删除旧alert数据失败", err)
		}
	}
}
