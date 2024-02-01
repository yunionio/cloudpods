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
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheSyncstatusTask{})
}

func (self *ElasticcacheSyncstatusTask) taskFailed(ctx context.Context, cache *models.SElasticcache, err jsonutils.JSONObject) {
	cache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_UNKNOWN, err.String())
	self.SetStageFailed(ctx, err)
	db.OpsLog.LogEvent(cache, db.ACT_SYNC_STATUS, cache.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, cache, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    cache,
		Action: notifyclient.ActionSyncStatus,
		IsFail: true,
	})
}

func (self *ElasticcacheSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SElasticcache)

	region, _ := cache.GetRegion()
	if region == nil {
		self.taskFailed(ctx, cache, jsonutils.NewString(fmt.Sprintf("failed to found cloudregion for elasticcache %s(%s)", cache.Name, cache.Id)))
		return
	}

	self.SetStage("OnElasticcacheSyncStatusComplete", nil)
	err := region.GetDriver().RequestSyncElasticcacheStatus(ctx, self.GetUserCred(), cache, self)
	if err != nil {
		self.taskFailed(ctx, cache, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *ElasticcacheSyncstatusTask) OnElasticcacheSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheSyncstatusTask) OnElasticcacheSyncStatusCompleteFailed(ctx context.Context, cache *models.SElasticcache, data jsonutils.JSONObject) {
	self.taskFailed(ctx, cache, data)
}
