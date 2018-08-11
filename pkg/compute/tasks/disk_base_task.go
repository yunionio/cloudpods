package tasks

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SDiskBaseTask struct {
	taskman.STask
}

func (self *SDiskBaseTask) getDisk() *models.SDisk {
	obj := self.GetObject()
	return obj.(*models.SDisk)
}

func (self *SDiskBaseTask) SetStageFailed(ctx context.Context, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *SDiskBaseTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		if !pendingUsage.IsEmpty() {
			disk := self.getDisk()
			models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &pendingUsage)
		}
	}
}

func (self *SDiskBaseTask) CleanHostSchedCache(disk *models.SDisk) {
	storage := disk.GetStorage()
	if hosts := storage.GetAllAttachingHosts(); hosts == nil {
		log.Errorf("get attaching host error")
	} else {
		for _, h := range hosts {
			if err := h.ClearSchedDescCache(); err != nil {
				log.Errorf("host CleanHostSchedCache error: %v", err)
			}
		}
	}
}
