package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type HostStorageAttachTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostStorageAttachTask{})
}

func (self *HostStorageAttachTask) taskFail(ctx context.Context, host *models.SHost, reason string) {
	if hoststorage := self.getHoststorage(host); hoststorage != nil {
		storage := hoststorage.GetStorage()
		hoststorage.Detach(ctx, self.GetUserCred())
		note := fmt.Sprintf("attach host %s failed: %s", host.Name, reason)
		db.OpsLog.LogEvent(storage, db.ACT_ATTACH, note, self.GetUserCred())
	}
	self.SetStageFailed(ctx, reason)
}

func (self *HostStorageAttachTask) getHoststorage(host *models.SHost) *models.SHoststorage {
	storageId, _ := self.GetParams().GetString("storage_id")
	hoststorages := host.GetHoststorages()
	for _, hoststorage := range hoststorages {
		if hoststorage.StorageId == storageId {
			return &hoststorage
		}
	}
	return nil
}

func (self *HostStorageAttachTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	hoststorage := self.getHoststorage(host)
	if hoststorage == nil {
		self.taskFail(ctx, host, "failed to find hoststorage")
		return
	}
	storage := hoststorage.GetStorage()
	self.SetStage("OnAttachStorageComplete", nil)
	err := host.GetHostDriver().RequestAttachStorage(ctx, hoststorage, host, storage, self)
	if err != nil {
		self.taskFail(ctx, host, err.Error())
	}
}

func (self *HostStorageAttachTask) OnAttachStorageComplete(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *HostStorageAttachTask) OnAttachStorageCompleteFailed(ctx context.Context, host *models.SHost, reason jsonutils.JSONObject) {
	self.taskFail(ctx, host, reason.String())
}
