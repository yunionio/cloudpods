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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ModelartsPoolCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolCreateTask{})
}

func (self *ModelartsPoolCreateTask) taskFailed(ctx context.Context, pool *models.SModelartsPool, err error) {
	pool.SetStatus(self.UserCred, api.MODELARTS_POOL_STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, pool, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)
	iRegion, err := pool.GetIRegion()
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "pool.GetIRegion"))
		return
	}

	opts := &cloudprovider.ModelartsPoolCreateOption{
		Name:         pool.Name,
		InstanceType: pool.InstanceType,
		WorkType:     pool.WorkType,
	}

	ipool, err := iRegion.CreateIModelartsPool(opts)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iProvider.CreateIModelartsPool"))
		return
	}
	err = db.SetExternalId(pool, self.GetUserCred(), ipool.GetGlobalId())
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "db.SetExternalId"))
		return
	}
	err = cloudprovider.WaitStatusWithDelay(ipool, api.MODELARTS_POOL_STATUS_RUNNING, 30*time.Second, 15*time.Second, 600*time.Second)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "db.WaitStatusWithDelay"))
		return
	}
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})

	pool.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}
