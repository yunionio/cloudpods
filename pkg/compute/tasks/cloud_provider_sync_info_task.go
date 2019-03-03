package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudProviderSyncInfoTask struct {
	taskman.STask
}

func init() {
	syncWorker := appsrv.NewWorkerManager("CloudProviderSyncInfoTaskWorkerManager", 2, 512, true)
	taskman.RegisterTaskAndWorker(CloudProviderSyncInfoTask{}, syncWorker)
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

func taskFail(ctx context.Context, task *CloudProviderSyncInfoTask, provider *models.SCloudprovider, reason string) {
	logclient.AddActionLogWithStartable(task, provider, getAction(task.Params), reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

func (self *CloudProviderSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)

	db.OpsLog.LogEvent(provider, db.ACT_SYNCING_HOST, "", self.UserCred)

	provider.MarkSyncing(self.UserCred)

	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	}
	syncRange.Normalize()

	provider.SyncCallSyncCloudproviderRegions(self.UserCred, &syncRange)

	self.OnSyncCloudProviderInfoComplete(ctx, provider, nil)
}

func (self *CloudProviderSyncInfoTask) OnSyncCloudProviderInfoComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)
	provider.MarkEndSync(self.UserCred)
	provider.CleanSchedCache()
	self.SetStageComplete(ctx, nil)
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, provider, getAction(self.Params), body, self.UserCred, true)
}

/*func logSyncFailed(provider *models.SCloudprovider, task taskman.ITask, reason string) {
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, reason, task.GetUserCred())
	logclient.AddActionLogWithStartable(task, provider, getAction(task.GetParams()), reason, task.GetUserCred(), false)
}

func syncCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	notes := fmt.Sprintf("Start sync host info ...")
	log.Infof(notes)
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_START, "", task.UserCred)

	if driver.GetFactory().IsOnPremise() {
		syncOnPremiseCloudProviderInfo(ctx, provider, task, driver, syncRange)
	} else {
		syncOutOfPremiseCloudProviderInfo(ctx, provider, task, driver, syncRange)
	}
}

func syncOnPremiseCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	cpr := models.CloudproviderRegionManager.FetchByIdsOrCreate(provider.Id, models.DEFAULT_REGION_ID)
	cpr.DoSync(ctx, task.UserCred, syncRange)
}

func syncOutOfPremiseCloudProviderInfo(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, syncRange *models.SSyncRange) {
	regions := driver.GetIRegions()

	externalIdPrefix := driver.GetCloudRegionExternalIdPrefix()
	localRegions, _, cloudProviderRegions, result := models.CloudregionManager.SyncRegions(ctx, task.UserCred, provider, externalIdPrefix, regions)
	msg := result.Result()
	log.Infof("SyncRegion result: %s", msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}

	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	// logclient.AddActionLog(provider, getAction(task.Params), "", task.UserCred, true)
	for i := 0; i < len(localRegions); i += 1 {
		if len(syncRange.Region) > 0 && !utils.IsInStringArray(localRegions[i].Id, syncRange.Region) {
			continue
		}

		cloudProviderRegions[i].DoSync(ctx, task.UserCred, syncRange)
	}
}

func syncVMDisks(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, driver cloudprovider.ICloudProvider, host *models.SHost, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM, syncRange *models.SSyncRange) {
	disks, err := remoteVM.GetIDisks()
	if err != nil {
		msg := fmt.Sprintf("GetIDisks for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMDisks(ctx, task.UserCred, driver, host, disks, provider.ProjectId, syncRange.ProjectSync)
	msg := result.Result()
	notes := fmt.Sprintf("syncVMDisks for VM %s result: %s", localVM.Name, msg)
	log.Infof(notes)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
	// logclient.AddActionLog(provider, getAction(task.Params), notes, task.UserCred, true)
}

func syncVMEip(ctx context.Context, provider *models.SCloudprovider, task *CloudProviderSyncInfoTask, localVM *models.SGuest, remoteVM cloudprovider.ICloudVM) {
	eip, err := remoteVM.GetIEIP()
	if err != nil {
		msg := fmt.Sprintf("GetIEIP for VM %s failed %s", remoteVM.GetName(), err)
		log.Errorf(msg)
		logSyncFailed(provider, task, msg)
		return
	}
	result := localVM.SyncVMEip(ctx, task.UserCred, provider, eip, provider.ProjectId)
	msg := result.Result()
	log.Infof("syncVMEip for VM %s result: %s", localVM.Name, msg)
	if result.IsError() {
		logSyncFailed(provider, task, msg)
		return
	}
	db.OpsLog.LogEvent(provider, db.ACT_SYNC_HOST_COMPLETE, msg, task.UserCred)
}*/
