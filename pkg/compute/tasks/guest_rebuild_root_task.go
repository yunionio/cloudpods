package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/osprofile"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestRebuildRootTask{})
	taskman.RegisterTask(KVMGuestRebuildRootTask{})
	taskman.RegisterTask(ManagedGuestRebuildRootTask{})
}

type GuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *GuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if jsonutils.QueryBoolean(self.Params, "need_stop", false) {
		self.SetStage("OnStopServerComplete", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, false, self.GetTaskId())
	} else {
		self.StartRebuildRootDisk(ctx, guest)
	}
}

func (self *GuestRebuildRootTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *GuestRebuildRootTask) StartRebuildRootDisk(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILDING_ROOT, nil, self.UserCred)
	gds := guest.CategorizeDisks()
	imageId, _ := self.Params.GetString("image_id")
	oldStatus := gds.Root.Status
	_, err := gds.Root.GetModelManager().TableSpec().Update(gds.Root, func() error {
		gds.Root.TemplateId = imageId
		gds.Root.Status = models.DISK_REBUILD
		return nil
	})
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	} else {
		db.OpsLog.LogEvent(gds.Root, db.ACT_UPDATE_STATUS,
			fmt.Sprintf("%s=>%s", oldStatus, models.DISK_REBUILD), self.UserCred)
	}

	self.SetStage("OnRebuildRootDiskComplete", nil)
	guest.SetStatus(self.UserCred, models.VM_REBUILD_ROOT, "")

	// clear logininfo
	loginParams := make(map[string]interface{})
	loginParams["login_account"] = "none"
	loginParams["login_key"] = "none"
	loginParams["login_key_timestamp"] = "none"
	guest.SetAllMetadata(ctx, loginParams, self.UserCred)

	guest.GetDriver().RequestRebuildRootDisk(ctx, guest, self)
}

func (self *GuestRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	imgId, _ := self.Params.GetString("image_id")
	imginfo, err := models.CachedimageManager.GetImageById(ctx, self.UserCred, imgId, false)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	}
	osprof, err := osprofile.GetOSProfileFromImageProperties(imginfo.Properties, guest.Hypervisor)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	}
	err = guest.SetMetadata(ctx, "__os_profile__", osprof, self.UserCred)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	}
	if guest.OsType != osprof.OSType {
		_, err := guest.GetModelManager().TableSpec().Update(guest, func() error {
			guest.OsType = osprof.OSType
			return nil
		})
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
			return
		}
	}
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT, "", self.UserCred)
	guest.NotifyServerEvent(notifyclient.SERVER_REBUILD_ROOT, notifyclient.PRIORITY_IMPORTANT, true)
	self.SetStage("OnSyncStatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT_FAIL, data, self.UserCred)
	guest.SetStatus(self.UserCred, models.VM_REBUILD_ROOT_FAIL, "")
	logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, data, self.UserCred, false)
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestRebuildRootTask) OnSyncStatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if guest.Status == models.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartComplete", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
	logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, "", self.UserCred, true)
}

func (self *GuestRebuildRootTask) OnGuestStartComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

/* -------------------------------------------------- */
/* ------------ KVMGuestRebuildRootTask ------------- */
/* -------------------------------------------------- */

type KVMGuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *KVMGuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	gds := guest.CategorizeDisks()
	self.SetStage("OnRebuildRootDiskComplete", nil)
	gds.Root.StartDiskCreateTask(ctx, self.UserCred, true, "", self.GetTaskId())
}

func (self *KVMGuestRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnGuestDeployComplete", nil)
	guest.SetStatus(self.UserCred, models.VM_DEPLOYING, "")
	// params := jsonutils.NewDict()
	// params.Set("reset_password", jsonutils.JSONTrue)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.GetParams(), "deploy", self.GetTaskId())
}

func (self *KVMGuestRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStageFailed(ctx, data.String())
	logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, data, self.UserCred, false)
}

func (self *KVMGuestRebuildRootTask) OnGuestDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStageComplete(ctx, nil)
	logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, nil, self.UserCred, true)
}

type ManagedGuestRebuildRootTask struct {
	SGuestBaseTask
}

func (self *ManagedGuestRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	diskCat := guest.CategorizeDisks()
	imageId := diskCat.Root.GetTemplateId()
	storage := diskCat.Root.GetStorage()
	cache := storage.GetStoragecache()

	self.SetStage("OnHostCacheImageComplete", nil)
	cache.StartImageCacheTask(ctx, self.UserCred, imageId, false, self.GetTaskId())
}

func (self *ManagedGuestRebuildRootTask) OnHostCacheImageComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("OnGuestDeployComplete", nil)
	guest.SetStatus(self.UserCred, models.VM_DEPLOYING, "rebuild deploy")
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "rebuild", self.GetTaskId())
}

func (self *ManagedGuestRebuildRootTask) OnHostCacheImageCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStageFailed(ctx, data.String())
	logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, data, self.UserCred, false)
}

func (self *ManagedGuestRebuildRootTask) OnGuestDeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
