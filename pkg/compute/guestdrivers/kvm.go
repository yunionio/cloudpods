package guestdrivers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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

func (self *SKVMGuestDriver) StartGuestDiskSnapshotTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDiskSnapshotTask", guest, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMGuestDriver) RequestDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, snapshotId, diskId string) error {
	url := fmt.Sprintf("/servers/%s/snapshot", guest.Id)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(diskId))
	body.Set("snapshot_id", jsonutils.NewString(snapshotId))
	header := http.Header{}
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	host := guest.GetHost()
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMGuestDriver) RequestDeleteSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	url := fmt.Sprintf("/servers/%s/delete-snapshot", guest.Id)
	header := http.Header{}
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	host := guest.GetHost()
	_, err := host.Request(task.GetUserCred(), "POST", url, header, params)
	return err
}

func (self *SKVMGuestDriver) RequestReloadDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	url := fmt.Sprintf("/servers/%s/reload-disk-snapshot", guest.Id)
	header := http.Header{}
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	host := guest.GetHost()
	_, err := host.Request(task.GetUserCred(), "POST", url, header, params)
	return err
}

func findVNCPort(results string) int {
	reg := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+):([\d]+)`)
	finds := reg.FindStringSubmatch(results)
	log.Debugf("finds=%s", finds)
	port, _ := strconv.Atoi(finds[2])
	return port
}

func (self *SKVMGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	url := fmt.Sprintf("/servers/%s/monitor", guest.Id)
	body := jsonutils.NewDict()
	var cmd string
	if guest.GetVdi() == "spice" {
		cmd = "info spice"
	} else {
		cmd = "info vnc"
	}
	body.Add(jsonutils.NewString(cmd), "cmd")
	ret, err := host.Request(userCred, "POST", url, nil, body)
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

	/***
	Server:
	address: 0.0.0.0:5901
	auth: none
	Client: none
		 ***/

	// info_vnc = result['results'].split('\n')
	// port = int(info_vnc[1].split(':')[-1].split()[0])

	port := findVNCPort(results)
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

	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")

	url := fmt.Sprintf("%s/servers/%s/stop", host.ManagerUri, guest.Id)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, nil, false)
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
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
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
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")

	config := jsonutils.NewDict()
	desc := self.GetJsonDescAtHost(ctx, guest, host)
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
	header.Set("X-Auth-Token", userCred.GetTokenString())
	header.Set("X-Region-Version", "v2")

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

func (self *SKVMGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
	return disk.StartDiskDeleteTask(ctx, task.GetUserCred(), task.GetTaskId(), isPurge)
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, guest, host)
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
		body.Add(jsonutils.JSONTrue, "fw_only")
	}
	url := fmt.Sprintf("/servers/%s/sync", guest.Id)
	header := http.Header{}
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, err := host.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SKVMGuestDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/suspend", host.ManagerUri, guest.Id)
	header := http.Header{}
	header.Add("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Add("X-Task-Id", task.GetTaskId())
	header.Add("X-Region-Version", "v2")
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, nil, false)
	return err
}

func (self *SKVMGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartGuestCreateDiskTask(ctx, task.GetUserCred(), task.GetParams(), task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	return guest.StartSyncstatus(ctx, task.GetUserCred(), task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "KVMGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}
