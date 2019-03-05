package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type HostStorageDetachTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostStorageDetachTask{})
}

func (self *HostStorageDetachTask) taskFail(ctx context.Context, host *models.SHost, reason string) {
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
	self.SetStageComplete(ctx, nil)
}

func (self *HostStorageDetachTask) OnDetachStorageCompleteFailed(ctx context.Context, host *models.SHost, reason jsonutils.JSONObject) {
	self.taskFail(ctx, host, reason.String())
}
