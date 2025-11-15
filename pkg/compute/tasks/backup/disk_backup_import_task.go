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

package backup

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBackupImportTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskBackupImportTask{})
}

func (importTask *DiskBackupImportTask) taskFailed(ctx context.Context, backup *models.SDiskBackup, reason jsonutils.JSONObject, status string) {
	reasonStr, _ := reason.GetString()
	backup.SetStatus(ctx, importTask.UserCred, status, reasonStr)
	logclient.AddActionLogWithStartable(importTask, backup, logclient.ACT_CREATE, reason, importTask.UserCred, false)
	importTask.SetStageFailed(ctx, reason)
}

func (importTask *DiskBackupImportTask) taksSuccess(ctx context.Context, backup *models.SDiskBackup, data *jsonutils.JSONDict) {
	backup.SetStatus(ctx, importTask.UserCred, api.BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(importTask, backup, logclient.ACT_CREATE, backup.GetShortDesc(ctx), importTask.UserCred, true)
	notifyclient.EventNotify(ctx, importTask.UserCred, notifyclient.SEventNotifyParam{
		Obj:    backup,
		Action: notifyclient.ActionCreate,
	})
	importTask.SetStageComplete(ctx, data)
}

func (importTask *DiskBackupImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)
	input := api.DiskBackupImportTaskInput{}
	err := importTask.Params.Unmarshal(&input)
	if err != nil {
		importTask.taskFailed(ctx, backup, jsonutils.NewString(err.Error()), api.BACKUP_STATUS_IMPORT_FAILED)
		return
	}

	backup.SetStatus(ctx, importTask.UserCred, api.BACKUP_STATUS_IMPORTING, "")

	importTask.SetStage("OnImportComplete", nil)
	taskman.LocalTaskRun(importTask, func() (jsonutils.JSONObject, error) {
		err := backup.DoImport(ctx, importTask.UserCred, input)
		if err != nil {
			return nil, errors.Wrap(err, "DoImport")
		}
		return nil, nil
	})
}

func (importTask *DiskBackupImportTask) OnImportComplete(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	importTask.taksSuccess(ctx, backup, nil)
}

func (importTask *DiskBackupImportTask) OnImportCompleteFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	importTask.taskFailed(ctx, backup, data, api.BACKUP_STATUS_IMPORT_FAILED)
}
