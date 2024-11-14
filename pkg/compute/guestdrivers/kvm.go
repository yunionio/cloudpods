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
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	host_api "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	guestdriver_types "yunion.io/x/onecloud/pkg/compute/guestdrivers/types"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SKVMGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_ON_PREMISE
	keys.Provider = api.CLOUD_PROVIDER_ONECLOUD
	keys.Brand = api.ONECLOUD_BRAND_ONECLOUD
	keys.Hypervisor = api.HYPERVISOR_KVM
	return keys
}

func (self *SKVMGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     true,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
		Storages: cloudprovider.Storage{
			SysDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_RBD, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_GPFS, MinSizeGb: options.Options.LocalSysDiskMinSizeGB, MaxSizeGb: options.Options.LocalSysDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
			DataDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_LOCAL, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_RBD, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_NFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_GPFS, MinSizeGb: options.Options.LocalDataDiskMinSizeGB, MaxSizeGb: options.Options.LocalDataDiskMaxSizeGB, StepSizeGb: 1, Resizable: true},
			},
		},
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
	obj, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		return errors.Wrapf(err, "failed to find snapshot %s", snapshotId)
	}
	snapshot := obj.(*models.SSnapshot)

	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/snapshot", host.ManagerUri, guest.Id)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(diskId))
	body.Set("snapshot_id", jsonutils.NewString(snapshotId))

	if snapshot.DiskBackupId != "" {
		backupObj, err := models.DiskBackupManager.FetchById(snapshot.DiskBackupId)
		if err != nil {
			return errors.Wrapf(err, "failed to find backup %s", snapshot.DiskBackupId)
		}
		backup := backupObj.(*models.SDiskBackup)
		body.Set("backup_disk_config", jsonutils.Marshal(backup.DiskConfig))
	}

	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestDeleteSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/delete-snapshot", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	return err
}

func (self *SKVMGuestDriver) RequestReloadDiskSnapshot(ctx context.Context, guest *models.SGuest, task taskman.ITask, params *jsonutils.JSONDict) error {
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/reload-disk-snapshot", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	return err
}

func findVNCPort(results string) int {
	vncInfo := strings.Split(results, "\n")
	addrParts := strings.Split(vncInfo[1], ":")
	port, _ := strconv.Atoi(strings.TrimSpace(addrParts[len(addrParts)-1]))
	return port
}

func findVNCPort2(results string) int {
	vncInfo := strings.Split(results, "\n")
	for i := 0; i < len(vncInfo); i++ {
		lineStr := strings.TrimSpace(vncInfo[i])
		if strings.HasSuffix(lineStr, "(ipv4)") {
			addrParts := strings.Split(lineStr, ":")
			v := addrParts[len(addrParts)-1]
			port, _ := strconv.Atoi(v[0 : len(v)-7])
			return port
		}
	}
	return -1
}

func (self *SKVMGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
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
		return nil, errors.Wrapf(err, "Fail to request VNC info")
	}
	results, err := ret.GetString("results")
	if len(results) == 0 {
		return nil, errors.Wrapf(err, "Can't get vnc information from host.")
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

	if guest.CheckQemuVersion(guest.GetMetadata(ctx, "__qemu_version", userCred), "2.12.1") && strings.HasSuffix(cmd, "vnc") {
		port = findVNCPort2(results)
	} else {
		port = findVNCPort(results)
	}
	if port < 5900 {
		return nil, httperrors.NewResourceNotReadyError("invalid vnc port %d", port)
	}

	if len(host.AccessIp) == 0 {
		return nil, httperrors.NewResourceNotReadyError("the host %s loses its ip address", host.Name)
	}

	password := guest.GetMetadata(ctx, "__vnc_password", userCred)

	result := &cloudprovider.ServerVncOutput{
		Host:       host.AccessIp,
		Protocol:   guest.GetVdi(),
		Port:       int64(port),
		Hypervisor: api.HYPERVISOR_KVM,
		Password:   password,
	}
	return result, nil
}

func (self *SKVMGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask, syncStatus bool) error {
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

func (self *SKVMGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	if len(desc.UserData) > 0 {
		// host 需要加密后的user-data以提供 http://169.254.169.254/latest/user-data 解密访问
		desc.UserData = base64.StdEncoding.EncodeToString([]byte(desc.UserData))
	}
	return jsonutils.Marshal(desc), nil
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

func (self *SKVMGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	header := self.getTaskRequestHeader(task)

	config := jsonutils.NewDict()
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	desc, err := drv.GetJsonDescAtHost(ctx, userCred, guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, body, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	if jsonutils.QueryBoolean(body, "is_running", false) {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return body, nil
		})
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	header := self.getTaskRequestHeader(task)

	url := fmt.Sprintf("%s/servers/%s/status", host.ManagerUri, guest.Id)
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
	if err != nil {
		return err
	}
	statusStr, _ := res.GetString("status")
	if len(statusStr) > 0 {
		// may be an old version host, use sync request
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			// delay response to ensure event order
			time.Sleep(time.Second)
			return res, nil
		})
	}
	return nil
}

func (self *SKVMGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	if ispId := guest.GetMetadata(ctx, api.BASE_INSTANCE_SNAPSHOT_ID, userCred); len(ispId) > 0 {
		ispM, err := models.InstanceSnapshotManager.FetchById(ispId)
		if err == nil {
			isp := ispM.(*models.SInstanceSnapshot)
			isp.DecRefCount(ctx, userCred)
		}
		guest.SetMetadata(ctx, api.BASE_INSTANCE_SNAPSHOT_ID, "", userCred)
	}
	return nil
}

func (self *SKVMGuestDriver) IsSupportEip() bool {
	return true
}

func (self *SKVMGuestDriver) IsSupportShutdownMode() bool {
	return true
}

func (self *SKVMGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return nil
}

func (self *SKVMGuestDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, eip *models.SElasticip, task taskman.ITask) error {
	defer task.ScheduleRun(nil)

	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	var guestnics []models.SGuestnetwork
	{
		netq := models.NetworkManager.Query().SubQuery()
		wirq := models.WireManager.Query().SubQuery()
		vpcq := models.VpcManager.Query().SubQuery()
		gneq := models.GuestnetworkManager.Query()
		q := gneq.Equals("guest_id", guest.Id).
			IsNullOrEmpty("eip_id")
		q = q.Join(netq, sqlchemy.Equals(netq.Field("id"), gneq.Field("network_id")))
		q = q.Join(wirq, sqlchemy.Equals(wirq.Field("id"), netq.Field("wire_id")))
		q = q.Join(vpcq, sqlchemy.Equals(vpcq.Field("id"), wirq.Field("vpc_id")))
		q = q.Filter(sqlchemy.NotEquals(vpcq.Field("id"), api.DEFAULT_VPC_ID))
		if err := db.FetchModelObjects(models.GuestnetworkManager, q, &guestnics); err != nil {
			return err
		}
		if len(guestnics) == 0 {
			return errors.Errorf("guest has no nics to associate eip")
		}
	}

	guestnic := &guestnics[0]
	lockman.LockObject(ctx, guestnic)
	defer lockman.ReleaseObject(ctx, guestnic)
	if _, err := db.Update(guestnic, func() error {
		guestnic.EipId = eip.Id
		return nil
	}); err != nil {
		return errors.Wrapf(err, "set associated eip for guestnic %s (guest:%s, network:%s)",
			guestnic.Ifname, guestnic.GuestId, guestnic.NetworkId)
	}

	if err := eip.AssociateInstance(ctx, userCred, api.EIP_ASSOCIATE_TYPE_SERVER, guest); err != nil {
		return errors.Wrapf(err, "associate eip %s(%s) to vm %s(%s)", eip.Name, eip.Id, guest.Name, guest.Id)
	}
	if err := eip.SetStatus(ctx, userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE); err != nil {
		return errors.Wrapf(err, "set eip status to %s", api.EIP_STATUS_ALLOCATE)
	}
	return nil
}

func (self *SKVMGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, cpuSockets, vmemSize int64) error {
	taskParams := task.GetParams()
	if jsonutils.QueryBoolean(taskParams, "guest_online", false) {
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
		if taskParams.Contains("cpu_numa_pin") {
			cpuNumaPin, _ := taskParams.Get("cpu_numa_pin")
			body.Set("cpu_numa_pin", cpuNumaPin)
		}
		host, _ := guest.GetHost()
		url := fmt.Sprintf("%s/servers/%s/hotplug-cpu-mem", host.ManagerUri, guest.Id)
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SKVMGuestDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	_, err := guest.SendMonitorCommand(
		ctx, task.GetUserCred(),
		&api.ServerMonitorInput{COMMAND: "system_reset"},
	)
	return err
}

func (self *SKVMGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	host, _ := guest.GetHost()
	header := task.GetTaskRequestHeader()
	url := fmt.Sprintf("%s/servers/%s/status", host.ManagerUri, guest.Id)
	task.SetStage("OnGetGuestStatus", nil)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "GET", url, header, nil, false)
	return err
}

func (self *SKVMGuestDriver) RequestAttachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	return guest.StartSyncTaskWithoutSyncstatus(
		ctx,
		task.GetUserCred(),
		jsonutils.QueryBoolean(task.GetParams(), "sync_desc_only", false),
		task.GetTaskId(),
	)
}

func (self *SKVMGuestDriver) RequestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, task taskman.ITask) error {
	disks := guest.CategorizeDisks()
	opts := api.DiskSaveInput{}
	task.GetParams().Unmarshal(&opts)
	return disks.Root.StartDiskSaveTask(ctx, userCred, opts, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestOpenForward(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, req *guestdriver_types.OpenForwardRequest) (*guestdriver_types.OpenForwardResponse, error) {
	var (
		host, _    = guest.GetHost()
		url        = fmt.Sprintf("%s/servers/%s/open-forward", host.ManagerUri, guest.Id)
		httpClient = httputils.GetDefaultClient()
		header     = mcclient.GetTokenHeaders(userCred)
		hostreq    = &host_api.GuestOpenForwardRequest{
			NetworkId: req.NetworkId,
			Proto:     req.Proto,
			Addr:      req.Addr,
			Port:      req.Port,
		}
		body = jsonutils.Marshal(hostreq)
	)
	_, respBody, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	hostresp := &host_api.GuestOpenForwardResponse{}
	if err := respBody.Unmarshal(hostresp); err != nil {
		return nil, errors.Wrap(err, "unmarshal host response")
	}
	resp := &guestdriver_types.OpenForwardResponse{
		Proto:     hostresp.Proto,
		ProxyAddr: hostresp.ProxyAddr,
		ProxyPort: hostresp.ProxyPort,
		Addr:      hostresp.Addr,
		Port:      hostresp.Port,
	}
	return resp, nil
}

func (self *SKVMGuestDriver) RequestCloseForward(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, req *guestdriver_types.CloseForwardRequest) (*guestdriver_types.CloseForwardResponse, error) {
	var (
		host, _    = guest.GetHost()
		url        = fmt.Sprintf("%s/servers/%s/close-forward", host.ManagerUri, guest.Id)
		httpClient = httputils.GetDefaultClient()
		header     = mcclient.GetTokenHeaders(userCred)
		hostreq    = &host_api.GuestCloseForwardRequest{
			NetworkId: req.NetworkId,
			Proto:     req.Proto,
			ProxyAddr: req.ProxyAddr,
			ProxyPort: req.ProxyPort,
		}
		body = jsonutils.Marshal(hostreq)
	)
	_, respBody, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	hostresp := &host_api.GuestCloseForwardResponse{}
	if err := respBody.Unmarshal(hostresp); err != nil {
		return nil, errors.Wrap(err, "unmarshal host response")
	}
	resp := &guestdriver_types.CloseForwardResponse{
		Proto:     hostresp.Proto,
		ProxyAddr: hostresp.ProxyAddr,
		ProxyPort: hostresp.ProxyPort,
	}
	return resp, nil
}

func (self *SKVMGuestDriver) RequestListForward(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, req *guestdriver_types.ListForwardRequest) (*guestdriver_types.ListForwardResponse, error) {
	var (
		host, _    = guest.GetHost()
		url        = fmt.Sprintf("%s/servers/%s/list-forward", host.ManagerUri, guest.Id)
		httpClient = httputils.GetDefaultClient()
		header     = mcclient.GetTokenHeaders(userCred)
		hostreq    = &host_api.GuestListForwardRequest{
			NetworkId: req.NetworkId,
			Proto:     req.Proto,
			Addr:      req.Addr,
			Port:      req.Port,
		}
		body = jsonutils.Marshal(hostreq)
	)
	_, respBody, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	hostresp := &host_api.GuestListForwardResponse{}
	if err := respBody.Unmarshal(hostresp); err != nil {
		return nil, errors.Wrap(err, "unmarshal host response")
	}
	var respForwards []guestdriver_types.OpenForwardResponse
	for i := range hostresp.Forwards {
		respForwards = append(respForwards, guestdriver_types.OpenForwardResponse{
			Proto:     hostresp.Forwards[i].Proto,
			ProxyAddr: hostresp.Forwards[i].ProxyAddr,
			ProxyPort: hostresp.Forwards[i].ProxyPort,
			Addr:      hostresp.Forwards[i].Addr,
			Port:      hostresp.Forwards[i].Port,
		})
	}
	resp := &guestdriver_types.ListForwardResponse{
		Forwards: respForwards,
	}
	return resp, nil
}

func (self *SKVMGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SKVMGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SKVMGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if guest.Hypervisor == api.HYPERVISOR_KVM {
		if guest.GetDiskIndex(disk.Id) <= 0 && guest.Status == api.VM_RUNNING {
			return fmt.Errorf("Cann't online resize root disk")
		}
		if guest.Status == api.VM_RUNNING && storage.StorageType == api.STORAGE_SLVM {
			return fmt.Errorf("shared lvm storage cann't online resize")
		}
	}

	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	desc, err := drv.GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
		body.Add(jsonutils.JSONTrue, "fw_only")
	}
	url := fmt.Sprintf("%s/servers/%s/sync", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/suspend", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, nil, false)
	return err
}

func (self *SKVMGuestDriver) RequestResumeOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, nil, false)
	return err
}

func (self *SKVMGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	input := new(api.ServerCreateInput)
	task.GetParams().Unmarshal(input)
	return guest.StartGuestCreateDiskTask(ctx, task.GetUserCred(), input.Disks, task.GetTaskId())
}

func (self *SKVMGuestDriver) NeedRequestGuestHotAddIso(ctx context.Context, guest *models.SGuest) bool {
	return guest.Status == api.VM_RUNNING
}

func (self *SKVMGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotRemoveIso(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) NeedRequestGuestHotAddVfd(ctx context.Context, guest *models.SGuest) bool {
	return guest.Status == api.VM_RUNNING
}

func (self *SKVMGuestDriver) RequestGuestHotAddVfd(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestGuestHotRemoveVfd(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
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
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	desc, err := drv.GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	body.Set("backup_nbd_server_uri", jsonutils.NewString(guest.GetMetadata(ctx, "backup_nbd_server_uri", task.GetUserCred())))
	url := fmt.Sprintf("%s/servers/%s/block-replication", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSlaveBlockStreamDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	body := jsonutils.NewDict()
	url := fmt.Sprintf("%s/servers/%s/slave-block-stream-disks", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
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

func (self *SKVMGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}

func (self *SKVMGuestDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return true, nil
}

func (self *SKVMGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SKVMGuestDriver) IsSupportLiveMigrate() bool {
	return true
}

func checkAssignHost(ctx context.Context, userCred mcclient.TokenCredential, preferHost string) error {
	iHost, _ := models.HostManager.FetchByIdOrName(ctx, userCred, preferHost)
	if iHost == nil {
		return httperrors.NewBadRequestError("Host %s not found", preferHost)
	}
	host := iHost.(*models.SHost)
	err := host.IsAssignable(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "IsAssignable")
	}
	return nil
}

func (self *SKVMGuestDriver) CheckMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput) error {
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if !input.IsRescueMode && guest.Status != api.VM_READY {
		return httperrors.NewServerStatusError("Cannot normal migrate guest in status %s, try rescue mode or server-live-migrate?", guest.Status)
	}
	if input.IsRescueMode {
		host, err := guest.GetHost()
		if err != nil {
			return err
		}
		if host.HostStatus != api.HOST_OFFLINE {
			return httperrors.NewBadRequestError("Host status %s, can't do rescue mode migration", host.HostStatus)
		}
		disks, err := guest.GetDisks()
		if err != nil {
			return errors.Wrapf(err, "GetDisks")
		}
		for _, disk := range disks {
			storage, _ := disk.GetStorage()
			if utils.IsInStringArray(
				storage.StorageType, api.STORAGE_LOCAL_TYPES) {
				return httperrors.NewBadRequestError("Rescue mode requires all disk store in shared storages")
			}
		}
	}
	devices, err := guest.GetIsolatedDevices()
	if err != nil {
		return errors.Wrapf(err, "GetIsolatedDevices")
	}
	if len(devices) > 0 {
		return httperrors.NewBadRequestError("Cannot migrate with isolated devices")
	}
	if len(input.PreferHostId) > 0 {
		err := checkAssignHost(ctx, userCred, input.PreferHostId)
		if err != nil {
			return errors.Wrap(err, "checkAssignHost")
		}
	}
	return nil
}

func (self *SKVMGuestDriver) CheckLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_SUSPEND}) {
		if input.MaxBandwidthMb != nil && *input.MaxBandwidthMb < 50 {
			return httperrors.NewBadRequestError("max bandwidth must gratethan 100M")
		}
		cdrom := guest.GetCdrom()
		if cdrom != nil && len(cdrom.ImageId) > 0 {
			return httperrors.NewBadRequestError("Cannot live migrate with cdrom")
		}
		devices, err := guest.GetIsolatedDevices()
		if err != nil {
			return errors.Wrapf(err, "GetIsolatedDevices")
		}
		if len(devices) > 0 {
			return httperrors.NewBadRequestError("Cannot live migrate with isolated devices")
		}
		if !guest.CheckQemuVersion(guest.GetQemuVersion(userCred), "1.1.2") {
			return httperrors.NewBadRequestError("Cannot do live migrate, too low qemu version")
		}
		if len(input.PreferHost) > 0 {
			err := checkAssignHost(ctx, userCred, input.PreferHost)
			if err != nil {
				return errors.Wrap(err, "checkAssignHost")
			}
		}
	}
	return nil
}

func (self *SKVMGuestDriver) RequestCancelLiveMigrate(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/cancel-live-migrate", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, jsonutils.NewDict(), false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) ValidateDetachNetwork(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	if guest.Status == api.VM_RUNNING && guest.GetMetadata(ctx, api.VM_METADATA_HOT_REMOVE_NIC, nil) != "enable" {
		return httperrors.NewBadRequestError("Guest %s can't hot remove nic", guest.GetName())
	}
	return nil
}

func (self *SKVMGuestDriver) ValidateChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, targetStorageId string) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_BLOCK_STREAM, api.VM_DISK_CHANGE_STORAGE}) {
		return httperrors.NewBadRequestError("Cannot change disk storage in status %s", guest.Status)
	}

	// backup guest not supported
	if guest.BackupHostId != "" {
		return httperrors.NewBadRequestError("Cannot change disk storage in backup guest %s", guest.GetName())
	}

	// storage must attached on guest's host
	host, err := guest.GetHost()
	if err != nil {
		return errors.Wrapf(err, "Get guest %s host", guest.GetName())
	}
	attachedStorages := host.GetAttachedEnabledHostStorages(nil)
	foundStorage := false
	for _, storage := range attachedStorages {
		if storage.GetId() == targetStorageId {
			foundStorage = true
		}
	}
	if !foundStorage {
		return httperrors.NewBadRequestError("Storage %s not attached or enabled on host %s", targetStorageId, host.GetName())
	}
	return nil
}

func (self *SKVMGuestDriver) RequestChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, input *api.ServerChangeDiskStorageInternalInput, task taskman.ITask) error {
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	body := jsonutils.Marshal(input)
	header := self.getTaskRequestHeader(task)
	url := fmt.Sprintf("%s/servers/%s/storage-clone-disk", host.ManagerUri, guest.GetId())
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) RequestSwitchToTargetStorageDisk(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, input *api.ServerChangeDiskStorageInternalInput, task taskman.ITask) error {
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	body := jsonutils.Marshal(input)
	header := self.getTaskRequestHeader(task)
	url := fmt.Sprintf("%s/servers/%s/live-change-disk", host.ManagerUri, guest.GetId())
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	return err
}

func (self *SKVMGuestDriver) validateVdiProtocol(vdi string) error {
	if !utils.IsInStringArray(vdi, []string{api.VM_VDI_PROTOCOL_VNC, api.VM_VDI_PROTOCOL_SPICE}) {
		return httperrors.NewInputParameterError("unsupported vdi protocol %s", vdi)
	}
	return nil
}

func (self *SKVMGuestDriver) validateVGA(ovdi, ovga string, nvdi, nvga *string) (vdi, vga string) {
	vdi = ovdi
	if nvdi != nil {
		vdi = *nvdi
	}
	if vdi != api.VM_VDI_PROTOCOL_VNC && vdi != api.VM_VDI_PROTOCOL_SPICE {
		vdi = api.VM_VDI_PROTOCOL_VNC
	}
	var candidateVga []string
	switch vdi {
	case api.VM_VDI_PROTOCOL_VNC:
		candidateVga = []string{api.VM_VIDEO_STANDARD, api.VM_VIDEO_QXL, api.VM_VIDEO_VIRTIO}
	case api.VM_VDI_PROTOCOL_SPICE:
		candidateVga = []string{api.VM_VIDEO_QXL, api.VM_VIDEO_VIRTIO}
	}
	vga = ovga
	if nvga != nil {
		vga = *nvga
	}
	if !utils.IsInStringArray(vga, candidateVga) {
		vga = candidateVga[0]
	}
	return
}

func (self *SKVMGuestDriver) validateMachineType(machine string, osArch string) error {
	var candidate []string
	if apis.IsARM(osArch) {
		candidate = []string{api.VM_MACHINE_TYPE_ARM_VIRT}
	} else {
		candidate = []string{api.VM_MACHINE_TYPE_PC, api.VM_MACHINE_TYPE_Q35}
	}
	if !utils.IsInStringArray(machine, candidate) {
		return httperrors.NewInputParameterError("Invalid machine type %q for arch %q", machine, osArch)
	}
	return nil
}

func (self *SKVMGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualizedGuestDriver.ValidateCreateData")
	}
	if input.Vdi != "" {
		err = self.validateVdiProtocol(input.Vdi)
		if err != nil {
			return nil, errors.Wrap(err, "validateVdiProtocol")
		}
	}

	if input.Vdi != "" || input.Vga != "" {
		input.Vdi, input.Vga = self.validateVGA("", "", &input.Vdi, &input.Vga)
	}

	if input.Machine != "" {
		if err := self.validateMachineType(input.Machine, input.OsArch); err != nil {
			return nil, errors.Wrap(err, "validateMachineType")
		}
	}

	for i := range input.Secgroups {
		if input.Secgroups[i] == api.SECGROUP_DEFAULT_ID {
			continue
		}
		secObj, err := validators.ValidateModel(ctx, userCred, models.SecurityGroupManager, &input.Secgroups[i])
		if err != nil {
			return nil, err
		}
		secgroup := secObj.(*models.SSecurityGroup)
		if secgroup.CloudregionId != api.DEFAULT_REGION_ID {
			return nil, httperrors.NewInputParameterError("invalid secgroup %s", secgroup.Name)
		}
	}

	return input, nil
}

func (self *SKVMGuestDriver) ValidateUpdateData(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential, input api.ServerUpdateInput) (api.ServerUpdateInput, error) {
	input, err := self.SVirtualizedGuestDriver.ValidateUpdateData(ctx, guest, userCred, input)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualizedGuestDriver.ValidateUpdateData")
	}

	if input.Vdi != nil {
		err = self.validateVdiProtocol(*input.Vdi)
		if err != nil {
			return input, errors.Wrap(err, "validateVdiProtocol")
		}
	}

	if input.Vga != nil || input.Vdi != nil {
		vdi, vga := self.validateVGA(guest.Vdi, guest.Vga, input.Vdi, input.Vga)
		input.Vdi = &vdi
		input.Vga = &vga
	}

	if input.Machine != nil {
		err := self.validateMachineType(*input.Machine, guest.OsArch)
		if err != nil {
			return input, errors.Wrap(err, "ValidateMachineType")
		}
	}

	return input, nil
}

func (self *SKVMGuestDriver) RequestSyncIsolatedDevice(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SKVMGuestDriver) RequestCPUSet(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, guest *models.SGuest, input *api.ServerCPUSetInput) (*api.ServerCPUSetResp, error) {
	url := fmt.Sprintf("%s/servers/%s/cpuset", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	body := jsonutils.Marshal(input)
	_, respBody, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	resp := new(api.ServerCPUSetResp)
	if respBody == nil {
		return resp, nil
	}
	if err := respBody.Unmarshal(resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal response")
	}
	return resp, nil
}

func (self *SKVMGuestDriver) RequestCPUSetRemove(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, guest *models.SGuest, input *api.ServerCPUSetRemoveInput) error {
	url := fmt.Sprintf("%s/servers/%s/cpuset-remove", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	body := jsonutils.Marshal(input)
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) QgaRequestGuestPing(ctx context.Context, header http.Header, host *models.SHost, guest *models.SGuest, async bool, input *api.ServerQgaTimeoutInput) error {
	url := fmt.Sprintf("%s/servers/%s/qga-guest-ping", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	body := jsonutils.NewDict()
	if input != nil {
		body.Set("timeout", jsonutils.NewInt(int64(input.Timeout)))
	}
	body.Set("async", jsonutils.NewBool(async))
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) QgaRequestGuestInfoTask(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/servers/%s/qga-guest-info-task", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, res, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	return res, nil
}

func (self *SKVMGuestDriver) QgaRequestSetNetwork(ctx context.Context, task taskman.ITask, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/servers/%s/qga-set-network", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := task.GetTaskRequestHeader()
	_, res, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	return res, nil
}

func (self *SKVMGuestDriver) QgaRequestGetNetwork(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/servers/%s/qga-get-network", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, res, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	return res, nil
}

func (self *SKVMGuestDriver) QgaRequestGetOsInfo(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/servers/%s/qga-get-os-info", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, res, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	return res, nil
}

func (self *SKVMGuestDriver) QgaRequestSetUserPassword(ctx context.Context, task taskman.ITask, host *models.SHost, guest *models.SGuest, input *api.ServerQgaSetPasswordInput) error {
	url := fmt.Sprintf("%s/servers/%s/qga-set-password", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := task.GetTaskRequestHeader()
	body := jsonutils.Marshal(input)
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) RequestQgaCommand(ctx context.Context, userCred mcclient.TokenCredential, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/servers/%s/qga-command", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := mcclient.GetTokenHeaders(userCred)
	_, res, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, errors.Wrap(err, "host request")
	}
	return res, nil
}

func (self *SKVMGuestDriver) FetchMonitorUrl(ctx context.Context, guest *models.SGuest) string {
	if options.Options.KvmMonitorAgentUseMetadataService && !guest.IsSriov() {
		return apis.MetaServiceMonitorAgentUrl
	}
	return self.SVirtualizedGuestDriver.FetchMonitorUrl(ctx, guest)
}

func (self *SKVMGuestDriver) RequestResetNicTrafficLimit(ctx context.Context, task taskman.ITask, host *models.SHost, guest *models.SGuest, input []api.ServerNicTrafficLimit) error {
	url := fmt.Sprintf("%s/servers/%s/reset-nic-traffic-limit", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := task.GetTaskRequestHeader()
	body := jsonutils.Marshal(input)
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSetNicTrafficLimit(ctx context.Context, task taskman.ITask, host *models.SHost, guest *models.SGuest, input []api.ServerNicTrafficLimit) error {
	url := fmt.Sprintf("%s/servers/%s/set-nic-traffic-limit", host.ManagerUri, guest.Id)
	httpClient := httputils.GetDefaultClient()
	header := task.GetTaskRequestHeader()
	body := jsonutils.Marshal(input)
	_, _, err := httputils.JSONRequest(httpClient, ctx, "POST", url, header, body, false)
	if err != nil {
		return errors.Wrap(err, "host request")
	}
	return nil
}

func (self *SKVMGuestDriver) RequestStartRescue(ctx context.Context, task taskman.ITask, body jsonutils.JSONObject, host *models.SHost, guest *models.SGuest) error {
	header := self.getTaskRequestHeader(task)
	client := httputils.GetDefaultClient()
	url := fmt.Sprintf("%s/servers/%s/start-rescue", host.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(client, ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}

	return nil
}

func (self *SKVMGuestDriver) ValidateSyncOSInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return httperrors.NewBadRequestError("can't sync guest os info in status %s", guest.Status)
	}
	return nil
}

func (kvm *SKVMGuestDriver) ValidateGuestChangeConfigInput(ctx context.Context, guest *models.SGuest, input api.ServerChangeConfigInput) (*api.ServerChangeConfigSettings, error) {
	confs, err := kvm.SBaseGuestDriver.ValidateGuestChangeConfigInput(ctx, guest, input)
	if err != nil {
		return nil, errors.Wrap(err, "SBaseGuestDriver.ValidateGuestChangeConfigInput")
	}

	if confs.ExtraCpuChanged() && guest.Status != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("Can't change extra cpus on vm status %s", guest.Status)
	}

	for i := range input.ResetTrafficLimits {
		input.ResetTrafficLimits[i].Mac = netutils.FormatMacAddr(input.ResetTrafficLimits[i].Mac)
		_, err := guest.GetGuestnetworkByMac(input.ResetTrafficLimits[i].Mac)
		if err != nil {
			return nil, errors.Wrapf(err, "get guest network by ResetTrafficLimits mac %s", input.ResetTrafficLimits[i].Mac)
		}
	}
	if len(input.ResetTrafficLimits) > 0 {
		confs.ResetTrafficLimits = input.ResetTrafficLimits
	}

	for i := range input.SetTrafficLimits {
		input.SetTrafficLimits[i].Mac = netutils.FormatMacAddr(input.SetTrafficLimits[i].Mac)
		_, err := guest.GetGuestnetworkByMac(input.SetTrafficLimits[i].Mac)
		if err != nil {
			return nil, errors.Wrapf(err, "get guest network by SetTrafficLimits mac %s", input.SetTrafficLimits[i].Mac)
		}
	}
	if len(input.SetTrafficLimits) > 0 {
		confs.SetTrafficLimits = input.SetTrafficLimits
	}

	return confs, nil
}

func (kvm *SKVMGuestDriver) ValidateGuestHotChangeConfigInput(ctx context.Context, guest *models.SGuest, confs *api.ServerChangeConfigSettings) (*api.ServerChangeConfigSettings, error) {
	if guest.GetMetadata(ctx, api.VM_METADATA_HOTPLUG_CPU_MEM, nil) != "enable" {
		return confs, errors.Wrap(errors.ErrInvalidStatus, "host plug cpu memory is disabled")
	}
	if apis.IsARM(guest.OsArch) {
		return confs, errors.Wrap(errors.ErrInvalidStatus, "cpu architecture is arm")
	}
	return confs, nil
}

func (kvm *SKVMGuestDriver) GetRandomNetworkTypes() []api.TNetworkType {
	return []api.TNetworkType{api.NETWORK_TYPE_GUEST, api.NETWORK_TYPE_HOSTLOCAL}
}
