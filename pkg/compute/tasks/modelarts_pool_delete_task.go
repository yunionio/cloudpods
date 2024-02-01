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

type ModelartsPoolDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolDeleteTask{})
}

func (modelartsDeleteTask *ModelartsPoolDeleteTask) taskFailed(ctx context.Context, status string, mp *models.SModelartsPool, err error) {
	mp.SetStatus(ctx, modelartsDeleteTask.UserCred, status, err.Error())
	db.OpsLog.LogEvent(mp, db.ACT_DELETE_FAIL, err, modelartsDeleteTask.UserCred)
	logclient.AddActionLogWithStartable(modelartsDeleteTask, mp, logclient.ACT_DELOCATE, err, modelartsDeleteTask.UserCred, false)
	modelartsDeleteTask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (modelartsDeleteTask *ModelartsPoolDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)

	if len(pool.ExternalId) == 0 {
		modelartsDeleteTask.taskComplete(ctx, pool)
		return
	}
	iMp, err := pool.GetIModelartsPool()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			modelartsDeleteTask.taskComplete(ctx, pool)
			return
		}
		modelartsDeleteTask.taskFailed(ctx, api.MODELARTS_POOL_STATUS_DELETE_FAILED, pool, errors.Wrapf(err, "iMp.GetIModelartsPoolById"))
		return
	}
	err = iMp.Delete()
	if err != nil {
		modelartsDeleteTask.taskFailed(ctx, api.MODELARTS_POOL_STATUS_DELETE_FAILED, pool, errors.Wrapf(err, "iMp.Delete"))
		return
	}
	err = cloudprovider.WaitStatus(iMp, api.MODELARTS_POOL_STATUS_UNKNOWN, time.Second*15, time.Minute*20)
	if err != nil {
		if errors.Cause(err) == errors.ErrTimeout {
			modelartsDeleteTask.taskFailed(ctx, api.MODELARTS_POOL_STATUS_TIMEOUT, pool, errors.Wrapf(err, "ErrTimeout"))
			return
		} else if errors.Cause(err) == errors.ErrNotFound {
			modelartsDeleteTask.taskComplete(ctx, pool)
			return
		} else {
			modelartsDeleteTask.taskFailed(ctx, api.MODELARTS_POOL_STATUS_DELETE_FAILED, pool, errors.Wrapf(err, "default:"))
			return
		}
	}
	modelartsDeleteTask.taskComplete(ctx, pool)
}

func (modelartsDeleteTask *ModelartsPoolDeleteTask) taskComplete(ctx context.Context, pool *models.SModelartsPool) {
	pool.RealDelete(ctx, modelartsDeleteTask.GetUserCred())
	notifyclient.EventNotify(ctx, modelartsDeleteTask.UserCred, notifyclient.SEventNotifyParam{
		Obj:    modelartsDeleteTask,
		Action: notifyclient.ActionDelete,
	})
	modelartsDeleteTask.SetStageComplete(ctx, nil)
}
