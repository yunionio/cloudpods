package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/utils"
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
	if guest.Hypervisor == models.HYPERVISOR_BAREMETAL && host.HostType != models.HOST_TYPE_BAREMETAL {
		self.OnGuestStopComplete(ctx, obj, data)
		return
	}
	self.SetStage("on_guest_stop_complete", nil)
	err := guest.GetDriver().RequestStopGuestForDelete(ctx, guest, self)
	if err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		// errMsg := jsonutils.NewString(err.Error())
		// self.OnGuestStopCompleteFailed(ctx, obj, errMsg)
		self.OnGuestStopComplete(ctx, obj, data)
	}
}

func (self *GuestDeleteTask) OnGuestStopComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	eip, _ := guest.GetEip()
	if eip != nil && eip.Mode != models.EIP_MODE_INSTANCE_PUBLICIP {
		// detach floating EIP only
		self.SetStage("on_eip_dissociate_complete", nil)
		eip.StartEipDissociateTask(ctx, self.UserCred, self.GetTaskId())
	} else {
		self.OnEipDissociateComplete(ctx, obj, nil)
	}
}

func (self *GuestDeleteTask) OnEipDissociateCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.OnFailed(ctx, guest, err)
}

func (self *GuestDeleteTask) OnEipDissociateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	if options.Options.EnablePendingDelete && !guest.PendingDeleted &&
		!jsonutils.QueryBoolean(self.Params, "purge", false) &&
		!jsonutils.QueryBoolean(self.Params, "override_pending_delete", false) {
		log.Debugf("XXXXXXX Do guest pending delete... XXXXXXX")
		guestStatus, _ := self.Params.GetString("guest_status")
		if !utils.IsInStringArray(guestStatus, []string{models.VM_SCHEDULE_FAILED, models.VM_NETWORK_FAILED, models.VM_DISK_FAILED,
			models.VM_CREATE_FAILED, models.VM_DEVICE_FAILED}) {
			self.StartPendingDeleteGuest(ctx, guest)
			return
		}
	}
	log.Debugf("XXXXXXX Do real delete on guest ... XXXXXXX")
	self.doStartDeleteGuest(ctx, guest)
}

func (self *GuestDeleteTask) OnGuestStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	self.OnGuestStopComplete(ctx, obj, err) // ignore stop error
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
	models.IsolatedDeviceManager.ReleaseDevicesOfGuest(guest, self.UserCred)
	self.SetStage("on_pending_delete_complete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDeleteTask) OnPendingDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsSystem {
		self.NotifyServerDeleted(ctx, guest)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteTask) StartDeleteGuest(ctx context.Context, guest *models.SGuest) {
	// No snapshot
	self.SetStage("on_guest_detach_disks_complete", nil)
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
	models.IsolatedDeviceManager.ReleaseDevicesOfGuest(guest, self.UserCred)
	host := guest.GetHost()
	if (host == nil || !host.Enabled) && jsonutils.QueryBoolean(self.Params, "purge", false) {
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
	guest.LeaveAllGroups(self.UserCred)
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
