package guestdrivers

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SESXiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SESXiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SESXiGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_ESXI
}

func (self *SESXiGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SESXiGuestDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SESXiGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
	err := disk.RealDelete(ctx, task.GetUserCred())
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}
