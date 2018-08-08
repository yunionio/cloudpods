package guestdrivers

import (
	"context"

	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/onecloud/pkg/mcclient"
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
