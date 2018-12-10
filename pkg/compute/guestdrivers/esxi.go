package guestdrivers

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
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

func (self *SESXiGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SESXiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SESXiGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SESXiGuestDriver) GetChangeConfigStatus() ([]string, error) {
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

func (self *SESXiGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SESXiGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SESXiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == models.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	/*if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_PUBLIC_CLOUD, models.STORAGE_CLOUD_SSD, models.STORAGE_CLOUD_EFFICIENCY}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}*/
	return nil
}

func (self *SESXiGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SESXiGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SESXiGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	log.Debugf("RequestDeployGuestOnHost: %s", config)

	agent, err := host.GetEsxiAgentHost()
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("No ESXi agent host")
	}

	diskCat := guest.CategorizeDisks()
	if diskCat.Root == nil {
		return fmt.Errorf("no root disk???")
	}
	storage := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("root disk has no storage???")
	}

	config.Add(jsonutils.NewString(host.AccessIp), "host_ip")
	config.Add(jsonutils.NewString(guest.Id), "guest_id")

	accessInfo, err := host.GetCloudaccount().GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}
	config.Add(jsonutils.Marshal(accessInfo), "datastore")

	url := "/disks/agent/deploy"

	body := jsonutils.NewDict()
	body.Add(config, "disk")

	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")

	_, err = agent.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SESXiGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	if data.Contains("host_ip") {
		hostIp, _ := data.GetString("host_ip")
		host, err := models.HostManager.GetHostByIp(hostIp)
		if err != nil {
			log.Errorf("fail to find host with IP %s: %s", hostIp, err)
			return err
		}
		if host.Id != guest.HostId {
			models.HostManager.ClearSchedDescCache(host.Id)
			models.HostManager.ClearSchedDescCache(guest.HostId)
			guest.OnScheduleToHost(ctx, task.GetUserCred(), host.Id)
		}
	}

	return self.SManagedVirtualizedGuestDriver.OnGuestDeployTaskDataReceived(ctx, guest, task, data)
}

func (self *SESXiGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SESXiGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ESXiGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SESXiGuestDriver) RequestRenewInstance(guest *models.SGuest, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, nil
}
