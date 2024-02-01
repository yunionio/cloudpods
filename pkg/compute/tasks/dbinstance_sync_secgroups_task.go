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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstanceSyncSecgroupsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceSyncSecgroupsTask{})
}

func (self *DBInstanceSyncSecgroupsTask) taskFailed(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_SYNC_SECGROUP_FAILED, err.Error())
	db.OpsLog.LogEvent(rds, db.ACT_SYNC_CONF, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, rds, logclient.ACT_SYNC_CONF, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceSyncSecgroupsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)

	driver, err := rds.GetRegionDriver()
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "GetRegionDriver"))
		return
	}

	self.SetStage("OnSyncSecurityGroupsComplete", nil)
	err = driver.RequestSyncRdsSecurityGroups(ctx, self.GetUserCred(), rds, self)
	if err != nil {
		self.taskFailed(ctx, rds, errors.Wrapf(err, "RequestSyncRdsSecurityGroups"))
		return
	}
}

func (self *DBInstanceSyncSecgroupsTask) OnSyncSecurityGroupsCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *DBInstanceSyncSecgroupsTask) OnSyncSecurityGroupsComplete(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStage("OnSyncComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *DBInstanceSyncSecgroupsTask) OnSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *DBInstanceSyncSecgroupsTask) OnSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
