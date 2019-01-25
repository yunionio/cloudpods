package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
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
	if guest.Hypervisor == models.HYPERVISOR_BAREMETAL && host != nil && host.HostType != models.HOST_TYPE_BAREMETAL {
		// if a fake server for converted hypervisor, then just skip stop
		self.OnGuestStopComplete(ctx, obj, data)
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

func (self *GuestDeleteTask) OnGuestStopComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	eip, _ := guest.GetEip()
	if eip != nil && eip.Mode != models.EIP_MODE_INSTANCE_PUBLICIP {
		// detach floating EIP only
		if jsonutils.QueryBoolean(self.Params, "purge", false) {
			// purge locally
			eip.Dissociate(ctx, self.UserCred)
			self.OnEipDissociateComplete(ctx, guest, nil)
		} else {
			self.SetStage("on_eip_dissociate_complete", nil)
			eip.StartEipDissociateTask(ctx, self.UserCred, false, self.GetTaskId())
		}
	} else {
		self.OnEipDissociateComplete(ctx, obj, nil)
	}
}

func (self *GuestDeleteTask) OnGuestStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	self.OnGuestStopComplete(ctx, obj, err) // ignore stop error
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
		self.doClearSecurityGroupComplete(ctx, guest)
		return
	}
	lastDisk := guestdisks[len(guestdisks)-1].GetDisk() // remove last detachable disk
	log.Debugf("lastDisk IsDetachable?? %v", lastDisk.IsDetachable())
	if !lastDisk.IsDetachable() {
		self.doClearSecurityGroupComplete(ctx, guest)
		return
	}
	guest.StartGuestDetachdiskTask(ctx, self.UserCred, lastDisk, true, self.GetTaskId())
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
	guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
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
			models.VM_SCHEDULE_FAILED, models.VM_NETWORK_FAILED, models.VM_DISK_FAILED,
			models.VM_CREATE_FAILED, models.VM_DEVICE_FAILED}) {
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
	guest.SetStatus(self.UserCred, models.VM_DELETING, "delete server after stop")
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATING, nil, self.UserCred)
	self.StartDeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) StartPendingDeleteGuest(ctx context.Context, guest *models.SGuest) {
	guest.DoPendingDelete(ctx, self.UserCred)
	self.SetStage("on_pending_delete_complete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDeleteTask) OnPendingDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsSystem {
		self.NotifyServerDeleted(ctx, guest)
	}
	self.SetStage("on_sync_guest_conf_complete", nil)
	guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
}

func (self *GuestDeleteTask) OnSyncGuestConfComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteTask) OnSyncGuestConfCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
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
		self.SetStage("on_guest_delete_complete", nil)
		guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetTaskId(), "")
	}
}

func (self *GuestDeleteTask) OnFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, models.VM_DELETE_FAIL, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_DELETE, err, self.UserCred, false)
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
	guest.RealDelete(ctx, self.UserCred)
	guest.RemoveAllMetadata(ctx, self.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_DELOCATE, nil, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_DELETE, nil, self.UserCred, true)
	if !guest.IsSystem && !guest.PendingDeleted {
		self.NotifyServerDeleted(ctx, guest)
	}
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteTask) NotifyServerDeleted(ctx context.Context, guest *models.SGuest) {
	guest.NotifyServerEvent(notifyclient.SERVER_DELETED, notifyclient.PRIORITY_IMPORTANT, false)
	guest.NotifyAdminServerEvent(ctx, notifyclient.SERVER_DELETED_ADMIN, notifyclient.PRIORITY_IMPORTANT)
}
