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

package guest

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaseGuestDeleteTask struct {
	SGuestBaseTask
}

var (
	STORAGEIDS = "storage_ids"
)

func init() {
	taskman.RegisterTask(BaseGuestDeleteTask{})
}

func (deleteTask *BaseGuestDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, _ := guest.GetHost()
	if guest.Hypervisor == api.HYPERVISOR_BAREMETAL && host != nil && host.HostType != api.HOST_TYPE_BAREMETAL {
		// if a fake server for converted hypervisor, then just skip stop
		deleteTask.OnGuestStopComplete(ctx, guest, data)
		return
	}
	drv, err := guest.GetDriver()
	if err != nil {
		deleteTask.OnGuestStopComplete(ctx, guest, data)
		return
	}
	if len(guest.BackupHostId) > 0 {
		deleteTask.SetStage("OnMasterHostStopGuestComplete", nil)
		if err := drv.RequestStopGuestForDelete(ctx, guest, nil, deleteTask); err != nil {
			log.Errorf("RequestStopGuestForDelete fail %s", err)
			deleteTask.OnMasterHostStopGuestComplete(ctx, guest, nil)
		}
	} else {
		deleteTask.SetStage("OnGuestStopComplete", nil)
		if err := drv.RequestStopGuestForDelete(ctx, guest, nil, deleteTask); err != nil {
			log.Errorf("RequestStopGuestForDelete fail %s", err)
			deleteTask.OnGuestStopComplete(ctx, guest, nil)
		}
	}
}

func (deleteTask *BaseGuestDeleteTask) OnMasterHostStopGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	deleteTask.SetStage("OnGuestStopComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	drv, err := guest.GetDriver()
	if err != nil {
		deleteTask.OnGuestStopComplete(ctx, guest, nil)
		return
	}
	err = drv.RequestStopGuestForDelete(ctx, guest, host, deleteTask)
	if err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		deleteTask.OnGuestStopComplete(ctx, guest, nil)
	}
}

func (deleteTask *BaseGuestDeleteTask) OnMasterHostStopGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	deleteTask.OnGuestStopComplete(ctx, guest, nil) // ignore stop error
}

func (deleteTask *BaseGuestDeleteTask) StartDeleteGuestSnapshots(ctx context.Context, guest *models.SGuest) {
	guest.StartDeleteGuestSnapshots(ctx, deleteTask.UserCred, deleteTask.GetTaskId())
}

func (deleteTask *BaseGuestDeleteTask) OnGuestStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(deleteTask.Params, "delete_snapshots", false) {
		deleteTask.SetStage("OnStartEipDissociate", nil)
		guest.StartDeleteGuestSnapshots(ctx, deleteTask.UserCred, deleteTask.Id)
		return
	}
	deleteTask.OnStartEipDissociate(ctx, guest, data)
}

func (deleteTask *BaseGuestDeleteTask) OnGuestStopCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	if len(guest.ExternalId) > 0 {
		_, e := guest.GetIVM(ctx)
		if errors.Cause(e) == cloudprovider.ErrNotFound {
			deleteTask.Params.Set("override_pending_delete", jsonutils.JSONTrue)
		}
	}
	deleteTask.OnGuestStopComplete(ctx, guest, err) // ignore stop error
}

func (deleteTask *BaseGuestDeleteTask) OnStartEipDissociateFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorf("Delete guest snapshots faield: %s", data)
	deleteTask.OnStartEipDissociate(ctx, guest, nil)
}

func (deleteTask *BaseGuestDeleteTask) OnStartEipDissociate(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if sourceGuestId := guest.GetMetadata(ctx, api.SERVER_META_CONVERT_FROM_ESXI, deleteTask.UserCred); len(sourceGuestId) > 0 {
		sourceGuest := models.GuestManager.FetchGuestById(sourceGuestId)
		if sourceGuest != nil &&
			sourceGuest.GetMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, deleteTask.UserCred) == guest.Id {
			sourceGuest.RemoveMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, deleteTask.UserCred)
			sourceGuest.StartSyncstatus(ctx, deleteTask.UserCred, "")
		}
	}
	eip, _ := guest.GetEipOrPublicIp()
	if eip != nil && eip.Mode != api.EIP_MODE_INSTANCE_PUBLICIP {
		// detach floating EIP only
		if jsonutils.QueryBoolean(deleteTask.Params, "purge", false) {
			// purge locally
			eip.Dissociate(ctx, deleteTask.UserCred)
			deleteTask.OnEipDissociateComplete(ctx, guest, nil)
		} else {
			deleteTask.SetStage("OnEipDissociateComplete", nil)
			autoDelete := jsonutils.QueryBoolean(deleteTask.GetParams(), "delete_eip", false)
			eip.StartEipDissociateTask(ctx, deleteTask.UserCred, autoDelete, deleteTask.GetTaskId())
		}
	} else {
		deleteTask.OnEipDissociateComplete(ctx, guest, nil)
	}
}

func (deleteTask *BaseGuestDeleteTask) OnEipDissociateCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	deleteTask.OnFailed(ctx, guest, err)
}

func (deleteTask *BaseGuestDeleteTask) OnEipDissociateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	deleteTask.SetStage("OnDiskDetachComplete", nil)
	deleteTask.OnDiskDetachComplete(ctx, obj, data)
}

// remove detachable disks
func (deleteTask *BaseGuestDeleteTask) OnDiskDetachComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Debugf("OnDiskDetachComplete")
	guest := obj.(*models.SGuest)

	guestdisksOrigin, _ := guest.GetGuestDisks()
	var guestdisks []models.SGuestdisk
	// clean dirty data
	for i := range guestdisksOrigin {
		if guestdisksOrigin[i].GetDisk() == nil {
			guestdisksOrigin[i].Detach(ctx, deleteTask.UserCred)
		} else {
			guestdisks = append(guestdisks, guestdisksOrigin[i])
		}
	}
	if len(guestdisks) == 0 {
		// on guest disks detached
		deleteTask.doClearGPUDevicesComplete(ctx, guest)
		return
	}
	// detach last detachable disk
	lastDisk := guestdisks[len(guestdisks)-1].GetDisk()
	deleteDisks := jsonutils.QueryBoolean(deleteTask.Params, "delete_disks", false)
	if deleteDisks {
		lastDisk.SetAutoDelete(lastDisk, deleteTask.GetUserCred(), true)
	}
	log.Debugf("lastDisk IsDetachable?? %v", lastDisk.IsDetachable())
	if !lastDisk.IsDetachable() {
		// no more disk need detach
		deleteTask.doClearGPUDevicesComplete(ctx, guest)
		return
	}
	purge := jsonutils.QueryBoolean(deleteTask.Params, "purge", false)
	guest.StartGuestDetachdiskTask(ctx, deleteTask.UserCred, lastDisk, true, deleteTask.GetTaskId(), purge, false)
}

func (deleteTask *BaseGuestDeleteTask) OnDiskDetachCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	deleteTask.OnFailed(ctx, guest, err)
}

// clean gpu devices
func (deleteTask *BaseGuestDeleteTask) doClearGPUDevicesComplete(ctx context.Context, guest *models.SGuest) {
	log.Debugf("doClearGPUDevicesComplete")
	models.IsolatedDeviceManager.ReleaseGPUDevicesOfGuest(ctx, guest, deleteTask.UserCred)
	if jsonutils.QueryBoolean(deleteTask.Params, "purge", false) {
		deleteTask.OnSyncConfigComplete(ctx, guest, nil)
	} else {
		deleteTask.SetStage("OnSyncConfigComplete", nil)
		guest.StartSyncTaskWithoutSyncstatus(ctx, deleteTask.UserCred, false, deleteTask.GetTaskId())
	}
}

func (deleteTask *BaseGuestDeleteTask) OnSyncConfigComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	// try to leave all groups
	guest.LeaveAllGroups(ctx, deleteTask.UserCred)
	// cleanup tap services and flows
	guest.CleanTapRecords(ctx, deleteTask.UserCred)

	isPurge := jsonutils.QueryBoolean(deleteTask.Params, "purge", false)
	overridePendingDelete := jsonutils.QueryBoolean(deleteTask.Params, "override_pending_delete", false)

	if options.Options.EnablePendingDelete && !isPurge && !overridePendingDelete {
		if guest.PendingDeleted {
			deleteTask.SetStageComplete(ctx, nil)
			return
		}
		log.Debugf("XXXXXXX Do guest pending delete... XXXXXXX")
		// pending detach
		guest.PendingDetachScalingGroup()
		guestStatus, _ := deleteTask.Params.GetString("guest_status")
		if !utils.IsInStringArray(guestStatus, []string{
			api.VM_SCHEDULE_FAILED, api.VM_NETWORK_FAILED,
			api.VM_CREATE_FAILED, api.VM_DEVICE_FAILED}) {
			deleteTask.StartPendingDeleteGuest(ctx, guest)
			return
		}
	}
	log.Debugf("XXXXXXX Do real delete on guest ... XXXXXXX")
	deleteTask.doStartDeleteGuest(ctx, guest)
}

func (deleteTask *BaseGuestDeleteTask) OnSyncConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	// guest := obj.(*models.SGuest)
	// deleteTask.OnFailed(ctx, guest, err)
	deleteTask.OnSyncConfigComplete(ctx, obj, err) // ignore sync config failed error
}

func (deleteTask *BaseGuestDeleteTask) OnGuestDeleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	deleteTask.OnFailed(ctx, guest, err)
}

func (deleteTask *BaseGuestDeleteTask) doStartDeleteGuest(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, deleteTask.UserCred, api.VM_DELETING, "delete server after stop")
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATING, guest.GetShortDesc(ctx), deleteTask.UserCred)
	deleteTask.StartDeleteGuest(ctx, guest)
}

func (deleteTask *BaseGuestDeleteTask) StartPendingDeleteGuest(ctx context.Context, guest *models.SGuest) {
	guest.DoPendingDelete(ctx, deleteTask.UserCred)
	deleteTask.SetStage("OnPendingDeleteComplete", nil)
	guest.StartSyncstatus(ctx, deleteTask.UserCred, deleteTask.GetTaskId())
}

func (deleteTask *BaseGuestDeleteTask) OnPendingDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	deleteTask.SetStageComplete(ctx, jsonutils.NewDict())
}

func (deleteTask *BaseGuestDeleteTask) OnPendingDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	deleteTask.OnPendingDeleteComplete(ctx, obj, nil)
}

func (deleteTask *BaseGuestDeleteTask) StartDeleteGuest(ctx context.Context, guest *models.SGuest) {
	// Temporary storageids to sync capacityUsed after delete
	{
		storages, _ := guest.GetStorages()
		storageIds := make([]string, len(storages))
		for i := range storages {
			storageIds[i] = storages[i].GetId()
		}
		deleteTask.Params.Set(STORAGEIDS, jsonutils.NewStringArray(storageIds))
	}
	// No snapshot
	deleteTask.SetStage("OnGuestDetachDisksComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		deleteTask.OnGuestDeleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	drv.RequestDetachDisksFromGuestForDelete(ctx, guest, deleteTask)
}

func (deleteTask *BaseGuestDeleteTask) OnGuestDetachDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	deleteTask.DoDeleteGuest(ctx, guest)
}

func (deleteTask *BaseGuestDeleteTask) OnGuestDetachDisksCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	deleteTask.OnGuestDeleteFailed(ctx, obj, data)
}

func (deleteTask *BaseGuestDeleteTask) DoDeleteGuest(ctx context.Context, guest *models.SGuest) {
	models.IsolatedDeviceManager.ReleaseDevicesOfGuest(ctx, guest, deleteTask.UserCred)
	host, _ := guest.GetHost()
	if guest.IsPrepaidRecycle() {
		err := host.BorrowIpAddrsFromGuest(ctx, deleteTask.UserCred, guest)
		if err != nil {
			msg := fmt.Sprintf("host.BorrowIpAddrsFromGuest fail %s", err)
			log.Errorf("%v", msg)
			deleteTask.OnGuestDeleteFailed(ctx, guest, jsonutils.NewString(msg))
			return
		}
		deleteTask.OnGuestDeleteComplete(ctx, guest, nil)
	} else if (host == nil || !host.GetEnabled()) && jsonutils.QueryBoolean(deleteTask.Params, "purge", false) {
		deleteTask.OnGuestDeleteComplete(ctx, guest, nil)
	} else {
		deleteTask.SetStage("OnGuestDeleteComplete", nil)
		guest.StartUndeployGuestTask(ctx, deleteTask.UserCred, deleteTask.GetTaskId(), "")
	}
}

func (deleteTask *BaseGuestDeleteTask) OnFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(ctx, deleteTask.UserCred, api.VM_DELETE_FAIL, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATE_FAIL, err, deleteTask.UserCred)
	logclient.AddActionLogWithStartable(deleteTask, guest, logclient.ACT_DELOCATE, err, deleteTask.UserCred, false)
	notifyclient.EventNotify(ctx, deleteTask.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionDelete,
		IsFail: true,
	})
	deleteTask.SetStageFailed(ctx, err)
}

func (deleteTask *BaseGuestDeleteTask) OnGuestDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	deleteTask.OnFailed(ctx, guest, err)
}

func (deleteTask *BaseGuestDeleteTask) OnGuestDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.RevokeAllNetworkSecgroups(ctx, deleteTask.UserCred)
	guest.DetachAllNetworks(ctx, deleteTask.UserCred)
	guest.EjectAllIso(deleteTask.UserCred)
	guest.EjectAllVfd(deleteTask.UserCred)
	guest.DeleteEip(ctx, deleteTask.UserCred)
	drv, _ := guest.GetDriver()
	if drv != nil {
		drv.OnDeleteGuestFinalCleanup(ctx, guest, deleteTask.UserCred)
	}
	deleteTask.DeleteGuest(ctx, guest)
}

func (deleteTask *BaseGuestDeleteTask) DeleteGuest(ctx context.Context, guest *models.SGuest) {
	data := jsonutils.NewDict()
	data.Set("real_delete", jsonutils.JSONTrue)
	deleteTask.SetStageComplete(ctx, data)
}

func (deleteTask *BaseGuestDeleteTask) NotifyServerDeleted(ctx context.Context, guest *models.SGuest) {
	guest.EventNotify(ctx, deleteTask.UserCred, notifyclient.ActionPendingDelete)
}
