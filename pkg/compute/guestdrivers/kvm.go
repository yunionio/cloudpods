package guestdrivers

import (
	"context"

	"github.com/yunionio/mcclient"

	"github.com/yunionio/oneclone/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/oneclone/pkg/compute/models"
)

type SKVMGuestDriver struct {
	SVirtualizedGuestDriver
}

func (self *SKVMGuestDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "GuestDetachAllDisksTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) {
	// guest.DeleteAllDisksInDB(ctx, userCred)
	// do nothing
}
