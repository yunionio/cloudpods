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

package tasks

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBackupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskBackupDeleteTask{})
}

func (tsk *DiskBackupDeleteTask) taskFailed(ctx context.Context, backup *models.SDiskBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	backup.SetStatus(tsk.UserCred, api.BACKUP_STATUS_DELETE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(tsk, backup, logclient.ACT_DELETE, reason, tsk.UserCred, false)
	tsk.SetStageFailed(ctx, reason)
}

func (tsk *DiskBackupDeleteTask) taskSuccess(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	backup.RealDelete(ctx, tsk.UserCred)
	logclient.AddActionLogWithStartable(tsk, backup, logclient.ACT_DELETE, backup.GetShortDesc(ctx), tsk.UserCred, true)
	tsk.SetStageComplete(ctx, nil)
}

func (tsk *DiskBackupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)
	rd, err := backup.GetRegionDriver()
	if err != nil {
		tsk.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
	tsk.SetStage("OnDeleteComplete", nil)
	if err := rd.RequestDeleteBackup(ctx, backup, tsk); err != nil {
		tsk.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
}

func (tsk *DiskBackupDeleteTask) OnDeleteComplete(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	tsk.taskSuccess(ctx, backup, nil)
}

func (tsk *DiskBackupDeleteTask) OnDeleteCompleteFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	log.Infof("params: %s", tsk.Params)
	log.Infof("OnDeleteFailed data: %s", data)
	if forceDelete := jsonutils.QueryBoolean(tsk.Params, "force_delete", false); !forceDelete {
		tsk.taskFailed(ctx, backup, data)
		return
	}
	reason, _ := data.GetString("__reason__")
	if !strings.Contains(reason, api.BackupStorageOffline) {
		tsk.taskFailed(ctx, backup, data)
		return
	}
	log.Infof("delete backup %s failed, force delete", backup.GetId())
	tsk.taskSuccess(ctx, backup, nil)
}
