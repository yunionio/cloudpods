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

package models

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/userdata"
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
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (self *SGuest) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "vnc")
}

func (self *SGuest) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM}) {
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
	data jsonutils.JSONObject,
) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "monitor")
}

func (self *SGuest) PerformMonitor(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM}) {
		cmd, err := data.GetString("command")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("command")
		}
		return self.SendMonitorCommand(ctx, userCred, cmd)
	}
	return nil, httperrors.NewInvalidStatusError("Cannot send command in status %s", self.Status)
}

func (self *SGuest) AllowPerformEvent(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, self, "event")
}

func (self *SGuest) PerformEvent(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	event, err := data.GetString("event")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("event")
	}
	if event == "GUEST_PANICKED" {
		kwargs := jsonutils.NewDict()
		kwargs.Set("reason", data)

		db.OpsLog.LogEvent(self, db.ACT_GUEST_PANICKED, data.String(), userCred)
		logclient.AddSimpleActionLog(self, logclient.ACT_GUEST_PANICKED, data.String(), userCred, true)
		self.NotifyServerEvent(
			userCred,
			notifyclient.SERVER_PANICKED,
			notify.NotifyPriorityNormal,
			false, kwargs, true,
		)
	}
	return nil, nil
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
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY}) {
		return nil, httperrors.NewInputParameterError("Cannot save image in status %s", self.Status)
	} else if !data.Contains("name") {
		return nil, httperrors.NewInputParameterError("Image name is required")
	} else if disks := self.CategorizeDisks(); disks.Root == nil {
		return nil, httperrors.NewInputParameterError("No root image")
	} else {
		kwargs := data.(*jsonutils.JSONDict)
		restart := (self.Status == api.VM_RUNNING) || jsonutils.QueryBoolean(data, "auto_start", false)
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

func (self *SGuest) AllowPerformSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {

	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "save-guest-image")
}

func (self *SGuest) PerformSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY}) {
		return nil, httperrors.NewBadRequestError("Cannot save image in status %s", self.Status)
	}
	if !data.Contains("name") && !data.Contains("generate_name") {
		return nil, httperrors.NewMissingParameterError("Image name is required")
	}
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Support only by KVM Hypervisor")
	}
	disks := self.CategorizeDisks()

	if disks.Root == nil {
		return nil, httperrors.NewInternalServerError("No root image")
	}

	// build images
	images := jsonutils.NewArray()
	diskList := append(disks.Data, disks.Root)
	for _, disk := range diskList {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(disk.DiskFormat), "disk_format")
		params.Add(jsonutils.NewInt(int64(disk.DiskSize)), "virtual_size")
		images.Add(params)
	}

	// build parameters
	kwargs := data.(*jsonutils.JSONDict)

	kwargs.Add(jsonutils.NewInt(int64(len(disks.Data)+1)), "image_number")
	properties := jsonutils.NewDict()
	if notes, err := kwargs.GetString("notes"); err != nil && len(notes) > 0 {
		properties.Add(jsonutils.NewString(notes), "notes")
	}
	osType := self.OsType
	if len(osType) == 0 {
		osType = "Linux"
	}
	properties.Add(jsonutils.NewString(osType), "os_type")
	kwargs.Add(properties, "properties")

	kwargs.Add(images, "images")

	s := auth.GetSession(ctx, userCred, options.Options.Region, "")
	ret, err := modules.GuestImages.Create(s, kwargs)
	if err != nil {
		return nil, err
	}
	imageIds, err := ret.Get("image_ids")
	if err != nil {
		return nil, fmt.Errorf("something wrong in glance")
	}
	tmp := imageIds.(*jsonutils.JSONArray)
	if tmp.Length() != len(disks.Data)+1 {
		return nil, fmt.Errorf("create subimage of guest image error")
	}
	taskParams := jsonutils.NewDict()
	if restart, _ := kwargs.Bool("auto_start"); restart {
		taskParams.Add(jsonutils.JSONTrue, "auto_start")
	}
	taskParams.Add(imageIds, "image_ids")
	return nil, self.StartGuestSaveGuestImage(ctx, userCred, taskParams, "")
}

func (self *SGuest) StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	return self.GetDriver().StartGuestSaveGuestImage(ctx, userCred, self, data, parentTaskId)
}

func (self *SGuest) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sync")

}

func (self *SGuest) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
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
	if self.GetHypervisor() != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
	}
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	isRescueMode := jsonutils.QueryBoolean(data, "rescue_mode", false)
	if !isRescueMode && self.Status != api.VM_READY {
		return nil, httperrors.NewServerStatusError("Cannot normal migrate guest in status %s, try rescue mode or server-live-migrate?", self.Status)
	}
	if isRescueMode {
		guestDisks := self.GetDisks()
		for _, guestDisk := range guestDisks {
			if utils.IsInStringArray(
				guestDisk.GetDisk().GetStorage().StorageType, api.STORAGE_LOCAL_TYPES) {
				return nil, httperrors.NewBadRequestError("Rescue mode requires all disk store in shared storages")
			}
		}
	}
	devices := self.GetIsolatedDevices()
	if len(devices) > 0 {
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

	autoStart := jsonutils.QueryBoolean(data, "auto_start", false)
	return nil, self.StartMigrateTask(ctx, userCred, isRescueMode, autoStart, self.Status, preferHostId, "")
}

func (self *SGuest) StartMigrateTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	isRescueMode, autoStart bool, guestStatus, preferHostId, parentTaskId string,
) error {
	self.SetStatus(userCred, api.VM_START_MIGRATE, "")
	data := jsonutils.NewDict()
	if isRescueMode {
		data.Set("is_rescue_mode", jsonutils.JSONTrue)
	}
	if len(preferHostId) > 0 {
		data.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	if autoStart {
		data.Set("auto_start", jsonutils.JSONTrue)
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
	if self.GetHypervisor() != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
	}
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup, can't migrate")
	}
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_SUSPEND}) {
		cdrom := self.getCdrom(false)
		if cdrom != nil && len(cdrom.ImageId) > 0 {
			return nil, httperrors.NewBadRequestError("Cannot live migrate with cdrom")
		}
		devices := self.GetIsolatedDevices()
		if devices != nil && len(devices) > 0 {
			return nil, httperrors.NewBadRequestError("Cannot live migrate with isolated devices")
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
	self.SetStatus(userCred, api.VM_START_MIGRATE, "")
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

func (self *SGuest) AllowPerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "clone")
}

func (self *SGuest) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Can't clone guest with backup guest")
	}

	if !self.GetDriver().IsSupportGuestClone() {
		return nil, httperrors.NewBadRequestError("Guest hypervisor %s does not support clone", self.Hypervisor)
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot clone VM in status %s", self.Status)
	}

	cloneInput, err := cmdline.FetchServerCreateInputByJSON(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input error %s", err)
	}
	if len(cloneInput.Name) == 0 {
		return nil, httperrors.NewMissingParameterError("name")
	}
	err = db.NewNameValidator(GuestManager, userCred, cloneInput.Name, "")
	if err != nil {
		return nil, err
	}

	createInput := self.ToCreateInput(userCred)
	createInput.Name = cloneInput.Name
	createInput.AutoStart = cloneInput.AutoStart

	createInput.EipBw = cloneInput.EipBw
	createInput.Eip = cloneInput.Eip
	createInput.EipChargeType = cloneInput.EipChargeType
	if err := GuestManager.validateEip(userCred, createInput, createInput.PreferRegion); err != nil {
		return nil, err
	}

	dataDict := createInput.JSON(createInput)
	// ownerId := db.SOwnerId{DomainId: createInput.Domain, ProjectId: createInput.Project}
	model, err := db.DoCreate(GuestManager, ctx, userCred, query, dataDict, userCred)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	func() {
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)

		model.PostCreate(ctx, userCred, userCred, query, dataDict)
	}()

	db.OpsLog.LogEvent(model, db.ACT_CREATE, model.GetShortDesc(ctx), userCred)
	logclient.AddActionLogWithContext(ctx, model, logclient.ACT_CREATE, "", userCred, true)

	pendingUsage, pendingRegionUsage := getGuestResourceRequirements(ctx, userCred, *createInput, userCred, 1, false)
	task, err := taskman.TaskManager.NewTask(ctx,
		"GuestCloneTask",
		model.(db.IStandaloneModel),
		userCred,
		dataDict,
		"", "",
		&pendingUsage, &pendingRegionUsage)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SGuest) AllowGetDetailsCreateParams(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "create-params")
}

func (self *SGuest) GetDetailsCreateParams(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := self.ToCreateInput(userCred)
	return input.JSON(input), nil
}

func (self *SGuest) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "deploy")
}

func (self *SGuest) saveOldPassword(ctx context.Context, userCred mcclient.TokenCredential) {
	loginKey := self.GetMetadata(api.VM_METADATA_LOGIN_KEY, userCred)
	if len(loginKey) > 0 {
		password, err := utils.DescryptAESBase64(self.Id, loginKey)
		if err == nil && len(password) <= 30 {
			for _, r := range password {
				if !unicode.IsPrint(r) {
					return
				}
			}
			self.SetMetadata(ctx, api.VM_METADATA_LAST_LOGIN_KEY, loginKey, userCred)
		}
	}
}

func (self *SGuest) GetOldPassword(ctx context.Context, userCred mcclient.TokenCredential) string {
	loginSecret := self.GetMetadata(api.VM_METADATA_LAST_LOGIN_KEY, userCred)
	password, _ := utils.DescryptAESBase64(self.Id, loginSecret)
	return password
}

func (self *SGuest) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	kwargs, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Parse query body error")
	}

	self.saveOldPassword(ctx, userCred)

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
		if (doRestart && self.Status == api.VM_RUNNING) || (self.Status != api.VM_RUNNING && (jsonutils.QueryBoolean(kwargs, "auto_start", false) || jsonutils.QueryBoolean(kwargs, "restart", false))) {
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

	attached, err := disk.isAttached()
	if err != nil {
		return httperrors.NewInternalServerError("isAttached check failed %s", err)
	}
	if attached {
		return httperrors.NewInputParameterError("Disk %s has been attached", disk.Name)
	}
	if len(disk.GetPathAtHost(self.GetHost())) == 0 {
		return httperrors.NewInputParameterError("Disk %s not belong the guest's host", disk.Name)
	}
	if disk.Status != api.DISK_READY {
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

	self.SetStatus(userCred, api.VM_ATTACH_DISK, "")

	if err := self.GetDriver().StartGuestAttachDiskTask(ctx, userCred, self, taskData, ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, fwOnly bool, parentTaskId string) error {

	data := jsonutils.NewDict()
	if fwOnly {
		data.Add(jsonutils.JSONTrue, "fw_only")
	} else if err := self.SetStatus(userCred, api.VM_SYNC_CONFIG, ""); err != nil {
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
	if self.Status == api.VM_RUNNING {
		err := self.StartSuspendTask(ctx, userCred, "")
		return nil, err
	}
	return nil, httperrors.NewInvalidStatusError("Cannot suspend VM in status %s", self.Status)
}

func (self *SGuest) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	err := self.SetStatus(userCred, api.VM_START_SUSPEND, "do suspend")
	if err != nil {
		return err
	}
	return self.GetDriver().StartSuspendTask(ctx, userCred, self, nil, parentTaskId)
}

func (self *SGuest) AllowPerformResume(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "resume")
}

func (self *SGuest) PerformResume(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerResumeInput) (jsonutils.JSONObject, error) {
	if self.Status == api.VM_SUSPEND {
		err := self.StartResumeTask(ctx, userCred, "")
		return nil, err
	}
	return nil, httperrors.NewInvalidStatusError("Cannot resume VM in status %s", self.Status)
}

func (self *SGuest) StartResumeTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	err := self.SetStatus(userCred, api.VM_RESUMING, "do resume")
	if err != nil {
		return err
	}
	return self.GetDriver().StartResumeTask(ctx, userCred, self, nil, parentTaskId)
}

func (self *SGuest) AllowPerformStart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "start")
}

func (self *SGuest) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_START_FAILED, api.VM_SAVE_DISK_FAILED, api.VM_SUSPEND}) {
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
	self.SetStatus(userCred, api.VM_START_DEPLOY, "")
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

func (self *SGuest) NotifyServerEvent(
	userCred mcclient.TokenCredential, event string, priority notify.TNotifyPriority,
	loginInfo bool, kwargs *jsonutils.JSONDict, notifyAdmin bool,
) {
	meta, err := self.GetAllMetadata(nil)
	if err != nil {
		return
	}
	if kwargs == nil {
		kwargs = jsonutils.NewDict()
	}

	kwargs.Add(jsonutils.NewString(self.Name), "name")
	kwargs.Add(jsonutils.NewString(self.Hypervisor), "hypervisor")
	host := self.GetHost()
	if host != nil {
		kwargs.Add(jsonutils.NewString(host.GetBrand()), "brand")
	}
	if loginInfo {
		kwargs.Add(jsonutils.NewString(self.getNotifyIps()), "ips")
		osName := meta[api.VM_METADATA_OS_NAME]
		if osName == "Windows" {
			kwargs.Add(jsonutils.JSONTrue, "windows")
		}
		loginAccount := meta[api.VM_METADATA_LOGIN_ACCOUNT]
		if len(loginAccount) > 0 {
			kwargs.Add(jsonutils.NewString(loginAccount), "account")
		}
		keypair := self.getKeypairName()
		if len(keypair) > 0 {
			kwargs.Add(jsonutils.NewString(keypair), "keypair")
		} else {
			loginKey := meta[api.VM_METADATA_LOGIN_KEY]
			if len(loginKey) > 0 {
				passwd, err := utils.DescryptAESBase64(self.Id, loginKey)
				if err == nil {
					kwargs.Add(jsonutils.NewString(passwd), "password")
				}
			}
		}
	}
	notifyclient.Notify([]string{userCred.GetUserId()}, false, priority, event, kwargs)
	if notifyAdmin {
		notifyclient.SystemNotify(priority, event, kwargs)
	}
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
		self.SetStatus(userCred, api.VM_START_STOP, "")
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
	cdrom := self.getCdrom(true)
	return cdrom.insertIso(imageId)
}

func (self *SGuest) InsertIsoSucc(imageId string, path string, size int, name string) bool {
	cdrom := self.getCdrom(false)
	return cdrom.insertIsoSucc(imageId, path, size, name)
}

func (self *SGuest) GetDetailsIso(userCred mcclient.TokenCredential) jsonutils.JSONObject {
	cdrom := self.getCdrom(false)
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
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	cdrom := self.getCdrom(false)
	if cdrom != nil && len(cdrom.ImageId) > 0 {
		return nil, httperrors.NewBadRequestError("CD-ROM not empty, please eject first")
	}
	imageId, _ := data.GetString("image_id")
	image, err := parseIsoInfo(ctx, userCred, imageId)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		err = self.StartInsertIsoTask(ctx, image.Id, false, self.HostId, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Insert ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformEjectiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformEjectiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	cdrom := self.getCdrom(false)
	if cdrom == nil || len(cdrom.ImageId) == 0 {
		return nil, httperrors.NewBadRequestError("No ISO to eject")
	}
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
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

func (self *SGuest) StartInsertIsoTask(ctx context.Context, imageId string, boot bool, hostId string, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.insertIso(imageId)

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	data.Add(jsonutils.NewString(hostId), "host_id")
	if boot {
		data.Add(jsonutils.JSONTrue, "boot")
	}

	task, err := taskman.TaskManager.NewTask(ctx, "GuestInsertIsoTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) IsDisksShared() bool {
	return self.getDefaultStorageType() == api.STORAGE_RBD
}

func (self *SGuest) StartGueststartTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string,
) error {
	if self.Hypervisor == api.HYPERVISOR_KVM && self.IsDisksShared() {
		return self.GuestSchedStartTask(ctx, userCred, data, parentTaskId)
	} else {
		return self.GuestNonSchedStartTask(ctx, userCred, data, parentTaskId)
	}
}

func (self *SGuest) GuestSchedStartTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string,
) error {
	self.SetStatus(userCred, api.VM_SCHEDULE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSchedStartTask", self, userCred, data,
		parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) GuestNonSchedStartTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string,
) error {
	self.SetStatus(userCred, api.VM_START_START, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStartTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGuestCreateTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput, pendingUsage quotas.IQuota, parentTaskId string) error {
	return self.GetDriver().StartGuestCreateTask(self, ctx, userCred, input.JSON(input), pendingUsage, parentTaskId)
}

func (self *SGuest) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return self.GetDriver().StartGuestSyncstatusTask(self, ctx, userCred, parentTaskId)
}

func (self *SGuest) StartAutoDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	db.OpsLog.LogEvent(self, db.ACT_DELETE, "auto-delete after stop", userCred)
	return self.StartDeleteGuestTask(ctx, userCred, parentTaskId, false, false, false)
}

func (self *SGuest) StartDeleteGuestTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string,
	isPurge, overridePendingDelete, deleteSnapshots bool,
) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.Status), "guest_status")
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	if deleteSnapshots {
		params.Add(jsonutils.JSONTrue, "delete_snapshots")
	}
	self.SetStatus(userCred, api.VM_START_DELETE, "")
	return self.GetDriver().StartDeleteGuestTask(ctx, userCred, self, params, parentTaskId)
}

func (self *SGuest) AllowPerformAddSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformAddSecgroup(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
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
	if len(originSecgroups)+len(secgrpJsonArray) > maxCount {
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
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
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
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot assign security rules in status %s", self.Status)
	}
	secgrpV := validators.NewModelIdOrNameValidator("secgrp", "secgroup", userCred)
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
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
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
	err = self.StartDeleteGuestTask(ctx, userCred, "", true, false, false)
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

// 重装系统(更换系统镜像)
func (self *SGuest) PerformRebuildRoot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ServerRebuildRootInput) (*api.SGuest, error) {
	imageId := input.GetImageName()

	if len(imageId) > 0 {
		img, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
		if err != nil {
			return nil, httperrors.NewNotFoundError("failed to find %s", imageId)
		}
		diskCat := self.CategorizeDisks()
		if img.MinDiskMB == 0 || img.Status != imageapi.IMAGE_STATUS_ACTIVE {
			return nil, httperrors.NewInputParameterError("invlid image")
		}
		if img.MinDiskMB > diskCat.Root.DiskSize {
			return nil, httperrors.NewInputParameterError("image size exceeds root disk size")
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

	autoStart := false
	if input.AutoStart != nil {
		autoStart = *input.AutoStart
	}
	var needStop = false
	if self.Status == api.VM_RUNNING {
		needStop = true
	}
	resetPasswd := true
	if input.ResetPassword != nil {
		resetPasswd = *input.ResetPassword
	}
	passwd := input.Password
	if len(passwd) > 0 {
		if !seclib2.MeetComplxity(passwd) {
			return nil, httperrors.NewWeakPasswordError()
		}
	}

	keypairStr := input.GetKeypairName()
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
		resetPasswd = true
	} else {
		err = self.setKeypairId(userCred, "")
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}

	allDisks := false
	if input.AllDisks != nil {
		allDisks = *input.AllDisks
	}

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
	self.SetStatus(userCred, api.VM_REBUILD_ROOT, "request start rebuild root")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestRebuildRootTask", self, userCred, data, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
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
	disksConf := make([]*api.DiskConfig, 0)
	diskSizes := make(map[string]int, 0)

	diskDefArray, err := cmdline.FetchDiskConfigsByJSON(data)
	if err != nil {
		return nil, err
	}
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
		disksConf = append(disksConf, diskInfo)
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
		if storage.GetCapacity() > 0 && storage.GetCapacity() < int64(size) {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, "Not eough storage space on current host", userCred, false)
			return nil, httperrors.NewBadRequestError("Not eough storage space on current host")
		}
	}
	pendingUsage := &SQuota{
		Storage: diskSize,
	}
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)

	err = self.CreateDisksOnHost(ctx, userCred, host, disksConf, pendingUsage, false, false, nil, nil, false)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	err = self.StartGuestCreateDiskTask(ctx, userCred, disksConf, "")
	return nil, err
}

func (self *SGuest) StartGuestCreateDiskTask(ctx context.Context, userCred mcclient.TokenCredential, disks []*api.DiskConfig, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.Marshal(disks), "disks")
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
		attached, err := self.isAttach2Disk(disk)
		if err != nil {
			return nil, httperrors.NewInternalServerError("check isAttach2Disk fail %s", err)
		}
		if attached {
			if disk.DiskType == api.DISK_TYPE_SYS {
				return nil, httperrors.NewUnsupportOperationError("Cannot detach sys disk")
			}
			detachDiskStatus, err := self.GetDriver().GetDetachDiskStatus()
			if err != nil {
				return nil, err
			}
			if keepDisk && !self.GetDriver().CanKeepDetachDisk() {
				return nil, httperrors.NewInputParameterError("Cannot keep detached disk")
			}

			err = self.GetDriver().ValidateDetachDisk(ctx, userCred, self, disk)
			if err != nil {
				return nil, err
			}

			if utils.IsInStringArray(self.Status, detachDiskStatus) {
				self.SetStatus(userCred, api.VM_DETACH_DISK, "")
				err = self.StartGuestDetachdiskTask(ctx, userCred, disk, keepDisk, "", false)
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

func (self *SGuest) StartGuestDetachdiskTask(
	ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, keepDisk bool, parentTaskId string, purge bool,
) error {
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(disk.Id), "disk_id")
	taskData.Add(jsonutils.NewBool(keepDisk), "keep_disk")
	taskData.Add(jsonutils.NewBool(purge), "purge")
	if utils.IsInStringArray(disk.Status, []string{api.DISK_INIT, api.DISK_ALLOC_FAILED}) {
		//删除非正常状态下的disk
		taskData.Add(jsonutils.JSONFalse, "keep_disk")
		db.Update(disk, func() error {
			disk.AutoDelete = true
			return nil
		})
	}
	disk.SetStatus(userCred, api.DISK_DETACHING, "")
	return self.GetDriver().StartGuestDetachdiskTask(ctx, userCred, self, taskData, parentTaskId)
}

func (self *SGuest) AllowPerformDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "detach-isolated-device")
}

func (self *SGuest) PerformDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != api.VM_READY {
		msg := "Only allowed to attach isolated device when guest is ready"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewInvalidStatusError(msg)
	}
	var detachAllDevice = jsonutils.QueryBoolean(data, "detach_all", false)
	if !detachAllDevice {
		device, err := data.GetString("device")
		if err != nil {
			msg := "Missing isolated device"
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
			return nil, httperrors.NewBadRequestError(msg)
		}
		err = self.startDetachIsolateDevice(ctx, userCred, device)
		if err != nil {
			return nil, err
		}
	} else {
		devs := self.GetIsolatedDevices()
		host := self.GetHost()
		lockman.LockObject(ctx, host)
		defer lockman.ReleaseObject(ctx, host)
		for i := 0; i < len(devs); i++ {
			err := self.detachIsolateDevice(ctx, userCred, &devs[i])
			if err != nil {
				return nil, err
			}
		}
	}
	if jsonutils.QueryBoolean(data, "auto_start", false) {
		return self.PerformStart(ctx, userCred, query, data)
	}
	return nil, nil
}

func (self *SGuest) startDetachIsolateDevice(ctx context.Context, userCred mcclient.TokenCredential, device string) error {
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msg := fmt.Sprintf("Isolated device %s not found", device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return httperrors.NewBadRequestError(msg)
	}
	dev := iDev.(*SIsolatedDevice)
	host := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	err = self.detachIsolateDevice(ctx, userCred, dev)
	return err
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
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != api.VM_READY {
		msg := "Only allowed to attach isolated device when guest is ready"
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewInvalidStatusError(msg)
	}
	var err error
	if data.Contains("device") {
		device, _ := data.GetString("device")
		err = self.startAttachIsolatedDevice(ctx, userCred, device)
	} else if data.Contains("model") {
		vmodel, _ := data.GetString("model")
		var count int64 = 1
		if data.Contains("count") {
			count, _ = data.Int("count")
		}
		if count < 1 {
			return nil, httperrors.NewBadRequestError("guest attach gpu count must > 0")
		}
		err = self.startAttachIsolatedDevices(ctx, userCred, vmodel, int(count))
	} else {
		return nil, httperrors.NewMissingParameterError("device||model")
	}

	if err != nil {
		return nil, err
	}
	if jsonutils.QueryBoolean(data, "auto_start", false) {
		return self.PerformStart(ctx, userCred, query, data)
	}
	return nil, nil
}

func (self *SGuest) startAttachIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, gpuModel string, count int) error {
	host := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	devs, err := IsolatedDeviceManager.GetDevsOnHost(host.Id, gpuModel, count)
	if err != nil {
		return httperrors.NewInternalServerError("fetch gpu failed %s", err)
	}
	if len(devs) == 0 || len(devs) != count {
		return httperrors.NewBadRequestError("guest %s host %s isolated device not enough", self.GetName(), host.GetName())
	}
	defer func() { go host.ClearSchedDescCache() }()
	for i := 0; i < len(devs); i++ {
		err = self.attachIsolatedDevice(ctx, userCred, &devs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) startAttachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, device string) error {
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msg := fmt.Sprintf("Isolated device %s not found", device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return httperrors.NewBadRequestError(msg)
	}
	dev := iDev.(*SIsolatedDevice)
	host := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	err = self.attachIsolatedDevice(ctx, userCred, dev)
	var msg string
	if err != nil {
		msg = err.Error()
	} else {
		go host.ClearSchedDescCache()
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, err == nil)
	return err
}

func (self *SGuest) attachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, dev *SIsolatedDevice) error {
	if len(dev.GuestId) > 0 {
		return fmt.Errorf("Isolated device already attached to another guest: %s", dev.GuestId)
	}
	if dev.HostId != self.HostId {
		return fmt.Errorf("Isolated device and guest are not located in the same host")
	}
	_, err := db.Update(dev, func() error {
		dev.GuestId = self.Id
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_ATTACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) AllowPerformSetIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "set-isolated-device")
}

func (self *SGuest) PerformSetIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("Only allowed to attach isolated device when guest is ready")
	}
	var addDevs []string
	{
		addDevices, err := data.Get("add_devices")
		if err == nil {
			arrAddDev, ok := addDevices.(*jsonutils.JSONArray)
			if ok {
				addDevs = arrAddDev.GetStringArray()
			} else {
				return nil, httperrors.NewInputParameterError("attach devices is not string array")
			}
		}
	}

	var delDevs []string
	{
		delDevices, err := data.Get("del_devices")
		if err == nil {
			arrDelDev, ok := delDevices.(*jsonutils.JSONArray)
			if ok {
				delDevs = arrDelDev.GetStringArray()
			} else {
				return nil, httperrors.NewInputParameterError("detach devices is not string array")
			}
		}
	}

	// detach first
	for i := 0; i < len(delDevs); i++ {
		err := self.startDetachIsolateDevice(ctx, userCred, delDevs[i])
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(addDevs); i++ {
		err := self.startAttachIsolatedDevice(ctx, userCred, addDevs[i])
		if err != nil {
			return nil, err
		}
	}
	if jsonutils.QueryBoolean(data, "auto_start", false) {
		return self.PerformStart(ctx, userCred, query, data)
	}
	return nil, nil
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

// Change IPaddress of a guestnetwork
// first detach the network, then attach a network with identity mac address but different IP configurations
// TODO change IP address of a teaming NIC may fail!!
func (self *SGuest) PerformChangeIpaddr(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_READY && self.Status != api.VM_RUNNING {
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
		return nil, httperrors.NewMissingParameterError("net_desc")
	}
	conf, err := cmdline.ParseNetworkConfigByJSON(netDesc, -1)
	if err != nil {
		return nil, err
	}
	if conf.BwLimit == 0 {
		conf.BwLimit = gn.BwLimit
	}
	if conf.Index == 0 {
		conf.Index = int(gn.Index)
	}
	conf, err = parseNetworkInfo(userCred, conf)
	if err != nil {
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
				if self.Status != api.VM_READY {
					// change mac
					return nil, httperrors.NewInvalidStatusError("cannot change mac when guest is running")
				}
				// check mac duplication
				cnt, err := GuestnetworkManager.Query().Equals("mac_addr", conf.Mac).CountWithError()
				if err != nil {
					return nil, httperrors.NewInternalServerError("check mac uniqueness fail %s", err)
				}
				if cnt > 0 {
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
		ngn, err := self.attach2NetworkDesc(ctx, userCred, host, conf, nil, nil)
		if err != nil {
			// recover detached guestnetwork
			conf2 := gn.ToNetworkConfig()
			if reserve {
				conf2.Reserved = true
			}
			_, err2 := self.attach2NetworkDesc(ctx, userCred, host, conf2, nil, nil)
			if err2 != nil {
				log.Errorf("recover detached network fail %s", err2)
			}
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
	if self.Status != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("Cannot detach network in status %s", self.Status)
	}
	reserve := jsonutils.QueryBoolean(data, "reserve", false)

	netStr, _ := data.GetString("net_id")
	if len(netStr) > 0 {
		netObj, err := NetworkManager.FetchByIdOrName(userCred, netStr)
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
	if self.Status == api.VM_READY {
		// owner_cred = self.get_owner_user_cred() >.<
		netDesc, err := data.Get("net_desc")
		if err != nil {
			return nil, httperrors.NewBadRequestError(err.Error())
		}
		conf, err := cmdline.ParseNetworkConfigByJSON(netDesc, -1)
		if err != nil {
			return nil, httperrors.NewBadRequestError(err.Error())
		}

		conf, err = parseNetworkInfo(userCred, conf)
		if err != nil {
			return nil, err
		}
		err = isValidNetworkInfo(userCred, conf)
		if err != nil {
			return nil, err
		}
		var inicCnt, enicCnt int
		if IsExitNetworkInfo(conf) {
			enicCnt = 1
			// ebw = conf.BwLimit
		} else {
			inicCnt = 1
			// ibw = conf.BwLimit
		}
		pendingUsage := &SRegionQuota{
			Port:  inicCnt,
			Eport: enicCnt,
			//Bw:    ibw,
			//Ebw:   ebw,
		}
		keys, err := self.GetRegionalQuotaKeys()
		if err != nil {
			return nil, err
		}
		pendingUsage.SetKeys(keys)
		err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
		if err != nil {
			return nil, httperrors.NewOutOfQuotaError(err.Error())
		}
		host := self.GetHost()
		_, err = self.attach2NetworkDesc(ctx, userCred, host, conf, pendingUsage, nil)
		if err != nil {
			quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
			return nil, httperrors.NewBadRequestError(err.Error())
		}
		host.ClearSchedDescCache()
		err = self.StartGuestDeployTask(ctx, userCred, nil, "deploy", "")
		if err != nil {
			quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
		}
		return nil, err
	}
	return nil, httperrors.NewBadRequestError("Cannot attach network in status %s", self.Status)
}

func (self *SGuest) AllowPerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "change-bandwidth")
}

func (self *SGuest) PerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
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

func (self *SGuest) AllowPerformModifySrcCheck(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "modify-src-check")
}

func (self *SGuest) PerformModifySrcCheck(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		msg := fmt.Sprintf("Cannot change seting in status %s", self.Status)
		return nil, httperrors.NewBadRequestError(msg)
	}

	argValue := func(name string, val *bool) error {
		if !data.Contains(name) {
			return nil
		}
		b, err := data.Bool(name)
		if err != nil {
			return errors.Wrapf(err, "fetch arg %s", name)
		}
		*val = b
		return nil
	}
	var (
		srcIpCheck  = self.SrcIpCheck.Bool()
		srcMacCheck = self.SrcMacCheck.Bool()
	)
	if err := argValue("src_ip_check", &srcIpCheck); err != nil {
		return nil, err
	}
	if err := argValue("src_mac_check", &srcMacCheck); err != nil {
		return nil, err
	}
	// default: both check on
	// switch: mac check off, also implies ip check off
	// router: mac check on, ip check off
	if !srcMacCheck && srcIpCheck {
		srcIpCheck = false
	}

	if srcIpCheck != self.SrcIpCheck.Bool() || srcMacCheck != self.SrcMacCheck.Bool() {
		diff, err := db.Update(self, func() error {
			self.SrcIpCheck = tristate.NewFromBool(srcIpCheck)
			self.SrcMacCheck = tristate.NewFromBool(srcMacCheck)
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_GUEST_SRC_CHECK, diff, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_SRC_CHECK, diff, userCred, true)
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
		sku, err := ServerSkuManager.FetchSkuByNameAndProvider(skuId, self.GetDriver().GetProvider(), true)
		if err != nil {
			return nil, err
		}

		if self.GetDriver().GetProvider() == api.CLOUD_PROVIDER_UCLOUD && !strings.HasPrefix(self.InstanceType, sku.InstanceTypeFamily) {
			return nil, httperrors.NewInputParameterError("Cannot change config with different instance family")
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

	if self.Status == api.VM_RUNNING && (cpuChanged || memChanged) && self.GetDriver().NeedStopForChangeSpec(self, cpuChanged, memChanged) {
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
	var newDiskIdx = 0
	var diskSizes = make(map[string]int, 0)
	var newDisks = make([]*api.DiskConfig, 0)
	var resizeDisks = jsonutils.NewArray()

	var inputDisks = make([]*api.DiskConfig, 0)
	if disksConf, err := data.Get("disks"); err == nil {
		if err = disksConf.Unmarshal(&inputDisks); err != nil {
			return nil, httperrors.NewInputParameterError("Unmarshal disks configure error %s", err)
		}
	}

	var diskIdx = 1
	for _, diskConf := range inputDisks {
		diskConf, err = parseDiskInfo(ctx, userCred, diskConf)
		if err != nil {
			return nil, httperrors.NewBadRequestError("Parse disk info error: %s", err)
		}
		if len(diskConf.Backend) == 0 {
			diskConf.Backend = self.getDefaultStorageType()
		}
		if diskConf.SizeMb > 0 {
			if diskIdx >= len(disks) {
				// 这里backeend为空时,qcloud有可能会选择local_ssd作为后端存储,会导致报错(主要是climc)
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
				newDisks = append(newDisks, diskConf)
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
					err = self.ValidateResizeDisk(disk, storage)
					if err != nil {
						return nil, httperrors.NewUnsupportOperationError(err.Error())
					}
					if !storage.IsEmulated && storage.GetFreeCapacity() < int64(addDisk) {
						return nil, httperrors.NewInsufficientResourceError("Not enough free space")
					}
					diskSizes[storage.Id] = diskSizes[storage.Id] + diskConf.SizeMb - oldSize
				}
			}
		}
		diskIdx += 1
	}

	provider, e := self.GetHost().GetProviderFactory()
	if e != nil || !provider.IsPublicCloud() {
		for storageId, needSize := range diskSizes {
			iStorage, err := StorageManager.FetchById(storageId)
			if err != nil {
				return nil, httperrors.NewBadRequestError("Fetch storage error: %s", err)
			}
			storage := iStorage.(*SStorage)
			if !storage.IsEmulated && storage.GetFreeCapacity() < int64(needSize) {
				return nil, httperrors.NewInsufficientResourceError("Not enough free space")
			}
		}
	} else {
		log.Debugf("Skip storage free capacity validating for public cloud: %s", provider.GetId())
	}

	if resizeDisks.Length() > 0 {
		confs.Add(resizeDisks, "resize")
	}
	if self.Status != api.VM_RUNNING && jsonutils.QueryBoolean(data, "auto_start", false) {
		confs.Add(jsonutils.NewBool(true), "auto_start")
	}
	if self.Status == api.VM_RUNNING {
		confs.Set("guest_online", jsonutils.JSONTrue)
	}

	err = self.GetDriver().ValidateChangeConfig(ctx, userCred, self, cpuChanged, memChanged, newDisks)
	if err != nil {
		return nil, err
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

	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	pendingUsage.SetKeys(keys)
	log.Debugf("ChangeConfig pendingUsage %s", jsonutils.Marshal(pendingUsage))
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, err
	}

	if len(newDisks) > 0 {
		err := self.CreateDisksOnHost(ctx, userCred, host, newDisks, pendingUsage, false, false, nil, nil, false)
		if err != nil {
			quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
			return nil, httperrors.NewBadRequestError("Create disk on host error: %s", err)
		}
		confs.Add(jsonutils.Marshal(newDisks), "create")
	}
	self.StartChangeConfigTask(ctx, userCred, confs, "", pendingUsage)
	return nil, nil
}

func (self *SGuest) confToSchedDesc(addCpu, addMem, addDisk int) *schedapi.ScheduleInput {
	guestDisks := self.GetDisks()
	diskInfo := guestDisks[0].ToDiskConfig()
	diskInfo.SizeMb = addDisk

	desc := &schedapi.ScheduleInput{
		ServerConfig: schedapi.ServerConfig{
			ServerConfigs: &api.ServerConfigs{
				Hypervisor: self.Hypervisor,
				PreferHost: self.HostId,
				Disks:      []*api.DiskConfig{diskInfo},
			},
			Memory:  addMem,
			Ncpu:    addCpu,
			Project: self.ProjectId,
			Domain:  self.DomainId,
		},
	}
	return desc
}

func (self *SGuest) StartChangeConfigTask(ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string, pendingUsage quotas.IQuota) error {
	self.SetStatus(userCred, api.VM_CHANGE_FLAVOR, "")
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
	if secgroup := self.getSecgroup(); secgroup != nil {
		return self.revokeSecgroup(ctx, userCred, secgroup)
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
	backends := []SLoadbalancerBackend{}
	q := LoadbalancerBackendManager.Query().Equals("backend_id", self.Id).IsFalse("pending_deleted")
	err := db.FetchModelObjects(LoadbalancerBackendManager, q, &backends)
	if err != nil {
		log.Errorf("failed to get backends for guest %s(%s)", self.Name, self.Id)
	}
	for i := 0; i < len(backends); i++ {
		backends[i].DoPendingDelete(ctx, userCred)
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
	if self.BillingType == billing_api.BILLING_TYPE_POSTPAID && !self.ExpiredAt.IsZero() {
		if err := self.CancelExpireTime(ctx, userCred); err != nil {
			return err
		}
	}
	err := self.SVirtualResourceBase.DoCancelPendingDelete(ctx, userCred)
	if err != nil {
		return err
	}
	return nil
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
	if self.Status == api.VM_RUNNING || self.Status == api.VM_STOP_FAILED {
		self.GetDriver().StartGuestResetTask(self, ctx, userCred, isHard, "")
		return nil, nil
	}
	return nil, httperrors.NewInvalidStatusError("Cannot reset VM in status %s", self.Status)
}

func (self *SGuest) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SGuest) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Guest has %d task active, can't sync status", count)
	}

	self.SetStatus(userCred, api.VM_SYNCING_STATUS, "perform_syncstatus")
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SGuest) isNotRunningStatus(status string) bool {
	if status == api.VM_READY || status == api.VM_SUSPEND {
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

	status, _ := data.GetString("status")
	if len(self.BackupHostId) > 0 && status == api.VM_RUNNING {
		if len(self.GetMetadata("__mirror_job_status", userCred)) == 0 {
			self.SetMetadata(ctx, "__mirror_job_status", "ready", userCred)
		}
	} else if ispId := self.GetMetadata("__base_instance_snapshot_id", userCred); len(ispId) > 0 {
		ispM, err := InstanceSnapshotManager.FetchById(ispId)
		if err == nil {
			isp := ispM.(*SInstanceSnapshot)
			isp.DecRefCount(ctx, userCred)
		}
		self.SetMetadata(ctx, "__base_instance_snapshot_id", "", userCred)
	}

	if preStatus != self.Status && !self.isNotRunningStatus(preStatus) && self.isNotRunningStatus(self.Status) {
		db.OpsLog.LogEvent(self, db.ACT_STOP, "", userCred)
		if self.Status == api.VM_READY && !self.DisableDelete.Bool() && self.ShutdownBehavior == api.SHUTDOWN_TERMINATE {
			err = self.StartAutoDeleteGuestTask(ctx, userCred, "")
			return nil, err
		}
	}
	return nil, nil
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
	if isForce || utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_STOP_FAILED}) {
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
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_STOP_FAILED}) || (isForce && self.Status == api.VM_STOPPING) {
		return nil, self.GetDriver().StartGuestRestartTask(self, ctx, userCred, isForce, "")
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do restart server in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformSendkeys(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "sendkeys")
}

func (self *SGuest) PerformSendkeys(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if self.Status != api.VM_RUNNING {
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
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
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

	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	eipVm := eip.GetAssociateVM()
	if eipVm != nil {
		return nil, httperrors.NewConflictError("eip has been associated")
	}

	if eipRegion.Id != instRegion.Id {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different region")
	}

	if len(eip.NetworkId) > 0 {
		gns, err := self.GetNetworks("")
		if err != nil {
			return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetNetworks"))
		}
		for _, gn := range gns {
			if gn.NetworkId == eip.NetworkId {
				return nil, httperrors.NewInputParameterError("cannot associate eip with same network")
			}
		}
	}

	eipZone := eip.GetZone()
	if eipZone != nil {
		insZone := self.getZone()
		if eipZone.Id != insZone.Id {
			return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different zone")
		}
	}

	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInputParameterError("server host is not found???")
	}

	if host.ManagerId != eip.ManagerId {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different provider")
	}

	self.SetStatus(userCred, api.VM_ASSOCIATE_EIP, "associate eip")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(self.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_SERVER), "instance_type")

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

	if eip.Mode != api.EIP_MODE_STANDALONE_EIP {
		return nil, httperrors.NewNotSupportedError("%s not support dissociate", eip.Mode)
	}

	autoDelete := jsonutils.QueryBoolean(data, "auto_delete", false)
	err = self.StartGuestDissociateEipTask(ctx, userCred, eip, autoDelete, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartGuestDissociateEipTask")
	}
	return nil, nil
}

func (self *SGuest) StartGuestDissociateEipTask(ctx context.Context, userCred mcclient.TokenCredential, eip *SElasticip, autoDelete bool, parentTaskId string) error {
	self.SetStatus(userCred, api.VM_DISSOCIATE_EIP, "associate eip")
	eip.SetStatus(userCred, api.EIP_STATUS_DISSOCIATE, "start to dissociate")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(autoDelete), "auto_delete")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDissociateEipTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		eip.SetStatus(userCred, api.EIP_STATUS_READY, "")
		return errors.Wrap(err, "NewTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "create-eip")
}

func (self *SGuest) PerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var bw int64
	chargeType, _ := data.GetString("charge_type")
	if len(chargeType) == 0 {
		chargeType = api.EIP_CHARGE_TYPE_DEFAULT
	}

	bw, _ = data.Int("bandwidth")
	if chargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		if bw == 0 {
			return nil, httperrors.NewMissingParameterError("bandwidth")
		}
	}

	if len(self.ExternalId) == 0 {
		return nil, httperrors.NewInvalidStatusError("Not a managed VM")
	}
	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("No host???")
	}

	_, err := host.GetDriver()
	if err != nil {
		return nil, httperrors.NewInvalidStatusError("No valid cloud provider")
	}

	region := host.GetRegion()
	if region == nil {
		return nil, httperrors.NewInvalidStatusError("No cloudregion???")
	}

	err = self.GetDriver().ValidateCreateEip(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	eipPendingUsage := &SRegionQuota{Eip: 1}
	keys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		return nil, err
	}
	eipPendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, eipPendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Out of eip quota: %s", err)
	}

	eip, err := ElasticipManager.NewEipForVMOnHost(ctx, userCred, self, host, int(bw), chargeType, eipPendingUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, false)
		return nil, httperrors.NewGeneralError(err)
	}

	err = eip.AllocateAndAssociateVM(ctx, userCred, self, "")
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (self *SGuest) setUserData(ctx context.Context, userCred mcclient.TokenCredential, data string) error {
	if err := userdata.ValidateUserdata(data); err != nil {
		return err
	}
	encodeData, err := userdata.Encode(data)
	if err != nil {
		return errors.Wrap(err, "encode guest userdata")
	}
	err = self.SetMetadata(ctx, "user_data", encodeData, userCred)
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

func (self *SGuest) AllowPerformSetQemuParams(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "set-qemu-params")
}

func (self *SGuest) PerformSetQemuParams(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isaSerial, err := data.GetString("disable_isa_serial")
	if err == nil {
		err = self.SetMetadata(ctx, "disable_isa_serial", isaSerial, userCred)
		if err != nil {
			return nil, err
		}
	}
	pvpanic, err := data.GetString("disable_pvpanic")
	if err == nil {
		err = self.SetMetadata(ctx, "disable_pvpanic", pvpanic, userCred)
		if err != nil {
			return nil, err
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
	if self.Status == api.VM_BLOCK_STREAM {
		return nil, httperrors.NewBadRequestError("Cannot swith to backup when guest in status %s", self.Status)
	}
	if len(self.BackupHostId) == 0 {
		return nil, httperrors.NewBadRequestError("Guest no backup host")
	}

	mirrorJobStatus := self.GetMetadata("__mirror_job_status", userCred)
	if mirrorJobStatus != "ready" {
		return nil, httperrors.NewBadRequestError("Guest can't switch to backup, mirror job not ready")
	}

	oldStatus := self.Status
	self.SetStatus(userCred, api.VM_SWITCH_TO_BACKUP, "Switch to backup")
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

func (manager *SGuestManager) getGuests(userCred mcclient.TokenCredential, data jsonutils.JSONObject) ([]SGuest, error) {
	_guests := []string{}
	data.Unmarshal(&_guests, "guests")
	if len(_guests) == 0 {
		return nil, httperrors.NewMissingParameterError("guests")
	}
	guests := []SGuest{}
	q1 := manager.Query().In("id", _guests)
	q2 := manager.Query().In("name", _guests)
	q2 = manager.FilterByOwner(q2, userCred, manager.NamespaceScope())
	q2 = manager.FilterBySystemAttributes(q2, userCred, data, manager.ResourceScope())
	q := sqlchemy.Union(q1, q2).Query().Distinct()
	err := db.FetchModelObjects(manager, q, &guests)
	if err != nil {
		return nil, err
	}
	guestStr := []string{}
	for _, guest := range guests {
		guestStr = append(guestStr, guest.Id)
		guestStr = append(guestStr, guest.Name)
	}

	for _, guest := range _guests {
		if !utils.IsInStringArray(guest, guestStr) {
			return nil, httperrors.NewResourceNotFoundError("failed to found guest %s", guest)
		}
	}
	return guests, nil
}

func (manager *SGuestManager) getUserMetadata(data jsonutils.JSONObject) (map[string]interface{}, error) {
	if !data.Contains("metadata") {
		return nil, httperrors.NewMissingParameterError("metadata")
	}
	metadata, err := data.GetMap("metadata")
	if err != nil {
		return nil, httperrors.NewInputParameterError("input data not key value dict")
	}
	dictStore := map[string]interface{}{}
	for k, v := range metadata {
		dictStore["user:"+k], _ = v.GetString()
	}
	return dictStore, nil
}

func (manager *SGuestManager) AllowPerformBatchUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "batch-user-metadata")
}

func (manager *SGuestManager) PerformBatchUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guests, err := manager.getGuests(userCred, data)
	if err != nil {
		return nil, err
	}
	metadata, err := manager.getUserMetadata(data)
	if err != nil {
		return nil, err
	}
	for _, guest := range guests {
		err := guest.SetUserMetadataValues(ctx, metadata, userCred)
		if err != nil {
			msg := fmt.Errorf("set guest %s(%s) user-metadata error: %v", guest.Name, guest.Id, err)
			return nil, httperrors.NewGeneralError(msg)
		}
	}
	return jsonutils.Marshal(guests), nil
}

func (manager *SGuestManager) AllowPerformBatchSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "batch-set-user-metadata")
}

func (manager *SGuestManager) PerformBatchSetUserMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guests, err := manager.getGuests(userCred, data)
	if err != nil {
		return nil, err
	}
	metadata, err := manager.getUserMetadata(data)
	if err != nil {
		return nil, err
	}
	for _, guest := range guests {
		err := guest.SetUserMetadataAll(ctx, metadata, userCred)
		if err != nil {
			msg := fmt.Errorf("set guest %s(%s) user-metadata error: %v", guest.Name, guest.Id, err)
			return nil, httperrors.NewGeneralError(msg)
		}
	}
	return jsonutils.Marshal(guests), nil
}

func (self *SGuest) AllowPerformBlockStreamFailed(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "block-stream-failed")
}

func (self *SGuest) PerformBlockStreamFailed(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		self.SetMetadata(ctx, "__mirror_job_status", "failed", userCred)
	}
	if self.Status == api.VM_BLOCK_STREAM || self.Status == api.VM_RUNNING {
		reason, _ := data.GetString("reason")
		logclient.AddSimpleActionLog(self, logclient.ACT_VM_BLOCK_STREAM, reason, userCred, false)
		return nil, self.SetStatus(userCred, api.VM_BLOCK_STREAM_FAIL, reason)
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
		err := guest.GuestStartAndSyncToBackup(ctx, userCred, "", guest.Status)
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
	iguest, err := manager.FetchById(guestId)
	if err == sql.ErrNoRows {
		ret := jsonutils.NewDict()
		ret.Set("guest_unknown_need_clean", jsonutils.JSONTrue)
		return ret, nil
	} else {
		if err != nil {
			return nil, httperrors.NewInternalServerError("Fetch guest error %s", err)
		} else {
			guest := iguest.(*SGuest)

			hostId, _ := data.GetString("host_id")
			if len(hostId) == 0 {
				return nil, httperrors.NewMissingParameterError("host_id")
			}

			if guest.HostId != hostId && guest.BackupHostId != hostId {
				return nil, guest.StartGuestDeleteOnHostTask(ctx, userCred, hostId, false, "")
			}
		}
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

func (guest *SGuest) GuestStartAndSyncToBackup(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId, guestStatus string,
) error {
	data := jsonutils.NewDict()
	data.Set("guest_status", jsonutils.NewString(guestStatus))
	task, err := taskman.TaskManager.NewTask(
		ctx, "GuestStartAndSyncToBackupTask", guest, userCred, data, parentTaskId, "", nil)
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
	if self.getDefaultStorageType() != api.STORAGE_LOCAL {
		return nil, httperrors.NewBadRequestError("Cannot create backup with shared storage")
	}
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Backup only support hypervisor kvm")
	}
	if len(self.GetIsolatedDevices()) > 0 {
		return nil, httperrors.NewBadRequestError("Cannot create backup with isolated devices")
	}
	hasSnapshot, err := self.GuestDisksHasSnapshot()
	if err != nil {
		return nil, httperrors.NewInternalServerError("GuestDisksHasSnapshot fail %s", err)
	}
	if hasSnapshot {
		return nil, httperrors.NewBadRequestError("Cannot create backup with snapshot")
	}

	req := self.getGuestBackupResourceRequirements(ctx, userCred)
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	req.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &req)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	params := data.(*jsonutils.JSONDict)
	params.Set("guest_status", jsonutils.NewString(self.Status))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateBackupTask", self, userCred, params, "", "", &req)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &req, &req, false)
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
	if backupHost.Status == api.HOST_OFFLINE && !jsonutils.QueryBoolean(data, "purge", false) {
		return nil, httperrors.NewBadRequestError("Backup host is offline")
	}

	taskData := jsonutils.NewDict()
	taskData.Set("purge", jsonutils.NewBool(jsonutils.QueryBoolean(data, "purge", false)))
	self.SetStatus(userCred, api.VM_DELETING_BACKUP, "delete backup server")
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestDeleteBackupTask", self, userCred, taskData, "", "", nil); err != nil {
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

func (self *SGuest) AllowPerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "cancel-expire")
}

func (self *SGuest) PerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("guest billing type %s not support cancel expire", self.BillingType)
	}
	if err := self.GetDriver().CancelExpireTime(ctx, userCred, self); err != nil {
		return nil, err
	}
	guestdisks := self.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		if err := disk.CancelExpireTime(ctx, userCred); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (self *SGuest) AllowPerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "postpaid-expire")
}

func (self *SGuest) PerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("guest billing type is %s", self.BillingType)
	}

	durationStr := jsonutils.GetAnyString(data, []string{"duration"})
	if len(durationStr) == 0 {
		return nil, httperrors.NewInputParameterError("missong duration")
	}

	bc, err := billing.ParseBillingCycle(durationStr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", durationStr, err)
	}

	if !self.GetDriver().IsSupportPostpaidExpire() {
		return nil, httperrors.NewBadRequestError("guest %s unsupport postpaid expire", self.Hypervisor)
	}

	err = self.SaveRenewInfo(ctx, userCred, &bc, nil, billing_api.BILLING_TYPE_POSTPAID)
	return nil, err
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
	self.SetStatus(userCred, api.VM_RENEWING, "")
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

func (self *SGuest) SaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	err := self.doSaveRenewInfo(ctx, userCred, bc, expireAt, billingType)
	if err != nil {
		return err
	}
	guestdisks := self.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		disk := guestdisks[i].GetDisk()
		err = disk.SaveRenewInfo(ctx, userCred, bc, expireAt, billingType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) doSaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	_, err := db.Update(self, func() error {
		if billingType == "" {
			billingType = billing_api.BILLING_TYPE_PREPAID
		}
		if self.BillingType == "" {
			self.BillingType = billingType
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
		log.Errorf("UpdateItem error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) CancelExpireTime(ctx context.Context, userCred mcclient.TokenCredential) error {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return fmt.Errorf("billing type %s not support cancel expire", self.BillingType)
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"update %s set expired_at = NULL and billing_cycle = NULL where id = ?",
			GuestManager.TableSpec().Name(),
		), self.Id,
	)
	if err != nil {
		return errors.Wrap(err, "guest cancel expire time")
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, "guest cancel expire time", userCred)
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

func (man *SGuestManager) PerformImport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	desc := &api.SImportGuestDesc{}
	if err := data.Unmarshal(desc); err != nil {
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
	if err := db.NewNameValidator(man, userCred, desc.Name, ""); err != nil {
		return nil, err
	}
	if hostObj, _ := HostManager.FetchByIdOrName(userCred, desc.HostId); hostObj == nil {
		return nil, httperrors.NewNotFoundError("Host %s not found", desc.HostId)
	} else {
		desc.HostId = hostObj.GetId()
	}
	guset, err := man.DoImport(ctx, userCred, desc)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(guset), nil
}

func (man *SGuestManager) DoImport(
	ctx context.Context, userCred mcclient.TokenCredential, desc *api.SImportGuestDesc,
) (*SGuest, error) {
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
	return gst, nil
}

func (man *SGuestManager) createImportGuest(ctx context.Context, userCred mcclient.TokenCredential, desc *api.SImportGuestDesc) (*SGuest, error) {
	model, err := db.NewModelObject(man)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	gst, ok := model.(*SGuest)
	if !ok {
		return nil, httperrors.NewGeneralError(fmt.Errorf("Can't convert %#v to *SGuest model", model))
	}
	gst.ProjectId = userCred.GetProjectId()
	gst.DomainId = userCred.GetProjectDomainId()
	gst.IsSystem = desc.IsSystem
	gst.Id = desc.Id
	gst.Name = desc.Name
	gst.HostId = desc.HostId
	gst.Status = api.VM_IMPORT
	gst.Hypervisor = desc.Hypervisor
	gst.VmemSize = desc.MemSizeMb
	gst.VcpuCount = desc.Cpu
	gst.BootOrder = desc.BootOrder
	gst.Description = desc.Description
	err = man.TableSpec().Insert(gst)
	return gst, err
}

func ToNetConfig(n *api.SImportNic, net *SNetwork) *api.NetworkConfig {
	return &api.NetworkConfig{
		Network: net.Id,
		Wire:    net.WireId,
		Address: n.Ip,
		Mac:     n.Mac,
		Driver:  n.Driver,
		BwLimit: n.BandWidth,
	}
}

func (self *SGuest) importNics(ctx context.Context, userCred mcclient.TokenCredential, nics []api.SImportNic) error {
	if len(nics) == 0 {
		return httperrors.NewInputParameterError("Empty import nics")
	}
	for _, nic := range nics {
		q := GuestnetworkManager.Query()
		count := q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("mac_addr"), nic.Mac),
			sqlchemy.Equals(q.Field("ip_addr"), nic.Ip)),
		).Count()
		if count > 0 {
			return httperrors.NewInputParameterError("ip %s or mac %s has been registered", nic.Ip, nic.Mac)
		}
		net, err := NetworkManager.GetOnPremiseNetworkOfIP(nic.Ip, "", tristate.None)
		if err != nil {
			return httperrors.NewNotFoundError("Not found network by ip %s", nic.Ip)
		}
		_, err = self.attach2NetworkDesc(ctx, userCred, self.GetHost(), ToNetConfig(&nic, net), nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func ToDiskConfig(d *api.SImportDisk) *api.DiskConfig {
	ret := &api.DiskConfig{
		SizeMb:  d.SizeMb,
		ImageId: d.TemplateId,
		Format:  d.Format,
		Driver:  d.Driver,
		DiskId:  d.DiskId,
		Cache:   d.CacheMode,
		Backend: d.Backend,
	}
	if len(d.Mountpoint) > 0 {
		ret.Mountpoint = d.Mountpoint
	}
	if len(d.Fs) > 0 {
		ret.Fs = d.Fs
	}
	return ret
}

func (self *SGuest) importDisks(ctx context.Context, userCred mcclient.TokenCredential, disks []api.SImportDisk) error {
	if len(disks) == 0 {
		return httperrors.NewInputParameterError("Empty import disks")
	}
	for _, disk := range disks {
		disk, err := self.createDiskOnHost(ctx, userCred, self.GetHost(), ToDiskConfig(&disk), nil, true, true, nil, nil, true)
		if err != nil {
			return err
		}
		disk.SetStatus(userCred, api.DISK_READY, "")
	}
	return nil
}

func (manager *SGuestManager) AllowPerformImportFromLibvirt(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowClassPerform(userCred, manager, "import-from-libvirt")
}

func (manager *SGuestManager) PerformImportFromLibvirt(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	host := &api.SLibvirtHostConfig{}
	if err := data.Unmarshal(host); err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal data error %s", err)
	}
	if len(host.XmlFilePath) == 0 {
		return nil, httperrors.NewInputParameterError("Some host config missing xml_file_path")
	}
	if len(host.HostIp) == 0 {
		return nil, httperrors.NewInputParameterError("Some host config missing host ip")
	}
	sHost, err := HostManager.GetHostByIp(host.HostIp)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid host ip %s", host.HostIp)
	}

	for _, server := range host.Servers {
		for mac, ip := range server.MacIp {
			_, err = net.ParseMAC(mac)
			if err != nil {
				return nil, httperrors.NewBadRequestError("Invalid server mac address %s", mac)
			}
			nIp := net.ParseIP(ip)
			if nIp == nil {
				return nil, httperrors.NewBadRequestError("Invalid server ip address %s", ip)
			}
			q := GuestnetworkManager.Query()
			count := q.Filter(sqlchemy.OR(
				sqlchemy.Equals(q.Field("mac_addr"), mac),
				sqlchemy.Equals(q.Field("ip_addr"), ip)),
			).Count()
			if count > 0 {
				return nil, httperrors.NewInputParameterError("ip %s or mac %s has been registered", mac, ip)
			}
		}
	}

	taskData := jsonutils.NewDict()
	taskData.Set("xml_file_path", jsonutils.NewString(host.XmlFilePath))
	taskData.Set("servers", jsonutils.Marshal(host.Servers))
	taskData.Set("monitor_path", jsonutils.NewString(host.MonitorPath))
	task, err := taskman.TaskManager.NewTask(ctx, "HostImportLibvirtServersTask", sHost, userCred,
		taskData, "", "", nil)
	if err != nil {
		return nil, httperrors.NewInternalServerError("NewTask error: %s", err)
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SGuest) AllowGetDetailsVirtInstall(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "virt-install")
}

func (self *SGuest) GetDetailsVirtInstall(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Hypervisor %s can't generate libvirt xml", self.Hypervisor)
	}

	var (
		vdiProtocol   string
		vdiListenPort int64
	)

	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM}) {
		vncInfo, err := self.GetDriver().GetGuestVncInfo(ctx, userCred, self, self.GetHost())
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		vdiProtocol, _ = vncInfo.GetString("protocol")
		vdiListenPort, _ = vncInfo.Int("port")
	}

	extraCmdline, _ := query.GetArray("extra_cmdline")
	libvirtBridge, _ := query.GetString("libvirt_bridge")
	if len(libvirtBridge) == 0 {
		libvirtBridge = "virbr0"
	}
	virtInstallCmd, err := self.GenerateVirtInstallCommandLine(
		vdiProtocol, vdiListenPort, extraCmdline, libvirtBridge)
	if err != nil {
		return nil, httperrors.NewInternalServerError("Generate xml failed: %s", err)
	}

	res := jsonutils.NewDict()
	res.Set("virt-install-command-line", jsonutils.NewString(virtInstallCmd))
	return res, nil
}

func (self *SGuest) GenerateVirtInstallCommandLine(
	vdiProtocol string, vdiListenPort int64, extraCmdline []jsonutils.JSONObject, libvirtBridge string,
) (string, error) {
	L := func(s string) string { return s + " \\\n" }

	cmd := L("virt-install")
	cmd += L("--cpu host")
	cmd += L("--boot cdrom,hd,network")
	cmd += L(fmt.Sprintf("--name %s", self.Name))
	cmd += L(fmt.Sprintf("--ram %d", self.VmemSize))
	cmd += L(fmt.Sprintf("--vcpus %d", self.VcpuCount))

	host := self.GetHost()

	// disks
	guestDisks := self.GetDisks()
	for _, guestDisk := range guestDisks {
		disk := guestDisk.GetDisk()
		cmd += L(
			fmt.Sprintf("--disk path=%s,bus=%s,cache=%s,io=%s",
				disk.GetPathAtHost(host),
				guestDisk.Driver,
				guestDisk.CacheMode,
				guestDisk.AioMode),
		)
	}

	// networks
	guestNetworks, err := self.GetNetworks("")
	if err != nil {
		return "", err
	}
	for _, guestNetwork := range guestNetworks {
		cmd += L(
			fmt.Sprintf("--network bridge,model=%s,source=%s,mac=%s",
				guestNetwork.Driver, libvirtBridge, guestNetwork.MacAddr),
		)
	}

	// isolated devices
	isolatedDevices := self.GetIsolatedDevices()
	for _, isolatedDev := range isolatedDevices {
		cmd += L(fmt.Sprintf("--hostdev %s", isolatedDev.Addr))
	}

	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM}) {
		// spice or vnc
		cmd += L(fmt.Sprintf("--graphics %s,listen=0.0.0.0,port=%d", vdiProtocol, vdiListenPort))
	} else {
		// generate xml, and need virsh define xml
		cmd += L("--print-xml")
	}
	cmd += L("--video vga")
	cmd += L("--wait 0")

	// some customized options, not verify
	for _, cmdline := range extraCmdline {
		cmd += L(cmdline.String())
	}

	// debug print
	cmd += "-d"
	return cmd, nil
}

func (self *SGuest) AllowPerformSyncFixNics(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "sync-fix-nics")
}

func (self *SGuest) PerformSyncFixNics(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	iVM, err := self.GetIVM()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	vnics, err := iVM.GetINics()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInternalServerError("host not found???")
	}
	iplistArray, err := data.Get("ip")
	if err != nil {
		return nil, httperrors.NewInputParameterError("missing field ip, list of ip")
	}
	iplist := iplistArray.(*jsonutils.JSONArray).GetStringArray()
	errs := make([]error, 0)
	for i := range vnics {
		ip := vnics[i].GetIP()
		if len(ip) == 0 {
			continue
		}
		_, err := host.getNetworkOfIPOnHost(ip)
		if err != nil {
			errs = append(errs, errors.Wrap(err, ip))
		}
	}
	if len(errs) > 0 {
		return nil, httperrors.NewInvalidStatusError(errors.NewAggregate(errs).Error())
	}
	result := self.SyncVMNics(ctx, userCred, host, vnics, iplist)
	if result.IsError() {
		return nil, httperrors.NewInternalServerError(result.Result())
	}
	return nil, nil
}

func (guest *SGuest) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guestdisks := guest.GetDisks()
	dataCopy := jsonutils.DeepCopy(data)
	for i := range guestdisks {
		disk := guestdisks[i].GetDisk()
		if disk == nil {
			return nil, httperrors.NewInternalServerError("some disk missing!!!")
		}
		_, err := disk.PerformChangeOwner(ctx, userCred, query, dataCopy)
		if err != nil {
			return nil, err
		}
	}

	if eip, _ := guest.GetEip(); eip != nil {
		_, err := eip.PerformChangeOwner(ctx, userCred, query, dataCopy)
		if err != nil {
			return nil, err
		}
	}
	return guest.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, data)
}

func (guest *SGuest) AllowPerformResizeDisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return guest.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, guest, "resize-disk")
}

func (guest *SGuest) PerformResizeDisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskStr, _ := data.GetString("disk")
	if len(diskStr) == 0 {
		return nil, httperrors.NewMissingParameterError("disk")
	}
	diskObj, err := DiskManager.FetchByIdOrName(userCred, diskStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(DiskManager.Keyword(), diskStr)
		} else {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	guestdisk := guest.GetGuestDisk(diskObj.GetId())
	if guestdisk == nil {
		return nil, httperrors.NewInvalidStatusError("disk %s not attached to server", diskStr)
	}
	disk := diskObj.(*SDisk)
	err = disk.doResize(ctx, userCred, data, guest)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (guest *SGuest) StartGuestDiskResizeTask(ctx context.Context, userCred mcclient.TokenCredential, diskId string, sizeMb int64, parentTaskId string, pendingUsage quotas.IQuota) error {
	guest.SetStatus(userCred, api.VM_START_RESIZE_DISK, "StartGuestDiskResizeTask")
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(sizeMb), "size")
	params.Add(jsonutils.NewString(diskId), "disk_id")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestResizeDiskTask", guest, userCred, params, parentTaskId, "", pendingUsage)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformIoThrottle(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "io-throttle")
}

func (self *SGuest) PerformIoThrottle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Hypervisor %s can't do io throttle", self.Hypervisor)
	}
	if self.Status != api.VM_RUNNING {
		return nil, httperrors.NewServerStatusError("Cannot do io throttle in status %s", self.Status)
	}
	bpsMb, err := data.Int("bps")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("bps")
	}
	if bpsMb < 0 {
		return nil, httperrors.NewInputParameterError("bps must > 0")
	}
	iops, err := data.Int("iops")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("iops")
	}
	if iops < 0 {
		return nil, httperrors.NewInputParameterError("iops must > 0")
	}
	return nil, self.StartBlockIoThrottleTask(ctx, userCred, bpsMb, iops)
}

func (self *SGuest) StartBlockIoThrottleTask(ctx context.Context, userCred mcclient.TokenCredential, bpsMb, iops int64) error {
	params := jsonutils.NewDict()
	params.Set("bps", jsonutils.NewInt(bpsMb))
	params.Set("iops", jsonutils.NewInt(iops))
	params.Set("old_status", jsonutils.NewString(self.Status))
	self.SetStatus(userCred, api.VM_IO_THROTTLE, "start block io throttle task")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestBlockIoThrottleTask", self, userCred, params, "", "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (manager *SGuestManager) AllowPerformBatchMigrate(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, manager, "batch-guest-migrate")
}

func (self *SGuest) validateForBatchMigrate(ctx context.Context) (*SGuest, error) {
	guest := GuestManager.FetchGuestById(self.Id)
	if guest.Hypervisor != api.HYPERVISOR_KVM {
		return guest, httperrors.NewBadRequestError("guest %s hypervisor %s can't migrate",
			guest.Name, guest.Hypervisor)
	}
	if len(guest.BackupHostId) > 0 {
		return guest, httperrors.NewBadRequestError("guest %s has backup, can't migrate", guest.Name)
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY, api.VM_UNKNOWN}) {
		return guest, httperrors.NewBadRequestError("guest %s status %s can't migrate", guest.Name, guest.Status)
	}
	if guest.Status == api.VM_RUNNING {
		if len(guest.GetIsolatedDevices()) > 0 {
			return guest, httperrors.NewBadRequestError(
				"guest %s status %s has isolated device, can't do migrate",
				guest.Name, guest.Status,
			)
		}
		cdrom := guest.getCdrom(false)
		if cdrom != nil && len(cdrom.ImageId) > 0 {
			return guest, httperrors.NewBadRequestError("cannot migrate with cdrom")
		}
	} else if guest.Status == api.VM_UNKNOWN {
		if guest.getDefaultStorageType() == api.STORAGE_LOCAL {
			return guest, httperrors.NewBadRequestError(
				"guest %s status %s can't migrate with local storage",
				guest.Name, guest.Status,
			)
		}
	}
	return guest, nil
}

func (manager *SGuestManager) PerformBatchMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := new(api.GuestBatchMigrateRequest)
	err := data.Unmarshal(params)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal input error %s", err)
	}
	if len(params.GuestIds) == 0 {
		return nil, httperrors.NewInputParameterError("missing guest id")
	}

	var preferHostId string
	if len(params.PreferHost) > 0 {
		if !db.IsAdminAllowPerform(userCred, manager, "assign-host") {
			return nil, httperrors.NewBadRequestError("Only system admin can assign host")
		}
		iHost, _ := HostManager.FetchByIdOrName(userCred, params.PreferHost)
		if iHost == nil {
			return nil, httperrors.NewBadRequestError("Host %s not found", params.PreferHost)
		}
		host := iHost.(*SHost)
		preferHostId = host.Id
	}

	guests := make([]SGuest, 0)
	q := GuestManager.Query().In("id", params.GuestIds)
	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, httperrors.NewInternalServerError(err.Error())
	}
	if len(guests) != len(params.GuestIds) {
		return nil, httperrors.NewBadRequestError("Check input guests is exist")
	}
	for i := 0; i < len(guests); i++ {
		lockman.LockObject(ctx, &guests[i])
		defer lockman.ReleaseObject(ctx, &guests[i])
		guest, err := guests[i].validateForBatchMigrate(ctx)
		if err != nil {
			return nil, err
		}
		guests[i] = *guest
	}

	var hostGuests = map[string][]*api.GuestBatchMigrateParams{}
	for i := 0; i < len(guests); i++ {
		bmp := &api.GuestBatchMigrateParams{
			Id:          guests[i].Id,
			LiveMigrate: guests[i].Status == api.VM_RUNNING,
			RescueMode:  guests[i].Status == api.VM_UNKNOWN,
			OldStatus:   guests[i].Status,
		}
		guests[i].SetStatus(userCred, api.VM_START_MIGRATE, "batch migrate")
		if _, ok := hostGuests[guests[i].HostId]; ok {
			hostGuests[guests[i].HostId] = append(hostGuests[guests[i].HostId], bmp)
		} else {
			hostGuests[guests[i].HostId] = []*api.GuestBatchMigrateParams{bmp}
		}
	}
	for hostId, params := range hostGuests {
		kwargs := jsonutils.NewDict()
		kwargs.Set("guests", jsonutils.Marshal(params))
		if len(preferHostId) > 0 {
			kwargs.Set("prefer_host_id", jsonutils.NewString(preferHostId))
		}
		host := HostManager.FetchHostById(hostId)
		manager.StartHostGuestsMigrateTask(ctx, userCred, host, kwargs, "")
	}
	return nil, nil
}

func (manager *SGuestManager) StartHostGuestsMigrateTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	host *SHost, kwargs *jsonutils.JSONDict, parentTaskId string,
) error {
	task, err := taskman.TaskManager.NewTask(ctx, "HostGuestsMigrateTask", host, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformInstanceSnapshot(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "instance-snapshot")
}

func (self *SGuest) validateCreateInstanceSnapshot(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (*SRegionQuota, error) {

	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("guest hypervisor %s can't create instance snapshot", self.Hypervisor)
	}

	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Can't do instance snapshot with backup guest")
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return nil, httperrors.NewInvalidStatusError("guest can't do snapshot in status %s", self.Status)
	}

	var name string
	ownerId := self.GetOwnerId()
	dataDict := data.(*jsonutils.JSONDict)
	nameHint, err := dataDict.GetString("generate_name")
	if err == nil {
		name, err = db.GenerateName(InstanceSnapshotManager, ownerId, nameHint)
		if err != nil {
			return nil, err
		}
		dataDict.Set("name", jsonutils.NewString(name))
	} else if name, err = dataDict.GetString("name"); err != nil {
		return nil, httperrors.NewMissingParameterError("name")
	}

	err = db.NewNameValidator(InstanceSnapshotManager, ownerId, name, "")
	if err != nil {
		return nil, err
	}

	disks := self.GetDisks()
	for i := 0; i < len(disks); i++ {
		if storage := disks[i].GetDisk().GetStorage(); utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
			count, err := SnapshotManager.GetDiskManualSnapshotCount(disks[i].DiskId)
			if err != nil {
				return nil, httperrors.NewInternalServerError(err.Error())
			}
			if count >= options.Options.DefaultMaxManualSnapshotCount {
				return nil, httperrors.NewBadRequestError("guests disk %d snapshot full, can't take anymore", i)
			}
		}
	}
	pendingUsage := &SRegionQuota{Snapshot: len(disks)}
	keys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		return nil, err
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}
	return pendingUsage, nil
}

// 1. validate guest status, guest hypervisor
// 2. validate every disk manual snapshot count
// 3. validate snapshot quota with disk count
func (self *SGuest) PerformInstanceSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	lockman.LockClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	pendingUsage, err := self.validateCreateInstanceSnapshot(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	name, _ := data.GetString("name")
	instanceSnapshot, err := InstanceSnapshotManager.CreateInstanceSnapshot(ctx, userCred, self, name, false)
	if err != nil {
		quotas.CancelPendingUsage(
			ctx, userCred, pendingUsage, pendingUsage, false)
		return nil, httperrors.NewInternalServerError("create instance snapshot failed: %s", err)
	}
	err = self.InstaceCreateSnapshot(ctx, userCred, instanceSnapshot, pendingUsage)
	if err != nil {
		quotas.CancelPendingUsage(
			ctx, userCred, pendingUsage, pendingUsage, false)
		return nil, httperrors.NewInternalServerError("start create snapshot task failed: %s", err)
	}
	return nil, nil
}

func (self *SGuest) InstaceCreateSnapshot(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	instanceSnapshot *SInstanceSnapshot,
	pendingUsage *SRegionQuota,
) error {
	self.SetStatus(userCred, api.VM_START_INSTANCE_SNAPSHOT, "instance snapshot")
	return instanceSnapshot.StartCreateInstanceSnapshotTask(ctx, userCred, pendingUsage, "")
}

func (self *SGuest) AllowPerformInstanceSnapshotReset(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "instance-snapshot")
}

func (self *SGuest) PerformInstanceSnapshotReset(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {

	if self.Status != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("guest can't do snapshot in status ", self.Status)
	}

	dataDict := data.(*jsonutils.JSONDict)
	instanceSnapshotV := validators.NewModelIdOrNameValidator(
		"instance_snapshot", "instance_snapshot", self.GetOwnerId(),
	)
	err := instanceSnapshotV.Validate(dataDict)
	if err != nil {
		return nil, err
	}
	instanceSnapshot := instanceSnapshotV.Model.(*SInstanceSnapshot)
	if instanceSnapshot.Status != api.INSTANCE_SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Instance sanpshot not ready")
	}

	err = self.StartSnapshotResetTask(ctx, userCred, instanceSnapshot)
	if err != nil {
		return nil, httperrors.NewInternalServerError("start snapshot reset failed %s", err)
	}

	return nil, nil
}

func (self *SGuest) StartSnapshotResetTask(
	ctx context.Context, userCred mcclient.TokenCredential, instanceSnapshot *SInstanceSnapshot) error {

	self.SetStatus(userCred, api.VM_START_SNAPSHOT_RESET, "start snapshot reset task")
	if task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotResetTask", instanceSnapshot, userCred, nil, "", "", nil,
	); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) AllowPerformSnapshotAndClone(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "snapshot-and-clone")
}

func (self *SGuest) PerformSnapshotAndClone(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	newlyGuestName, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("name")
	}
	count, err := data.Int("count")
	if err != nil {
		count = 1
	} else if count <= 0 {
		return nil, httperrors.NewInputParameterError("count must > 0")
	}

	lockman.LockClass(ctx, InstanceSnapshotManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, InstanceSnapshotManager, self.ProjectId)

	// validate create instance snapshot and set snapshot pending usage
	snapshotUsage, err := self.validateCreateInstanceSnapshot(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	// set guest pending usage
	pendingUsage, pendingRegionUsage, err := self.getGuestUsage(int(count))
	keys, err := self.GetQuotaKeys()
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		return nil, err
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		return nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}
	regionKeys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, err
	}
	pendingRegionUsage.SetKeys(regionKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingRegionUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, err
	}
	// migrate snapshotUsage into regionUsage, then discard snapshotUsage
	pendingRegionUsage.Snapshot = snapshotUsage.Snapshot

	instanceSnapshotName, err := db.GenerateName(InstanceSnapshotManager, self.GetOwnerId(),
		fmt.Sprintf("%s-%s", newlyGuestName, rand.String(8)))
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, false)
		return nil, httperrors.NewInternalServerError("Generate snapshot name failed %s", err)
	}
	instanceSnapshot, err := InstanceSnapshotManager.CreateInstanceSnapshot(
		ctx, userCred, self, instanceSnapshotName,
		jsonutils.QueryBoolean(data, "auto_delete_instance_snapshot", false))
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, false)
		return nil, httperrors.NewInternalServerError("create instance snapshot failed: %s", err)
	}
	// isnapshotshot create success, cancel pending usage
	quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, true)

	err = self.StartInstanceSnapshotAndCloneTask(
		ctx, userCred, newlyGuestName, &pendingUsage, &pendingRegionUsage, instanceSnapshot, data.(*jsonutils.JSONDict))
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) StartInstanceSnapshotAndCloneTask(
	ctx context.Context, userCred mcclient.TokenCredential, newlyGuestName string,
	pendingUsage *SQuota, pendingRegionUsage *SRegionQuota, instanceSnapshot *SInstanceSnapshot, data *jsonutils.JSONDict) error {

	params := jsonutils.NewDict()
	params.Set("guest_params", data)
	if task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotAndCloneTask", instanceSnapshot, userCred, params, "", "", pendingUsage, pendingRegionUsage); err != nil {
		return err
	} else {
		self.SetStatus(userCred, api.VM_START_INSTANCE_SNAPSHOT, "instance snapshot")
		task.ScheduleRun(nil)
		return nil
	}
}

func (manager *SGuestManager) CreateGuestFromInstanceSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential, guestParams *jsonutils.JSONDict, isp *SInstanceSnapshot,
) (*SGuest, *jsonutils.JSONDict, error) {
	lockman.LockClass(ctx, manager, isp.ProjectId)
	defer lockman.ReleaseClass(ctx, manager, isp.ProjectId)

	guestName, err := guestParams.GetString("name")
	if err != nil {
		return nil, nil, fmt.Errorf("No new guest name provider")
	}
	if guestName, err = db.GenerateName(manager, isp.GetOwnerId(), guestName); err != nil {
		return nil, nil, err
	}

	guestParams.Set("name", jsonutils.NewString(guestName))
	guestParams.Set("instance_snapshot_id", jsonutils.NewString(isp.Id))
	iGuest, err := db.DoCreate(manager, ctx, userCred, nil, guestParams, isp.GetOwnerId())
	if err != nil {
		return nil, nil, err
	}
	guest := iGuest.(*SGuest)
	func() {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		guest.PostCreate(ctx, userCred, guest.GetOwnerId(), nil, guestParams)
	}()

	return guest, guestParams, nil
}

func (self *SGuest) AllowGetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowGetSpec(userCred, self, "jnlp")
}

func (self *SGuest) GetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_BAREMETAL {
		return nil, httperrors.NewInvalidStatusError("not a baremetal server")
	}
	host := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("no valid host")
	}
	if !host.IsBaremetal {
		return nil, httperrors.NewInvalidStatusError("host is not a baremetal")
	}
	return host.GetDetailsJnlp(ctx, userCred, query)
}

func (guest *SGuest) StartDeleteGuestSnapshots(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDeleteSnapshotsTask", guest, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) AllowPerformBindGroups(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {

	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "bind-groups")
}

func (self *SGuest) PerformBindGroups(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	groupIdSet, err := self.checkGroups(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	groupGuests, err := GroupguestManager.FetchByGuestId(self.Id)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_BIND, nil, userCred, false)
		return nil, err
	}

	for i := range groupGuests {
		groupId := groupGuests[i].GroupId
		if groupIdSet.Has(groupId) {
			groupIdSet.Delete(groupId)
		}
	}

	for _, groupId := range groupIdSet.UnsortedList() {
		_, err := GroupguestManager.Attach(ctx, groupId, self.Id)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_BIND, nil, userCred, false)
			return nil, errors.Wrapf(err, "fail to attch group %s to guest %s", groupId, self.Id)
		}
	}
	// ignore error
	err = self.ClearSchedDescCache()
	if err != nil {
		log.Errorf("fail to clear scheduler desc cache after unbinding groups successfully")
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_BIND, nil, userCred, true)
	return nil, nil
}

func (self *SGuest) AllowPerformUnbindGroups(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "unbind-groups")
}

func (self *SGuest) PerformUnbindGroups(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	groupIdSet, err := self.checkGroups(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	groupGuests, err := GroupguestManager.FetchByGuestId(self.Id)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_UNBIND, nil, userCred, false)
		return nil, err
	}

	for i := range groupGuests {
		joint := groupGuests[i]
		if !groupIdSet.Has(joint.GroupId) {
			continue
		}
		err := joint.Detach(ctx, userCred)
		if err != nil {
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_UNBIND, nil, userCred, false)
			return nil, errors.Wrapf(err, "fail to detach group %s to guest %s", joint.GroupId, self.Id)
		}
	}
	// ignore error
	err = self.ClearSchedDescCache()
	if err != nil {
		log.Errorf("fail to clear scheduler desc cache after binding groups successfully")
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_INSTANCE_GROUP_UNBIND, nil, userCred, true)
	return nil, nil
}

func (self *SGuest) checkGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (sets.String, error) {

	groupIdArr := jsonutils.GetArrayOfPrefix(data, "group")
	if len(groupIdArr) == 0 {
		return nil, httperrors.NewMissingParameterError("group.0 group.1 ... ")
	}

	groupIdSet := sets.NewString()
	for i := range groupIdArr {
		groupIdStr, _ := groupIdArr[i].GetString()
		model, err := GroupManager.FetchByIdOrName(userCred, groupIdStr)
		if err == sql.ErrNoRows {
			return nil, httperrors.NewInputParameterError("no such group %s", groupIdStr)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch group by id or name %s", groupIdStr)
		}
		group := model.(*SGroup)
		if group.ProjectId != self.ProjectId {
			return nil, httperrors.NewForbiddenError("group and guest should belong to same project")
		}
		if group.Enabled.IsFalse() {
			return nil, httperrors.NewForbiddenError("can not bind or unbind disabled instance group")
		}
		groupIdSet.Insert(group.GetId())
	}

	return groupIdSet, nil
}
