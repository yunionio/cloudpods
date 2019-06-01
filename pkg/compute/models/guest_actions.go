package models

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (self *SGuest) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "vnc")
}

func (self *SGuest) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_BLOCK_STREAM}) {
		host := self.GetHost()
		if host == nil {
			return nil, httperrors.NewInternalServerError("Host missing")
		}
		retval, err := self.GetDriver().GetGuestVncInfo(ctx, userCred, self, host)
		if err != nil {
			return nil, err
		}
		retval.Add(jsonutils.NewString(self.Id), "id")
		return retval, nil
	} else {
		return jsonutils.NewDict(), nil
	}
}

func (self *SGuest) AllowPerformMonitor(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "monitor")
}

func (self *SGuest) PerformMonitor(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_BLOCK_STREAM}) {
		cmd, err := data.GetString("command")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("command")
		}
		return self.SendMonitorCommand(ctx, userCred, cmd)
	}
	return nil, httperrors.NewInvalidStatusError("Cannot send command in status %s", self.Status)
}

func (self *SGuest) AllowGetDetailsDesc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "desc")
}

func (self *SGuest) GetDetailsDesc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("No host for server")
	}
	desc := self.GetDriver().GetJsonDescAtHost(ctx, userCred, self, host)
	return desc, nil
}

func (self *SGuest) AllowPerformSaveImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "save-image")
}

func (self *SGuest) PerformSaveImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY}) {
		return nil, httperrors.NewInputParameterError("Cannot save image in status %s", self.Status)
	} else if !data.Contains("name") {
		return nil, httperrors.NewInputParameterError("Image name is required")
	} else if disks := self.CategorizeDisks(); disks.Root == nil {
		return nil, httperrors.NewInputParameterError("No root image")
	} else {
		kwargs := data.(*jsonutils.JSONDict)
		restart := (self.Status == VM_RUNNING) || jsonutils.QueryBoolean(data, "auto_start", false)
		properties := jsonutils.NewDict()
		if notes, err := data.GetString("notes"); err != nil && len(notes) > 0 {
			properties.Add(jsonutils.NewString(notes), "notes")
		}
		osType := self.OsType
		if len(osType) == 0 {
			osType = "Linux"
		}
		properties.Add(jsonutils.NewString(osType), "os_type")
		kwargs.Add(properties, "properties")
		kwargs.Add(jsonutils.NewBool(restart), "restart")

		lockman.LockObject(ctx, disks.Root)
		defer lockman.ReleaseObject(ctx, disks.Root)

		if imageId, err := disks.Root.PrepareSaveImage(ctx, userCred, kwargs); err != nil {
			return nil, err
		} else {
			kwargs.Add(jsonutils.NewString(imageId), "image_id")
		}
		return nil, self.StartGuestSaveImage(ctx, userCred, kwargs, "")
	}
}

func (self *SGuest) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	return self.GetDriver().StartGuestSaveImage(ctx, userCred, self, data, parentTaskId)
}

func (self *SGuest) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync")

}

func (self *SGuest) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING}) {
		return nil, httperrors.NewResourceBusyError("Cannot sync in status %s", self.Status)
	}
	if err := self.StartSyncTask(ctx, userCred, false, ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) GetQemuVersion(userCred mcclient.TokenCredential) string {
	return self.GetMetadata("__qemu_version", userCred)
}

// if qemuVer >= compareVer return true
func (self *SGuest) CheckQemuVersion(qemuVer, compareVer string) bool {
	if len(qemuVer) == 0 {
		return false
	}

	compareVersion := strings.Split(compareVer, ".")
	guestVersion := strings.Split(qemuVer, ".")
	var i = 0
	for ; i < len(guestVersion); i++ {
		if i >= len(compareVersion) {
			return true
		}
		v, _ := strconv.ParseInt(guestVersion[i], 10, 0)
		compareV, _ := strconv.ParseInt(compareVersion[i], 10, 0)
		if v < compareV {
			return false
		} else if v > compareV {
			return true
		}
	}
	if i < len(compareVersion)-1 {
		return false
	}
	return true
}

func (self *SGuest) AllowPerformMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "migrate")
}

func (self *SGuest) PerformMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.GetHypervisor() != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
	}
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	isRescueMode := jsonutils.QueryBoolean(data, "rescue_mode", false)
	if !isRescueMode && self.Status != VM_READY {
		return nil, httperrors.NewServerStatusError("Cannot normal migrate guest in status %s, try rescue mode or server-live-migrate?", self.Status)
	}
	if isRescueMode {
		guestDisks := self.GetDisks()
		for _, guestDisk := range guestDisks {
			if utils.IsInStringArray(
				guestDisk.GetDisk().GetStorage().StorageType, STORAGE_LOCAL_TYPES) {
				return nil, httperrors.NewBadRequestError("Rescue mode requires all disk store in shared storages")
			}
		}
	}
	devices := self.GetIsolatedDevices()
	if devices != nil && len(devices) > 0 {
		return nil, httperrors.NewBadRequestError("Cannot migrate with isolated devices")
	}
	var preferHostId string
	preferHost, _ := data.GetString("prefer_host")
	if len(preferHost) > 0 {
		if !db.IsAdminAllowPerform(userCred, self, "assign-host") {
			return nil, httperrors.NewBadRequestError("Only system admin can assign host")
		}
		iHost, _ := HostManager.FetchByIdOrName(userCred, preferHost)
		if iHost == nil {
			return nil, httperrors.NewBadRequestError("Host %s not found", preferHost)
		}
		host := iHost.(*SHost)
		preferHostId = host.Id
	}
	err := self.StartMigrateTask(ctx, userCred, isRescueMode, self.Status, preferHostId, "")
	return nil, err
}

func (self *SGuest) StartMigrateTask(ctx context.Context, userCred mcclient.TokenCredential, isRescueMode bool, guestStatus, preferHostId, parentTaskId string) error {
	data := jsonutils.NewDict()
	if isRescueMode {
		data.Set("is_rescue_mode", jsonutils.JSONTrue)
	}
	if len(preferHostId) > 0 {
		data.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	data.Set("guest_status", jsonutils.NewString(guestStatus))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestMigrateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformLiveMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "live-migrate")
}

func (self *SGuest) PerformLiveMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.GetHypervisor() != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
	}
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_SUSPEND}) {
		cdrom := self.getCdrom()
		if cdrom != nil && len(cdrom.ImageId) > 0 {
			return nil, httperrors.NewBadRequestError("Cannot migrate with cdrom")
		}
		devices := self.GetIsolatedDevices()
		if devices != nil && len(devices) > 0 {
			return nil, httperrors.NewBadRequestError("Cannot migrate with isolated devices")
		}
		if !self.CheckQemuVersion(self.GetQemuVersion(userCred), "1.1.2") {
			return nil, httperrors.NewBadRequestError("Cannot do live migrate, too low qemu version")
		}
		var preferHostId string
		preferHost, _ := data.GetString("prefer_host")
		if len(preferHost) > 0 {
			if !db.IsAdminAllowPerform(userCred, self, "assign-host") {
				return nil, httperrors.NewBadRequestError("Only system admin can assign host")
			}
			iHost, _ := HostManager.FetchByIdOrName(userCred, preferHost)
			if iHost == nil {
				return nil, httperrors.NewBadRequestError("Host %s not found", preferHost)
			}
			host := iHost.(*SHost)
			preferHostId = host.Id
		}
		err := self.StartGuestLiveMigrateTask(ctx, userCred, self.Status, preferHostId, "")
		return nil, err
	}
	return nil, httperrors.NewBadRequestError("Cannot live migrate in status %s", self.Status)
}

func (self *SGuest) StartGuestLiveMigrateTask(ctx context.Context, userCred mcclient.TokenCredential, guestStatus, preferHostId, parentTaskId string) error {
	self.SetStatus(userCred, VM_START_MIGRATE, "")
	data := jsonutils.NewDict()
	if len(preferHostId) > 0 {
		data.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	data.Set("guest_status", jsonutils.NewString(guestStatus))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestLiveMigrateTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "deploy")
}

func (self *SGuest) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	kwargs, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Parse query body error")
	}

	if kwargs.Contains("__delete_keypair__") || kwargs.Contains("keypair") {
		var kpId string

		if kwargs.Contains("keypair") {
			keypair, _ := kwargs.GetString("keypair")
			iKp, err := KeypairManager.FetchByIdOrName(userCred, keypair)
			if err != nil {
				return nil, err
			}
			if iKp == nil {
				return nil, fmt.Errorf("Fetch keypair error")
			}
			kp := iKp.(*SKeypair)
			kpId = kp.Id
		}

		if self.KeypairId != kpId {
			okey := self.getKeypair()
			if okey != nil {
				kwargs.Set("delete_public_key", jsonutils.NewString(okey.PublicKey))
			}

			diff, err := db.Update(self, func() error {
				self.KeypairId = kpId
				return nil
			})
			if err != nil {
				log.Errorf("update keypair fail: %s", err)
				return nil, httperrors.NewInternalServerError(err.Error())
			}

			db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)

			kwargs.Set("reset_password", jsonutils.JSONTrue)
		}
	}

	var resetPasswd bool
	passwdStr, _ := kwargs.GetString("password")
	if len(passwdStr) > 0 {
		if !seclib2.MeetComplxity(passwdStr) {
			return nil, httperrors.NewWeakPasswordError()
		}
		resetPasswd = true
	} else {
		resetPasswd = jsonutils.QueryBoolean(kwargs, "reset_password", false)
	}
	if resetPasswd {
		kwargs.Set("reset_password", jsonutils.JSONTrue)
	} else {
		kwargs.Set("reset_password", jsonutils.JSONFalse)
	}

	// 变更密码/密钥时需要Restart才能生效。更新普通字段不需要Restart, Azure需要在运行状态下操作
	doRestart := false
	if resetPasswd {
		doRestart = self.GetDriver().IsNeedRestartForResetLoginInfo()
	}

	deployStatus, err := self.GetDriver().GetDeployStatus()
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	if utils.IsInStringArray(self.Status, deployStatus) {
		if (doRestart && self.Status == VM_RUNNING) || (self.Status != VM_RUNNING && (jsonutils.QueryBoolean(kwargs, "auto_start", false) || jsonutils.QueryBoolean(kwargs, "restart", false))) {
			kwargs.Set("restart", jsonutils.JSONTrue)
		} else {
			// 避免前端直接传restart参数, 越过校验
			kwargs.Set("restart", jsonutils.JSONFalse)
		}
		err := self.StartGuestDeployTask(ctx, userCred, kwargs, "deploy", "")
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, httperrors.NewServerStatusError("Cannot deploy in status %s", self.Status)
}

func (self *SGuest) AllowPerformAttachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "attachdisk")
}

func (self *SGuest) ValidateAttachDisk(ctx context.Context, disk *SDisk) error {
	storage := disk.GetStorage()
	if provider := storage.GetCloudprovider(); provider != nil {
		host := self.GetHost()
		if provider.Id != host.ManagerId {
			return httperrors.NewInputParameterError("Disk %s and guest not belong to the same account", disk.Name)
		}
		if storage.ZoneId != host.ZoneId {
			return httperrors.NewInputParameterError("Disk %s and guest not belong to the same zone", disk.Name)
		}
	}

	if disk.isAttached() {
		return httperrors.NewInputParameterError("Disk %s has been attached", disk.Name)
	}
	if len(disk.GetPathAtHost(self.GetHost())) == 0 {
		return httperrors.NewInputParameterError("Disk %s not belong the guest's host", disk.Name)
	}
	if disk.Status != DISK_READY {
		return httperrors.NewInputParameterError("Disk in %s not able to attach", disk.Status)
	}
	guestStatus, err := self.GetDriver().GetAttachDiskStatus()
	if err != nil {
		return err
	}
	if !utils.IsInStringArray(self.Status, guestStatus) {
		return httperrors.NewInputParameterError("Guest %s not support attach disk in status %s", self.Name, self.Status)
	}
	return nil
}

func (self *SGuest) PerformAttachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskId, err := data.GetString("disk_id")
	if err != nil {
		log.Errorln(err)
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	disk, err := DiskManager.FetchByIdOrName(userCred, diskId)
	if err != nil && err != sql.ErrNoRows {
		log.Errorln(err)
		return nil, err
	}
	if disk == nil {
		return nil, httperrors.NewResourceNotFoundError("Disk %s not found", diskId)
	}
	if err := self.ValidateAttachDisk(ctx, disk.(*SDisk)); err != nil {
		return nil, err
	}

	taskData := data.(*jsonutils.JSONDict)
	taskData.Set("disk_id", jsonutils.NewString(disk.GetId()))

	if err := self.GetDriver().StartGuestAttachDiskTask(ctx, userCred, self, taskData, ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, fwOnly bool, parentTaskId string) error {

	data := jsonutils.NewDict()
	if fwOnly {
		data.Add(jsonutils.JSONTrue, "fw_only")
	} else if err := self.SetStatus(userCred, VM_SYNC_CONFIG, ""); err != nil {
		log.Errorln(err)
		return err
	}
	return self.doSyncTask(ctx, data, userCred, parentTaskId)
}

func (self *SGuest) StartSyncTaskWithoutSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, fwOnly bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("without_sync_status", jsonutils.JSONTrue)
	data.Set("fw_only", jsonutils.NewBool(fwOnly))
	return self.doSyncTask(ctx, data, userCred, parentTaskId)
}

func (self *SGuest) doSyncTask(ctx context.Context, data *jsonutils.JSONDict, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSyncConfTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformSuspend(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "suspend")
}

func (self *SGuest) PerformSuspend(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == VM_RUNNING {
		err := self.StartSuspendTask(ctx, userCred, "")
		return nil, err
	}
	return nil, httperrors.NewInvalidStatusError("Cannot suspend VM in status %s", self.Status)
}

func (self *SGuest) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	err := self.SetStatus(userCred, VM_SUSPEND, "do suspend")
	if err != nil {
		return err
	}
	return self.GetDriver().StartSuspendTask(ctx, userCred, self, nil, parentTaskId)
}

func (self *SGuest) AllowPerformStart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "start")
}

func (self *SGuest) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{VM_READY, VM_START_FAILED, VM_SAVE_DISK_FAILED, VM_SUSPEND}) {
		if self.isAllDisksReady() {
			var kwargs *jsonutils.JSONDict
			if data != nil {
				kwargs = data.(*jsonutils.JSONDict)
			}
			err := self.GetDriver().PerformStart(ctx, userCred, self, kwargs)
			return nil, err
		} else {
			return nil, httperrors.NewInvalidStatusError("Some disk not ready")
		}
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do start server in status %s", self.Status)
	}
}

func (self *SGuest) StartGuestDeployTask(ctx context.Context, userCred mcclient.TokenCredential, kwargs *jsonutils.JSONDict, action string, parentTaskId string) error {
	self.SetStatus(userCred, VM_START_DEPLOY, "")
	if kwargs == nil {
		kwargs = jsonutils.NewDict()
	}
	kwargs.Add(jsonutils.NewString(action), "deploy_action")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDeployTask", self, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) NotifyServerEvent(userCred mcclient.TokenCredential, event string, priority notify.TNotifyPriority, loginInfo bool) {
	meta, err := self.GetAllMetadata(nil)
	if err != nil {
		return
	}
	kwargs := jsonutils.NewDict()
	kwargs.Add(jsonutils.NewString(self.Name), "name")
	kwargs.Add(jsonutils.NewString(self.Hypervisor), "hypervisor")
	if loginInfo {
		kwargs.Add(jsonutils.NewString(self.getNotifyIps()), "ips")
		osName := meta["os_name"]
		if osName == "Windows" {
			kwargs.Add(jsonutils.JSONTrue, "windows")
		}
		loginAccount := meta["login_account"]
		if len(loginAccount) > 0 {
			kwargs.Add(jsonutils.NewString(loginAccount), "account")
		}
		keypair := self.getKeypairName()
		if len(keypair) > 0 {
			kwargs.Add(jsonutils.NewString(keypair), "keypair")
		} else {
			loginKey := meta["login_key"]
			if len(loginKey) > 0 {
				passwd, err := utils.DescryptAESBase64(self.Id, loginKey)
				if err == nil {
					kwargs.Add(jsonutils.NewString(passwd), "password")
				}
			}
		}
	}
	notifyclient.Notify(userCred.GetUserId(), false, priority, event, kwargs)
}

func (self *SGuest) NotifyAdminServerEvent(ctx context.Context, event string, priority notify.TNotifyPriority) {
	kwargs := jsonutils.NewDict()
	kwargs.Add(jsonutils.NewString(self.Name), "name")
	kwargs.Add(jsonutils.NewString(self.Hypervisor), "hypervisor")
	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		kwargs.Add(jsonutils.NewString(tc.Name), "tenant")
	} else {
		kwargs.Add(jsonutils.NewString(self.ProjectId), "tenant")
	}
	notifyclient.SystemNotify(priority, event, kwargs)
}

func (self *SGuest) StartGuestStopTask(ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	if len(parentTaskId) == 0 {
		self.SetStatus(userCred, VM_START_STOP, "")
	}
	params := jsonutils.NewDict()
	if isForce {
		params.Add(jsonutils.JSONTrue, "is_force")
	}
	if len(parentTaskId) > 0 {
		params.Add(jsonutils.JSONTrue, "subtask")
	}
	return self.GetDriver().StartGuestStopTask(self, ctx, userCred, params, parentTaskId)
}

func (self *SGuest) insertIso(imageId string) bool {
	cdrom := self.getCdrom()
	return cdrom.insertIso(imageId)
}

func (self *SGuest) InsertIsoSucc(imageId string, path string, size int, name string) bool {
	cdrom := self.getCdrom()
	return cdrom.insertIsoSucc(imageId, path, size, name)
}

func (self *SGuest) GetDetailsIso(userCred mcclient.TokenCredential) jsonutils.JSONObject {
	cdrom := self.getCdrom()
	desc := jsonutils.NewDict()
	if len(cdrom.ImageId) > 0 {
		desc.Set("image_id", jsonutils.NewString(cdrom.ImageId))
		desc.Set("status", jsonutils.NewString("inserting"))
	}
	if len(cdrom.Path) > 0 {
		desc.Set("name", jsonutils.NewString(cdrom.Name))
		desc.Set("size", jsonutils.NewInt(int64(cdrom.Size)))
		desc.Set("status", jsonutils.NewString("ready"))
	}
	return desc
}

func (self *SGuest) AllowPerformInsertiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformInsertiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cdrom := self.getCdrom()
	if cdrom == nil {
		return nil, fmt.Errorf("Failed to get cdrom")
	}

	if len(cdrom.ImageId) > 0 {
		return nil, httperrors.NewBadRequestError("CD-ROM not empty, please eject first")
	}
	imageId, _ := data.GetString("image_id")
	image, err := parseIsoInfo(ctx, userCred, imageId)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_READY}) {
		err = self.StartInsertIsoTask(ctx, image.Id, self.HostId, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Insert ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformEjectiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformEjectiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	cdrom := self.getCdrom()
	if cdrom == nil {
		return nil, fmt.Errorf("Failed to get cdrom")
	}
	if len(cdrom.ImageId) == 0 {
		return nil, httperrors.NewBadRequestError("No ISO to eject")
	}
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_READY}) {
		err := self.StartEjectisoTask(ctx, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Eject ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) StartEjectisoTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestEjectISOTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartInsertIsoTask(ctx context.Context, imageId string, hostId string, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.insertIso(imageId)

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	data.Add(jsonutils.NewString(hostId), "host_id")

	task, err := taskman.TaskManager.NewTask(ctx, "GuestInsertIsoTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGueststartTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, VM_START_START, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStartTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGuestCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error {
	return self.GetDriver().StartGuestCreateTask(self, ctx, userCred, params, pendingUsage, parentTaskId)
}

func (self *SGuest) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return self.GetDriver().StartGuestSyncstatusTask(self, ctx, userCred, parentTaskId)
}

func (self *SGuest) StartAutoDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	db.OpsLog.LogEvent(self, db.ACT_DELETE, "auto-delete after stop", userCred)
	return self.StartDeleteGuestTask(ctx, userCred, parentTaskId, false, false)
}

func (self *SGuest) StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, isPurge bool, overridePendingDelete bool) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.Status), "guest_status")
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	self.SetStatus(userCred, VM_START_DELETE, "")
	return self.GetDriver().StartDeleteGuestTask(ctx, userCred, self, params, parentTaskId)
}

func (self *SGuest) AllowPerformAddSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformAddSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING, VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot assign security group for this guest %s", self.Name)
	}

	secgrpJsonArray := jsonutils.GetArrayOfPrefix(data, "secgrp")
	if len(secgrpJsonArray) == 0 {
		return nil, httperrors.NewInputParameterError("Missing secgrp.0 secgrp.1 ... parameters")
	}

	originSecgroups := self.GetSecgroups()
	if len(originSecgroups)+len(secgrpJsonArray) >= maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	originSecgroupIds := []string{}
	for _, secgroup := range originSecgroups {
		originSecgroupIds = append(originSecgroupIds, secgroup.Id)
	}

	newSecgroups := []*SSecurityGroup{}
	newSecgroupNames := []string{}
	for idx := 0; idx < len(secgrpJsonArray); idx++ {
		secgroupId, _ := secgrpJsonArray[idx].GetString()
		secgrp, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewInputParameterError("failed to find secgroup %s for params %s", secgroupId, fmt.Sprintf("secgrp.%d", idx))
			}
			return nil, httperrors.NewGeneralError(err)
		}

		if err := SecurityGroupManager.ValidateName(secgrp.GetName()); err != nil {
			return nil, httperrors.NewInputParameterError("The secgroup name %s does not meet the requirements, please change the name", secgrp.GetName())
		}

		if utils.IsInStringArray(secgrp.GetId(), originSecgroupIds) {
			return nil, httperrors.NewInputParameterError("security group %s has already been assigned to guest %s", secgrp.GetName(), self.Name)
		}
		newSecgroups = append(newSecgroups, secgrp.(*SSecurityGroup))
		newSecgroupNames = append(newSecgroupNames, secgrp.GetName())
	}

	for _, secgroup := range newSecgroups {
		if _, err := GuestsecgroupManager.newGuestSecgroup(ctx, userCred, self, secgroup); err != nil {
			return nil, httperrors.NewInputParameterError(err.Error())
		}
	}

	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_ASSIGNSECGROUP, fmt.Sprintf("secgroups: %s", strings.Join(newSecgroupNames, ",")), userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) AllowPerformAssignSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "assign-secgroup")
}

func (self *SGuest) AllowPerformRevokeSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "revoke-secgroup")
}

func (self *SGuest) saveDefaultSecgroupId(userCred mcclient.TokenCredential, secGrpId string) error {
	if secGrpId != self.SecgrpId {
		diff, err := db.Update(self, func() error {
			self.SecgrpId = secGrpId
			return nil
		})
		if err != nil {
			log.Errorf("saveDefaultSecgroupId fail %s", err)
			return err
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (self *SGuest) revokeSecgroup(ctx context.Context, userCred mcclient.TokenCredential, secgroup *SSecurityGroup) error {
	if secgroup == nil {
		return fmt.Errorf("failed to revoke null secgroup")
	}
	if self.SecgrpId != secgroup.Id {
		return GuestsecgroupManager.DeleteGuestSecgroup(ctx, userCred, self, secgroup)
	}
	secgroups := self.GetSecgroups()
	if len(secgroups) <= 1 {
		return self.saveDefaultSecgroupId(userCred, "default")
	}
	for _, _secgroup := range secgroups {
		// 从guestsecgroups中移除一个安全组，并将guest的 secgroupId 设为此安全组ID
		if _secgroup.Id != secgroup.Id {
			err := GuestsecgroupManager.DeleteGuestSecgroup(ctx, userCred, self, &_secgroup)
			if err != nil {
				return err
			}
			return self.saveDefaultSecgroupId(userCred, _secgroup.Id)
		}
	}
	return nil
}

func (self *SGuest) PerformRevokeSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING, VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot revoke security rules in status %s", self.Status)
	}

	revokeSecgroups := []*SSecurityGroup{}
	secgrpJsonArray := jsonutils.GetArrayOfPrefix(data, "secgrp")
	originSecgroups := self.GetSecgroups()
	originSecgroupIds := []string{}
	for _, originSecgroup := range originSecgroups {
		originSecgroupIds = append(originSecgroupIds, originSecgroup.Id)
	}

	if len(secgrpJsonArray) == 0 {
		revokeSecgroups = append(revokeSecgroups, self.getSecgroup())
	}

	revokeSecgroupNames := []string{}
	for idx := 0; idx < len(secgrpJsonArray); idx++ {
		secgroupId, _ := secgrpJsonArray[idx].GetString()
		secgrp, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewInputParameterError("failed to find secgroup %s for params %s", secgroupId, fmt.Sprintf("secgrp.%d", idx))
			}
			return nil, httperrors.NewGeneralError(err)
		}
		if !utils.IsInStringArray(secgrp.GetId(), originSecgroupIds) {
			return nil, httperrors.NewInputParameterError("security group %s not assigned to guest %s", secgrp.GetName(), self.Name)
		}
		revokeSecgroups = append(revokeSecgroups, secgrp.(*SSecurityGroup))
		revokeSecgroupNames = append(revokeSecgroupNames, secgrp.GetName())
	}

	for _, secgroup := range revokeSecgroups {
		if err := self.revokeSecgroup(ctx, userCred, secgroup); err != nil {
			return nil, err
		}
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_REVOKESECGROUP, fmt.Sprintf("secgroups: %s", strings.Join(revokeSecgroupNames, ",")), userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) PerformAssignSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING, VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}
	secgrpV := validators.NewModelIdOrNameValidator("secgrp", "secgroup", userCred.GetProjectId())
	if err := secgrpV.Validate(data.(*jsonutils.JSONDict)); err != nil {
		return nil, err
	}

	err := SecurityGroupManager.ValidateName(secgrpV.Model.GetName())
	if err != nil {
		return nil, httperrors.NewInputParameterError("The secgroup name %s does not meet the requirements, please change the name", secgrpV.Model.GetName())
	}

	err = self.saveDefaultSecgroupId(userCred, secgrpV.Model.GetId())
	if err != nil {
		return nil, err
	}

	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_ASSIGNSECGROUP, fmt.Sprintf("secgroup: %s", secgrpV.Model.GetName()), userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) AllowPerformSetSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "set-secgroup")
}

func (self *SGuest) PerformSetSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING, VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}
	secgrpJsonArray := jsonutils.GetArrayOfPrefix(data, "secgrp")
	if len(secgrpJsonArray) == 0 {
		return nil, httperrors.NewInputParameterError("Missing secgrp.0 secgrp.1 ... parameters")
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot assign security group for this guest %s", self.Name)
	}

	if len(secgrpJsonArray) > maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	setSecgroups := []*SSecurityGroup{}
	setSecgroupNames := []string{}
	for idx := 0; idx < len(secgrpJsonArray); idx++ {
		secgroupId, _ := secgrpJsonArray[idx].GetString()
		secgrp, err := SecurityGroupManager.FetchByIdOrName(userCred, secgroupId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewInputParameterError("failed to find secgroup %s for params %s", secgroupId, fmt.Sprintf("secgrp.%d", idx))
			}
			return nil, httperrors.NewGeneralError(err)
		}

		if err := SecurityGroupManager.ValidateName(secgrp.GetName()); err != nil {
			return nil, httperrors.NewInputParameterError("The secgroup name %s does not meet the requirements, please change the name", secgrp.GetName())
		}

		setSecgroups = append(setSecgroups, secgrp.(*SSecurityGroup))
		setSecgroupNames = append(setSecgroupNames, secgrp.GetName())
	}

	err := self.RevokeAllSecgroups(ctx, userCred)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(setSecgroups); i++ {
		if i == 0 {
			err = self.saveDefaultSecgroupId(userCred, setSecgroups[i].Id)
		} else {
			_, err = GuestsecgroupManager.newGuestSecgroup(ctx, userCred, self, setSecgroups[i])
		}
		if err != nil {
			return nil, httperrors.NewInputParameterError(err.Error())
		}
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_SETSECGROUP, fmt.Sprintf("secgroups: %s", strings.Join(setSecgroupNames, ",")), userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SGuest) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidatePurgeCondition(ctx)
	if err != nil {
		return nil, err
	}
	host := self.GetHost()
	if host != nil && host.Enabled {
		return nil, httperrors.NewInvalidStatusError("Cannot purge server on enabled host")
	}
	err = self.StartDeleteGuestTask(ctx, userCred, "", true, false)
	return nil, err
}

func (self *SGuest) setKeypairId(userCred mcclient.TokenCredential, keypairId string) error {
	diff, err := db.Update(self, func() error {
		self.KeypairId = keypairId
		return nil
	})
	if err != nil {
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return err
}

func (self *SGuest) AllowPerformRebuildRoot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "rebuild-root")
}

func (self *SGuest) PerformRebuildRoot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	imageId, _ := data.GetString("image_id")

	if len(imageId) > 0 {
		img, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
		if err != nil {
			return nil, httperrors.NewNotFoundError("failed to find %s", imageId)
		}
		osType, _ := img.Properties["os_type"]
		osName := self.GetMetadata("os_name", userCred)
		if len(osName) == 0 && len(osType) == 0 && strings.ToLower(osType) != strings.ToLower(osName) {
			return nil, httperrors.NewBadRequestError("Cannot switch OS between %s-%s", osName, osType)
		}
		imageId = img.Id
	}

	rebuildStatus, err := self.GetDriver().GetRebuildRootStatus()
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}

	if !self.GetDriver().IsRebuildRootSupportChangeImage() && len(imageId) > 0 {
		templateId := self.GetTemplateId()
		if len(templateId) == 0 {
			return nil, httperrors.NewBadRequestError("No template for root disk, cannot rebuild root")
		}
		if imageId != templateId {
			return nil, httperrors.NewInputParameterError("%s not support rebuild root with a different image", self.GetDriver().GetHypervisor())
		}
	}

	if !utils.IsInStringArray(self.Status, rebuildStatus) {
		return nil, httperrors.NewInvalidStatusError("Cannot reset root in status %s", self.Status)
	}

	autoStart := jsonutils.QueryBoolean(data, "auto_start", false)
	var needStop = false
	if self.Status == VM_RUNNING {
		needStop = true
	}
	resetPasswd := jsonutils.QueryBoolean(data, "reset_password", true)
	passwd, _ := data.GetString("password")
	if len(passwd) > 0 {
		if !seclib2.MeetComplxity(passwd) {
			return nil, httperrors.NewWeakPasswordError()
		}
	}

	keypairStr := jsonutils.GetAnyString(data, []string{"keypair", "keypair_id"})
	if len(keypairStr) > 0 {
		keypairObj, err := KeypairManager.FetchByIdOrName(userCred, keypairStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("keypair %s not found", keypairStr)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		if self.KeypairId != keypairObj.GetId() {
			err = self.setKeypairId(userCred, keypairObj.GetId())
			if err != nil {
				return nil, httperrors.NewGeneralError(err)
			}
		}
	}

	allDisks := jsonutils.QueryBoolean(data, "all_disks", false)

	return nil, self.StartRebuildRootTask(ctx, userCred, imageId, needStop, autoStart, passwd, resetPasswd, allDisks)
}

func (self *SGuest) GetTemplateId() string {
	gdc := self.CategorizeDisks()
	if gdc.Root != nil {
		return gdc.Root.GetTemplateId()
	}
	return ""
}

func (self *SGuest) StartRebuildRootTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, needStop, autoStart bool, passwd string, resetPasswd bool, allDisk bool) error {
	data := jsonutils.NewDict()
	if len(imageId) == 0 {
		imageId = self.GetTemplateId()
		if len(imageId) == 0 {
			return httperrors.NewBadRequestError("No template for root disk, cannot rebuild root")
		}
	}
	data.Set("image_id", jsonutils.NewString(imageId))
	if needStop {
		data.Set("need_stop", jsonutils.JSONTrue)
	}
	if autoStart {
		data.Set("auto_start", jsonutils.JSONTrue)
	}
	if resetPasswd {
		data.Set("reset_password", jsonutils.JSONTrue)
	} else {
		data.Set("reset_password", jsonutils.JSONFalse)
	}
	if len(passwd) > 0 {
		data.Set("password", jsonutils.NewString(passwd))
	}

	if allDisk {
		data.Set("all_disks", jsonutils.JSONTrue)
	} else {
		data.Set("all_disks", jsonutils.JSONFalse)
	}
	self.SetStatus(userCred, VM_REBUILD_ROOT, "request start rebuild root")
	if self.GetHypervisor() == HYPERVISOR_BAREMETAL {
		task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerRebuildRootTask", self, userCred, data, "", "", nil)
		if err != nil {
			return err
		}
		task.ScheduleRun(nil)
	} else {
		task, err := taskman.TaskManager.NewTask(ctx, "GuestRebuildRootTask", self, userCred, data, "", "", nil)
		if err != nil {
			return err
		}
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) DetachDisk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential) {
	guestdisk := self.GetGuestDisk(disk.Id)
	if guestdisk != nil {
		guestdisk.Detach(ctx, userCred)
	}
}

func (self *SGuest) AllowPerformCreatedisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "createdisk")
}

func (self *SGuest) PerformCreatedisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskSize := 0
	disksConf := jsonutils.NewDict()
	diskSizes := make(map[string]int, 0)

	diskDefArray := jsonutils.GetArrayOfPrefix(data, "disk")
	if len(diskDefArray) == 0 {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, "No Disk Info Provided", userCred, false)
		return nil, httperrors.NewBadRequestError("No Disk Info Provided")
	}

	for diskIdx := 0; diskIdx < len(diskDefArray); diskIdx += 1 {
		diskInfo, err := parseDiskInfo(ctx, userCred, diskDefArray[diskIdx])
		if err != nil {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
			return nil, httperrors.NewBadRequestError(err.Error())
		}
		if len(diskInfo.Backend) == 0 {
			diskInfo.Backend = self.getDefaultStorageType()
		}
		disksConf.Set(fmt.Sprintf("disk.%d", diskIdx), jsonutils.Marshal(diskInfo))
		if _, ok := diskSizes[diskInfo.Backend]; !ok {
			diskSizes[diskInfo.Backend] = diskInfo.SizeMb
		} else {
			diskSizes[diskInfo.Backend] += diskInfo.SizeMb
		}
		diskSize += diskInfo.SizeMb
	}
	host := self.GetHost()
	if host == nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, "No valid host", userCred, false)
		return nil, httperrors.NewBadRequestError("No valid host")
	}
	for backend, size := range diskSizes {
		storage := host.GetLeastUsedStorage(backend)
		if storage == nil {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, "No valid storage on current host", userCred, false)
			return nil, httperrors.NewBadRequestError("No valid storage on current host")
		}
		if storage.GetCapacity() > 0 && storage.GetCapacity() < size {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, "Not eough storage space on current host", userCred, false)
			return nil, httperrors.NewBadRequestError("Not eough storage space on current host")
		}
	}
	pendingUsage := &SQuota{
		Storage: diskSize,
	}
	err := QuotaManager.CheckSetPendingQuota(ctx, userCred, self.ProjectId, pendingUsage)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)

	err = self.CreateDisksOnHost(ctx, userCred, host, disksConf, pendingUsage, false, false)
	if err != nil {
		QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, nil, pendingUsage)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	err = self.StartGuestCreateDiskTask(ctx, userCred, disksConf, "")
	return nil, err
}

func (self *SGuest) StartGuestCreateDiskTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateDiskTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformDetachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "detachdisk")
}

func (self *SGuest) PerformDetachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskId, err := data.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	keepDisk := jsonutils.QueryBoolean(data, "keep_disk", false)
	iDisk, err := DiskManager.FetchByIdOrName(userCred, diskId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("failed to find disk %s", diskId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	disk := iDisk.(*SDisk)
	if disk != nil {
		if self.isAttach2Disk(disk) {
			if disk.DiskType == DISK_TYPE_SYS {
				return nil, httperrors.NewUnsupportOperationError("Cannot detach sys disk")
			}
			detachDiskStatus, err := self.GetDriver().GetDetachDiskStatus()
			if err != nil {
				return nil, err
			}
			if keepDisk && !self.GetDriver().CanKeepDetachDisk() {
				return nil, httperrors.NewInputParameterError("Cannot keep detached disk")
			}
			if utils.IsInStringArray(self.Status, detachDiskStatus) {
				err = self.StartGuestDetachdiskTask(ctx, userCred, disk, keepDisk, "")
				return nil, err
			} else {
				return nil, httperrors.NewInvalidStatusError("Server in %s not able to detach disk", self.Status)
			}
		} else {
			return nil, httperrors.NewInvalidStatusError("Disk %s not attached", diskId)
		}
	}
	return nil, httperrors.NewResourceNotFoundError("Disk %s not found", diskId)
}

func (self *SGuest) StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, keepDisk bool, parentTaskId string) error {
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(disk.Id), "disk_id")
	taskData.Add(jsonutils.NewBool(keepDisk), "keep_disk")
	if utils.IsInStringArray(disk.Status, []string{DISK_INIT, DISK_ALLOC_FAILED}) {
		//删除非正常状态下的disk
		taskData.Add(jsonutils.JSONFalse, "keep_disk")
		db.Update(disk, func() error {
			disk.AutoDelete = true
			return nil
		})
	}
	disk.SetStatus(userCred, DISK_DETACHING, "")
	return self.GetDriver().StartGuestDetachdiskTask(ctx, userCred, self, taskData, parentTaskId)
}

func (self *SGuest) AllowPerformDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "detach-isolated-device")
}

func (self *SGuest) PerformDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != VM_READY {
		msg := "Only allowed to attach isolated device when guest is ready"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewInvalidStatusError(msg)
	}
	device, err := data.GetString("device")
	if err != nil {
		msg := "Missing isolated device"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewBadRequestError(msg)
	}
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msg := fmt.Sprintf("Isolated device %s not found", device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewBadRequestError(msg)
	}
	dev := iDev.(*SIsolatedDevice)
	host := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	err = self.detachIsolateDevice(ctx, userCred, dev)
	return nil, err
}

func (self *SGuest) detachIsolateDevice(ctx context.Context, userCred mcclient.TokenCredential, dev *SIsolatedDevice) error {
	if dev.GuestId != self.Id {
		msg := "Isolated device is not attached to this guest"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return httperrors.NewBadRequestError(msg)
	}
	_, err := db.Update(dev, func() error {
		dev.GuestId = ""
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) AllowPerformAttachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "attach-isolated-device")
}

func (self *SGuest) PerformAttachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != VM_READY {
		msg := "Only allowed to attach isolated device when guest is ready"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewInvalidStatusError(msg)
	}
	device, err := data.GetString("device")
	if err != nil {
		msg := "Missing isolated device"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewBadRequestError(msg)
	}
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msg := fmt.Sprintf("Isolated device %s not found", device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewBadRequestError(msg)
	}
	dev := iDev.(*SIsolatedDevice)
	host := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	err = self.attachIsolatedDevice(ctx, userCred, dev)
	var msg string
	if err != nil {
		msg = err.Error()
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, err == nil)
	return nil, err
}

func (self *SGuest) AllowPerformChangeIpaddr(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-ipaddr")
}

func (self *SGuest) findGuestnetworkByInfo(ipStr string, macStr string, index int64) (*SGuestnetwork, error) {
	if len(ipStr) > 0 {
		gn, err := self.GetGuestnetworkByIp(ipStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("ip %s not found", ipStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		return gn, nil
	} else if len(macStr) > 0 {
		gn, err := self.GetGuestnetworkByMac(macStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("mac %s not found", macStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		return gn, nil
	} else {
		gns, err := self.GetNetworks("")
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		if index >= 0 && index < int64(len(gns)) {
			return &gns[index], nil
		}
		return nil, httperrors.NewInputParameterError("no either ip_addr or mac specified")
	}
}

func (self *SGuest) PerformChangeIpaddr(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != VM_READY && self.Status != VM_RUNNING {
		return nil, httperrors.NewInvalidStatusError("Cannot change network ip_addr in status %s", self.Status)
	}

	reserve := jsonutils.QueryBoolean(data, "reserve", false)

	ipStr, _ := data.GetString("ip_addr")
	macStr, _ := data.GetString("mac")
	index, _ := data.Int("index")

	gn, err := self.findGuestnetworkByInfo(ipStr, macStr, index)
	if err != nil {
		return nil, err
	}

	netDesc, err := data.Get("net_desc")
	if err != nil {
		log.Errorf("net_desc not found")
		return nil, httperrors.NewMissingParameterError("net_desc")
	}
	conf, err := parseNetworkInfo(userCred, netDesc)
	if err != nil {
		log.Errorf("parseNetworkInfo fail %s", err)
		return nil, err
	}
	err = isValidNetworkInfo(userCred, conf)
	if err != nil {
		return nil, err
	}
	host := self.GetHost()

	ngn, err := func() ([]SGuestnetwork, error) {
		lockman.LockRawObject(ctx, GuestnetworkManager.KeywordPlural(), "")
		defer lockman.ReleaseRawObject(ctx, GuestnetworkManager.KeywordPlural(), "")

		if len(conf.Mac) > 0 {
			if conf.Mac != gn.MacAddr {
				if self.Status != VM_READY {
					// change mac
					return nil, httperrors.NewInvalidStatusError("cannot change mac when guest is running")
				}
				// check mac duplication
				if GuestnetworkManager.Query().Equals("mac_addr", conf.Mac).Count() > 0 {
					return nil, httperrors.NewConflictError("mac addr %s has been occupied", conf.Mac)
				}
			} else {
				if conf.Address == gn.IpAddr { // ip addr is the same, noop
					return nil, nil
				}
			}
		} else {
			conf.Mac = gn.MacAddr
		}

		err = self.detachNetworks(ctx, userCred, []SGuestnetwork{*gn}, reserve, false)
		if err != nil {
			return nil, err
		}
		conf.Ifname = gn.Ifname
		ngn, err := self.attach2NetworkDesc(ctx, userCred, host, conf, nil)
		if err != nil {
			return nil, httperrors.NewBadRequestError(err.Error())
		}

		return ngn, nil
	}()

	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CHANGE_NIC, err, userCred, false)
		return nil, err
	}

	notes := jsonutils.NewDict()
	if gn != nil {
		notes.Add(jsonutils.NewString(gn.IpAddr), "prev_ip")
	}
	if ngn != nil {
		notes.Add(jsonutils.NewString(ngn[0].IpAddr), "ip")
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CHANGE_NIC, notes, userCred, true)

	err = self.StartSyncTask(ctx, userCred, true, "")
	return nil, err
}

func (self *SGuest) AllowPerformDetachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "detachnetwork")
}

func (self *SGuest) PerformDetachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != VM_READY {
		return nil, httperrors.NewInvalidStatusError("Cannot detach network in status %s", self.Status)
	}
	reserve := jsonutils.QueryBoolean(data, "reserve", false)

	netStr, _ := data.GetString("net_id")
	if len(netStr) > 0 {
		netObj, err := NetworkManager.FetchById(netStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(NetworkManager.Keyword(), netStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		gns, err := self.GetNetworks(netObj.GetId())
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		err = self.detachNetworks(ctx, userCred, gns, reserve, true)
		return nil, err
	}
	ipStr, _ := data.GetString("ip_addr")
	if len(ipStr) > 0 {
		gn, err := self.GetGuestnetworkByIp(ipStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("ip %s not found", ipStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		err = self.detachNetworks(ctx, userCred, []SGuestnetwork{*gn}, reserve, true)
		return nil, err
	}
	macStr, _ := data.GetString("mac")
	if len(macStr) > 0 {
		gn, err := self.GetGuestnetworkByMac(macStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("mac %s not found", macStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		err = self.detachNetworks(ctx, userCred, []SGuestnetwork{*gn}, reserve, true)
		return nil, err
	}
	return nil, httperrors.NewInputParameterError("no either ip_addr, mac or network specified")
}

func (self *SGuest) AllowPerformAttachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "attachnetwork")
}

func (self *SGuest) PerformAttachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == VM_READY {
		// owner_cred = self.get_owner_user_cred() >.<
		netDesc, err := data.Get("net_desc")
		if err != nil {
			return nil, httperrors.NewBadRequestError(err.Error())
		}
		conf, err := parseNetworkInfo(userCred, netDesc)
		if err != nil {
			return nil, err
		}
		err = isValidNetworkInfo(userCred, conf)
		if err != nil {
			return nil, err
		}
		var inicCnt, enicCnt, ibw, ebw int
		if isExitNetworkInfo(conf) {
			enicCnt = 1
			ebw = conf.BwLimit
		} else {
			inicCnt = 1
			ibw = conf.BwLimit
		}
		pendingUsage := &SQuota{
			Port:  inicCnt,
			Eport: enicCnt,
			Bw:    ibw,
			Ebw:   ebw,
		}
		projectId := self.GetOwnerProjectId()
		err = QuotaManager.CheckSetPendingQuota(ctx, userCred, projectId, pendingUsage)
		if err != nil {
			return nil, httperrors.NewOutOfQuotaError(err.Error())
		}
		host := self.GetHost()
		_, err = self.attach2NetworkDesc(ctx, userCred, host, conf, pendingUsage)
		if err != nil {
			QuotaManager.CancelPendingUsage(ctx, userCred, projectId, nil, pendingUsage)
			return nil, httperrors.NewBadRequestError(err.Error())
		}
		host.ClearSchedDescCache()
		err = self.StartGuestDeployTask(ctx, userCred, nil, "deploy", "")
		return nil, err
	}
	return nil, httperrors.NewBadRequestError("Cannot attach network in status %s", self.Status)
}

func (self *SGuest) AllowPerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-bandwidth")
}

func (self *SGuest) PerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING}) {
		msg := fmt.Sprintf("Cannot change bandwidth in status %s", self.Status)
		return nil, httperrors.NewBadRequestError(msg)
	}

	bandwidth, err := data.Int("bandwidth")
	if err != nil || bandwidth < 0 {
		return nil, httperrors.NewBadRequestError("Bandwidth must be non-negative")
	}

	ipStr, _ := data.GetString("ip_addr")
	macStr, _ := data.GetString("mac")
	index, _ := data.Int("index")
	guestnic, err := self.findGuestnetworkByInfo(ipStr, macStr, index)
	if err != nil {
		return nil, err
	}

	if guestnic.BwLimit != int(bandwidth) {
		diff, err := db.Update(guestnic, func() error {
			guestnic.BwLimit = int(bandwidth)
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_CHANGE_BANDWIDTH, diff, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CHANGE_BANDWIDTH, diff, userCred, true)
		return nil, self.StartSyncTask(ctx, userCred, false, "")
	}
	return nil, nil
}

func (self *SGuest) AllowPerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-config")
}

func (self *SGuest) PerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.GetDriver().AllowReconfigGuest() {
		return nil, httperrors.NewInvalidStatusError("Not allow to change config")
	}

	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup not allow to change config")
	}

	changeStatus, err := self.GetDriver().GetChangeConfigStatus()
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	if !utils.IsInStringArray(self.Status, changeStatus) {
		return nil, httperrors.NewInvalidStatusError("Cannot change config in %s", self.Status)
	}

	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("No valid host")
	}

	var addCpu, addMem int
	var cpuChanged, memChanged bool

	confs := jsonutils.NewDict()
	skuId := jsonutils.GetAnyString(data, []string{"instance_type", "sku", "flavor"})
	if len(skuId) > 0 {
		sku, err := ServerSkuManager.FetchSkuByNameAndHypervisor(skuId, self.GetHypervisor(), true)
		if err != nil {
			return nil, err
		}

		if sku.GetName() != self.InstanceType {
			confs.Add(jsonutils.NewString(sku.GetName()), "instance_type")
			confs.Add(jsonutils.NewInt(int64(sku.CpuCoreCount)), "vcpu_count")
			confs.Add(jsonutils.NewInt(int64(sku.MemorySizeMB)), "vmem_size")

			if sku.CpuCoreCount != int(self.VcpuCount) {
				cpuChanged = true
				addCpu = sku.CpuCoreCount - int(self.VcpuCount)
			}
			if sku.MemorySizeMB != self.VmemSize {
				memChanged = true
				addMem = sku.MemorySizeMB - self.VmemSize
			}
		}

	} else {
		vcpuCount, err := data.GetString("vcpu_count")
		if err == nil {
			nVcpu, err := strconv.ParseInt(vcpuCount, 10, 0)
			if err != nil {
				return nil, httperrors.NewBadRequestError("Params vcpu_count parse error")
			}

			if nVcpu != int64(self.VcpuCount) {
				cpuChanged = true
				addCpu = int(nVcpu - int64(self.VcpuCount))
				err = confs.Add(jsonutils.NewInt(nVcpu), "vcpu_count")
				if err != nil {
					return nil, httperrors.NewBadRequestError("Params vcpu_count parse error")
				}
			}
		}
		vmemSize, err := data.GetString("vmem_size")
		if err == nil {
			if !regutils.MatchSize(vmemSize) {
				return nil, httperrors.NewBadRequestError("Memory size must be number[+unit], like 256M, 1G or 256")
			}
			nVmem, err := fileutils.GetSizeMb(vmemSize, 'M', 1024)
			if err != nil {
				httperrors.NewBadRequestError("Params vmem_size parse error")
			}
			if nVmem != self.VmemSize {
				memChanged = true
				addMem = nVmem - self.VmemSize
				err = confs.Add(jsonutils.NewInt(int64(nVmem)), "vmem_size")
				if err != nil {
					return nil, httperrors.NewBadRequestError("Params vmem_size parse error")
				}
			}
		}
	}

	if self.Status == VM_RUNNING && (cpuChanged || memChanged) && self.GetDriver().NeedStopForChangeSpec(self) {
		return nil, httperrors.NewInvalidStatusError("cannot change CPU/Memory spec in status %s", self.Status)
	}

	if addCpu < 0 {
		addCpu = 0
	}
	if addMem < 0 {
		addMem = 0
	}

	disks := self.GetDisks()
	var addDisk int
	var diskIdx = 1
	var newDiskIdx = 0
	var diskSizes = make(map[string]int, 0)
	var newDisks = jsonutils.NewDict()
	var resizeDisks = jsonutils.NewArray()
	for {
		diskNum := fmt.Sprintf("disk.%d", diskIdx)
		diskDesc, err := data.Get(diskNum)
		if err != nil || diskDesc == jsonutils.JSONNull {
			break
		}
		diskConf, err := parseDiskInfo(ctx, userCred, diskDesc)
		if err != nil {
			return nil, httperrors.NewBadRequestError("Parse disk info error: %s", err)
		}
		if len(diskConf.Backend) == 0 {
			diskConf.Backend = self.getDefaultStorageType()
		}
		if diskConf.SizeMb > 0 {
			if diskIdx >= len(disks) {
				storage := host.GetLeastUsedStorage(diskConf.Backend)
				if storage == nil {
					return nil, httperrors.NewResourceNotReadyError("host not connect storage %s", diskConf.Backend)
				}
				_, ok := diskSizes[storage.Id]
				if !ok {
					diskSizes[storage.Id] = 0
				}
				diskSizes[storage.Id] = diskSizes[storage.Id] + diskConf.SizeMb
				diskConf.Storage = storage.Id
				newDisks.Add(jsonutils.Marshal(diskConf), fmt.Sprintf("disk.%d", newDiskIdx))
				newDiskIdx += 1
				addDisk += diskConf.SizeMb
			} else {
				disk := disks[diskIdx].GetDisk()
				oldSize := disk.DiskSize
				if diskConf.SizeMb < oldSize {
					return nil, httperrors.NewInputParameterError("Cannot reduce disk size")
				} else if diskConf.SizeMb > oldSize {
					arr := jsonutils.NewArray(jsonutils.NewString(disks[diskIdx].DiskId), jsonutils.NewInt(int64(diskConf.SizeMb)))
					resizeDisks.Add(arr)
					addDisk += diskConf.SizeMb - oldSize
					storage := disks[diskIdx].GetDisk().GetStorage()
					_, ok := diskSizes[storage.Id]
					if !ok {
						diskSizes[storage.Id] = 0
					}
					diskSizes[storage.Id] = diskSizes[storage.Id] + diskConf.SizeMb - oldSize
				}
			}
		}
		diskIdx += 1
	}

	provider, e := self.GetHost().GetProviderFactory()
	if e != nil || provider.IsOnPremise() {
		for storageId, needSize := range diskSizes {
			iStorage, err := StorageManager.FetchById(storageId)
			if err != nil {
				return nil, httperrors.NewBadRequestError("Fetch storage error: %s", err)
			}
			storage := iStorage.(*SStorage)
			if storage.GetFreeCapacity() < needSize {
				return nil, httperrors.NewInsufficientResourceError("Not enough free space")
			}
		}
	} else {
		log.Debugf("Skip storage free capacity validating for public cloud: %s", provider.GetId())
	}

	if newDisks.Length() > 0 {
		confs.Add(newDisks, "create")
	}
	if resizeDisks.Length() > 0 {
		confs.Add(resizeDisks, "resize")
	}
	if self.Status != VM_RUNNING && jsonutils.QueryBoolean(data, "auto_start", false) {
		confs.Add(jsonutils.NewBool(true), "auto_start")
	}
	if self.Status == VM_RUNNING {
		confs.Set("guest_online", jsonutils.JSONTrue)
	}

	// schedulr forecast
	schedDesc := self.confToSchedDesc(addCpu, addMem, addDisk)
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	canChangeConf, err := modules.SchedManager.DoScheduleForecast(s, schedDesc, 1)
	if err != nil {
		return nil, err
	}
	if !canChangeConf {
		return nil, httperrors.NewBadRequestError("Host resource is not enough")
	}

	log.Debugf("%s", confs.String())

	pendingUsage := &SQuota{}
	if addCpu > 0 {
		pendingUsage.Cpu = addCpu
	}
	if addMem > 0 {
		pendingUsage.Memory = addMem
	}
	if addDisk > 0 {
		pendingUsage.Storage = addDisk
	}
	if !pendingUsage.IsEmpty() {
		err := QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), pendingUsage)
		if err != nil {
			return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
		}
	}

	if newDisks.Length() > 0 {
		err := self.CreateDisksOnHost(ctx, userCred, host, newDisks, pendingUsage, false, false)
		if err != nil {
			QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, nil, pendingUsage)
			return nil, httperrors.NewBadRequestError("Create disk on host error: %s", err)
		}
	}
	self.StartChangeConfigTask(ctx, userCred, confs, "", pendingUsage)
	return nil, nil
}

func (self *SGuest) confToSchedDesc(addCpu, addMem, addDisk int) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Set("vmem_size", jsonutils.NewInt(int64(addMem)))
	desc.Set("vcpu_count", jsonutils.NewInt(int64(addCpu)))
	guestDisks := self.GetDisks()
	diskInfo := guestDisks[0].ToDiskInfo()
	diskInfo.Size = int64(addDisk)
	desc.Set("disk.0", jsonutils.Marshal(diskInfo))
	desc.Set("prefer_host_id", jsonutils.NewString(self.HostId))
	desc.Set("hypervisor", jsonutils.NewString(self.GetHypervisor()))
	desc.Set("owner_tenant_id", jsonutils.NewString(self.ProjectId))
	return desc
}

func (self *SGuest) StartChangeConfigTask(ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string, pendingUsage quotas.IQuota) error {
	self.SetStatus(userCred, VM_CHANGE_FLAVOR, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestChangeConfigTask", self, userCred, data, parentTaskId, "", pendingUsage)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) RevokeAllSecgroups(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := GuestsecgroupManager.DeleteGuestSecgroup(ctx, userCred, self, nil)
	if err != nil {
		return err
	}
	err = self.revokeSecgroup(ctx, userCred, self.getSecgroup())
	if err != nil {
		return err
	}
	return nil
}

func (self *SGuest) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	eip, _ := self.GetEip()
	if eip != nil {
		eip.DoPendingDelete(ctx, userCred)
	}

	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		if !disk.IsDetachable() {
			disk.DoPendingDelete(ctx, userCred)
		} else {
			log.Warningf("detachable disk on pending delete guests!!! should be removed earlier")
			self.DetachDisk(ctx, disk, userCred)
		}
	}
	self.SVirtualResourceBase.DoPendingDelete(ctx, userCred)
}

func (model *SGuest) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, model, "cancel-delete")
}

func (self *SGuest) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		disk.DoCancelPendingDelete(ctx, userCred)
	}
	return self.SVirtualResourceBase.DoCancelPendingDelete(ctx, userCred)
}

func (self *SGuest) StartUndeployGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, targetHostId string) error {
	data := jsonutils.NewDict()
	if len(targetHostId) > 0 {
		data.Add(jsonutils.NewString(targetHostId), "target_host_id")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "GuestUndeployTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformReset(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "reset")
}

func (self *SGuest) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isHard := jsonutils.QueryBoolean(data, "is_hard", false)
	if self.Status == VM_RUNNING || self.Status == VM_STOP_FAILED {
		self.GetDriver().StartGuestResetTask(self, ctx, userCred, isHard, "")
		return nil, nil
	}
	return nil, httperrors.NewInvalidStatusError("Cannot reset VM in status %s", self.Status)
}

func (self *SGuest) AllowPerformDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "disk-snapshot")
}

func (self *SGuest) PerformDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest has backup, can't create snapshot")
	}
	if !utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do snapshot when VM in status %s", self.Status)
	}
	diskId, err := data.GetString("disk_id")
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	name, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	err = ValidateSnapshotName(self.Hypervisor, name, userCred.GetProjectId())
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	if self.GetGuestDisk(diskId) == nil {
		return nil, httperrors.NewNotFoundError("Guest disk %s not found", diskId)
	}
	if self.GetHypervisor() == HYPERVISOR_KVM {
		q := SnapshotManager.Query()
		cnt := q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
			sqlchemy.Equals(q.Field("created_by"), MANUAL),
			sqlchemy.Equals(q.Field("fake_deleted"), false))).Count()
		if cnt >= options.Options.DefaultMaxManualSnapshotCount {
			return nil, httperrors.NewBadRequestError("Disk %s snapshot full, cannot take any more", diskId)
		}
		pendingUsage := &SQuota{Snapshot: 1}
		err = QuotaManager.CheckSetPendingQuota(ctx, userCred, self.ProjectId, pendingUsage)
		if err != nil {
			return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
		}
		snapshot, err := SnapshotManager.CreateSnapshot(ctx, userCred, MANUAL, diskId, self.Id, "", name)
		QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, nil, pendingUsage)
		if err != nil {
			return nil, err
		}
		err = self.StartDiskSnapshot(ctx, userCred, diskId, snapshot.Id)
		return nil, err
	} else {
		snapshot, err := SnapshotManager.CreateSnapshot(ctx, userCred, MANUAL, diskId, self.Id, "", name)
		if err != nil {
			return nil, err
		}
		err = self.StartDiskSnapshot(ctx, userCred, diskId, snapshot.Id)
		return nil, err
	}

}

func (self *SGuest) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SGuest) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	self.SetStatus(userCred, VM_SYNCING_STATUS, "perform_syncstatus")
	err := self.StartSyncstatus(ctx, userCred, "")
	return nil, err
}

func (self *SGuest) isNotRunningStatus(status string) bool {
	if status == VM_READY || status == VM_SUSPEND {
		return true
	}
	return false
}

func (self *SGuest) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	preStatus := self.Status
	_, err := self.SVirtualResourceBase.PerformStatus(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	if preStatus != self.Status && !self.isNotRunningStatus(preStatus) && self.isNotRunningStatus(self.Status) {
		db.OpsLog.LogEvent(self, db.ACT_STOP, "", userCred)
		if self.Status == VM_READY && !self.DisableDelete.Bool() && self.ShutdownBehavior == SHUTDOWN_TERMINATE {
			err = self.StartAutoDeleteGuestTask(ctx, userCred, "")
			return nil, err
		}
	}
	return nil, nil
}

func (self *SGuest) StartDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, diskId, snapshotId string) error {
	self.SetStatus(userCred, VM_START_SNAPSHOT, "StartDiskSnapshot")
	params := jsonutils.NewDict()
	params.Set("disk_id", jsonutils.NewString(diskId))
	params.Set("snapshot_id", jsonutils.NewString(snapshotId))
	return self.GetDriver().StartGuestDiskSnapshotTask(ctx, userCred, self, params)
}

func (self *SGuest) AllowPerformStop(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "stop")
}

func (self *SGuest) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// XXX if is force, force stop guest
	var isForce = jsonutils.QueryBoolean(data, "is_force", false)
	if isForce || utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_STOP_FAILED}) {
		return nil, self.StartGuestStopTask(ctx, userCred, isForce, "")
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot stop server in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformRestart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "restart")
}

func (self *SGuest) PerformRestart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isForce := jsonutils.QueryBoolean(data, "is_force", false)
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_STOP_FAILED}) || (isForce && self.Status == VM_STOPPING) {
		return nil, self.GetDriver().StartGuestRestartTask(self, ctx, userCred, isForce, "")
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do restart server in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformSendkeys(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sendkeys")
}

func (self *SGuest) PerformSendkeys(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != VM_RUNNING {
		return nil, httperrors.NewInvalidStatusError("Cannot send keys in status %s", self.Status)
	}
	keys, err := data.GetString("keys")
	if err != nil {
		log.Errorln(err)
		return nil, httperrors.NewMissingParameterError("keys")
	}
	err = self.VerifySendKeys(keys)
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	cmd := fmt.Sprintf("sendkey %s", keys)
	duration, err := data.Int("duration")
	if err == nil {
		cmd = fmt.Sprintf("%s %d", cmd, duration)
	}
	_, err = self.SendMonitorCommand(ctx, userCred, cmd)
	return nil, err
}

func (self *SGuest) VerifySendKeys(keyStr string) error {
	keys := strings.Split(keyStr, "-")
	for _, key := range keys {
		if !self.IsLegalKey(key) {
			return fmt.Errorf("Unknown key '%s'", key)
		}
	}
	return nil
}

func (self *SGuest) IsLegalKey(key string) bool {
	singleKeys := "1234567890abcdefghijklmnopqrstuvwxyz"
	legalKeys := []string{"ctrl", "ctrl_r", "alt", "alt_r", "shift", "shift_r",
		"delete", "esc", "insert", "print", "spc",
		"f1", "f2", "f3", "f4", "f5", "f6",
		"f7", "f8", "f9", "f10", "f11", "f12",
		"home", "pgup", "pgdn", "end",
		"up", "down", "left", "right",
		"tab", "minus", "equal", "backspace", "backslash",
		"bracket_left", "bracket_right", "backslash",
		"semicolon", "apostrophe", "grave_accent", "ret",
		"comma", "dot", "slash",
		"caps_lock", "num_lock", "scroll_lock"}
	if len(key) > 1 && !utils.IsInStringArray(key, legalKeys) {
		return false
	} else if len(key) == 1 && !strings.Contains(singleKeys, key) {
		return false
	}
	return true
}

func (self *SGuest) SendMonitorCommand(ctx context.Context, userCred mcclient.TokenCredential, cmd string) (jsonutils.JSONObject, error) {
	host := self.GetHost()
	url := fmt.Sprintf("%s/servers/%s/monitor", host.ManagerUri, self.Id)
	header := http.Header{}
	header.Add("X-Auth-Token", userCred.GetTokenString())
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(cmd), "cmd")
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, err
	}
	ret := res.(*jsonutils.JSONDict)
	return ret, nil
}

func (self *SGuest) AllowPerformAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "associate-eip")
}

func (self *SGuest) PerformAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{VM_READY, VM_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("cannot associate eip in status %s", self.Status)
	}

	eip, err := self.GetEip()
	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if eip != nil {
		return nil, httperrors.NewInvalidStatusError("already associate with eip")
	}
	eipStr := jsonutils.GetAnyString(data, []string{"eip", "eip_id"})
	if len(eipStr) == 0 {
		return nil, httperrors.NewMissingParameterError("eip_id")
	}
	eipObj, err := ElasticipManager.FetchByIdOrName(userCred, eipStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("eip %s not found", eipStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	eip = eipObj.(*SElasticip)
	eipRegion := eip.GetRegion()
	instRegion := self.getRegion()

	if eip.Mode == EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	eipVm := eip.GetAssociateVM()
	if eipVm != nil {
		return nil, httperrors.NewConflictError("eip has been associated")
	}

	if eipRegion.Id != instRegion.Id {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different region")
	}

	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInputParameterError("server host is not found???")
	}

	if host.ManagerId != eip.ManagerId {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different provider")
	}

	self.SetStatus(userCred, VM_ASSOCIATE_EIP, "associate eip")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(self.Id), "instance_id")
	params.Add(jsonutils.NewString(EIP_ASSOCIATE_TYPE_SERVER), "instance_type")

	err = eip.StartEipAssociateTask(ctx, userCred, params, "")

	return nil, err
}

func (self *SGuest) AllowPerformDissociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "dissociate-eip")
}

func (self *SGuest) PerformDissociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	eip, err := self.GetEip()
	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if eip == nil {
		return nil, httperrors.NewInvalidStatusError("No eip to dissociate")
	}

	self.SetStatus(userCred, VM_DISSOCIATE_EIP, "associate eip")

	autoDelete := jsonutils.QueryBoolean(data, "auto_delete", false)

	err = eip.StartEipDissociateTask(ctx, userCred, autoDelete, "")
	if err != nil {
		log.Errorf("fail to start dissociate task %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

func (self *SGuest) AllowPerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "create-eip")
}

func (self *SGuest) PerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	bw, err := data.Int("bandwidth")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("bandwidth")
	}

	chargeType, _ := data.GetString("charge_type")
	if len(chargeType) == 0 {
		chargeType = EIP_CHARGE_TYPE_DEFAULT
	}

	if len(self.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("Not a managed VM")
	}
	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("No host???")
	}

	_, err = host.GetDriver()
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("No valid cloud provider")
	}

	region := host.GetRegion()
	if region == nil {
		return nil, httperrors.NewInvalidStatusError("No cloudregion???")
	}

	eipPendingUsage := &SQuota{Eip: 1}
	err = QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), eipPendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Out of eip quota: %s", err)
	}

	err = ElasticipManager.AllocateEipAndAssociateVM(ctx, userCred, self, int(bw), chargeType, eipPendingUsage)
	if err != nil {
		QuotaManager.CancelPendingUsage(ctx, userCred, userCred.GetProjectId(), eipPendingUsage, eipPendingUsage)
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (self *SGuest) setUserData(ctx context.Context, userCred mcclient.TokenCredential, data string) error {
	data = base64.StdEncoding.EncodeToString([]byte(data))
	if len(data) > 16*1024 {
		return fmt.Errorf("User data is limited to 16 KB.")
	}
	err := self.SetMetadata(ctx, "user_data", data, userCred)
	if err != nil {
		return err
	}
	return nil
}

func (self *SGuest) AllowPerformUserData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "userdata")
}

func (self *SGuest) PerformUserData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userData, err := data.GetString("user_data")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("user_data")
	}
	err = self.setUserData(ctx, userCred, userData)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(self.HostId) > 0 {
		err = self.StartSyncTask(ctx, userCred, false, "")
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	return nil, nil
}

func (self *SGuest) SwitchToBackup(userCred mcclient.TokenCredential) error {
	diff, err := db.Update(self, func() error {
		self.HostId, self.BackupHostId = self.BackupHostId, self.HostId
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (self *SGuest) AllowPerformSwitchToBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "switch-to-backup")
}

func (self *SGuest) PerformSwitchToBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == VM_BLOCK_STREAM {
		return nil, httperrors.NewBadRequestError("Cannot swith to backup when guest in status %s", self.Status)
	}
	if len(self.BackupHostId) == 0 {
		return nil, httperrors.NewBadRequestError("Guest no backup host")
	}
	oldStatus := self.Status
	self.SetStatus(userCred, VM_SWITCH_TO_BACKUP, "Switch to backup")
	deleteBackup := jsonutils.QueryBoolean(data, "delete_backup", false)
	purgeBackup := jsonutils.QueryBoolean(data, "purge_backup", false)

	taskData := jsonutils.NewDict()
	taskData.Set("old_status", jsonutils.NewString(oldStatus))
	taskData.Set("delete_backup", jsonutils.NewBool(deleteBackup))
	taskData.Set("purge_backup", jsonutils.NewBool(purgeBackup))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSwitchToBackupTask", self, userCred, taskData, "", "", nil); err != nil {
		log.Errorln(err)
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (manager *SGuestManager) AllowPerformDirtyServerStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "dirty-server-start")
}

func (manager *SGuestManager) PerformDirtyServerStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guestId, err := data.GetString("guest_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("guest_id")
	}
	guest := manager.FetchGuestById(guestId)
	if guest == nil {
		return nil, httperrors.NewNotFoundError("Guest %s not found", guestId)
	}
	hostId, _ := data.GetString("host_id")
	if len(hostId) == 0 {
		return nil, httperrors.NewMissingParameterError("host_id")
	}

	if guest.HostId == hostId {
		// master guest
		err := guest.StartGueststartTask(ctx, userCred, nil, "")
		return nil, err
	} else if guest.BackupHostId == hostId {
		// slave guest
		err := guest.GuestStartAndSyncToBackup(ctx, userCred, nil, "")
		return nil, err
	}
	return nil, nil
}

func (manager *SGuestManager) AllowPerformDirtyServerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "dirty-server-verify")
}

func (manager *SGuestManager) PerformDirtyServerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guestId, err := data.GetString("guest_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("guest_id")
	}
	guest := manager.FetchGuestById(guestId)
	if guest == nil {
		return nil, httperrors.NewNotFoundError("Guest %s not found", guestId)
	}
	hostId, _ := data.GetString("host_id")
	if len(hostId) == 0 {
		return nil, httperrors.NewMissingParameterError("host_id")
	}

	if guest.HostId != hostId && guest.BackupHostId != hostId {
		return nil, guest.StartGuestDeleteOnHostTask(ctx, userCred, hostId, false, "")
	}
	return nil, nil
}

func (self *SGuest) StartGuestDeleteOnHostTask(ctx context.Context, userCred mcclient.TokenCredential, hostId string, purge bool, parentTaskId string) error {
	taskData := jsonutils.NewDict()
	taskData.Set("host_id", jsonutils.NewString(hostId))
	taskData.Set("purge", jsonutils.NewBool(purge))
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestDeleteOnHostTask", self, userCred, taskData, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (guest *SGuest) GuestStartAndSyncToBackup(ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStartAndSyncToBackupTask", guest, userCred, data, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformCreateBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "create-backup")
}

func (self *SGuest) PerformCreateBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Already have backup server")
	}
	if self.getDefaultStorageType() != STORAGE_LOCAL {
		return nil, httperrors.NewBadRequestError("Cannot create backup with shared storage")
	}
	if self.Hypervisor != HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Backup only support hypervisor kvm")
	}
	if len(self.GetIsolatedDevices()) > 0 {
		return nil, httperrors.NewBadRequestError("Cannot create backup with isolated devices")
	}
	if self.GuestDisksHasSnapshot() {
		return nil, httperrors.NewBadRequestError("Cannot create backup with snapshot")
	}

	req := self.getGuestBackupResourceRequirements(ctx, userCred)
	err := QuotaManager.CheckSetPendingQuota(ctx, userCred, self.GetOwnerProjectId(), &req)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	params := data.(*jsonutils.JSONDict)
	params.Set("guest_status", jsonutils.NewString(self.Status))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateBackupTask", self, userCred, params, "", "", &req)
	if err != nil {
		QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, nil, &req)
		log.Errorln(err)
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (self *SGuest) AllowPerformDeleteBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "delete-backup")
}

func (self *SGuest) PerformDeleteBackup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) == 0 {
		return nil, httperrors.NewBadRequestError("Guest without backup")
	}
	backupHost := HostManager.FetchHostById(self.BackupHostId)
	if backupHost == nil {
		return nil, httperrors.NewNotFoundError("Guest backup host not found")
	}
	if backupHost.Status == HOST_OFFLINE && !jsonutils.QueryBoolean(data, "purge", false) {
		return nil, httperrors.NewBadRequestError("Backup host is offline")
	}

	taskData := jsonutils.NewDict()
	taskData.Set("purge", jsonutils.NewBool(jsonutils.QueryBoolean(data, "purge", false)))
	taskData.Set("host_id", jsonutils.NewString(self.BackupHostId))
	taskData.Set("failed_status", jsonutils.NewString(VM_BACKUP_DELETE_FAILED))

	self.SetStatus(userCred, VM_DELETING_BACKUP, "delete backup server")
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestDeleteOnHostTask", self, userCred, taskData, "", "", nil); err != nil {
		log.Errorln(err)
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (self *SGuest) CreateBackupDisks(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateBackupDisksTask", self, userCred, nil, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) StartCreateBackup(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, kwargs *jsonutils.JSONDict) error {
	if kwargs == nil {
		kwargs = jsonutils.NewDict()
	}
	kwargs.Add(jsonutils.NewString("create"), "deploy_action")
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestDeployBackupTask", self, userCred, kwargs, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformMirrorJobFailed(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "mirror-job-failed")
}

func (self *SGuest) PerformMirrorJobFailed(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) == 0 {
		return nil, nil
	} else {
		return nil, self.SetStatus(userCred, VM_MIRROR_FAIL, "OnSyncToBackup")
	}
}

func (self *SGuest) AllowPerformSetExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "set-extra-option")
}

func (self *SGuest) PerformSetExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	key, err := data.GetString("key")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("key")
	}
	value, _ := data.GetString("value")
	extraOptions := self.GetExtraOptions(userCred)
	extraOptions.Set(key, jsonutils.NewString(value))
	return nil, self.SetExtraOptions(ctx, userCred, extraOptions)
}

func (self *SGuest) GetExtraOptions(userCred mcclient.TokenCredential) *jsonutils.JSONDict {
	options := self.GetMetadataJson("extra_options", userCred)
	o, ok := options.(*jsonutils.JSONDict)
	if ok {
		return o
	}
	return jsonutils.NewDict()
}

func (self *SGuest) SetExtraOptions(ctx context.Context, userCred mcclient.TokenCredential, extraOptions *jsonutils.JSONDict) error {
	return self.SetMetadata(ctx, "extra_options", extraOptions, userCred)
}

func (self *SGuest) AllowPerformDelExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "del-extra-option")
}

func (self *SGuest) PerformDelExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	key, err := data.GetString("key")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("key")
	}
	extraOptions := self.GetExtraOptions(userCred)
	extraOptions.Remove(key)
	return nil, self.SetExtraOptions(ctx, userCred, extraOptions)
}

func (self *SGuest) AllowPerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "renew")
}

func (self *SGuest) PerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	durationStr := jsonutils.GetAnyString(data, []string{"duration"})
	if len(durationStr) == 0 {
		return nil, httperrors.NewInputParameterError("missong duration")
	}

	bc, err := billing.ParseBillingCycle(durationStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", durationStr, err)
	}

	if !self.GetDriver().IsSupportedBillingCycle(bc) {
		return nil, httperrors.NewInputParameterError("unsupported duration %s", durationStr)
	}

	err = self.startGuestRenewTask(ctx, userCred, durationStr, "")
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (self *SGuest) startGuestRenewTask(ctx context.Context, userCred mcclient.TokenCredential, duration string, parentTaskId string) error {
	self.SetStatus(userCred, VM_RENEWING, "")
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(duration), "duration")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestRenewTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("fail to crate GuestRenewTask %s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) SaveRenewInfo(ctx context.Context, userCred mcclient.TokenCredential, bc *billing.SBillingCycle, expireAt *time.Time) error {
	err := self.doSaveRenewInfo(ctx, userCred, bc, expireAt)
	if err != nil {
		return err
	}
	guestdisks := self.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		if disk.BillingType == BILLING_TYPE_PREPAID {
			err = disk.SaveRenewInfo(ctx, userCred, bc, expireAt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SGuest) doSaveRenewInfo(ctx context.Context, userCred mcclient.TokenCredential, bc *billing.SBillingCycle, expireAt *time.Time) error {
	_, err := db.Update(self, func() error {
		if self.BillingType != BILLING_TYPE_PREPAID {
			self.BillingType = BILLING_TYPE_PREPAID
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		log.Errorf("Update error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) AllowPerformStreamDisksComplete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "stream-disks-complete")
}

func (self *SGuest) PerformStreamDisksComplete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	for _, disk := range self.GetDisks() {
		d := disk.GetDisk()
		if len(d.SnapshotId) > 0 {
			SnapshotManager.AddRefCount(d.SnapshotId, -1)
			d.SetMetadata(ctx, "merge_snapshot", jsonutils.JSONFalse, userCred)
		}
	}
	return nil, nil
}

func (man *SGuestManager) AllowPerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, man, "import")
}

type SImportNic struct {
	Index     int    `json:"index"`
	Bridge    string `json:"bridge"`
	Domain    string `json:"domain"`
	Ip        string `json:"ip"`
	Vlan      int    `json:"vlan"`
	Driver    string `json:"driver"`
	Masklen   int    `json:"masklen"`
	Virtual   bool   `json:"virtual"`
	Manual    bool   `json:"manual"`
	WireId    string `json:"wire_id"`
	NetId     string `json:"net_id"`
	Mac       string `json:"mac"`
	BandWidth int    `json:"bw"`
	Dns       string `json:"dns"`
	Net       string `json:"net"`
	Interface string `json:"interface"`
	Gateway   string `json:"gateway"`
	Ifname    string `json:"ifname"`
}

func (n SImportNic) ToNetConfig(net *SNetwork) *SNetworkConfig {
	return &SNetworkConfig{
		Network: net.Id,
		Wire:    net.WireId,
		Address: n.Ip,
		Mac:     n.Mac,
		Driver:  n.Driver,
		BwLimit: n.BandWidth,
	}
}

type SImportDisk struct {
	Index      int    `json:"index"`
	DiskId     string `json:"disk_id"`
	Driver     string `json:"driver"`
	CacheMode  string `json:"cache_mode"`
	AioMode    string `json:"aio_mode"`
	SizeMb     int    `json:"size"`
	Format     string `json:"format"`
	Fs         string `json:"fs"`
	Mountpoint string `json:"mountpoint"`
	Dev        string `json:"dev"`
	TemplateId string `json:"template_id"`
}

func (d SImportDisk) ToDiskConfig() *SDiskConfig {
	ret := &SDiskConfig{
		SizeMb:  d.SizeMb,
		ImageId: d.TemplateId,
		Format:  d.Format,
		Driver:  d.Driver,
		DiskId:  d.DiskId,
		Cache:   d.CacheMode,
	}
	if len(d.Mountpoint) > 0 {
		ret.Mountpoint = d.Mountpoint
	}
	if len(d.Fs) > 0 {
		ret.Fs = d.Fs
	}
	return ret
}

type SImportGuestDesc struct {
	Id          string            `json:"uuid"`
	Name        string            `json:"name"`
	Nics        []SImportNic      `json:"nics"`
	Disks       []SImportDisk     `json:"disks"`
	Metadata    map[string]string `json:"metadata"`
	MemSizeMb   int               `json:"mem"`
	Cpu         int               `json:"cpu"`
	TemplateId  string            `json:"template_id"`
	ImagePath   string            `json:"image_path"`
	Vdi         string            `json:"vdi"`
	Hypervisor  string            `json:"hypervisor"`
	HostId      string            `json:"host"`
	BootOrder   string            `json:"boot_order"`
	IsSystem    bool              `json:"is_system"`
	Description string            `json:"description"`
}

func (man *SGuestManager) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	desc := SImportGuestDesc{}
	if err := data.Unmarshal(&desc); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid desc: %s", data.String())
	}
	if len(desc.Id) == 0 {
		return nil, httperrors.NewInputParameterError("Server Id is empty")
	}
	if len(desc.Name) == 0 {
		return nil, httperrors.NewInputParameterError("Server Name is empty")
	}
	if obj, _ := man.FetchByIdOrName(userCred, desc.Id); obj != nil {
		return nil, httperrors.NewInputParameterError("Server %s already exists", desc.Id)
	}
	if err := db.NewNameValidator(man, userCred.GetProjectId(), desc.Name); err != nil {
		return nil, err
	}
	if hostObj, _ := HostManager.FetchByIdOrName(userCred, desc.HostId); hostObj == nil {
		return nil, httperrors.NewNotFoundError("Host %s not found", desc.HostId)
	} else {
		desc.HostId = hostObj.GetId()
	}
	// 1. create import guest on host
	gst, err := man.createImportGuest(ctx, userCred, desc)
	if err != nil {
		return nil, err
	}
	// 2. import networks
	if err := gst.importNics(ctx, userCred, desc.Nics); err != nil {
		return nil, err
	}
	// 3. import disks
	if err := gst.importDisks(ctx, userCred, desc.Disks); err != nil {
		return nil, err
	}
	// 4. set metadata
	for k, v := range desc.Metadata {
		gst.SetMetadata(ctx, k, v, userCred)
	}
	return jsonutils.Marshal(gst), nil
}

func (man *SGuestManager) createImportGuest(ctx context.Context, userCred mcclient.TokenCredential, desc SImportGuestDesc) (*SGuest, error) {
	model, err := db.NewModelObject(man)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	gst, ok := model.(*SGuest)
	if !ok {
		return nil, httperrors.NewGeneralError(fmt.Errorf("Can't convert %#v to *SGuest model", model))
	}
	gst.ProjectId = userCred.GetProjectId()
	gst.IsSystem = desc.IsSystem
	gst.Id = desc.Id
	gst.Name = desc.Name
	gst.HostId = desc.HostId
	gst.Status = "import"
	gst.Hypervisor = desc.Hypervisor
	gst.VmemSize = desc.MemSizeMb
	gst.VcpuCount = int8(desc.Cpu)
	gst.BootOrder = desc.BootOrder
	gst.Description = desc.Description
	err = man.TableSpec().Insert(gst)
	return gst, err
}

func (self *SGuest) importNics(ctx context.Context, userCred mcclient.TokenCredential, nics []SImportNic) error {
	if len(nics) == 0 {
		return httperrors.NewInputParameterError("Empty import nics")
	}
	for _, nic := range nics {
		net, err := NetworkManager.GetOnPremiseNetworkOfIP(nic.Ip, "", tristate.None)
		if err != nil {
			return httperrors.NewNotFoundError("Not found network by ip %s", nic.Ip)
		}
		_, err = self.attach2NetworkDesc(ctx, userCred, self.GetHost(), nic.ToNetConfig(net), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) importDisks(ctx context.Context, userCred mcclient.TokenCredential, disks []SImportDisk) error {
	if len(disks) == 0 {
		return httperrors.NewInputParameterError("Empty import disks")
	}
	for _, disk := range disks {
		disk, err := self.createDiskOnHost(ctx, userCred, self.GetHost(), disk.ToDiskConfig(), nil, true, true)
		if err != nil {
			return err
		}
		disk.SetStatus(userCred, DISK_READY, "")
	}
	return nil
}
