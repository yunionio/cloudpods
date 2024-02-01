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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceBackupPackTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupPackTask{})
}

func (self *InstanceBackupPackTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_PACK_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_PACK, reason, self.UserCred, false)
	db.OpsLog.LogEvent(ib, db.ACT_PACK_FAIL, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupPackTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup) {
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_PACK, nil, self.UserCred, true)
	db.OpsLog.LogEvent(ib, db.ACT_PACK, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupPackTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	packageName, _ := self.GetParams().GetString("package_name")
	self.SetStage("OnPackComplete", nil)
	err := ib.GetRegionDriver().RequestPackInstanceBackup(ctx, ib, self, packageName)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceBackupPackTask) OnPackComplete(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	log.Infof("OnPackComplete %s", data)
	notifyclient.NotifyImportantWithCtx(
		ctx,
		[]string{self.UserCred.GetUserId()},
		false,
		"BACKUP_PACK_COMPLETE",
		data,
	)
	self.taskSuccess(ctx, ib)
}

func (self *InstanceBackupPackTask) OnPackCompleteFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}
