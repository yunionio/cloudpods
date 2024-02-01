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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KubeNodePoolCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KubeNodePoolCreateTask{})
}

func (self *KubeNodePoolCreateTask) taskFail(ctx context.Context, pool *models.SKubeNodePool, err error) {
	pool.SetStatus(ctx, self.UserCred, apis.STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(pool, db.ACT_ALLOCATE, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, pool, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KubeNodePoolCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	pool := obj.(*models.SKubeNodePool)

	region, err := pool.GetRegion()
	if err != nil {
		self.taskFail(ctx, pool, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnKubeNodePoolCreateComplate", nil)
	err = region.GetDriver().RequestCreateKubeNodePool(ctx, self.UserCred, pool, self)
	if err != nil {
		self.taskFail(ctx, pool, errors.Wrapf(err, "RequestCreateKubeNodePool"))
		return
	}
}

func (self *KubeNodePoolCreateTask) OnKubeNodePoolCreateComplate(ctx context.Context, pool *models.SKubeNodePool, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *KubeNodePoolCreateTask) OnKubeNodePoolCreateComplateFailed(ctx context.Context, pool *models.SKubeNodePool, reason jsonutils.JSONObject) {
	self.taskFail(ctx, pool, errors.Errorf(reason.String()))
}
