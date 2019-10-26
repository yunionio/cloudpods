// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SKVMGuestDriver struct {
	SVirtualizedGuestDriver
}

func init() {
	driver := SKVMGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SKVMGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_KVM
}

func (self *SKVMGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SKVMGuestDriver) GetQuotaPlatformID() []string {
	return []string{
		api.CLOUD_ENV_ON_PREMISE,
		api.CLOUD_PROVIDER_ONECLOUD,
		api.HYPERVISOR_KVM,
	}
}

func (self *SKVMGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SKVMGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
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
	for i := 0; i < len(vncInfo); i++ {
		if strings.HasSuffix(vncInfo[i], "(ipv4)") {
			addrParts := strings.Split(vncInfo[i], ":")
			v := addrParts[len(addrParts)-1]
			port, _ := strconv.Atoi(v[0 : len(v)-7])
			return port
		}
	}
	return -1
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
		log.Errorln(err)
		return nil, err
	}
	results, _ := ret.GetString("results")
	if len(results) == 0 {
		err = fmt.Errorf("Can't get vnc information from host.")
		return nil, err
	}
	// info_vnc = result['results'].split('\n')
	// port = int(info_vnc[1].split(':')[-1].split()[0])

	/*													$ QEMU 2.9.1
	info spice			$ QEMU 2.12.1 monitor			Server:
	Server:				(qemu) info vnc					address: 0.0.0.0:5901
	address: *:5921		info vnc						auth: none
	migrated: false		default:						Client: none
	auth: spice			Server: :::5902 (ipv6)			$ QEMU 2.12.1 monitor without ipv6
	compiled: 0.13.3	Auth: none (Sub: none)			(qemu) info vnc
	mouse-mode: server	Server: 0.0.0.0:5902 (ipv4)		info vnc
	Channels: none		Auth: none (Sub: none)			default:
														Server: 0.0.0.0:5902 (ipv4)
														Auth: none (Sub: none)
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

	if guest.HostId != host.Id {
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

func (self *SKVMGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SKVMGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	log.Debugf("RequestDeployGuestOnHost: %s", config)
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
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, userCred, guest, host)
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

func (self *SKVMGuestDriver) NeedStopForChangeSpec(guest *models.SGuest) bool {
	return guest.GetMetadata("hotplug_cpu_mem", nil) != "enable"
}

func (self *SKVMGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error {
	if jsonutils.QueryBoolean(task.GetParams(), "guest_online", false) {
		addCpu := vcpuCount - int64(guest.VcpuCount)
		addMem := vmemSize - int64(guest.VmemSize)
		if addCpu < 0 || addMem < 0 {
			return fmt.Errorf("KVM guest doesn't support online reduce cpu or mem")
		}
		header := task.GetTaskRequestHeader()
		body := jsonutils.NewDict()
		if vcpuCount > int64(guest.VcpuCount) {
			body.Set("add_cpu", jsonutils.NewInt(addCpu))
		}
		if vmemSize > int64(guest.VmemSize) {
			body.Set("add_mem", jsonutils.NewInt(addMem))
		}
		host := guest.GetHost()
		url := fmt.Sprintf("%s/servers/%s/hotplug-cpu-mem", host.ManagerUri, guest.Id)
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SKVMGuestDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	_, err := guest.SendMonitorCommand(ctx, task.GetUserCred(), "system_reset")
	return err
}

func (self *SKVMGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	host := guest.GetHost()
	header := task.GetTaskRequestHeader()
	url := fmt.Sprintf("%s/servers/%s/status", host.ManagerUri, guest.Id)
	_, res, _ := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
	if res != nil {
		status, _ := res.GetString("status")
		if status == "notfound" {
			taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
				return nil, nil
			})
			return nil
		}

	}
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SKVMGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host)
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
	input := new(api.ServerCreateInput)
	task.GetParams().Unmarshal(input)
	return guest.StartGuestCreateDiskTask(ctx, task.GetUserCred(), input.Disks, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotRemoveIso(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
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
	host := guest.GetHost()
	desc := guest.GetDriver().GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host)
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	body.Set("backup_nbd_server_uri", jsonutils.NewString(guest.GetMetadata("backup_nbd_server_uri", task.GetUserCred())))
	url := fmt.Sprintf("%s/servers/%s/drive-mirror", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

// kvm guest must add cpu first
// if body has add_cpu_failed indicate dosen't exec add mem
// 1. cpu added part of request --> add_cpu_failed: true && added_cpu: count
// 2. cpu added all of request add mem failed --> add_mem_failed: true
func (self *SKVMGuestDriver) OnGuestChangeCpuMemFailed(ctx context.Context, guest *models.SGuest, data *jsonutils.JSONDict, task taskman.ITask) error {
	var cpuAdded int64
	if jsonutils.QueryBoolean(data, "add_cpu_failed", false) {
		cpuAdded, _ = data.Int("added_cpu")
	} else if jsonutils.QueryBoolean(data, "add_mem_failed", false) {
		vcpuCount, _ := task.GetParams().Int("vcpu_count")
		if vcpuCount-int64(guest.VcpuCount) > 0 {
			cpuAdded = vcpuCount - int64(guest.VcpuCount)
		}
	}
	if cpuAdded > 0 {
		_, err := db.Update(guest, func() error {
			guest.VcpuCount = guest.VcpuCount + int(cpuAdded)
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_CHANGE_FLAVOR,
			fmt.Sprintf("Change config task failed but added cpu count %d", cpuAdded), task.GetUserCred())
		logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_CHANGE_FLAVOR,
			fmt.Sprintf("Change config task failed but added cpu count %d", cpuAdded), task.GetUserCred(), false)

		models.HostManager.ClearSchedDescCache(guest.HostId)
	}
	return nil
}

func (self *SKVMGuestDriver) CancelExpireTime(
	ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	return guest.CancelExpireTime(ctx, userCred)
}

func (self *SKVMGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}
