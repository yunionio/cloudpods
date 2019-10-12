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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudAccountSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountSyncInfoTask{})
}

func (self *CloudAccountSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)

	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNCING_HOST, "", self.UserCred)
	// cloudaccount.MarkSyncing(self.UserCred)

	// do sync
	tenant, _ := self.GetParams().GetString("tenant")
	err := cloudaccount.SyncCallSyncAccountTask(ctx, self.UserCred, tenant)

	if err != nil {
		cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
		db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
		return
	}

	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	} else {
		syncRange.FullSync = true
		syncRange.DeepSync = true
	}

	if !syncRange.NeedSyncInfo() {
		self.OnCloudaccountSyncComplete(ctx, obj, nil)
		return
	}

	cloudproviders := cloudaccount.GetEnabledCloudproviders()

	if len(cloudproviders) > 0 {
		self.SetStage("on_cloudaccount_sync_complete", nil)
		for i := range cloudproviders {
			cloudproviders[i].StartSyncCloudProviderInfoTask(ctx, self.UserCred, &syncRange, self.GetId())
		}
	} else {
		self.OnCloudaccountSyncComplete(ctx, obj, nil)
	}
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, "", self.UserCred, true)
}
