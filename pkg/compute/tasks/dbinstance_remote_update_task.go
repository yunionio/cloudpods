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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DBInstanceRemoteUpdateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(DBInstanceRemoteUpdateTask{})
}

func (self *DBInstanceRemoteUpdateTask) taskFail(ctx context.Context, dbinstance *models.SDBInstance, reason jsonutils.JSONObject) {
	dbinstance.SetStatus(self.UserCred, api.DBINSTANCE_UPDATE_TAGS_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *DBInstanceRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SDBInstance)
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	if err := instance.GetRegion().GetDriver().RequestRemoteUpdateDBInstance(ctx, self.GetUserCred(), instance, replaceTags, self); err != nil {
		self.taskFail(ctx, instance, jsonutils.NewString(err.Error()))
	}
}

func (self *DBInstanceRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
