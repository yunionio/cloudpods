package guestdrivers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SKVMGuestDriver struct {
	SVirtualizedGuestDriver
}

func init() {
	driver := SKVMGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SKVMGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_KVM
}

func (self *SKVMGuestDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "GuestDetachAllDisksTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "KVMGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/snapshot", host.ManagerUri, guest.Id)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(diskId))
	body.Set("snapshot_id", jsonutils.NewString(snapshotId))
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestDeleteSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/delete-snapshot", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	return err
}

func (self *SKVMGuestDriver) RequestReloadDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/reload-disk-snapshot", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	return err
}

func findVNCPort(results string) int {
	vncInfo := strings.Split(results, "\n")
	addrParts := strings.Split(vncInfo[1], ":")
	port, _ := strconv.Atoi(addrParts[len(addrParts)-1])
	return port
}

func findVNCPort2(results string) int {
	vncInfo := strings.Split(results, "\n")
	addrParts := strings.Split(vncInfo[3], ":")
	v := addrParts[len(addrParts)-1]
	port, _ := strconv.Atoi(v[0 : len(v)-7])
	return port
}

func (self *SKVMGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	url := fmt.Sprintf("/servers/%s/monitor", guest.Id)
	body := jsonutils.NewDict()
	var cmd string
	if guest.GetVdi() == "spice" {
		cmd = "info spice"
	} else {
		cmd = "info vnc"
	}
	body.Add(jsonutils.NewString(cmd), "cmd")
	ret, err := host.Request(ctx, userCred, "POST", url, nil, body)
	if err != nil {
		err = fmt.Errorf("Fail to request VNC info %s", err)
		log.Errorf(err.Error())
		return nil, err
	}
	results, _ := ret.GetString("results")
	if len(results) == 0 {
		err = fmt.Errorf("Can't get vnc information from host.")
		return nil, err
	}
	// info_vnc = result['results'].split('\n')
	// port = int(info_vnc[1].split(':')[-1].split()[0])

	/*													QEMU 2.9.1
	info spice			QEMU 2.12.1 monitor				Server:
	Server:				(qemu) info vnc					address: 0.0.0.0:5901
	address: *:5921		info vnc						auth: none
	migrated: false		default:						Client: none
	auth: spice			Server: :::5902 (ipv6)
	compiled: 0.13.3	Auth: none (Sub: none)
	mouse-mode: server	Server: 0.0.0.0:5902 (ipv4)
	Channels: none		Auth: none (Sub: none)
	*/
	var port int
	if guest.CheckQemuVersion(guest.GetMetadata("__qemu_version", userCred), "2.12.1") && strings.HasSuffix(cmd, "vnc") {
		port = findVNCPort2(results)
	} else {
		port = findVNCPort(results)
	}

	retval := jsonutils.NewDict()
	retval.Add(jsonutils.NewString(host.AccessIp), "host")
	retval.Add(jsonutils.NewString(guest.GetVdi()), "protocol")
	retval.Add(jsonutils.NewInt(int64(port)), "port")
	return retval, nil
}

func (self *SKVMGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	body := jsonutils.NewDict()
	params := task.GetParams()
	timeout, err := params.Int("timeout")
	if err != nil {
		timeout = 30
	}
	isForce, err := params.Bool("is_force")
	if isForce {
		timeout = 0
	}
	body.Add(jsonutils.NewInt(timeout), "timeout")

	header := self.getTaskRequestHeader(task)

	url := fmt.Sprintf("%s/servers/%s/stop", host.ManagerUri, guest.Id)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	body := jsonutils.NewDict()

	// XXXXXXXX
	if guest.HostId != host.Id && guest.BackupHostId != host.Id {
		body.Set("migrated", jsonutils.JSONTrue)
	}
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, body, false)
	if err != nil {
		return err
	}
	delayClean := jsonutils.QueryBoolean(res, "delay_clean", false)
	if res != nil && delayClean {
		return nil
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SKVMGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	log.Debugf("RequestDeployGuestOnHost: %s", config)
	if config.Contains("container") {
		// ...
	}
	action, err := config.GetString("action")
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/servers/%s/%s", host.ManagerUri, guest.Id, action)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SKVMGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	header := self.getTaskRequestHeader(task)

	config := jsonutils.NewDict()
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, guest, host)
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (self *SKVMGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	header := http.Header{}
	header.Set(mcclient.AUTH_TOKEN, userCred.GetTokenString())
	header.Set(mcclient.REGION_VERSION, "v2")

	url := fmt.Sprintf("%s/servers/%s/status", host.ManagerUri, guest.Id)
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (self *SKVMGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SKVMGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, vcpuCount, vmemSize int64) error {
	// pass
	return nil
}

func (self *SKVMGuestDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	_, err := guest.SendMonitorCommand(ctx, task.GetUserCred(), "system_reset")
	return err
}

func (self *SKVMGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SKVMGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_ADMIN}, nil
}

func (self *SKVMGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY, models.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, guest, host)
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
		body.Add(jsonutils.JSONTrue, "fw_only")
	}
	url := fmt.Sprintf("%s/servers/%s/sync", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/suspend", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, nil, false)
	return err
}

func (self *SKVMGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartGuestCreateDiskTask(ctx, task.GetUserCred(), task.GetParams(), task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "KVMGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) RequestSyncToBackup(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	body := jsonutils.NewDict()
	body.Set("backup_nbd_server_uri", jsonutils.NewString(guest.GetMetadata("backup_nbd_server_uri", task.GetUserCred())))
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/drive-mirror", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}
