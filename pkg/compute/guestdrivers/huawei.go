package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/utils"
)

type SHuaweiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SHuaweiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SHuaweiGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_HUAWEI
}

func (self *SHuaweiGuestDriver) GetDefaultSysDiskBackend() string {
	return models.STORAGE_HUAWEI_SATA
}

func (self *SHuaweiGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SHuaweiGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}

	for _, stype := range []string{models.STORAGE_HUAWEI_SATA, models.STORAGE_HUAWEI_SAS, models.STORAGE_HUAWEI_SSD} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SHuaweiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SHuaweiGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SHuaweiGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
}

func (self *SHuaweiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{models.VM_RUNNING, models.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_HUAWEI_SATA, models.STORAGE_HUAWEI_SAS, models.STORAGE_HUAWEI_SSD}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func (self *SHuaweiGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// todo: implement me
	return nil
}

func (self *SHuaweiGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	// todo: implement me
	return false
}
