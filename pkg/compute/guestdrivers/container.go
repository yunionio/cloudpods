package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var (
	containerUseKubectlError error = httperrors.NewUnsupportOperationError("Not supported, please use kubectl")
)

type SContainerDriver struct {
	SVirtualizedGuestDriver
}

func init() {
	driver := SContainerDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SContainerDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	vmemSize, vcpuCount, err := models.ValidateMemCpuData(data)
	if err != nil {
		return nil, err
	}
	if vmemSize > 0 {
		data.Add(jsonutils.NewInt(int64(vmemSize)), "vmem_size")
	}
	if vcpuCount > 0 {
		data.Add(jsonutils.NewInt(int64(vcpuCount)), "vcpu_count")
	}
	return data, nil
}

func (self *SContainerDriver) newUnsupportOperationError(option string) error {
	return httperrors.NewUnsupportOperationError("Container not support %s", option)
}

func (self *SContainerDriver) GetHypervisor() string {
	return models.HYPERVISOR_CONTAINER
}

func (self *SContainerDriver) GetDefaultSysDiskBackend() string {
	return models.STORAGE_LOCAL
}

func (self *SContainerDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SContainerDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewUnsupportOperationError("")
}

func (self *SContainerDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	// always return running
	status := jsonutils.NewDict()
	status.Add(jsonutils.NewString("running"), "status")
	return status, nil
}

func (self *SContainerDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SContainerDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	return nil, self.newUnsupportOperationError("VNC")
}

func (self *SContainerDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	//guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	// do nothing here
	return nil
}

func (self *SContainerDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, nil, false)
	return err
}

func (self *SContainerDriver) OnGuestDeployTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	guest.SetStatus(task.GetUserCred(), models.VM_RUNNING, "on deploy complete")
	task.SetStageComplete(ctx, nil)
	return nil
}

func (self *SContainerDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SContainerDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	config.Add(jsonutils.JSONTrue, "k8s_pod")
	action, err := config.GetString("action")
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/servers/%s/%s", host.ManagerUri, guest.Id, action)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	return err
}

func (self *SContainerDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// clean disk records in DB
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (self *SContainerDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return self.newUnsupportOperationError("create disk")
}

func (self *SContainerDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, vcpuCount, vmemSize int64) error {
	return self.newUnsupportOperationError("change config")
}

func (self *SContainerDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	return self.newUnsupportOperationError("rebuild root")
}

func (self *SContainerDriver) GetRandomNetworkTypes() []string {
	return []string{models.SERVER_TYPE_CONTAINER, models.SERVER_TYPE_GUEST}
}

func (self *SContainerDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}
