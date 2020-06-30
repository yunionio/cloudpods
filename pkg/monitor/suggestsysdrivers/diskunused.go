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

type DiskUnused struct {
	*baseDriver
}

func NewDiskUnusedDriver() models.ISuggestSysRuleDriver {
	return &DiskUnused{
		baseDriver: newBaseDriver(
			monitor.DISK_UNUSED,
			monitor.DISK_MONITOR_RES_TYPE,
			monitor.DELETE_DRIVER_ACTION,
			monitor.DISK_MONITOR_SUGGEST,
		),
	}
}

func (drv *DiskUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, drv)
}

func (drv *DiskUnused) Run(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, setting)
}

func (drv *DiskUnused) GetLatestAlerts(rule *models.SSuggestSysRule, instance *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	duration, _ := time.ParseDuration(rule.TimeFrom)
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewBool(true), "unused")
	disks, err := ListAllResources(&modules.Disks, query)
	if err != nil {
		return nil, err
	}
	diskUnusedArr := make([]jsonutils.JSONObject, 0)
	for _, disk := range disks {
		id, _ := disk.GetString("id")
		logInput := logInput{
			ObjId:   id,
			ObjType: "disk",
			Limit:   "0",
			Scope:   "system",
			Action:  db.ACT_DETACH,
		}
		latestTime, err := getResourceObjLatestUsedTime(disk, logInput)
		if err != nil {
			continue
		}

		if time.Now().Add(-duration).Sub(latestTime) < 0 {
			continue
		}
		suggestSysAlert, err := getSuggestSysAlertFromJson(disk, drv)
		if err != nil {
			return diskUnusedArr, errors.Wrap(err, "getEIPUnused's alertData Unmarshal error")
		}

		input := &monitor.SSuggestSysAlertSetting{
			DiskUnused: &monitor.DiskUnused{},
		}
		suggestSysAlert.MonitorConfig = jsonutils.Marshal(input)
		if instance != nil {
			suggestSysAlert.MonitorConfig = jsonutils.Marshal(instance)
		}

		problem := jsonutils.NewDict()
		rtnTime := fmt.Sprintf("%.1fm", time.Now().Sub(latestTime).Minutes())
		problem.Add(jsonutils.NewString(rtnTime), "diskUnused time")
		suggestSysAlert.Problem = problem
		diskUnusedArr = append(diskUnusedArr, jsonutils.Marshal(suggestSysAlert))
	}
	return diskUnusedArr, nil
}

func (drv *DiskUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	obj := new(monitor.DiskUnused)
	input.DiskUnused = obj
	return nil
}

func (drv *DiskUnused) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert, params *jsonutils.JSONDict) error {
	suggestSysAlert.SetStatus(userCred, monitor.SUGGEST_ALERT_START_DELETE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ResolveUnusedTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (drv *DiskUnused) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	_, err := modules.Disks.Delete(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		log.Errorln("delete unused error", err)
		return err
	}
	return nil
}
