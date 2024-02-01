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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceDeleteTask{})
}

func (self *DBInstanceDeleteTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(rds, db.ACT_DELETE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_DELETE, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.DeleteDBInstance(ctx, rds)
}

func (self *DBInstanceDeleteTask) DeleteDBInstance(ctx context.Context, rds *models.SDBInstance) {
	irds, err := rds.GetIDBInstance(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.DeleteDBInstanceComplete(ctx, rds)
			return
		}
		self.taskFailed(ctx, rds, err)
		return
	}

	if !jsonutils.QueryBoolean(self.Params, "purge", false) {
		err = irds.Delete()
		if err != nil {
			self.taskFailed(ctx, rds, err)
			return
		}
		err = cloudprovider.WaitDeleted(irds, time.Second*10, time.Minute*50)
		if err != nil {
			self.taskFailed(ctx, rds, errors.Wrapf(err, "WaitDeleted"))
		}
	}

	self.DeleteDBInstanceComplete(ctx, rds)
}

func (self *DBInstanceDeleteTask) DeleteDBInstanceComplete(ctx context.Context, rds *models.SDBInstance) {
	region, err := rds.GetRegion()
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetRegion"))
		return
	}
	if !region.GetDriver().IsSupportKeepDBInstanceManualBackup() || jsonutils.QueryBoolean(self.Params, "purge", false) {
		err = rds.RealDelete(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, rds, errors.Wrap(err, "Delete"))
			return
		}
		//notifyclient.NotifyWebhook(ctx, self.UserCred, rds, notifyclient.ActionDelete)
		notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
			Obj:    rds,
			Action: notifyclient.ActionDelete,
		})
		self.SetStageComplete(ctx, nil)
		return
	}

	self.DeleteBackups(ctx, rds, nil)
	//notifyclient.NotifyWebhook(ctx, self.UserCred, rds, notifyclient.ActionDelete)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    rds,
		Action: notifyclient.ActionDelete,
	})
}

func (self *DBInstanceDeleteTask) DeleteBackups(ctx context.Context, instance *models.SDBInstance, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(self.Params, "keep_backup", false) {
		backups, _ := instance.GetDBInstanceBackupByMode(api.BACKUP_MODE_MANUAL)
		for i := range backups {
			backups[i].StartDBInstanceBackupDeleteTask(ctx, self.UserCred, "")
		}
	}
	err := instance.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "instance.Purge"))
		return
	}
	self.SetStageComplete(ctx, nil)
}
