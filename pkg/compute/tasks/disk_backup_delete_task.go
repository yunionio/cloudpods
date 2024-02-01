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

func (self *DiskBackupDeleteTask) taskFailed(ctx context.Context, backup *models.SDiskBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	backup.SetStatus(ctx, self.UserCred, api.BACKUP_STATUS_DELETE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, backup, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *DiskBackupDeleteTask) taskSuccess(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	backup.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskBackupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	backup := obj.(*models.SDiskBackup)
	self.SetStage("OnDelete", nil)
	rd, err := backup.GetRegionDriver()
	if err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
		return
	}
	if err := rd.RequestDeleteBackup(ctx, backup, self); err != nil {
		self.taskFailed(ctx, backup, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskBackupDeleteTask) OnDelete(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	self.taskSuccess(ctx, backup, nil)
}

func (self *DiskBackupDeleteTask) OnDeleteFailed(ctx context.Context, backup *models.SDiskBackup, data jsonutils.JSONObject) {
	log.Infof("params: %s", self.Params)
	log.Infof("OnDeleteFailed data: %s", data)
	if forceDelete := jsonutils.QueryBoolean(self.Params, "force_delete", false); !forceDelete {
		self.taskFailed(ctx, backup, data)
	}
	reason, _ := data.GetString("__reason__")
	if !strings.Contains(reason, api.BackupStorageOffline) {
		self.taskFailed(ctx, backup, data)
	}
	log.Infof("delete backup %s failed, force delete", backup.GetId())
	self.taskSuccess(ctx, backup, nil)
}
