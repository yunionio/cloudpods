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
)

type DBInstanceRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstanceRemoteUpdateTask{})
}

func (self *DBInstanceRemoteUpdateTask) taskFail(ctx context.Context, rds *models.SDBInstance, err error) {
	rds.SetStatus(ctx, self.UserCred, api.DBINSTANCE_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstanceRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	rds := obj.(*models.SDBInstance)
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	region, err := rds.GetRegion()
	if err != nil {
		self.taskFail(ctx, rds, errors.Wrapf(err, "GetRegion"))
		return
	}

	if err := region.GetDriver().RequestRemoteUpdateDBInstance(ctx, self.GetUserCred(), rds, replaceTags, self); err != nil {
		self.taskFail(ctx, rds, err)
		return
	}
}

func (self *DBInstanceRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	rds.StartDBInstanceSyncTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *DBInstanceRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.taskFail(ctx, rds, errors.Errorf(data.String()))
}

func (self *DBInstanceRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DBInstanceRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, rds *models.SDBInstance, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
