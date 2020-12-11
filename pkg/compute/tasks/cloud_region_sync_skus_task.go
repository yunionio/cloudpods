package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudRegionSyncSkusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudRegionSyncSkusTask{})
}

func (self *CloudRegionSyncSkusTask) taskFailed(ctx context.Context, region *models.SCloudregion, msg string) {
	db.OpsLog.LogEvent(region, db.ACT_SYNC_CLOUD_SKUS, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, region, logclient.ACT_CLOUD_SYNC, msg, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(msg))
}

func (self *CloudRegionSyncSkusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	region := obj.(*models.SCloudregion)
	res, _ := self.GetParams().GetString("resource")
	meta, err := models.FetchSkuResourcesMeta()
	if err != nil {
		self.taskFailed(ctx, region, err.Error())
		return
	}

	type SyncFunc func(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, extSkuMeta *models.SSkuResourcesMeta) compare.SyncResult
	var syncFunc SyncFunc
	switch res {
	case models.ServerSkuManager.Keyword():
		syncFunc = models.SyncServerSkusByRegion
	case models.ElasticcacheSkuManager.Keyword():
		syncFunc = models.ElasticcacheSkuManager.SyncElasticcacheSkus
	case models.DBInstanceSkuManager.Keyword():
		syncFunc = models.DBInstanceSkuManager.SyncDBInstanceSkus
	}

	if syncFunc != nil {
		result := syncFunc(ctx, self.GetUserCred(), region, meta)
		log.Infof("Sync %s %s skus for region %s result: %s", region.Provider, res, region.Name, result.Result())
		if result.IsError() {
			self.taskFailed(ctx, region, result.Result())
			return
		}
	}

	self.SetStageComplete(ctx, nil)
}
