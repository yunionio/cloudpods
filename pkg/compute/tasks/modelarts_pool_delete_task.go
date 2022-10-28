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

	apis "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ModelartsPoolDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolDeleteTask{})
}

func (self *ModelartsPoolDeleteTask) taskFailed(ctx context.Context, mp *models.SModelartsPool, err error) {
	mp.SetStatus(self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(mp, db.ACT_DELETE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, mp, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)

	if len(pool.ExternalId) == 0 {
		self.taskComplete(ctx, pool)
		return
	}
	iMp, err := pool.GetIModelartsPool()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, pool)
			return
		}
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iMp.GetIModelartsPoolById"))
		return
	}
	err = iMp.Delete()
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iMp.Delete"))
		return
	}
	err = cloudprovider.WaitDeleted(iMp, time.Second*10, time.Minute*10)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iMp.WaitDeleted"))
		return
	}
	self.taskComplete(ctx, pool)
}

func (self *ModelartsPoolDeleteTask) taskComplete(ctx context.Context, pool *models.SModelartsPool) {
	pool.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
