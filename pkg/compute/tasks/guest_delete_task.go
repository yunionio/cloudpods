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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDeleteTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDeleteTask{})
}

func (self *GuestDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := guest.GetHost()
	if guest.Hypervisor == api.HYPERVISOR_BAREMETAL && host != nil && host.HostType != api.HOST_TYPE_BAREMETAL {
		// if a fake server for converted hypervisor, then just skip stop
		self.OnGuestStopComplete(ctx, guest, data)
		return
	}
	if len(guest.BackupHostId) > 0 {
		self.SetStage("OnMasterHostStopGuestComplete", nil)
		if err := guest.GetDriver().RequestStopGuestForDelete(ctx, guest, nil, self); err != nil {
			log.Errorf("RequestStopGuestForDelete fail %s", err)
			self.OnMasterHostStopGuestComplete(ctx, guest, nil)
		}
	} else {
		self.SetStage("OnGuestStopComplete", nil)
		if err := guest.GetDriver().RequestStopGuestForDelete(ctx, guest, nil, self); err != nil {
			log.Errorf("RequestStopGuestForDelete fail %s", err)
			self.OnGuestStopComplete(ctx, guest, nil)
		}
	}
}

func (self *GuestDeleteTask) OnMasterHostStopGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnGuestStopComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	err := guest.GetDriver().RequestStopGuestForDelete(ctx, guest, host, self)
	if err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		self.OnGuestStopComplete(ctx, guest, nil)
	}
}

func (self *GuestDeleteTask) OnMasterHostStopGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnGuestStopComplete(ctx, guest, nil) // ignore stop error
}

func (self *GuestDeleteTask) StartDeleteGuestSnapshots(ctx context.Context, guest *models.SGuest) {
	guest.StartDeleteGuestSnapshots(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDeleteTask) OnGuestStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(self.Params, "delete_snapshots", false) {
		self.SetStage("OnStartEipDissociate", nil)
		guest.StartDeleteGuestSnapshots(ctx, self.UserCred, self.Id)
		return
	}
	self.OnStartEipDissociate(ctx, guest, data)
}

func (self *GuestDeleteTask) OnGuestStopCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	self.OnGuestStopComplete(ctx, guest, err) // ignore stop error
}

func (self *GuestDeleteTask) OnStartEipDissociateFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorf("Delete guest snapshots faield: %s", data)
	self.OnStartEipDissociate(ctx, guest, nil)
}

func (self *GuestDeleteTask) OnStartEipDissociate(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	eip, _ := guest.GetEip()
	if eip != nil && eip.Mode != api.EIP_MODE_INSTANCE_PUBLICIP {
		// detach floating EIP only
		if jsonutils.QueryBoolean(self.Params, "purge", false) {
			// purge locally
			eip.Dissociate(ctx, self.UserCred)
			self.OnEipDissociateComplete(ctx, guest, nil)
		} else {
			self.SetStage("OnEipDissociateComplete", nil)
			eip.StartEipDissociateTask(ctx, self.UserCred, false, self.GetTaskId())
		}
	} else {
		self.OnEipDissociateComplete(ctx, guest, nil)
	}
}

func (self *GuestDeleteTask) OnEipDissociateCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
}

func (self *GuestDeleteTask) OnEipDissociateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("OnDiskDetachComplete", nil)
	self.OnDiskDetachComplete(ctx, obj, data)
}

// remove detachable disks
func (self *GuestDeleteTask) OnDiskDetachComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Debugf("OnDiskDetachComplete")
	guest := obj.(*models.SGuest)

	guestdisks := guest.GetDisks()
	if len(guestdisks) == 0 {
		// on guest disks detached
		self.doClearSecurityGroupComplete(ctx, guest)
		return
	}
	// detach last detachable disk
	lastDisk := guestdisks[len(guestdisks)-1].GetDisk()
	log.Debugf("lastDisk IsDetachable?? %v", lastDisk.IsDetachable())
	if !lastDisk.IsDetachable() {
		// no more disk need detach
		self.doClearSecurityGroupComplete(ctx, guest)
		return
	}
	purge := jsonutils.QueryBoolean(self.Params, "purge", false)
	guest.StartGuestDetachdiskTask(ctx, self.UserCred, lastDisk, true, self.GetTaskId(), purge)
}

func (self *GuestDeleteTask) OnDiskDetachCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
}

// revoke all secgroups
func (self *GuestDeleteTask) doClearSecurityGroupComplete(ctx context.Context, guest *models.SGuest) {
	log.Debugf("doClearSecurityGroupComplete")
	models.IsolatedDeviceManager.ReleaseDevicesOfGuest(ctx, guest, self.UserCred)
	guest.RevokeAllSecgroups(ctx, self.UserCred)
	// sync revoked secgroups to remote cloud
	self.SetStage("OnSyncConfigComplete", nil)
	guest.StartSyncTaskWithoutSyncstatus(ctx, self.UserCred, false, self.GetTaskId())
}

func (self *GuestDeleteTask) OnSyncConfigComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	overridePendingDelete := jsonutils.QueryBoolean(self.Params, "override_pending_delete", false)

	if options.Options.EnablePendingDelete && !isPurge && !overridePendingDelete {
		if guest.PendingDeleted {
			self.SetStageComplete(ctx, nil)
			return
		}
		log.Debugf("XXXXXXX Do guest pending delete... XXXXXXX")
		guestStatus, _ := self.Params.GetString("guest_status")
		if !utils.IsInStringArray(guestStatus, []string{
			api.VM_SCHEDULE_FAILED, api.VM_NETWORK_FAILED, api.VM_DISK_FAILED,
			api.VM_CREATE_FAILED, api.VM_DEVICE_FAILED}) {
			self.StartPendingDeleteGuest(ctx, guest)
			return
		}
	}
	log.Debugf("XXXXXXX Do real delete on guest ... XXXXXXX")
	self.doStartDeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) OnSyncConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	// guest := obj.(*models.SGuest)
	// self.OnFailed(ctx, guest, err)
	self.OnSyncConfigComplete(ctx, obj, err) // ignore sync config failed error
}

func (self *GuestDeleteTask) OnGuestDeleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
}

func (self *GuestDeleteTask) doStartDeleteGuest(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_DELETING, "delete server after stop")
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATING, nil, self.UserCred)
	self.StartDeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) StartPendingDeleteGuest(ctx context.Context, guest *models.SGuest) {
	guest.DoPendingDelete(ctx, self.UserCred)
	self.SetStage("OnPendingDeleteComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDeleteTask) OnPendingDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsSystem {
		self.NotifyServerDeleted(ctx, guest)
	}
	// self.SetStage("on_sync_guest_conf_complete", nil)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_PENDING_DELETE, nil, self.UserCred, true)
	// guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteTask) OnPendingDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnPendingDeleteComplete(ctx, obj, nil)
}

func (self *GuestDeleteTask) StartDeleteGuest(ctx context.Context, guest *models.SGuest) {
	// No snapshot
	self.SetStage("OnGuestDetachDisksComplete", nil)
	guest.GetDriver().RequestDetachDisksFromGuestForDelete(ctx, guest, self)
}

func (self *GuestDeleteTask) OnGuestDetachDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.DoDeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) OnGuestDetachDisksCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnGuestDeleteFailed(ctx, obj, data)
}

func (self *GuestDeleteTask) DoDeleteGuest(ctx context.Context, guest *models.SGuest) {
	models.IsolatedDeviceManager.ReleaseDevicesOfGuest(ctx, guest, self.UserCred)
	host := guest.GetHost()
	if guest.IsPrepaidRecycle() {
		err := host.BorrowIpAddrsFromGuest(ctx, self.UserCred, guest)
		if err != nil {
			msg := fmt.Sprintf("host.BorrowIpAddrsFromGuest fail %s", err)
			log.Errorf(msg)
			self.OnGuestDeleteFailed(ctx, guest, jsonutils.NewString(msg))
			return
		}
		self.OnGuestDeleteComplete(ctx, guest, nil)
	} else if (host == nil || !host.Enabled) && jsonutils.QueryBoolean(self.Params, "purge", false) {
		self.OnGuestDeleteComplete(ctx, guest, nil)
	} else {
		self.SetStage("OnGuestDeleteComplete", nil)
		guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetTaskId(), "")
	}
}

func (self *GuestDeleteTask) OnFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_DELETE_FAIL, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestDeleteTask) OnGuestDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
}

func (self *GuestDeleteTask) OnGuestDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.LeaveAllGroups(ctx, self.UserCred)
	guest.DetachAllNetworks(ctx, self.UserCred)
	guest.EjectIso(self.UserCred)
	guest.DeleteEip(ctx, self.UserCred)
	guest.GetDriver().OnDeleteGuestFinalCleanup(ctx, guest, self.UserCred)
	self.DeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) DeleteGuest(ctx context.Context, guest *models.SGuest) {
	isPendingDeleted := guest.PendingDeleted
	guest.RealDelete(ctx, self.UserCred)
	// guest.RemoveAllMetadata(ctx, self.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATE, nil, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	if !guest.IsSystem && !isPendingDeleted {
		self.NotifyServerDeleted(ctx, guest)
	}
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteTask) NotifyServerDeleted(ctx context.Context, guest *models.SGuest) {
	guest.NotifyServerEvent(
		self.UserCred,
		notifyclient.SERVER_DELETED,
		notify.NotifyPriorityImportant,
		false, nil, false,
	)
	guest.NotifyAdminServerEvent(ctx, notifyclient.SERVER_DELETED_ADMIN, notify.NotifyPriorityImportant)
}
