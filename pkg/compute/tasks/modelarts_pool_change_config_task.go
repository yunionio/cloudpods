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

type ModelartsPoolChangeConfigTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolChangeConfigTask{})
}

func (self *ModelartsPoolChangeConfigTask) taskFailed(ctx context.Context, mp *models.SModelartsPool, err error) {
	mp.SetStatus(ctx, self.UserCred, api.MODELARTS_POOL_STATUS_CHANGE_CONFIG_FAILED, err.Error())
	db.OpsLog.LogEvent(mp, db.ACT_CHANGE_CONFIG, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, mp, logclient.ACT_CHANGE_CONFIG, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolChangeConfigTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	pool := obj.(*models.SModelartsPool)
	iMp, err := pool.GetIModelartsPool()
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iMp.GetIModelartsPoolById"))
		return
	}
	input := &api.ModelartsPoolChangeConfigInput{}
	self.GetParams().Unmarshal(input)
	opts := &cloudprovider.ModelartsPoolChangeConfigOptions{}
	opts.NodeCount = input.NodeCount
	err = iMp.ChangeConfig(opts)
	if err != nil {
		self.taskFailed(ctx, pool, errors.Wrapf(err, "iMp.ChangeConfig"))
		return
	}
	// withDelay
	time.Sleep(30 * time.Second)
	err = cloudprovider.WaitMultiStatus(iMp, []string{api.MODELARTS_POOL_STATUS_RUNNING, api.MODELARTS_POOL_STATUS_CHANGE_CONFIG_FAILED}, 15*time.Second, 2*time.Hour)
	if err != nil {
		pool.SetStatus(ctx, self.UserCred, api.MODELARTS_POOL_STATUS_TIMEOUT, err.Error())
		db.OpsLog.LogEvent(pool, db.ACT_CHANGE_CONFIG, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, pool, logclient.ACT_CHANGE_CONFIG, err, self.UserCred, false)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	pool.SyncWithCloudModelartsPool(ctx, self.GetUserCred(), iMp)
	self.taskComplete(ctx, pool)
}

func (self *ModelartsPoolChangeConfigTask) taskComplete(ctx context.Context, pool *models.SModelartsPool) {
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionChangeConfig,
	})
	self.SetStageComplete(ctx, nil)
}
