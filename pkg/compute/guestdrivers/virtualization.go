package guestdrivers

import (
	"context"
	"fmt"
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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

func (self *SVirtualizedGuestDriver) PrepareDiskRaidConfig(host *models.SHost, params *jsonutils.JSONDict) error {
	// do nothing
	return nil
}

func (self *SVirtualizedGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig) (*models.SNetwork, string, int8, models.IPAddlocationDirection) {
	net, _ := host.GetNetworkWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	return net, netConfig.Mac, -1, models.IPAllocationStepdown
}

func (self *SVirtualizedGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *models.SNetworkConfig, pendingUsage quotas.IQuota) error {
	var wirePattern *regexp.Regexp
	if len(netConfig.Wire) > 0 {
		wirePattern = regexp.MustCompile(netConfig.Wire)
	}
	hostwires := host.GetHostwires()
	netsAvaiable := make([]models.SNetwork, 0)
	for i := 0; i < len(hostwires); i += 1 {
		hostwire := hostwires[i]
		wire := hostwire.GetWire()

		if wire == nil {
			continue
		}

		log.Debugf("Wire %#v", wire)

		if wirePattern != nil && !wirePattern.MatchString(wire.Id) && wirePattern.MatchString(wire.Name) {
			continue
		}
		var net *models.SNetwork
		if netConfig.Private {
			net, _ = wire.GetCandidatePrivateNetwork(userCred, netConfig.Exit, models.SERVER_TYPE_GUEST)
		} else {
			net, _ = wire.GetCandidatePublicNetwork(netConfig.Exit, models.SERVER_TYPE_GUEST)
		}
		if net != nil {
			netsAvaiable = append(netsAvaiable, *net)
		}
	}
	if len(netsAvaiable) == 0 {
		return fmt.Errorf("No appropriate host virtual network...")
	}
	selNet := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, models.SERVER_TYPE_GUEST)
	if selNet == nil {
		return fmt.Errorf("Not enough address in virtual network")
	}
	err := guest.Attach2Network(ctx, userCred, selNet, pendingUsage, netConfig.Address, netConfig.Mac, netConfig.Driver, netConfig.BwLimit, netConfig.Vip, -1, netConfig.Reserved, models.IPAllocationDefault, false)
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

func (self *SVirtualizedGuestDriver) OnGuestDeployTaskComplete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	if jsonutils.QueryBoolean(task.GetParams(), "restart", false) {
		task.SetStage("OnDeployStartGuestComplete", nil)
		return guest.StartGueststartTask(ctx, task.GetUserCred(), nil, task.GetTaskId())
	} else {
		guest.SetStatus(task.GetUserCred(), models.VM_READY, "ready")
		task.SetStageComplete(ctx, nil)
		return nil
	}
}

func (self *SVirtualizedGuestDriver) StartGuestSyncstatusTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSyncstatusTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVirtualizedGuestDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	guestStatus, _ := task.GetParams().GetString("guest_status")
	if guestStatus == models.VM_RUNNING {
		host := guest.GetHost()
		if host != nil && host.Enabled && host.HostStatus == models.HOST_ONLINE && !jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return guest.StartGuestStopTask(ctx, task.GetUserCred(), true, task.GetTaskId())
		}
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
	return data, nil
}

func (self *SVirtualizedGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SVirtualizedGuestDriver) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, data *jsonutils.JSONDict) error {
	return guest.StartGueststartTask(ctx, userCred, data, "")
}

func (self *SVirtualizedGuestDriver) CheckDiskTemplateOnStorage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, storageId string, task taskman.ITask) error {
	storage := models.StorageManager.FetchStorageById(storageId)
	if storage == nil {
		return fmt.Errorf("No such storage?? %s", storageId)
	}
	cache := storage.GetStoragecache()
	if cache == nil {
		return fmt.Errorf("Cache is missing from storage")
	}
	return cache.StartImageCacheTask(ctx, userCred, imageId, false, task.GetTaskId())
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
