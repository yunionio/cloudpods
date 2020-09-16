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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/dbinit"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type SnapshotUnused struct {
	*baseDriver
}

func NewSnapshotUnusedDriver() models.ISuggestSysRuleDriver {
	return &SnapshotUnused{
		baseDriver: newBaseDriver(
			monitor.SNAPSHOT_UNUSED,
			monitor.SNAPSHOT_MONITOR_RES_TYPE,
			monitor.DELETE_DRIVER_ACTION,
			monitor.MonitorSuggest("释放未使用的快照"),
			*dbinit.SnapShotUnusedCreateInput,
		),
	}
}

func (rule *SnapshotUnused) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, rule)
}

func (drv *SnapshotUnused) Run(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, setting)
}

func (drv *SnapshotUnused) GetLatestAlerts(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	duration, _ := time.ParseDuration(rule.TimeFrom)

	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("snapshot_id.isnotnull()"), "filter.0")
	query.Add(jsonutils.NewString("snapshot_id.isnotempty()"), "filter.1")
	unusedSnapshotDisks, err := ListAllResources(&modules.Disks, query)
	if err != nil {
		return nil, errors.Wrap(err, "list unused snapshot disks")
	}
	snapshotQuery := jsonutils.NewDict()
	snapshotQuery.Add(jsonutils.NewString("ref_count.le(0)"))
	snapshots, err := ListAllResources(&modules.Snapshots, snapshotQuery)
	if err != nil {
		return nil, errors.Wrap(err, "list snapshots")
	}
	snapshots = drv.filterUnusedSnapshots(snapshots, unusedSnapshotDisks)
	snapshots = drv.filterSnapshotsByTime(snapshots, duration)

	unusedResult := make([]jsonutils.JSONObject, 0)
	for _, snapshot := range snapshots {
		alert, err := getSuggestSysAlertFromJson(snapshot, drv)
		if err != nil {
			return unusedResult, errors.Wrap(err, "get unused snapshot")
		}
		updateAt, _ := snapshot.GetTime("updated_at")
		problems := []monitor.SuggestAlertProblem{
			monitor.SuggestAlertProblem{
				Type:        "snapshotUnused time",
				Description: fmt.Sprintf("%.1fm", time.Now().Sub(updateAt).Minutes()),
			},
		}
		alert.Problem = jsonutils.Marshal(&problems)
		unusedResult = append(unusedResult, jsonutils.Marshal(alert))
	}
	return unusedResult, nil
}

func (rule *SnapshotUnused) filterUnusedSnapshots(snapshots []jsonutils.JSONObject, disks []jsonutils.JSONObject) []jsonutils.JSONObject {
	unused := make([]jsonutils.JSONObject, 0)
	for _, snapshot := range snapshots {
		if !rule.isSnapshotUsedByDisks(snapshot, disks) {
			unused = append(unused, snapshot)
		}
	}
	return unused
}

func (rule *SnapshotUnused) filterSnapshotsByTime(snapshots []jsonutils.JSONObject, duration time.Duration) []jsonutils.JSONObject {
	result := make([]jsonutils.JSONObject, 0)
	for _, snapshot := range snapshots {
		updateAt, _ := snapshot.GetTime("updated_at")
		if time.Now().Add(-duration).Sub(updateAt) < 0 {
			continue
		}
		result = append(result, snapshot)
	}
	return result
}

func (rule *SnapshotUnused) isSnapshotUsedByDisks(snapshot jsonutils.JSONObject, disks []jsonutils.JSONObject) bool {
	for _, disk := range disks {
		snapshotId, _ := snapshot.GetString("id")
		diskSnapshotId, _ := disk.GetString("snapshot_id")
		if diskSnapshotId != "" && snapshotId == diskSnapshotId {
			return true
		}
	}
	return false
}

func (rule *SnapshotUnused) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	return nil
}

func (rule *SnapshotUnused) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential,
	suggestSysAlert *models.SSuggestSysAlert, params *jsonutils.JSONDict) error {
	suggestSysAlert.SetStatus(userCred, monitor.SUGGEST_ALERT_START_DELETE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ResolveUnusedTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (rule *SnapshotUnused) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	_, err := modules.Snapshots.Delete(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		log.Errorln("delete unused error", err)
		return errors.Wrapf(err, "delete unused snapshot %s", data.ResId)
	}
	return nil
}
