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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVirtualizedGuestDriver struct {
	SBaseGuestDriver
}

func (d *SVirtualizedGuestDriver) DoScheduleSKUFilter() bool {
	return false
}

func (self *SVirtualizedGuestDriver) GetMaxVCpuCount() int {
	return 128
}

func (self *SVirtualizedGuestDriver) GetMaxVMemSizeGB() int {
	return 512
}

func (self *SVirtualizedGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, params []*api.BaremetalDiskConfig, disks []*api.DiskConfig) ([]*api.DiskConfig, error) {
	// do nothing
	return nil, nil
}

func (self *SVirtualizedGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig) (*models.SNetwork, []models.SNicConfig, api.IPAllocationDirection, bool, error) {
	net, err := host.GetNetworkWithId(netConfig.Network, netConfig.Reserved)
	if err != nil {
		return nil, nil, "", false, errors.Wrapf(err, "get network with id %q, reserverd %v", netConfig.Network, netConfig.Reserved)
	}
	nicConfs := []models.SNicConfig{
		{
			Mac:    netConfig.Mac,
			Index:  -1,
			Ifname: netConfig.Ifname,
		},
	}
	if netConfig.RequireTeaming || netConfig.TryTeaming {
		nicConfs = append(nicConfs, models.SNicConfig{
			Mac:    "",
			Index:  -1,
			Ifname: "",
		})
	}
	// reUse := false
	// if len(netConfig.Address) > 0 && !options.Options.EnablePreAllocateIpAddr && !utils.IsInStringArray(host.GetProviderName(), []string{api.CLOUD_PROVIDER_ONECLOUD, api.CLOUD_PROVIDER_VMWARE, api.CLOUD_PROVIDER_CLOUDPODS}) {
	//	reUse = true
	// }
	return net, nicConfs, api.IPAllocationStepdown, false, nil
}

func (self *SVirtualizedGuestDriver) GetRandomNetworkTypes() []api.TNetworkType {
	return []api.TNetworkType{api.NETWORK_TYPE_GUEST}
}

func (self *SVirtualizedGuestDriver) wireAvaiableForGuest(guest *models.SGuest, wire *models.SWire) (bool, error) {
	if guest.BackupHostId == "" {
		return true, nil
	} else {
		backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
		attached := backupHost.IsAttach2Wire(wire.Id)
		return attached, nil
	}
}

func (self *SVirtualizedGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]models.SGuestnetwork, error) {
	var wirePattern *regexp.Regexp
	if len(netConfig.Wire) > 0 {
		wirePattern = regexp.MustCompile(netConfig.Wire)
	}
	hostNetifs := host.GetHostNetInterfaces()
	netsAvaiable := make([]models.SNetwork, 0)
	driver, err := guest.GetDriver()
	if err != nil {
		return nil, err
	}
	netTypes := driver.GetRandomNetworkTypes()
	if len(netConfig.NetType) > 0 {
		netTypes = []api.TNetworkType{api.TNetworkType(netConfig.NetType)}
	}

	var sriovWires []string
	if netConfig.SriovDevice != nil {
		if netConfig.SriovDevice.Id != "" {
			idev, err := models.IsolatedDeviceManager.FetchById(netConfig.SriovDevice.Id)
			if err != nil {
				return nil, errors.Wrap(err, "fetch isolated device")
			}
			dev, _ := idev.(*models.SIsolatedDevice)
			sriovWires = []string{dev.WireId}
		} else {
			wires, err := models.IsolatedDeviceManager.FindUnusedNicWiresByModel(netConfig.SriovDevice.Model)
			if err != nil {
				return nil, errors.Wrap(err, "FindUnusedNicWiresByModel")
			}
			sriovWires = wires
		}
	}

	for i := 0; i < len(hostNetifs); i += 1 {
		hostNetif := hostNetifs[i]
		wire := hostNetif.GetWire()
		if wire == nil {
			continue
		}

		if ok, err := self.wireAvaiableForGuest(guest, wire); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		if netConfig.SriovDevice != nil && !utils.IsInStringArray(wire.Id, sriovWires) {
			continue
		}

		// !!
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
		}
	}
	if len(netsAvaiable) == 0 {
		return nil, fmt.Errorf("No appropriate host virtual network...")
	}
	if len(netConfig.Address) > 0 || len(netConfig.Address6) > 0 {
		addr, _ := netutils.NewIPV4Addr(netConfig.Address)
		addr6, _ := netutils.NewIPV6Addr(netConfig.Address6)
		netsAvaiableForAddr := make([]models.SNetwork, 0)
		for i := range netsAvaiable {
			if (len(netConfig.Address) == 0 || netsAvaiable[i].IsAddressInRange(addr)) && (len(netConfig.Address6) == 0 || netsAvaiable[i].IsAddress6InRange(addr6)) {
				netsAvaiableForAddr = append(netsAvaiableForAddr, netsAvaiable[i])
			}
		}
		if len(netsAvaiableForAddr) == 0 {
			if netConfig.RequireDesignatedIP {
				return nil, fmt.Errorf("No virtual network for IP %s", netConfig.Address)
			}
		} else {
			netsAvaiable = netsAvaiableForAddr
		}
	}
	selNet := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, netTypes)
	if selNet == nil {
		return nil, fmt.Errorf("Not enough address in virtual network")
	}
	nicConfs := make([]models.SNicConfig, 1)
	nicConfs[0] = models.SNicConfig{
		Mac:    netConfig.Mac,
		Index:  -1,
		Ifname: netConfig.Ifname,
	}
	if netConfig.RequireTeaming || netConfig.TryTeaming {
		nicConf := models.SNicConfig{
			Mac:    "",
			Index:  -1,
			Ifname: "",
		}
		nicConfs = append(nicConfs, nicConf)
	}
	gn, err := guest.Attach2Network(ctx, userCred, models.Attach2NetworkArgs{
		Network:             selNet,
		PendingUsage:        pendingUsage,
		IpAddr:              netConfig.Address,
		Ip6Addr:             netConfig.Address6,
		NicDriver:           netConfig.Driver,
		BwLimit:             netConfig.BwLimit,
		Virtual:             netConfig.Vip,
		TryReserved:         netConfig.Reserved,
		AllocDir:            api.IPAllocationDefault,
		RequireDesignatedIP: netConfig.RequireDesignatedIP,
		RequireIPv6:         netConfig.RequireIPv6,
		NicConfs:            nicConfs,

		IsDefault: netConfig.IsDefault,
	})
	return gn, err
}

func (self *SVirtualizedGuestDriver) GetStorageTypes() []string {
	return nil
}

func (self *SVirtualizedGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	if len(storageIds) == 0 {
		return host.GetLeastUsedStorage(diskConfig.Backend), nil
	}
	ss, err := models.StorageManager.FetchStorageByIds(storageIds)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch storages by ids: %v", storageIds)
	}
	var candidates []models.SStorage
	if len(diskConfig.Medium) > 0 {
		// try to find mediumType matched storage
		for i := range ss {
			if ss[i].MediumType == diskConfig.Medium {
				candidates = append(candidates, ss[i])
			}
		}
		if len(candidates) == 0 {
			candidates = ss
		}
	} else {
		candidates = ss
	}
	return models.ChooseLeastUsedStorage(candidates, ""), nil
}

func (self *SVirtualizedGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, bootIndex *int8, task taskman.ITask, guest *models.SGuest) error {
	return guest.StartInsertIsoTask(ctx, 0, imageId, true, bootIndex, guest.HostId, task.GetUserCred(), task.GetTaskId())
}

func (self *SVirtualizedGuestDriver) StartGuestStopTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	taskName := "GuestStopTask"
	if guest.BackupHostId != "" {
		taskName = "HAGuestStopTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (self *SVirtualizedGuestDriver) StartGuestResetTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error {
	var taskName = "GuestSoftResetTask"
	if isHard {
		taskName = "GuestHardResetTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
	return disk.StartDiskDeleteTask(ctx, task.GetUserCred(), task.GetTaskId(), isPurge,
		jsonutils.QueryBoolean(task.GetParams(), "override_pending_delete", false), false)
}

func (self *SVirtualizedGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return models.StartResourceSyncStatusTask(ctx, userCred, guest, "GuestSyncstatusTask", parentTaskId)
}

func (self *SVirtualizedGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest,
	host *models.SHost, task taskman.ITask) error {
	if host == nil {
		host, _ = guest.GetHost()
	}
	if host != nil && host.GetEnabled() && host.HostStatus == api.HOST_ONLINE {
		return guest.StartGuestStopTask(ctx, task.GetUserCred(), 0, true, false, task.GetTaskId())
	}
	if host != nil && !jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
		return fmt.Errorf("fail to contact host")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) ValidateCreateDataOnHost(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *models.SHost, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if host.HostStatus != api.HOST_ONLINE {
		return nil, httperrors.NewInvalidStatusError("Host %s is not online", bmName)
	}
	input.PreferHost = host.Id
	if host.IsPrepaidRecycle() {
		input.VmemSize = host.MemSize
		input.VcpuCount = int(host.CpuCount)

		cnt, err := host.GetGuestCount()
		if err != nil {
			return nil, httperrors.NewInternalServerError("GetGuestCount fail %s", err)
		}
		if cnt >= 1 {
			return nil, httperrors.NewInsufficientResourceError("host has been occupied")
		}
	}
	return input, nil
}

func (self *SVirtualizedGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict, parentTaskId string) error {
	return guest.StartGueststartTask(ctx, userCred, data, parentTaskId)
}

func (self *SVirtualizedGuestDriver) CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, storageId string, task taskman.ITask) error {
	storage := models.StorageManager.FetchStorageById(storageId)
	if storage == nil {
		return fmt.Errorf("No such storage?? %s", storageId)
	}
	cache := storage.GetStoragecache()
	if cache == nil {
		return fmt.Errorf("Cache is missing from storage")
	}
	input := api.CacheImageInput{
		ImageId:      imageId,
		Format:       format,
		ParentTaskId: task.GetTaskId(),
	}
	return cache.StartImageCacheTask(ctx, userCred, input)
}

func (self *SVirtualizedGuestDriver) CanKeepDetachDisk() bool {
	return true
}

func (self *SVirtualizedGuestDriver) StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDetachDiskTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestAttachDiskTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSuspendTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) StartResumeTask(ctx context.Context, userCred mcclient.TokenCredential,
	guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestResumeTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	guest.SetStatus(ctx, userCred, api.VM_START_SAVE_DISK, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSaveImageTask", guest, userCred, params, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SVirtualizedGuestDriver) StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	guest.SetStatus(ctx, userCred, api.VM_START_SAVE_DISK, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSaveGuestImageTask", guest, userCred, params, parentTaskId,
		"", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}
