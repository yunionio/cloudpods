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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	self.SetStage("OnCloudaccountSyncReady", nil)

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		// do sync
		err := cloudaccount.SyncCallSyncAccountTask(ctx, self.UserCred)

		if err != nil {
			if errors.Cause(err) != httperrors.ErrConflict {
				log.Debugf("no other sync task, mark end sync for all cloudproviders")
				cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
			}
			return nil, errors.Wrap(err, "SyncCallSyncAccountTask")
		}
		return nil, nil
	})
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncReadyFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
	self.SetStageFailed(ctx, jsonutils.Marshal(err))
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncReady(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)

	driver, err := cloudaccount.GetProvider()
	if err != nil {
		cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
		db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
		self.SetStageFailed(ctx, jsonutils.Marshal(err))
		logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
		return
	}

	if cloudprovider.IsSupportProject(driver) {
		projects, err := driver.GetIProjects()
		if err != nil {
			msg := fmt.Sprintf("GetIProjects for cloudaccount %s failed %s", cloudaccount.GetName(), err)
			log.Errorf(msg)
		} else {
			result := models.ExternalProjectManager.SyncProjects(ctx, self.GetUserCred(), cloudaccount, projects)
			log.Infof("Sync project for cloudaccount %s result: %s", cloudaccount.GetName(), result.Result())
		}
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

	if syncRange.FullSync && cloudprovider.IsSupportDnsZone(driver) {
		dnsZones, err := driver.GetICloudDnsZones()
		if err != nil {
			log.Errorf("failed to get dns zones for account %s error: %v", cloudaccount.Name, err)
		} else {
			localZones, remoteZones, result := cloudaccount.SyncDnsZones(ctx, self.GetUserCred(), dnsZones)
			log.Infof("Sync dns zones for cloudaccount %s result: %s", cloudaccount.GetName(), result.Result())
			for i := 0; i < len(localZones); i++ {
				func() {
					lockman.LockObject(ctx, &localZones[i])
					defer lockman.ReleaseObject(ctx, &localZones[i])

					if localZones[i].Deleted {
						return
					}

					syncDnsRecordSets(ctx, self.GetUserCred(), cloudaccount, &localZones[i], remoteZones[i])
				}()
			}
		}
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

func syncDnsRecordSets(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, localDnsZone *models.SDnsZone, remoteDnsZone cloudprovider.ICloudDnsZone) {
	result := localDnsZone.SyncDnsRecordSets(ctx, userCred, account.Provider, remoteDnsZone)
	log.Infof("Sync dns records for dns zone %s result: %s", localDnsZone.GetName(), result.Result())
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, "", self.UserCred, true)
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	cloudaccount.MarkEndSyncWithLock(ctx, self.UserCred)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
	self.SetStageFailed(ctx, err)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
}
