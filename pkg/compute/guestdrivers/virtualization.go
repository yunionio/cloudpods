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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

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

func (self *SVirtualizedGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, params []*api.BaremetalDiskConfig) error {
	// do nothing
	return nil
}

func (self *SVirtualizedGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig) (*models.SNetwork, []models.SNicConfig, models.IPAddlocationDirection) {
	net, _ := host.GetNetworkWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
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
	return net, nicConfs, models.IPAllocationStepdown
}

func (self *SVirtualizedGuestDriver) GetRandomNetworkTypes() []string {
	return []string{api.NETWORK_TYPE_GUEST}
}

func (self *SVirtualizedGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]models.SGuestnetwork, error) {
	var wirePattern *regexp.Regexp
	if len(netConfig.Wire) > 0 {
		wirePattern = regexp.MustCompile(netConfig.Wire)
	}
	hostwires := host.GetHostwires()
	netsAvaiable := make([]models.SNetwork, 0)
	netTypes := guest.GetDriver().GetRandomNetworkTypes()
	if len(netConfig.NetType) > 0 {
		netTypes = []string{netConfig.NetType}
	}
	for i := 0; i < len(hostwires); i += 1 {
		hostwire := hostwires[i]
		wire := hostwire.GetWire()

		if wire == nil {
			log.Errorf("host wire is nil?????")
			continue
		}

		// !!
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
		}
	}
	if len(netsAvaiable) == 0 {
		return nil, fmt.Errorf("No appropriate host virtual network...")
	}
	if len(netConfig.Address) > 0 {
		addr, _ := netutils.NewIPV4Addr(netConfig.Address)
		netsAvaiableForAddr := make([]models.SNetwork, 0)
		for i := range netsAvaiable {
			if netsAvaiable[i].IsAddressInRange(addr) {
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
	gn, err := guest.Attach2Network(ctx, userCred, selNet, pendingUsage, netConfig.Address, netConfig.Driver, netConfig.BwLimit, netConfig.Vip, netConfig.Reserved, models.IPAllocationDefault, netConfig.RequireDesignatedIP, nicConfs)
	return gn, err
}

func (self *SVirtualizedGuestDriver) GetStorageTypes() []string {
	return nil
}

func (self *SVirtualizedGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	if len(storageIds) == 0 {
		return host.GetLeastUsedStorage(backend)
	}
	return models.StorageManager.FetchStorageById(storageIds[0])
}

func (self *SVirtualizedGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartInsertIsoTask(ctx, imageId, guest.HostId, task.GetUserCred(), task.GetTaskId())
}

func (self *SVirtualizedGuestDriver) StartGuestStopTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStopTask", guest, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
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
		jsonutils.QueryBoolean(task.GetParams(), "override_pending_delete", false))
}

func (self *SVirtualizedGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSyncstatusTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest,
	host *models.SHost, task taskman.ITask) error {
	if host == nil {
		host = guest.GetHost()
	}
	if host != nil && host.Enabled && host.HostStatus == api.HOST_ONLINE {
		return guest.StartGuestStopTask(ctx, task.GetUserCred(), true, task.GetTaskId())
	}
	if host != nil && !jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
		return fmt.Errorf("fail to contact host")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	return input, nil
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

func (self *SVirtualizedGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict) error {
	return guest.StartGueststartTask(ctx, userCred, data, "")
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
	return cache.StartImageCacheTask(ctx, userCred, imageId, format, false, task.GetTaskId())
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

func (self *SVirtualizedGuestDriver) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	guest.SetStatus(userCred, api.VM_START_SAVE_DISK, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSaveImageTask", guest, userCred, params, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}
