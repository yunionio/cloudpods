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
	"strings"
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

func (modelartsCreateTask *ModelartsPoolCreateTask) taskFailed(ctx context.Context, pool *models.SModelartsPool, status string, err error) {
	pool.SetStatus(modelartsCreateTask.UserCred, status, err.Error())
	db.OpsLog.LogEvent(pool, db.ACT_ALLOCATE, err, modelartsCreateTask.UserCred)
	logclient.AddActionLogWithStartable(modelartsCreateTask, pool, logclient.ACT_ALLOCATE, err, modelartsCreateTask.UserCred, false)
	modelartsCreateTask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (modelartsCreateTask *ModelartsPoolCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)
	iRegion, err := pool.GetIRegion()
	if err != nil {
		modelartsCreateTask.taskFailed(ctx, pool, api.MODELARTS_POOL_STATUS_CREATE_FAILED, errors.Wrapf(err, "pool.GetIRegion"))
		return
	}

	opts := &cloudprovider.ModelartsPoolCreateOption{
		Name:         pool.Name,
		InstanceType: pool.InstanceType,
		WorkType:     pool.WorkType,
		NodeCount:    pool.NodeCount,
		Cidr:         pool.Cidr,
	}

	ipool, err := iRegion.CreateIModelartsPool(opts)
	if err != nil {
		modelartsCreateTask.taskFailed(ctx, pool, api.MODELARTS_POOL_STATUS_CREATE_FAILED, errors.Wrapf(err, "iRegion.CreateIModelartsPool"))
		return
	}
	err = db.SetExternalId(pool, modelartsCreateTask.GetUserCred(), ipool.GetGlobalId())
	if err != nil {
		modelartsCreateTask.taskFailed(ctx, pool, api.MODELARTS_POOL_STATUS_CREATE_FAILED, errors.Wrapf(err, "db.SetExternalId"))
		return
	}
	modelartsCreateTask.SetStage("OnModelartsPoolCreateComplete", nil)
	// withDelay
	modelartsCreateTask.WaitStatus(ctx, ipool, pool)
}

func (modelartsCreateTask *ModelartsPoolCreateTask) taskComplete(ctx context.Context, pool *models.SModelartsPool) {
	pool.SetStatus(modelartsCreateTask.GetUserCred(), api.MODELARTS_POOL_STATUS_RUNNING, "")
	notifyclient.EventNotify(ctx, modelartsCreateTask.UserCred, notifyclient.SEventNotifyParam{
		Obj:    modelartsCreateTask,
		Action: notifyclient.ActionCreate,
	})
}

func (modelartsCreateTask *ModelartsPoolCreateTask) WaitStatus(ctx context.Context, ipool cloudprovider.ICloudModelartsPool, pool *models.SModelartsPool) error {
	taskman.LocalTaskRun(modelartsCreateTask, func() (jsonutils.JSONObject, error) {
		time.Sleep(2 * time.Minute)
		err := cloudprovider.WaitMultiStatus(ipool, []string{api.MODELARTS_POOL_STATUS_RUNNING, api.MODELARTS_POOL_STATUS_CREATE_FAILED}, 15*time.Second, 2*time.Hour)
		if err != nil {
			return nil, err
		}
		if ipool.GetStatus() == api.MODELARTS_POOL_STATUS_CREATE_FAILED {
			return nil, errors.Error(ipool.GetStatusMessage())
		} else {
			return nil, nil
		}
	})
	return nil
}

func (modelartsCreateTask *ModelartsPoolCreateTask) OnModelartsPoolCreateCompleteFailed(ctx context.Context, modelarts *models.SModelartsPool, err jsonutils.JSONObject) {
	if strings.Contains(err.String(), errors.ErrTimeout.Error()) {
		modelartsCreateTask.taskFailed(ctx, modelarts, api.MODELARTS_POOL_STATUS_TIMEOUT, errors.Errorf(err.String()))
	} else {
		modelartsCreateTask.taskFailed(ctx, modelarts, api.MODELARTS_POOL_STATUS_CREATE_FAILED, errors.Errorf(err.String()))
	}
}

func (modelartsCreateTask *ModelartsPoolCreateTask) OnModelartsPoolCreateComplete(ctx context.Context, modelarts *models.SModelartsPool, body jsonutils.JSONObject) {
	modelartsCreateTask.taskComplete(ctx, modelarts)
}
