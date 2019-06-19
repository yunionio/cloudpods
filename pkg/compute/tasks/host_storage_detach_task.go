package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type HostStorageDetachTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostStorageDetachTask{})
}

func (self *HostStorageDetachTask) taskFail(ctx context.Context, host *models.SHost, reason string) {
	var hoststorage = new(models.SHoststorage)
	storageId, _ := self.GetParams().GetString("storage_id")
	err := models.HoststorageManager.Query().Equals("host_id", host.Id).Equals("storage_id", storageId).First(hoststorage)
	if err == nil {
		storage := hoststorage.GetStorage()
		note := fmt.Sprintf("detach host %s failed: %s", host.Name, reason)
		db.OpsLog.LogEvent(storage, db.ACT_DETACH_FAIL, note, self.GetUserCred())
		logclient.AddActionLogWithContext(ctx, storage, logclient.ACT_DETACH_HOST, note, self.GetUserCred(), false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *HostStorageDetachTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	storageId, _ := self.GetParams().GetString("storage_id")
	_storage, err := models.StorageManager.FetchById(storageId)
	if err != nil {
		self.taskFail(ctx, host, err.Error())
		return
	}
	storage := _storage.(*models.SStorage)
	self.SetStage("OnDetachStorageComplete", nil)
	err = host.GetHostDriver().RequestDetachStorage(ctx, host, storage, self)
	if err != nil {
		self.taskFail(ctx, host, err.Error())
	}
}

func (self *HostStorageDetachTask) OnDetachStorageComplete(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	storageId, _ := self.GetParams().GetString("storage_id")
	storage := models.StorageManager.FetchStorageById(storageId)
	db.OpsLog.LogEvent(storage, db.ACT_DETACH, "", self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, storage, logclient.ACT_DETACH_HOST,
		fmt.Sprintf("Detach host %s success", host.Name), self.GetUserCred(), true)
	self.SetStageComplete(ctx, nil)
	storage.SyncStatusWithHosts()
}

func (self *HostStorageDetachTask) OnDetachStorageCompleteFailed(ctx context.Context, host *models.SHost, reason jsonutils.JSONObject) {
	self.taskFail(ctx, host, reason.String())
}
