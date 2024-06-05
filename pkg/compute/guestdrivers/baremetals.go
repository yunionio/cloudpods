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
	"path/filepath"
	"regexp"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
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

func (self *SBaremetalGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SBaremetalGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
	}
}

func (self *SBaremetalGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_ON_PREMISE
	keys.Provider = api.CLOUD_PROVIDER_ONECLOUD
	keys.Brand = api.ONECLOUD_BRAND_ONECLOUD
	keys.Hypervisor = api.HYPERVISOR_BAREMETAL
	return keys
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

func (self *SBaremetalGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, confs []*api.BaremetalDiskConfig, disks []*api.DiskConfig) ([]*api.DiskConfig, error) {
	baremetalStorage := models.ConvertStorageInfo2BaremetalStorages(host.StorageInfo)
	if baremetalStorage == nil {
		return nil, fmt.Errorf("Convert storage info error")
	}
	if len(confs) == 0 {
		parsedConf, _ := baremetal.ParseDiskConfig("")
		confs = []*api.BaremetalDiskConfig{&parsedConf}
	}
	layouts, err := baremetal.CalculateLayout(confs, baremetalStorage)
	if err != nil {
		return nil, err
	}
	err = host.UpdateDiskConfig(userCred, layouts)
	if err != nil {
		return nil, err
	}
	allocable, extra := baremetal.CheckDisksAllocable(layouts, disks)
	if !allocable {
		return nil, fmt.Errorf("baremetal.CheckDisksAllocable not allocable")
	}
	return extra, nil
}

func (self *SBaremetalGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SBaremetalGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return nil, httperrors.NewUnsupportOperationError("Cannot change config for baremtal")
}

func (self *SBaremetalGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SBaremetalGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	return httperrors.NewUnsupportOperationError("Cannot resize disk for baremtal")
}

func (self *SBaremetalGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig) (*models.SNetwork, []models.SNicConfig, api.IPAllocationDirection, bool, error) {
	netifs, net, err := host.GetNetinterfacesWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	if err != nil {
		return nil, nil, "", false, errors.Wrap(err, "get host netinterfaces")
	}
	if netifs != nil {
		nicCnt := 1
		if netConfig.RequireTeaming || netConfig.TryTeaming {
			nicCnt = 2
		}
		if len(netifs) < nicCnt {
			if netConfig.RequireTeaming {
				return net, nil, "", false, errors.Errorf("not enough network interfaces, want %d got %d", nicCnt, len(netifs))
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
		reuseAddr := false
		hn := host.GetAttach2Network(netConfig.Network)
		if hn != nil && options.Options.BaremetalServerReuseHostIp {
			if netConfig.Address == "" || netConfig.Address == hn.IpAddr {
				// try to reuse host network IP address
				netConfig.Address = hn.IpAddr
				reuseAddr = true
			}
		}

		return net, nicConfs, api.IPAllocationStepup, reuseAddr, nil
	}
	return net, nil, "", false, nil
}

func (self *SBaremetalGuestDriver) GetRandomNetworkTypes() []api.TNetworkType {
	return []api.TNetworkType{api.NETWORK_TYPE_BAREMETAL, api.NETWORK_TYPE_GUEST}
}

func (self *SBaremetalGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]models.SGuestnetwork, error) {
	netifs := host.GetHostNetInterfaces()
	netsAvaiable := make([]models.SNetwork, 0)
	netifIndexs := make(map[string][]models.SNetInterface, 0)

	drv, err := guest.GetDriver()
	if err != nil {
		return nil, err
	}

	netTypes := drv.GetRandomNetworkTypes()
	if len(netConfig.NetType) > 0 {
		netTypes = []api.TNetworkType{api.TNetworkType(netConfig.NetType)}
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
			net, _ = wire.GetCandidatePrivateNetwork(ctx, userCred, userCred, models.NetworkManager.AllowScope(userCred), netConfig.Exit, netTypes)
		} else {
			net, _ = wire.GetCandidateAutoAllocNetwork(ctx, userCred, userCred, models.NetworkManager.AllowScope(userCred), netConfig.Exit, netTypes)
		}
		if net != nil {
			netsAvaiable = append(netsAvaiable, *net)
			if _, exist := netifIndexs[net.WireId]; !exist {
				netifIndexs[net.WireId] = make([]models.SNetInterface, 0)
			}
			netifIndexs[net.WireId] = append(netifIndexs[net.WireId], netifs[idx])
		}
	}
	if len(netsAvaiable) == 0 {
		return nil, fmt.Errorf("No appropriate host virtual network...")
	}
	net := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, netTypes)
	if net != nil {
		netifs := netifIndexs[net.WireId]
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
		address := ""
		reuseAddr := false
		hn := host.GetAttach2Network(net.Id)
		if hn != nil && options.Options.BaremetalServerReuseHostIp {
			// try to reuse host network IP address
			address = hn.IpAddr
			reuseAddr = true
		}
		return guest.Attach2Network(ctx, userCred, models.Attach2NetworkArgs{
			Network:             net,
			PendingUsage:        pendingUsage,
			IpAddr:              address,
			NicDriver:           netConfig.Driver,
			BwLimit:             netConfig.BwLimit,
			Virtual:             netConfig.Vip,
			TryReserved:         false,
			AllocDir:            api.IPAllocationStepup,
			RequireDesignatedIP: false,
			UseDesignatedIP:     reuseAddr,
			NicConfs:            nicConfs,

			IsDefault: netConfig.IsDefault,
		})
	}
	return nil, fmt.Errorf("No appropriate host virtual network...")
}

func (self *SBaremetalGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_BAREMETAL,
	}
}

func (self *SBaremetalGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	if len(storageIds) != 0 {
		return models.StorageManager.FetchStorageById(storageIds[0]), nil
	}
	bs := host.GetBaremetalstorage()
	if bs == nil {
		return nil, nil
	}
	return bs.GetStorage(), nil
}

func (self *SBaremetalGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	diskCat := guest.CategorizeDisks()
	var imageId string
	if diskCat.Root != nil {
		imageId = diskCat.Root.GetTemplateId()
	}
	if len(imageId) == 0 {
		task.ScheduleRun(nil)
		return nil
	}
	storage, _ := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	storageCache := storage.GetStoragecache()
	if storageCache == nil {
		return fmt.Errorf("no valid storage cache")
	}
	input := api.CacheImageInput{
		ImageId:      imageId,
		Format:       "qcow2",
		ParentTaskId: task.GetTaskId(),
	}
	return storageCache.StartImageCacheTask(ctx, task.GetUserCred(), input)
}

func (self *SBaremetalGuestDriver) NeedRequestGuestHotAddIso(ctx context.Context, guest *models.SGuest) bool {
	return true
}

func (self *SBaremetalGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	host, _ := guest.GetHost()
	return host.StartInsertIsoTask(ctx, task.GetUserCred(), filepath.Base(path), boot, task.GetTaskId())
}

func (self *SBaremetalGuestDriver) RequestGuestHotRemoveIso(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host, _ := guest.GetHost()
	return host.StartEjectIsoTask(ctx, task.GetUserCred(), task.GetTaskId())
}

func (self *SBaremetalGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, bootIndex *int8, task taskman.ITask, guest *models.SGuest) error {
	return guest.StartInsertIsoTask(ctx, 0, imageId, true, nil, guest.HostId, task.GetUserCred(), task.GetTaskId())
}

func (self *SBaremetalGuestDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	desc := guest.GetJsonDescAtBaremetal(ctx, host)
	config := jsonutils.NewDict()
	config.Set("desc", jsonutils.Marshal(desc))
	params := task.GetParams()
	if params.Length() > 0 {
		config.Add(params, "params")
	}
	headers := task.GetTaskRequestHeader()
	url := fmt.Sprintf("/baremetals/%s/servers/%s/start", host.Id, guest.Id)
	_, err := host.BaremetalSyncRequest(ctx, "POST", url, headers, config)
	return err
}

func (self *SBaremetalGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest,
	host *models.SHost, task taskman.ITask) error {
	if host == nil {
		host, _ = guest.GetHost()
	}
	guestStatus, _ := task.GetParams().GetString("guest_status")
	overridePendingDelete := jsonutils.QueryBoolean(task.GetParams(), "override_pending_delete", false)
	purge := jsonutils.QueryBoolean(task.GetParams(), "purge", false)
	if host != nil && host.GetEnabled() &&
		(guestStatus == api.VM_RUNNING || strings.Index(guestStatus, "stop") >= 0) &&
		options.Options.EnablePendingDelete &&
		!guest.PendingDeleted &&
		!overridePendingDelete &&
		!purge {
		return guest.StartGuestStopTask(ctx, task.GetUserCred(), 0, true, false, task.GetTaskId())
	}
	if host != nil && !host.GetEnabled() && !purge {
		return errors.Errorf("fail to contact baremetal")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask, syncStatus bool) error {
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
	return task.ScheduleRun(nil)
}

func (self *SBaremetalGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/servers/%s", host.Id, guest.Id)
	headers := task.GetTaskRequestHeader()
	_, err := host.BaremetalSyncRequest(ctx, "DELETE", url, headers, nil)
	return err
}

func (self *SBaremetalGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) error {
	return errors.Wrap(httperrors.ErrNotSupported, "baremetal doesn't support RequestSyncstatusOnHost")
}

func (self *SBaremetalGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return models.StartResourceSyncStatusTask(ctx, userCred, guest, "BaremetalServerSyncStatusTask", parentTaskId)
}

func (self *SBaremetalGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.BaremetalDiskConfigs) != 0 {
		if err := baremetal.ValidateDiskConfigs(input.BaremetalDiskConfigs); err != nil {
			return nil, httperrors.NewInputParameterError("Invalid raid config: %v", err)
		}
	}
	if input.BaremetalRootDiskMatcher != nil {
		if err := baremetal.ValidateRootDiskMatcher(input.BaremetalRootDiskMatcher); err != nil {
			return nil, httperrors.NewInputParameterError("Invalid root disk matcher: %v", err)
		}
	}
	//if len(input.Disks) <= 0 {
	//	return nil, httperrors.NewInputParameterError("Root disk must be present")
	//}

	//disk0 := input.Disks[0]

	//if disk0.ImageId == "" {
	//	return nil, httperrors.NewInputParameterError("Root disk must have templete")
	//}
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

func (self *SBaremetalGuestDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, params *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	desc := guest.GetJsonDescAtBaremetal(ctx, host)
	return jsonutils.Marshal(desc), nil
}

func (self *SBaremetalGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	ret := &cloudprovider.ServerVncOutput{}
	ret.HostId = host.Id
	zone, _ := host.GetZone()
	ret.Zone = zone.Name
	return ret, nil
}

func (self *SBaremetalGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SBaremetalGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict, parentTaskId string) error {
	return guest.StartGueststartTask(ctx, userCred, data, "")
}

func (self *SBaremetalGuestDriver) CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, storageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
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

func (self *SBaremetalGuestDriver) IsDisableImageCache(gst *models.SGuest) (bool, error) {
	host, err := gst.GetHost()
	if err != nil {
		return false, errors.Wrapf(err, "Get guest %s(%s) host", gst.GetName(), gst.GetId())
	}
	agent := host.GetAgent(api.AgentTypeBaremetal)
	if agent == nil {
		return false, errors.Wrapf(errors.ErrNotFound, "get host %s(%s) agent", host.GetName(), host.GetId())
	}
	return agent.DisableImageCache, nil
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
	} else if val == "deploy" && jsonutils.QueryBoolean(task.GetParams(), "restart", false) {
		config.Set("on_finish", jsonutils.NewString("shutdown"))
	}

	disableCache, err := self.IsDisableImageCache(guest)
	if err != nil {
		return errors.Wrap(err, "check IsDisableImageCache")
	}
	config.Set("disable_image_cache", jsonutils.NewBool(disableCache))

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
	return fmt.Errorf("Cannot detach disk from a baremetal server")
}

func (self *SBaremetalGuestDriver) StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot attach disk to a baremetal server")
}

func (self *SBaremetalGuestDriver) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot suspend a baremetal server")
}

func (self *SBaremetalGuestDriver) StartResumeTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot resume a baremetal server")
}

func (self *SBaremetalGuestDriver) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return httperrors.NewUnsupportOperationError("Cannot save image for baremtal")
}

func (self *SBaremetalGuestDriver) StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {

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
	baremetal, _ := guest.GetHost()
	if baremetal != nil {
		return baremetal.UpdateDiskConfig(userCred, nil)
	}
	return nil
}

func (self *SBaremetalGuestDriver) IsSupportGuestClone() bool {
	return false
}

func (self *SBaremetalGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	host, _ := guest.GetHost()
	if host == nil {
		return false, errors.Wrap(httperrors.ErrNotFound, "no host")
	}
	ipmiInfo, err := host.GetIpmiInfo()
	if err != nil {
		return false, errors.Wrap(err, "host.GetIpmiInfo")
	}
	return ipmiInfo.CdromBoot, nil
}

func (self *SBaremetalGuestDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return false, nil
}
