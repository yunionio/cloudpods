package guestdrivers

import (
	"context"
	"fmt"
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SVirtualizedGuestDriver struct {
	SBaseGuestDriver
}

func (self *SVirtualizedGuestDriver) GetMaxVCpuCount() int {
	return 128
}

func (self *SVirtualizedGuestDriver) GetMaxVMemSizeGB() int {
	return 512
}

func (self *SVirtualizedGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, params *jsonutils.JSONDict) error {
	// do nothing
	return nil
}

func (self *SVirtualizedGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig) (*models.SNetwork, string, int8, models.IPAddlocationDirection) {
	net, _ := host.GetNetworkWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	return net, netConfig.Mac, -1, models.IPAllocationStepdown
}

func (self *SVirtualizedGuestDriver) GetRandomNetworkTypes() []string {
	return []string{models.NETWORK_TYPE_GUEST}
}

func (self *SVirtualizedGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig, pendingUsage quotas.IQuota) error {
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

		log.Debugf("Wire %#v", wire)

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
		return fmt.Errorf("No appropriate host virtual network...")
	}
	selNet := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, netTypes)
	if selNet == nil {
		return fmt.Errorf("Not enough address in virtual network")
	}
	err := guest.Attach2Network(ctx, userCred, selNet, pendingUsage, netConfig.Address, netConfig.Mac, netConfig.Driver, netConfig.BwLimit, netConfig.Vip, -1, netConfig.Reserved, models.IPAllocationDefault, false, netConfig.Ifname)
	return err
}

func (self *SVirtualizedGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	return host.GetLeastUsedStorage(backend)
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

func (self *SVirtualizedGuestDriver) OnGuestDeployTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	if jsonutils.QueryBoolean(task.GetParams(), "restart", false) {
		task.SetStage("OnDeployStartGuestComplete", nil)
		return guest.StartGueststartTask(ctx, task.GetUserCred(), nil, task.GetTaskId())
	}
	task.SetStage("OnDeployGuestSyncstatusComplete", nil)
	return guest.StartSyncstatus(ctx, task.GetUserCred(), task.GetTaskId())
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
	if host != nil && host.Enabled && host.HostStatus == models.HOST_ONLINE {
		return guest.StartGuestStopTask(ctx, task.GetUserCred(), true, task.GetTaskId())
	}
	if host != nil && !jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
		return fmt.Errorf("fail to contact host")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SVirtualizedGuestDriver) ValidateCreateHostData(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *models.SHost, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if host.HostStatus != models.HOST_ONLINE {
		return nil, httperrors.NewInvalidStatusError("Host %s is not online", bmName)
	}
	data.Add(jsonutils.NewString(host.Id), "prefer_host_id")
	if host.IsPrepaidRecycle() {
		data.Set("vmem_size", jsonutils.NewInt(int64(host.MemSize)))
		data.Set("vcpu_count", jsonutils.NewInt(int64(host.CpuCount)))

		if host.GetGuestCount() >= 1 {
			return nil, httperrors.NewInsufficientResourceError("host has been occupied")
		}
	}
	return data, nil
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
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSaveImageTask", guest, userCred, params, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SVirtualizedGuestDriver) StartGuestDiskSnapshotTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDiskSnapshotTask", guest, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
