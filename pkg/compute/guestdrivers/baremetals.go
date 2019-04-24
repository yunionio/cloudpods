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
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SBaremetalGuestDriver struct {
	SBaseGuestDriver
}

func init() {
	driver := SBaremetalGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SBaremetalGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAREMETAL
}

func (self *SBaremetalGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SBaremetalGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SBaremetalGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SBaremetalGuestDriver) GetMaxVCpuCount() int {
	return 1024
}

func (self *SBaremetalGuestDriver) GetMaxVMemSizeGB() int {
	return 4096
}

func (self *SBaremetalGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, confs []*api.BaremetalDiskConfig) error {
	baremetalStorage := models.ConvertStorageInfo2BaremetalStorages(host.StorageInfo)
	if baremetalStorage == nil {
		return fmt.Errorf("Convert storage info error")
	}
	if len(confs) == 0 {
		parsedConf, _ := baremetal.ParseDiskConfig("")
		nConfs := []*api.BaremetalDiskConfig{&parsedConf}
		layouts, err := baremetal.CalculateLayout(nConfs, baremetalStorage)
		if err != nil {
			return err
		}
		return host.UpdateDiskConfig(userCred, layouts)
	} else {
		layouts, err := baremetal.CalculateLayout(confs, baremetalStorage)
		if err != nil {
			return err
		}
		return host.UpdateDiskConfig(userCred, layouts)
	}
}

func (self *SBaremetalGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SBaremetalGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return nil, httperrors.NewUnsupportOperationError("Cannot change config for baremtal")
}

func (self *SBaremetalGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SBaremetalGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	return httperrors.NewUnsupportOperationError("Cannot resize disk for baremtal")
}

func (self *SBaremetalGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig) (*models.SNetwork, []models.SNicConfig, models.IPAddlocationDirection) {
	netifs, net := host.GetNetinterfacesWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	if netifs != nil {
		nicCnt := 1
		if netConfig.RequireTeaming || netConfig.TryTeaming {
			nicCnt = 2
		}
		if len(netifs) < nicCnt {
			if netConfig.RequireTeaming {
				return net, nil, ""
			}
			nicCnt = len(netifs)
		}
		nicConfs := make([]models.SNicConfig, 0)
		for i := 0; i < nicCnt; i += 1 {
			nicConf := models.SNicConfig{
				Mac:    netifs[i].Mac,
				Index:  netifs[i].Index,
				Ifname: "",
			}
			nicConfs = append(nicConfs, nicConf)
		}
		return net, nicConfs, models.IPAllocationStepup
	}
	return net, nil, ""
}

func (self *SBaremetalGuestDriver) GetRandomNetworkTypes() []string {
	return []string{api.NETWORK_TYPE_BAREMETAL, api.NETWORK_TYPE_GUEST}
}

func (self *SBaremetalGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]models.SGuestnetwork, error) {
	netifs := host.GetNetInterfaces()
	netsAvaiable := make([]models.SNetwork, 0)
	netifIndexs := make(map[string][]models.SNetInterface, 0)

	netTypes := guest.GetDriver().GetRandomNetworkTypes()
	if len(netConfig.NetType) > 0 {
		netTypes = []string{netConfig.NetType}
	}
	var wirePattern *regexp.Regexp
	if len(netConfig.Wire) > 0 {
		wirePattern = regexp.MustCompile(netConfig.Wire)
	}
	for idx, netif := range netifs {
		if !netif.IsUsableServernic() {
			continue
		}
		wire := netif.GetWire()
		if wire == nil {
			continue
		}
		if wirePattern != nil && !wirePattern.MatchString(wire.Id) && !wirePattern.MatchString(wire.Name) {
			continue
		}
		var net *models.SNetwork
		if netConfig.Private {
			net, _ = wire.GetCandidatePrivateNetwork(userCred, netConfig.Exit, netTypes)
		} else {
			net, _ = wire.GetCandidatePublicNetwork(netConfig.Exit, netTypes)
		}
		if net != nil {
			netsAvaiable = append(netsAvaiable, *net)
			if _, exist := netifIndexs[net.Id]; !exist {
				netifIndexs[net.Id] = make([]models.SNetInterface, 0)
			}
			netifIndexs[net.Id] = append(netifIndexs[net.Id], netifs[idx])
		}
	}
	if len(netsAvaiable) == 0 {
		return nil, fmt.Errorf("No appropriate host virtual network...")
	}
	net := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, netTypes)
	if net != nil {
		netifs := netifIndexs[net.Id]
		nicConfs := make([]models.SNicConfig, 0)
		nicCnt := 1
		if netConfig.RequireTeaming || netConfig.TryTeaming {
			nicCnt = 2
		}
		if len(netifs) < nicCnt {
			if netConfig.RequireTeaming {
				return nil, fmt.Errorf("not enough network interfaces, want %d got %d", nicCnt, len(netifs))
			}
			nicCnt = len(netifs)
		}
		for i := 0; i < nicCnt; i += 1 {
			nicConf := models.SNicConfig{
				Mac:    netifs[i].Mac,
				Index:  netifs[i].Index,
				Ifname: "",
			}
			nicConfs = append(nicConfs, nicConf)
		}
		return guest.Attach2Network(ctx, userCred, net, pendingUsage, "", netConfig.Driver, netConfig.BwLimit, netConfig.Vip, false, models.IPAllocationStepup, false, nicConfs)
	}
	return nil, fmt.Errorf("No appropriate host virtual network...")
}

func (self *SBaremetalGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_BAREMETAL,
	}
}

func (self *SBaremetalGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	if len(storageIds) != 0 {
		return models.StorageManager.FetchStorageById(storageIds[0])
	}
	bs := host.GetBaremetalstorage()
	if bs == nil {
		return nil
	}
	return bs.GetStorage()
}

func (self *SBaremetalGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil) // skip
	return nil
}

func (self *SBaremetalGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *models.SGuest, task taskman.ITask) error {
	task.ScheduleRun(nil) // skip
	return nil
}

func (self *SBaremetalGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtBaremetal(ctx, host)
	config := jsonutils.NewDict()
	config.Set("desc", desc)
	headers := task.GetTaskRequestHeader()
	url := fmt.Sprintf("/baremetals/%s/servers/%s/start", host.Id, guest.Id)
	return host.BaremetalSyncRequest(ctx, "POST", url, headers, config)
}

func (self *SBaremetalGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest,
	host *models.SHost, task taskman.ITask) error {
	if host == nil {
		host = guest.GetHost()
	}
	guestStatus, _ := task.GetParams().GetString("guest_status")
	overridePendingDelete := jsonutils.QueryBoolean(task.GetParams(), "override_pending_delete", false)
	purge := jsonutils.QueryBoolean(task.GetParams(), "purge", false)
	if host != nil && host.Enabled &&
		(guestStatus == api.VM_RUNNING || strings.Index(guestStatus, "stop") >= 0) &&
		options.Options.EnablePendingDelete &&
		!guest.PendingDeleted &&
		!overridePendingDelete &&
		!purge {
		return guest.StartGuestStopTask(ctx, task.GetUserCred(), true, task.GetTaskId())
	}
	if host != nil && !host.Enabled && !purge {
		return fmt.Errorf("fail to contact baremetal")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	body := jsonutils.NewDict()
	timeout, err := task.GetParams().Int("timeout")
	if err != nil {
		timeout = 30
	}
	if jsonutils.QueryBoolean(task.GetParams(), "is_force", false) || jsonutils.QueryBoolean(task.GetParams(), "reset", false) {
		timeout = 0
	}
	body.Set("timeout", jsonutils.NewInt(timeout))
	headers := task.GetTaskRequestHeader()
	url := fmt.Sprintf("/baremetals/%s/servers/%s/stop", host.Id, guest.Id)
	_, err = host.BaremetalSyncRequest(ctx, "POST", url, headers, body)
	return err
}

func (self *SBaremetalGuestDriver) StartGuestStopTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStopTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/servers/%s", host.Id, guest.Id)
	headers := task.GetTaskRequestHeader()
	_, err := host.BaremetalSyncRequest(ctx, "DELETE", url, headers, nil)
	return err
}

func (self *SBaremetalGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("baremetal doesn't support RequestSyncstatusOnHost")
}

func (self *SBaremetalGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerSyncStatusTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	/*var confs = make([]api.BaremetalDiskConfig, 0)*/
	//jsonArray := jsonutils.GetArrayOfPrefix(data, "baremetal_disk_config")
	//for i := 0; i < len(jsonArray); i += 1 {
	//// for data.Contains(fmt.Sprintf("baremetal_disk_config.%d", len(confs))) {
	//desc, _ := jsonArray[i].GetString() // data.GetString(fmt.Sprintf("baremetal_disk_config.%d", len(confs)))
	//data.Remove(fmt.Sprintf("baremetal_disk_config.%d", i))
	//bmConf, err := baremetal.ParseDiskConfig(desc)
	//if err != nil {
	//return nil, httperrors.NewInputParameterError("baremetal_disk_config.%d", i)
	//}
	//confs = append(confs, bmConf)
	//}
	//if len(confs) == 0 {
	//bmConf, _ := baremetal.ParseDiskConfig("")
	//confs = append(confs, bmConf)
	//}
	/*data.Set("baremetal_disk_config", jsonutils.Marshal(confs))*/
	if len(input.Disks) <= 0 {
		return nil, httperrors.NewInputParameterError("Root disk must be present")
	}

	disk0 := input.Disks[0]

	if disk0.ImageId == "" {
		return nil, httperrors.NewInputParameterError("Root disk must have templete")
	}
	return input, nil
}

func (self *SBaremetalGuestDriver) ValidateCreateDataOnHost(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *models.SHost, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if host.HostType != api.HOST_TYPE_BAREMETAL || !host.IsBaremetal {
		return nil, httperrors.NewInputParameterError("Host %s is not a baremetal", bmName)
	}
	if !utils.IsInStringArray(host.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_START_CONVERT}) {
		return nil, httperrors.NewInvalidStatusError("Baremetal %s is not ready", bmName)
	}
	if host.GetBaremetalServer() != nil {
		return nil, httperrors.NewInsufficientResourceError("Baremetal %s is occupied", bmName)
	}
	input.VmemSize = host.MemSize
	input.VcpuCount = int(host.CpuCount)
	return input, nil
}

func (self *SBaremetalGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtBaremetal(ctx, host)
}

func (self *SBaremetalGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(host.Id), "host_id")
	zone := host.GetZone()
	data.Add(jsonutils.NewString(zone.Name), "zone")
	return data, nil
}

func (self *SBaremetalGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict) error {
	return guest.StartGueststartTask(ctx, userCred, data, "")
}

func (self *SBaremetalGuestDriver) CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, storageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) OnGuestDeployTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	task.SetStage("OnDeployGuestSyncstatusComplete", nil)
	return guest.StartSyncstatus(ctx, task.GetUserCred(), task.GetTaskId())
}

func (self *SBaremetalGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	if data.Contains("disks") {
		disks, _ := data.GetArray("disks")
		for i := 0; i < len(disks); i++ {
			diskId, _ := disks[i].GetString("disk_id")
			iDisk, _ := models.DiskManager.FetchById(diskId)
			if iDisk == nil {
				return fmt.Errorf("OnGuestDeployTaskDataReceived fetch disk error")
			}
			disk := iDisk.(*models.SDisk)
			diskSize, _ := disks[i].Int("size")
			notes := fmt.Sprintf("%s=>%s", disk.Status, api.DISK_READY)
			_, err := db.Update(disk, func() error {
				if disk.DiskSize < int(diskSize) {
					disk.DiskSize = int(diskSize)
				}
				disk.DiskFormat = "raw"
				disk.Status = api.DISK_READY
				return nil
			})
			if err != nil {
				return err
			}
			db.OpsLog.LogEvent(disk, db.ACT_UPDATE_STATUS, notes, task.GetUserCred())
			logclient.AddActionLogWithStartable(task, disk, logclient.ACT_VM_SYNC_STATUS, nil, task.GetUserCred(), false)
			db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), task.GetUserCred())
			if disks[i].Contains("dev") {
				dev, _ := disks[i].GetString("dev")
				disk.SetMetadata(ctx, "dev", dev, task.GetUserCred())
			}
		}
	}
	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SBaremetalGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	val, _ := config.GetString("action")
	if len(val) == 0 {
		val = "deploy"
	}
	if val == "rebuild" && jsonutils.QueryBoolean(task.GetParams(), "auto_start", false) {
		config.Set("on_finish", jsonutils.NewString("restart"))
	}
	url := fmt.Sprintf("/baremetals/%s/servers/%s/%s", host.Id, guest.Id, val)
	headers := task.GetTaskRequestHeader()
	_, err = host.BaremetalSyncRequest(ctx, "POST", url, headers, config)
	return err
}

func (self *SBaremetalGuestDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SBaremetalGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot detach disk from a baremetal serer")
}

func (self *SBaremetalGuestDriver) StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot attach disk to a baremetal serer")
}

func (self *SBaremetalGuestDriver) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot suspend a baremetal serer")
}

func (self *SBaremetalGuestDriver) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return httperrors.NewUnsupportOperationError("Cannot save image for baremtal")
}

func (self *SBaremetalGuestDriver) StartGuestResetTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerResetTask", guest, userCred, nil, "", parentTaskId, nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	err := guest.DeleteAllDisksInDB(ctx, userCred)
	if err != nil {
		return err
	}
	baremetal := guest.GetHost()
	if baremetal != nil {
		return baremetal.UpdateDiskConfig(userCred, nil)
	}
	return nil
}

func (self *SBaremetalGuestDriver) IsSupportGuestClone() bool {
	return false
}
