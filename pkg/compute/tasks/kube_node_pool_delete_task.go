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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type KubeNodePoolDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(KubeNodePoolDeleteTask{})
}

func (self *KubeNodePoolDeleteTask) taskFailed(ctx context.Context, pool *models.SKubeNodePool, err error) {
	pool.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(pool, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, pool, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *KubeNodePoolDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SKubeNodePool)

	ipool, err := pool.GetIKubeNodePool(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, pool)
			return
		}
		self.taskFailed(ctx, pool, errors.Wrapf(err, "GetIKubeNodePool"))
		return
	}

	err = ipool.Delete()
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "Delete"))
		return
	}

	err = cloudprovider.WaitDeleted(ipool, time.Second*10, time.Minute*10)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "WaitDeleted"))
		return
	}
	self.taskComplete(ctx, pool)
}

func (self *KubeNodePoolDeleteTask) taskComplete(ctx context.Context, pool *models.SKubeNodePool) {
	logclient.AddActionLogWithStartable(self, pool, logclient.ACT_DELETE, pool.GetShortDesc(ctx), self.UserCred, true)
	pool.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
