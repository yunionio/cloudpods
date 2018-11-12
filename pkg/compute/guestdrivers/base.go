package guestdrivers

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseGuestDriver struct {
}

func (self *SBaseGuestDriver) StartGuestCreateTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error {
	taskName, _ := data.GetString("__task__")
	if len(taskName) > 0 {
		data.Remove("__task__")
		log.Infof("Start embedded guest start task")
		switch taskName {
		case taskman.CONVERT_TASK:
			hostId, _ := data.GetString("prefer_host_id")
			if len(hostId) == 0 {
				hostId, _ = data.GetString("prefer_baremetal_id")
			}
			if len(hostId) == 0 {
				return fmt.Errorf("Not target baremetal for convert task")
			}
			host, err := models.HostManager.FetchById(hostId)
			if err != nil {
				return fmt.Errorf("Cannot find host")
			}
			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(guest.Id), "server_id")
			params.Add(data, "server_params")
			task, err := taskman.TaskManager.NewTask(ctx, "BaremetalConvertHypervisorTask", host, userCred, params, parentTaskId, "", pendingUsage)
			if err != nil {
				return err
			}
			task.ScheduleRun(nil)
		}
	} else {
		task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateTask", guest, userCred, data, parentTaskId, "", pendingUsage)
		if err != nil {
			return err
		}
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SBaseGuestDriver) OnGuestCreateTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	if jsonutils.QueryBoolean(task.GetParams(), "auto_start", false) {
		task.SetStage("on_auto_start_guest", nil)
		return guest.StartGueststartTask(ctx, task.GetUserCred(), nil, task.GetTaskId())
	} else {
		task.SetStage("on_sync_status_complete", nil)
		return guest.StartSyncstatus(ctx, task.GetUserCred(), task.GetTaskId())
	}
}

func (self *SBaseGuestDriver) StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDeleteTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaseGuestDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaseGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (self *SBaseGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaseGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaseGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{}, fmt.Errorf("This Guest driver dose not implement GetDetachDiskStatus")
}

func (self *SBaseGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{}, fmt.Errorf("This Guest driver dose not implement GetAttachDiskStatus")
}

func (self *SBaseGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{}, fmt.Errorf("This Guest driver dose not implement GetRebuildRootStatus")
}

func (self *SBaseGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{}, fmt.Errorf("This Guest driver dose not implement GetChangeConfigStatus")
}

func (self *SBaseGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{}, fmt.Errorf("This Guest driver dose not implement GetDeployStatus")
}

func (self *SBaseGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) StartGuestResetTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SBaseGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, vcpuCount, vmemSize int64) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) StartGuestDiskSnapshotTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestDeleteSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) RequestReloadDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseGuestDriver) getTaskRequestHeader(task taskman.ITask) http.Header {
	header := http.Header{}
	header.Set(mcclient.AUTH_TOKEN, task.GetUserCred().GetTokenString())
	header.Set(mcclient.TASK_ID, task.GetTaskId())
	header.Set(mcclient.REGION_VERSION, "v2")
	return header
}
