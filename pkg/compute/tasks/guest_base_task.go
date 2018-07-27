package tasks

import (
	"context"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type SGuestBaseTask struct {
	taskman.STask
}

func (self *SGuestBaseTask) getGuest() *models.SGuest {
	obj := self.GetObject()
	return obj.(*models.SGuest)
}

func (self *SGuestBaseTask) SetStageFailed(ctx context.Context, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.STask.SetStageFailed(ctx, reason)
}

func (self *SGuestBaseTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		if !pendingUsage.IsEmpty() {
			guest := self.getGuest()
			models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, guest.ProjectId, &pendingUsage, &pendingUsage)
		}
	}
}
