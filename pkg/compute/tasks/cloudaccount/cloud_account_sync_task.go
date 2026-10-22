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

package cloudaccount

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
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

	if cloudaccount.Provider == api.CLOUD_PROVIDER_VMWARE || cloudaccount.Provider == api.CLOUD_PROVIDER_PROXMOX {
		cloudaccount.SetStatus(ctx, self.UserCred, api.CLOUD_PROVIDER_SYNC_NETWORK, "StartSyncOnPremiseNetworkTask")
		zone, _ := self.Params.GetString("zone")
		var err error
		if cloudaccount.Provider == api.CLOUD_PROVIDER_PROXMOX {
			err = cloudaccount.PrepareProxmoxHostNetwork(ctx, self.UserCred, zone)
		} else {
			err = cloudaccount.PrepareEsxiHostNetwork(ctx, self.UserCred, zone)
		}
		if err != nil {
			d := jsonutils.NewDict()
			d.Set("error", jsonutils.NewString(err.Error()))
			db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_NETWORK_FAILED, d, self.UserCred)
			cloudaccount.SetStatus(ctx, self.UserCred, api.CLOUD_PROVIDER_SYNC_NETWORK_FAILED, "sync network failed")
			logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUDACCOUNT_SYNC_NETWORK, d, self.UserCred, false)
			cloudaccount.MarkEndSync(self.UserCred, false)
			self.SetStageFailed(ctx, d)
			return
		} else {
			cloudaccount.SetStatus(ctx, self.UserCred, api.CLOUD_PROVIDER_INIT, "sync network sucess")
			logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUDACCOUNT_SYNC_NETWORK, cloudaccount.GetShortDesc(ctx), self.UserCred, true)
		}
	}

	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNCING_HOST, "", self.UserCred)

	self.SetStage("OnCloudaccountSyncReady", nil)

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		// do sync
		err := cloudaccount.SyncCallSyncAccountTask(ctx, self.UserCred)
		if err != nil {
			if errors.Cause(err) == httperrors.ErrConflict {
				log.Errorf("account %s(%s) alread in syncing", cloudaccount.Name, cloudaccount.Provider)
			}
			// 进入同步任务前已经mark sync, 这里需要清理下状态
			cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred, false)
			return nil, errors.Wrap(err, "SyncCallSyncAccountTask")
		}
		return nil, nil
	})
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncReadyFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	if !strings.Contains(err.String(), "ConflictError") {
		db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, err)
}

func (self *CloudAccountSyncInfoTask) GetSyncRange(ctx context.Context) models.SSyncRange {
	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	} else {
		syncRange.FullSync = true
		syncRange.DeepSync = true
	}
	return syncRange
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncReady(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)

	syncRange := self.GetSyncRange(ctx)

	if len(syncRange.AccountSyncTaskId) > 0 {
		if aObj, err := db.FetchById(models.CloudaccountSyncTaskManager, syncRange.AccountSyncTaskId); err == nil {
			if aObj.(*models.SCloudaccountSyncTask).IsCanceled() {
				self.OnCloudaccountSyncComplete(ctx, obj, nil)
				return
			}
		}
	}

	if !syncRange.NeedSyncInfo() {
		self.OnCloudaccountSyncComplete(ctx, obj, nil)
		return
	}

	cloudproviders := cloudaccount.GetEnabledCloudproviders()

	if len(cloudproviders) > 0 {
		self.SetStage("OnCloudaccountSyncComplete", nil)
		for i := range cloudproviders {
			cloudproviders[i].StartSyncCloudProviderInfoTask(ctx, self.UserCred, &syncRange, self.GetId())
		}
	} else {
		self.OnCloudaccountSyncComplete(ctx, obj, nil)
	}
}

func (self *CloudAccountSyncInfoTask) finishAccountSyncTask(ctx context.Context, syncRange models.SSyncRange, fail bool, reason string) {
	if len(syncRange.AccountSyncTaskId) == 0 {
		return
	}
	obj, err := db.FetchById(models.CloudaccountSyncTaskManager, syncRange.AccountSyncTaskId)
	if err != nil {
		return
	}
	task := obj.(*models.SCloudaccountSyncTask)
	// only close account task when it has no active children (probe-only / no provider sync)
	children, _ := models.CloudaccountSyncTaskManager.FetchChildren(task.Id)
	active := 0
	for i := range children {
		if children[i].Status == api.CloudaccountSyncTaskStatusQueued || children[i].Status == api.CloudaccountSyncTaskStatusRunning {
			active++
		}
	}
	if active > 0 {
		return
	}
	status := api.CloudaccountSyncTaskStatusDone
	if fail {
		status = api.CloudaccountSyncTaskStatusFail
	}
	if task.Status == api.CloudaccountSyncTaskStatusCancel {
		return
	}
	_, _ = db.Update(task, func() error {
		task.Status = status
		now := time.Now().UTC()
		if task.StartedAt.IsZero() {
			task.StartedAt = now
		}
		task.EndedAt = now
		return nil
	})
	_ = reason
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	syncRange := self.GetSyncRange(ctx)
	cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred, syncRange.NeedSyncInfo())
	self.finishAccountSyncTask(ctx, syncRange, false, "")
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, syncRange, self.UserCred, true)
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	syncRange := self.GetSyncRange(ctx)
	cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred, syncRange.NeedSyncInfo())
	self.finishAccountSyncTask(ctx, syncRange, true, err.String())
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
	self.SetStageFailed(ctx, err)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
}
