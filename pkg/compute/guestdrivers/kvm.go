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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	host_api "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	guestdriver_types "yunion.io/x/onecloud/pkg/compute/guestdrivers/types"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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

func (self *SKVMGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
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
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/snapshot", host.ManagerUri, guest.Id)
	body := jsonutils.NewDict()
	body.Set("disk_id", jsonutils.NewString(diskId))
	body.Set("snapshot_id", jsonutils.NewString(snapshotId))
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
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

	if guest.CheckQemuVersion(guest.GetMetadata("__qemu_version", userCred), "2.12.1") && strings.HasSuffix(cmd, "vnc") {
		port = findVNCPort2(results)
	} else {
		port = findVNCPort(results)
	}

	result := &cloudprovider.ServerVncOutput{
		Host:       host.AccessIp,
		Protocol:   guest.GetVdi(),
		Port:       int64(port),
		Hypervisor: api.HYPERVISOR_ESXI,
	}
	return result, nil
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

func (self *SKVMGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
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
	desc, err := guest.GetDriver().GetJsonDescAtHost(ctx, userCred, guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	config.Add(desc, "desc")
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	url := fmt.Sprintf("%s/servers/%s/start", host.ManagerUri, guest.Id)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	return nil
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
	if ispId := guest.GetMetadata(api.BASE_INSTANCE_SNAPSHOT_ID, userCred); len(ispId) > 0 {
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

func (self *SKVMGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
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
	if err := eip.SetStatus(userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE); err != nil {
		return errors.Wrapf(err, "set eip status to %s", api.EIP_STATUS_ALLOCATE)
	}
	return nil
}

func (self *SKVMGuestDriver) NeedStopForChangeSpec(guest *models.SGuest, cpuChanged, memChanged bool) bool {
	return guest.GetMetadata("hotplug_cpu_mem", nil) != "enable" ||
		(memChanged && guest.GetMetadata("__hugepage", nil) == "native") ||
		apis.IsARM(guest.OsArch)
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
	_, err := guest.SendMonitorCommand(ctx, task.GetUserCred(), "system_reset")
	return err
}

func (self *SKVMGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	host, _ := guest.GetHost()
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
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SKVMGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SKVMGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if guest.GetDiskIndex(disk.Id) <= 0 && guest.Status == api.VM_RUNNING {
		return fmt.Errorf("Cann't online resize root disk")
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SKVMGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	desc, err := guest.GetDriver().GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host, nil)
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

func (self *SKVMGuestDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host, _ := guest.GetHost()
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

func (self *SKVMGuestDriver) NeedRequestGuestHotAddIso(ctx context.Context, guest *models.SGuest) bool {
	return guest.Status == api.VM_RUNNING
}

func (self *SKVMGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
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
	host, _ := guest.GetHost()
	desc, err := guest.GetDriver().GetJsonDescAtHost(ctx, task.GetUserCred(), guest, host, nil)
	if err != nil {
		return errors.Wrapf(err, "GetJsonDescAtHost")
	}
	body := jsonutils.NewDict()
	body.Add(desc, "desc")
	body.Set("backup_nbd_server_uri", jsonutils.NewString(guest.GetMetadata("backup_nbd_server_uri", task.GetUserCred())))
	url := fmt.Sprintf("%s/servers/%s/drive-mirror", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
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

func (self *SKVMGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return true, nil
}

func (self *SKVMGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SKVMGuestDriver) IsSupportLiveMigrate() bool {
	return true
}

func checkAssignHost(userCred mcclient.TokenCredential, preferHost string) error {
	iHost, _ := models.HostManager.FetchByIdOrName(userCred, preferHost)
	if iHost == nil {
		return httperrors.NewBadRequestError("Host %s not found", preferHost)
	}
	host := iHost.(*models.SHost)
	err := host.IsAssignable(userCred)
	if err != nil {
		return errors.Wrap(err, "IsAssignable")
	}
	return nil
}

func (self *SKVMGuestDriver) CheckMigrate(guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestMigrateInput) error {
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if !input.IsRescueMode && guest.Status != api.VM_READY {
		return httperrors.NewServerStatusError("Cannot normal migrate guest in status %s, try rescue mode or server-live-migrate?", guest.Status)
	}
	if input.IsRescueMode {
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
	if len(input.PreferHost) > 0 {
		err := checkAssignHost(userCred, input.PreferHost)
		if err != nil {
			return errors.Wrap(err, "checkAssignHost")
		}
	}
	return nil
}

func (self *SKVMGuestDriver) CheckLiveMigrate(guest *models.SGuest, userCred mcclient.TokenCredential, input api.GuestLiveMigrateInput) error {
	if len(guest.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_SUSPEND}) {
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
			err := checkAssignHost(userCred, input.PreferHost)
			if err != nil {
				return errors.Wrap(err, "checkAssignHost")
			}
		}
	}
	return nil
}

func (self *SKVMGuestDriver) ValidateDetachNetwork(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest) error {
	if guest.Status == api.VM_RUNNING && guest.GetMetadata("hot_remove_nic", nil) != "enable" {
		return httperrors.NewBadRequestError("Guest %s can't hot remove nic", guest.GetName())
	}
	return nil
}

func (self *SKVMGuestDriver) ValidateChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, input *api.ServerChangeDiskStorageInput) error {
	// kvm guest must in ready status
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
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
		if storage.GetId() == input.TargetStorageId {
			foundStorage = true
		}
	}
	if !foundStorage {
		return httperrors.NewBadRequestError("Storage %s not attached or enabled on host %s", input.TargetStorageId, host.GetName())
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
