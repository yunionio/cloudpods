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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudProviderSyncInfoTask struct {
	taskman.STask
}

var syncLocalTaskWorkerMan *appsrv.SWorkerManager

func InitCloudproviderSyncWorkers(count int) {
	syncWorker := appsrv.NewWorkerManager("CloudProviderSyncInfoTaskWorkerManager", count, 512, true)
	taskman.RegisterTaskAndWorker(CloudProviderSyncInfoTask{}, syncWorker)
	syncLocalTaskWorkerMan = appsrv.NewWorkerManager("CloudProviderSyncLocalTaskWorkerManager", count, 512, false)
}

func getAction(params *jsonutils.JSONDict) string {
	fullSync := jsonutils.QueryBoolean(params, "full_sync", false)
	if !fullSync {
		syncRangeJson, _ := params.Get("sync_range")
		if syncRangeJson != nil {
			fullSync = jsonutils.QueryBoolean(syncRangeJson, "full_sync", false)
		}
	}

	action := ""

	if fullSync {
		action = logclient.ACT_CLOUD_FULLSYNC
	} else {
		action = logclient.ACT_CLOUD_SYNC
	}
	return action
}

func (self *CloudProviderSyncInfoTask) GetSyncRange(ctx context.Context) models.SSyncRange {
	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	}
	syncRange.Normalize(ctx)
	return syncRange
}

func (self *CloudProviderSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)

	self.SetStage("OnSyncCloudProviderPreInfoComplete", nil)

	syncRange := self.GetSyncRange(ctx)

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		return nil, models.SyncCloudproviderResources(ctx, self.GetUserCred(), provider, &syncRange)
	})
}

func (self *CloudProviderSyncInfoTask) OnSyncCloudProviderPreInfoComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	syncRange := self.GetSyncRange(ctx)

	db.OpsLog.LogEvent(provider, db.ACT_SYNCING_HOST, "", self.UserCred)
	self.SetStage("OnSyncCloudProviderInfoComplete", nil)

	taskman.LocalTaskRunWithWorkers(self, func() (jsonutils.JSONObject, error) {
		provider.SyncCallSyncCloudproviderRegions(ctx, self.UserCred, syncRange)
		return nil, nil
	}, syncLocalTaskWorkerMan)
}

func (self *CloudProviderSyncInfoTask) OnSyncCloudProviderPreInfoCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	log.Errorf("faild to sync provider quotas %s", body.String())
	self.OnSyncCloudProviderPreInfoComplete(ctx, obj, body)
}

func (self *CloudProviderSyncInfoTask) OnSyncCloudProviderInfoComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, provider, getAction(self.Params), body, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *CloudProviderSyncInfoTask) OnSyncCloudProviderInfoCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_FAILED, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, provider, getAction(self.Params), body, self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}
