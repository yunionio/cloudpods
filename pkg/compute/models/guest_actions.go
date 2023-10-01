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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/userdata"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	guestdriver_types "yunion.io/x/onecloud/pkg/compute/guestdrivers/types"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (self *SGuest) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	ret := &cloudprovider.ServerVncOutput{}
	if self.PowerStates == api.VM_POWER_STATES_ON ||
		utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM, api.VM_MIGRATING}) {
		host, err := self.GetHost()
		if err != nil {
			return nil, httperrors.NewInternalServerError(errors.Wrapf(err, "GetHost").Error())
		}
		if options.Options.ForceUseOriginVnc {
			input.Origin = true
		}
		ret, err = self.GetDriver().GetGuestVncInfo(ctx, userCred, self, host, input)
		if err != nil {
			return nil, err
		}
		ret.Id = self.Id
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CONSOLE, ret, userCred, true)
		return ret, nil
	}
	return ret, nil
}

func (self *SGuest) PreCheckPerformAction(
	ctx context.Context, userCred mcclient.TokenCredential,
	action string, query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	if err := self.SVirtualResourceBase.PreCheckPerformAction(ctx, userCred, action, query, data); err != nil {
		return err
	}
	if self.Hypervisor == api.HYPERVISOR_KVM {
		host, _ := self.GetHost()
		if host != nil && (host.HostStatus == api.HOST_OFFLINE || !host.Enabled.Bool()) &&
			utils.IsInStringArray(action,
				[]string{
					"start", "restart", "stop", "reset", "rebuild-root",
					"change-config", "instance-snapshot", "snapshot-and-clone",
					"attach-isolated-device", "detach-isolated-deivce",
					"insert-iso", "eject-iso", "deploy", "create-backup",
				}) {
			return httperrors.NewInvalidStatusError(
				"host status %s and enabled %v, can't do server %s", host.HostStatus, host.Enabled.Bool(), action)
		}
	}
	return nil
}

func (self *SGuest) PerformMonitor(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerMonitorInput,
) (jsonutils.JSONObject, error) {
	if self.PowerStates == api.VM_POWER_STATES_ON ||
		utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM, api.VM_MIGRATING}) {
		if input.COMMAND == "" {
			return nil, httperrors.NewMissingParameterError("command")
		}
		return self.SendMonitorCommand(ctx, userCred, input)
	}
	return nil, httperrors.NewInvalidStatusError("Cannot send command in status %s", self.Status)
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
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionServerPanicked,
			IsFail: true,
		})
	}
	return nil, nil
}

func (self *SGuest) GetDetailsDesc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	host, _ := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInvalidStatusError("No host for server")
	}
	return self.GetDriver().GetJsonDescAtHost(ctx, userCred, self, host, nil)
}

func (self *SGuest) PerformSaveImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerSaveImageInput) (api.ServerSaveImageInput, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return input, httperrors.NewInputParameterError("Cannot save image in status %s", self.Status)
	}
	input.Restart = (self.Status == api.VM_RUNNING) || input.AutoStart
	if len(input.Name) == 0 && len(input.GenerateName) == 0 {
		return input, httperrors.NewInputParameterError("Image name is required")
	}
	disks := self.CategorizeDisks()
	if disks.Root == nil {
		return input, httperrors.NewInputParameterError("No root image")
	}
	input.OsType = self.OsType
	if len(input.OsType) == 0 {
		input.OsType = "Linux"
	}
	input.OsArch = self.OsArch
	if apis.IsARM(self.OsArch) {
		if osArch := self.GetMetadata(ctx, "os_arch", nil); len(osArch) == 0 {
			host, _ := self.GetHost()
			input.OsArch = host.CpuArchitecture
		}
	}

	factory, _ := cloudprovider.GetProviderFactory(self.GetDriver().GetProvider())
	if factory == nil || factory.IsOnPremise() { // OneCloud or VMware
		lockman.LockObject(ctx, disks.Root)
		defer lockman.ReleaseObject(ctx, disks.Root)

		var err error
		input.ImageId, err = disks.Root.PrepareSaveImage(ctx, userCred, input)
		if err != nil {
			return input, errors.Wrapf(err, "PrepareSaveImage")
		}
	}
	if len(input.Name) == 0 {
		input.Name = input.GenerateName
	}

	logclient.AddSimpleActionLog(self, logclient.ACT_SAVE_IMAGE, input, userCred, true)
	return input, self.StartGuestSaveImage(ctx, userCred, input, "")
}

func (self *SGuest) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerSaveImageInput, parentTaskId string) error {
	return self.GetDriver().StartGuestSaveImage(ctx, userCred, self, jsonutils.Marshal(input).(*jsonutils.JSONDict), parentTaskId)
}

func (self *SGuest) PerformSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.ServerSaveGuestImageInput) (jsonutils.JSONObject, error) {

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY}) {
		return nil, httperrors.NewBadRequestError("Cannot save image in status %s", self.Status)
	}
	if len(input.Name) == 0 && len(input.GenerateName) == 0 {
		return nil, httperrors.NewMissingParameterError("Image name is required")
	}
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Support only by KVM Hypervisor")
	}
	disks := self.CategorizeDisks()

	if disks.Root == nil {
		return nil, httperrors.NewInternalServerError("No root image")
	}

	if len(self.EncryptKeyId) > 0 && (input.EncryptKeyId == nil || len(*input.EncryptKeyId) == 0) {
		// server encrypted, so image must be encrypted
		input.EncryptKeyId = &self.EncryptKeyId
	} else if len(self.EncryptKeyId) > 0 && input.EncryptKeyId != nil && len(*input.EncryptKeyId) > 0 && self.EncryptKeyId != *input.EncryptKeyId {
		return nil, errors.Wrap(httperrors.ErrConflict, "input encrypt key not match with server encrypt key")
	}

	diskList := append(disks.Data, disks.Root)

	kwargs := imageapi.GuestImageCreateInput{}
	kwargs.GuestImageCreateInputBase = input.GuestImageCreateInputBase
	kwargs.Properties = make(map[string]string)
	if len(kwargs.ProjectId) == 0 {
		kwargs.ProjectId = self.ProjectId
	}

	for _, disk := range diskList {
		kwargs.Images = append(kwargs.Images, imageapi.GuestImageCreateInputSubimage{
			DiskFormat:  disk.DiskFormat,
			VirtualSize: disk.DiskSize,
		})
	}

	if len(input.Notes) > 0 {
		kwargs.Properties["notes"] = input.Notes
	}

	osType := self.OsType
	if len(osType) == 0 {
		osType = "Linux"
	}
	kwargs.Properties["os_type"] = osType

	if apis.IsARM(self.OsArch) {
		var osArch string
		if osArch = self.GetMetadata(ctx, "os_arch", nil); len(osArch) == 0 {
			host, _ := self.GetHost()
			osArch = host.CpuArchitecture
		}
		kwargs.Properties["os_arch"] = osArch
		kwargs.OsArch = self.OsArch
	}

	s := auth.GetSession(ctx, userCred, consts.GetRegion())
	ret, err := image.GuestImages.Create(s, jsonutils.Marshal(kwargs))
	if err != nil {
		return nil, err
	}
	guestImageId, _ := ret.GetString("id")
	// set class metadata
	cm, err := self.GetAllClassMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetAllClassMetadata")
	}
	if len(cm) > 0 {
		_, err = image.GuestImages.PerformAction(s, guestImageId, "set-class-metadata", jsonutils.Marshal(cm))
		if err != nil {
			return nil, errors.Wrapf(err, "unable to SetClassMetadata for guest image %s", guestImageId)
		}
	}
	guestImageInfo := struct {
		RootImage  imageapi.SubImageInfo
		DataImages []imageapi.SubImageInfo
	}{}
	ret.Unmarshal(&guestImageInfo)

	if len(guestImageInfo.DataImages) != len(disks.Data) {
		return nil, fmt.Errorf("create subimage of guest image error")
	}
	imageIds := make([]string, 0, len(guestImageInfo.DataImages)+1)
	for _, info := range guestImageInfo.DataImages {
		imageIds = append(imageIds, info.ID)
	}
	imageIds = append(imageIds, guestImageInfo.RootImage.ID)
	taskParams := jsonutils.NewDict()
	if input.AutoStart != nil && *input.AutoStart {
		taskParams.Add(jsonutils.JSONTrue, "auto_start")
	}
	taskParams.Add(jsonutils.Marshal(imageIds), "image_ids")
	log.Infof("before StartGuestSaveGuestImage image_ids: %s", imageIds)
	return nil, self.StartGuestSaveGuestImage(ctx, userCred, taskParams, "")
}

func (self *SGuest) StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	return self.GetDriver().StartGuestSaveGuestImage(ctx, userCred, self, data, parentTaskId)
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
	return self.GetMetadata(context.Background(), "__qemu_version", userCred)
}

func (self *SGuest) GetQemuCmdline(userCred mcclient.TokenCredential) string {
	return self.GetMetadata(context.Background(), "__qemu_cmdline", userCred)
}

func (self *SGuest) GetDetailsQemuInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ServerQemuInfo, error) {
	version := self.GetQemuVersion(userCred)
	cmdline := self.GetQemuCmdline(userCred)
	return &api.ServerQemuInfo{
		Version: version,
		Cmdline: cmdline,
	}, nil
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

func (self *SGuest) validateMigrate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	migrateInput *api.GuestMigrateInput,
	liveMigrateInput *api.GuestLiveMigrateInput,
) error {
	isLiveMigrate := false
	if liveMigrateInput != nil {
		isLiveMigrate = true
	}

	if isLiveMigrate {
		// do live migrate check
		if !self.GetDriver().IsSupportLiveMigrate() {
			return httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
		}
		if err := self.GetDriver().CheckLiveMigrate(ctx, self, userCred, *liveMigrateInput); err != nil {
			return err
		}
		if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_SUSPEND}) {
			if len(liveMigrateInput.PreferHostId) > 0 {
				iHost, _ := HostManager.FetchByIdOrName(userCred, liveMigrateInput.PreferHostId)
				if iHost == nil {
					return httperrors.NewBadRequestError("Host %s not found", liveMigrateInput.PreferHostId)
				}
				host := iHost.(*SHost)
				liveMigrateInput.PreferHostId = host.Id
			}
			return nil
		}
		return httperrors.NewBadRequestError("Cannot live migrate in status %s", self.Status)
	} else {
		// do migrate check
		if !self.GetDriver().IsSupportMigrate() {
			return httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.GetHypervisor())
		}
		if !migrateInput.IsRescueMode && self.Status != api.VM_READY {
			return httperrors.NewServerStatusError("Cannot normal migrate guest in status %s, try rescue mode or server-live-migrate?", self.Status)
		}
		if err := self.GetDriver().CheckMigrate(ctx, self, userCred, *migrateInput); err != nil {
			return err
		}
		if len(migrateInput.PreferHostId) > 0 {
			iHost, _ := HostManager.FetchByIdOrName(userCred, migrateInput.PreferHostId)
			if iHost == nil {
				return httperrors.NewBadRequestError("Host %s not found", migrateInput.PreferHostId)
			}
			host := iHost.(*SHost)
			migrateInput.PreferHostId = host.Id
		}
		return nil
	}
}

func (self *SGuest) validateConvertToKvm(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	migrateInput *api.GuestMigrateInput,
) error {
	if len(migrateInput.PreferHostId) > 0 {
		iHost, _ := HostManager.FetchByIdOrName(userCred, migrateInput.PreferHostId)
		if iHost == nil {
			return httperrors.NewBadRequestError("Host %s not found", migrateInput.PreferHostId)
		}
		host := iHost.(*SHost)
		migrateInput.PreferHostId = host.Id
	}
	if self.Status != api.VM_READY {
		return httperrors.NewServerStatusError("can't convert guest in status %s", self.Status)
	}
	return nil
}

func (self *SGuest) PerformMigrateForecast(ctx context.Context, userCred mcclient.TokenCredential, _ jsonutils.JSONObject, input *api.ServerMigrateForecastInput) (jsonutils.JSONObject, error) {
	var (
		mInput  *api.GuestMigrateInput     = nil
		lmInput *api.GuestLiveMigrateInput = nil
	)

	if input.ConvertToKvm {
		mInput = &api.GuestMigrateInput{PreferHostId: input.PreferHostId}
		if err := self.validateConvertToKvm(ctx, userCred, mInput); err != nil {
			return nil, err
		}
		input.PreferHostId = mInput.PreferHostId
	} else {
		if input.LiveMigrate {
			lmInput = &api.GuestLiveMigrateInput{
				PreferHostId: input.PreferHostId,
				SkipCpuCheck: &input.SkipCpuCheck,
			}
			if err := self.validateMigrate(ctx, userCred, nil, lmInput); err != nil {
				return nil, err
			}
			input.PreferHostId = lmInput.PreferHostId
		} else {
			mInput = &api.GuestMigrateInput{
				PreferHostId: input.PreferHostId,
				IsRescueMode: input.IsRescueMode,
			}
			if err := self.validateMigrate(ctx, userCred, mInput, nil); err != nil {
				return nil, err
			}
			input.PreferHostId = mInput.PreferHostId
		}
	}

	schedParams := self.GetSchedMigrateParams(userCred, input)
	if input.ConvertToKvm {
		schedParams.Hypervisor = api.HYPERVISOR_KVM
	}

	s := auth.GetAdminSession(ctx, options.Options.Region)
	_, res, err := scheduler.SchedManager.DoScheduleForecast(s, schedParams, 1)
	if err != nil {
		return nil, errors.Wrap(err, "Do schedule migrate forecast")
	}

	return res, nil
}

func (self *SGuest) GetSchedMigrateParams(
	userCred mcclient.TokenCredential,
	input *api.ServerMigrateForecastInput,
) *schedapi.ScheduleInput {
	schedDesc := self.ToSchedDesc()
	if input.PreferHostId != "" {
		schedDesc.ServerConfig.PreferHost = input.PreferHostId
	}
	if input.LiveMigrate {
		schedDesc.LiveMigrate = input.LiveMigrate
		if self.GetMetadata(context.Background(), "__cpu_mode", userCred) != api.CPU_MODE_QEMU {
			host, _ := self.GetHost()
			schedDesc.CpuDesc = host.CpuDesc
			schedDesc.CpuMicrocode = host.CpuMicrocode
			schedDesc.CpuMode = api.CPU_MODE_HOST
		} else {
			schedDesc.CpuMode = api.CPU_MODE_QEMU
		}
		schedDesc.SkipCpuCheck = &input.SkipCpuCheck
		host, _ := self.GetHost()
		if host != nil {
			schedDesc.TargetHostKernel, _ = host.SysInfo.GetString("kernel_version")
			schedDesc.SkipKernelCheck = &input.SkipKernelCheck
			schedDesc.HostMemPageSizeKB = host.PageSizeKB
		}
	}
	schedDesc.ReuseNetwork = true
	return schedDesc
}

func (self *SGuest) PerformMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.GuestMigrateInput) (jsonutils.JSONObject, error) {
	if err := self.validateMigrate(ctx, userCred, input, nil); err != nil {
		return nil, err
	}

	return nil, self.StartMigrateTask(ctx, userCred, input.IsRescueMode, input.AutoStart, self.Status, input.PreferHostId, "")
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
	dedicateMigrateTask := "GuestMigrateTask"
	if self.GetHypervisor() != api.HYPERVISOR_KVM {
		dedicateMigrateTask = "ManagedGuestMigrateTask" //托管私有云
	}
	if task, err := taskman.TaskManager.NewTask(ctx, dedicateMigrateTask, self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) PerformLiveMigrate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.GuestLiveMigrateInput) (jsonutils.JSONObject, error) {
	if err := self.validateMigrate(ctx, userCred, nil, input); err != nil {
		return nil, err
	}
	if input.EnableTLS == nil {
		input.EnableTLS = &options.Options.EnableTlsMigration
	}
	return nil, self.StartGuestLiveMigrateTask(ctx, userCred,
		self.Status, input.PreferHostId, input.SkipCpuCheck,
		input.SkipKernelCheck, input.EnableTLS, input.QuicklyFinish, input.MaxBandwidthMb, input.KeepDestGuestOnFailed, "",
	)
}

func (self *SGuest) StartGuestLiveMigrateTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	guestStatus, preferHostId string,
	skipCpuCheck, skipKernelCheck, enableTLS, quicklyFinish *bool,
	maxBandwidthMb *int64, keepDestGuestOnFailed *bool, parentTaskId string,
) error {
	self.SetStatus(userCred, api.VM_START_MIGRATE, "")
	data := jsonutils.NewDict()
	if len(preferHostId) > 0 {
		data.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	if skipCpuCheck != nil {
		data.Set("skip_cpu_check", jsonutils.NewBool(*skipCpuCheck))
	}
	if skipKernelCheck != nil {
		data.Set("skip_kernel_check", jsonutils.NewBool(*skipKernelCheck))
	}
	if enableTLS != nil {
		data.Set("enable_tls", jsonutils.NewBool(*enableTLS))
	}
	if quicklyFinish != nil {
		data.Set("quickly_finish", jsonutils.NewBool(*quicklyFinish))
	}
	if maxBandwidthMb != nil {
		data.Set("max_bandwidth_mb", jsonutils.NewInt(*maxBandwidthMb))
	}
	if keepDestGuestOnFailed != nil {
		data.Set("keep_dest_guest_on_failed", jsonutils.NewBool(*keepDestGuestOnFailed))
	}

	data.Set("guest_status", jsonutils.NewString(guestStatus))
	dedicateMigrateTask := "GuestLiveMigrateTask"
	if self.GetHypervisor() != api.HYPERVISOR_KVM {
		dedicateMigrateTask = "ManagedGuestLiveMigrateTask" //托管私有云
	}
	if task, err := taskman.TaskManager.NewTask(ctx, dedicateMigrateTask, self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) PerformSetLiveMigrateParams(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerSetLiveMigrateParamsInput,
) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_LIVE_MIGRATING {
		return nil, httperrors.NewServerStatusError("cannot set migrate params in status %s", self.Status)
	}
	if input.MaxBandwidthMB == nil && input.DowntimeLimitMS == nil {
		return nil, httperrors.NewInputParameterError("empty input")
	}

	arguments := map[string]interface{}{}
	if input.MaxBandwidthMB != nil {
		arguments["max-bandwidth"] = *input.MaxBandwidthMB * 1024 * 1024
	}
	if input.DowntimeLimitMS != nil {
		arguments["downtime-limit"] = *input.DowntimeLimitMS
	}
	cmd := map[string]interface{}{
		"execute":   "migrate-set-parameters",
		"arguments": arguments,
	}
	log.Infof("set live migrate params input: %s", jsonutils.Marshal(cmd).String())
	monitorInput := &api.ServerMonitorInput{
		COMMAND: jsonutils.Marshal(cmd).String(),
		QMP:     true,
	}
	return self.SendMonitorCommand(ctx, userCred, monitorInput)
}

func (self *SGuest) PerformCancelLiveMigrate(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_LIVE_MIGRATING {
		return nil, httperrors.NewServerStatusError("cannot set migrate params in status %s", self.Status)
	}

	return nil, self.GetDriver().RequestCancelLiveMigrate(ctx, self, userCred)
}

func (self *SGuest) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.IsEncrypted() {
		return nil, httperrors.NewForbiddenError("cannot clone encrypted server")
	}
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
	err = db.NewNameValidator(GuestManager, userCred, cloneInput.Name, nil)
	if err != nil {
		return nil, err
	}

	createInput := self.ToCreateInput(ctx, userCred)
	createInput.Name = cloneInput.Name
	createInput.AutoStart = cloneInput.AutoStart

	createInput.EipBw = cloneInput.EipBw
	createInput.Eip = cloneInput.Eip
	createInput.EipChargeType = cloneInput.EipChargeType
	if err := GuestManager.validateEip(userCred, createInput, createInput.PreferRegion, createInput.PreferManager); err != nil {
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

func (self *SGuest) PerformSetPassword(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerSetPasswordInput) (jsonutils.JSONObject, error) {
	if self.Hypervisor == api.HYPERVISOR_KVM && self.Status == api.VM_RUNNING {
		inputQga := &api.ServerQgaSetPasswordInput{
			Username: input.Username,
			Password: input.Password,
		}
		if inputQga.Username == "" {
			if self.OsType == osprofile.OS_TYPE_WINDOWS {
				inputQga.Username = api.VM_DEFAULT_WINDOWS_LOGIN_USER
			} else {
				inputQga.Username = api.VM_DEFAULT_LINUX_LOGIN_USER
			}
		}
		if inputQga.Password == "" && input.ResetPassword {
			inputQga.Password = seclib2.RandomPassword2(12)
		}
		return self.PerformQgaSetPassword(ctx, userCred, query, inputQga)
	} else {
		inputDeploy := api.ServerDeployInput{}
		inputDeploy.AutoStart = input.AutoStart
		inputDeploy.Password = input.Password
		inputDeploy.ResetPassword = input.ResetPassword
		return self.PerformDeploy(ctx, userCred, query, inputDeploy)
	}
}

func (self *SGuest) GetDetailsCreateParams(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := self.ToCreateInput(ctx, userCred)
	return input.JSON(input), nil
}

func (self *SGuest) saveOldPassword(ctx context.Context, userCred mcclient.TokenCredential) {
	loginKey := self.GetMetadata(ctx, api.VM_METADATA_LOGIN_KEY, userCred)
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
	loginSecret := self.GetMetadata(ctx, api.VM_METADATA_LAST_LOGIN_KEY, userCred)
	password, _ := utils.DescryptAESBase64(self.Id, loginSecret)
	return password
}

func (self *SGuest) PerformDeploy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ServerDeployInput,
) (jsonutils.JSONObject, error) {
	self.saveOldPassword(ctx, userCred)

	if input.DeleteKeypair || len(input.KeypairId) > 0 {
		if len(input.KeypairId) > 0 {
			_, err := validators.ValidateModel(userCred, KeypairManager, &input.KeypairId)
			if err != nil {
				return nil, err
			}
		}

		if self.KeypairId != input.KeypairId {
			okey := self.getKeypair()
			if okey != nil {
				input.DeletePublicKey = okey.PublicKey
			}

			diff, err := db.Update(self, func() error {
				self.KeypairId = input.KeypairId
				return nil
			})
			if err != nil {
				return nil, httperrors.NewInternalServerError("update keypairId %v", err)
			}

			db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
			input.ResetPassword = true
		}
	}

	if len(input.Password) > 0 {
		err := seclib2.ValidatePassword(input.Password)
		if err != nil {
			return nil, err
		}
		input.ResetPassword = true
	}

	// 变更密码/密钥时需要Restart才能生效。更新普通字段不需要Restart, Azure需要在运行状态下操作
	doRestart := false
	if input.ResetPassword {
		doRestart = self.GetDriver().IsNeedRestartForResetLoginInfo()
	}

	deployStatus, err := self.GetDriver().GetDeployStatus()
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}

	if utils.IsInStringArray(self.Status, deployStatus) {
		if (doRestart && self.Status == api.VM_RUNNING) || (self.Status != api.VM_RUNNING && (input.AutoStart || input.Restart)) {
			input.Restart = true
		} else {
			// 避免前端直接传restart参数, 越过校验
			input.Restart = false
		}
		err := self.StartGuestDeployTask(ctx, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), "deploy", "")
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, httperrors.NewServerStatusError("Cannot deploy in status %s", self.Status)
}

func (self *SGuest) ValidateAttachDisk(ctx context.Context, disk *SDisk) error {
	storage, _ := disk.GetStorage()
	host, _ := self.GetHost()
	if provider := storage.GetCloudprovider(); provider != nil {
		if provider.Id != host.ManagerId {
			return httperrors.NewInputParameterError("Disk %s and guest not belong to the same account", disk.Name)
		}
		if storage.ZoneId != host.ZoneId {
			return httperrors.NewInputParameterError("Disk %s and guest not belong to the same zone", disk.Name)
		}
	}

	if disk.IsLocal() {
		attached, err := disk.isAttached()
		if err != nil {
			return httperrors.NewInternalServerError("isAttached check failed %s", err)
		}
		if attached {
			return httperrors.NewInputParameterError("Disk %s has been attached", disk.Name)
		}
	}

	if len(disk.GetPathAtHost(host)) == 0 {
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
	ok, err := self.IsInSameClass(ctx, &disk.SStandaloneAnonResourceBase)
	if err != nil {
		return err
	}
	if self.EncryptKeyId != disk.EncryptKeyId {
		return errors.Wrapf(httperrors.ErrConflict, "conflict encryption key between server and disk")
	}
	if !ok {
		return httperrors.NewForbiddenError("the class metadata of guest and disk is different")
	}
	return nil
}

func (self *SGuest) PerformAttachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerAttachDiskInput) (jsonutils.JSONObject, error) {
	if len(input.DiskId) == 0 {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	if input.BootIndex != nil {
		if isDup, err := self.isBootIndexDuplicated(*input.BootIndex); err != nil {
			return nil, err
		} else if isDup {
			return nil, httperrors.NewInputParameterError("boot index %d is duplicated", *input.BootIndex)
		}
	}

	diskObj, err := validators.ValidateModel(userCred, DiskManager, &input.DiskId)
	if err != nil {
		return nil, err
	}

	if err := self.ValidateAttachDisk(ctx, diskObj.(*SDisk)); err != nil {
		return nil, err
	}

	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(input.DiskId), "disk_id")
	if input.BootIndex != nil && *input.BootIndex >= 0 {
		taskData.Add(jsonutils.NewInt(int64(*input.BootIndex)), "boot_index")
	}

	self.SetStatus(userCred, api.VM_ATTACH_DISK, "")
	return nil, self.GetDriver().StartGuestAttachDiskTask(ctx, userCred, self, taskData, "")
}

func (self *SGuest) StartRestartNetworkTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, ip string, inBlockStream bool) error {
	data := jsonutils.NewDict()
	data.Set("ip", jsonutils.NewString(ip))
	data.Set("in_block_stream", jsonutils.NewBool(inBlockStream))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestRestartNetworkTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) StartQgaRestartNetworkTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, device string, ipMask string, gateway string, prevIp string, inBlockStream bool) error {
	data := jsonutils.NewDict()
	data.Set("device", jsonutils.NewString(device))
	data.Set("ip_mask", jsonutils.NewString(ipMask))
	data.Set("gateway", jsonutils.NewString(gateway))
	data.Set("prev_ip", jsonutils.NewString(prevIp))
	data.Set("in_block_stream", jsonutils.NewBool(inBlockStream))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestQgaRestartNetworkTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) startSyncTask(ctx context.Context, userCred mcclient.TokenCredential, firewallOnly bool, parentTaskId string, data *jsonutils.JSONDict) error {
	if firewallOnly {
		data.Add(jsonutils.JSONTrue, "fw_only")
	} else if err := self.SetStatus(userCred, api.VM_SYNC_CONFIG, ""); err != nil {
		log.Errorln(err)
		return err
	}
	return self.doSyncTask(ctx, data, userCred, parentTaskId)
}

func (self *SGuest) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, firewallOnly bool, parentTaskId string) error {
	return self.startSyncTask(ctx, userCred, firewallOnly, parentTaskId, jsonutils.NewDict())
}

func (self *SGuest) StartSyncTaskWithoutSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, fwOnly bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("without_sync_status", jsonutils.JSONTrue)
	data.Set("fw_only", jsonutils.NewBool(fwOnly))
	return self.doSyncTask(ctx, data, userCred, parentTaskId)
}

func (self *SGuest) doSyncTask(ctx context.Context, data *jsonutils.JSONDict, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSyncConfTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
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

func (self *SGuest) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_START_FAILED, api.VM_SAVE_DISK_FAILED, api.VM_SUSPEND}) {
		if err := self.ValidateEncryption(ctx, userCred); err != nil {
			return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
		}
		if !self.guestDisksStorageTypeIsShared() {
			host, _ := self.GetHost()
			guestsMem, err := host.GetNotReadyGuestsMemorySize()
			if err != nil {
				return nil, err
			}
			if float32(guestsMem+self.VmemSize) > host.GetVirtualMemorySize() {
				return nil, httperrors.NewInsufficientResourceError("host virtual memory not enough")
			}
		}
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

func (self *SGuest) StartGuestDeployTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	kwargs *jsonutils.JSONDict, action string, parentTaskId string,
) error {
	self.SetStatus(userCred, api.VM_START_DEPLOY, "")
	if kwargs == nil {
		kwargs = jsonutils.NewDict()
	}
	kwargs.Add(jsonutils.NewString(action), "deploy_action")

	taskName := "GuestDeployTask"
	if self.BackupHostId != "" {
		taskName = "HAGuestDeployTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) EventNotify(ctx context.Context, userCred mcclient.TokenCredential, action noapi.SAction) {
	detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
		if action != notifyclient.ActionCreate && action != notifyclient.ActionRebuildRoot && action != notifyclient.ActionResetPassword {
			return
		}
		meta, err := self.GetAllMetadata(ctx, userCred)
		if err != nil {
			return
		}
		details.Add(jsonutils.NewString(self.getNotifyIps()), "ips")
		osName := meta[api.VM_METADATA_OS_NAME]
		if osName == "Windows" {
			details.Add(jsonutils.JSONTrue, "windows")
		}
		loginAccount := meta[api.VM_METADATA_LOGIN_ACCOUNT]
		if len(loginAccount) > 0 {
			details.Add(jsonutils.NewString(loginAccount), "login_account")
		}
		keypair := self.getKeypairName()
		if len(keypair) > 0 {
			details.Add(jsonutils.NewString(keypair), "keypair")
		} else {
			loginKey := meta[api.VM_METADATA_LOGIN_KEY]
			if len(loginKey) > 0 {
				passwd, err := utils.DescryptAESBase64(self.Id, loginKey)
				if err == nil {
					details.Add(jsonutils.NewString(passwd), "password")
				}
			}
		}
	}
	var resourceType string
	if self.Hypervisor == api.HYPERVISOR_BAREMETAL {
		resourceType = noapi.TOPIC_RESOURCE_BAREMETAL
	}
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:                 self,
		ResourceType:        resourceType,
		Action:              action,
		ObjDetailsDecorator: detailsDecro,
		AdvanceDays:         0,
	})
}

func (self *SGuest) NotifyServerEvent(
	ctx context.Context, userCred mcclient.TokenCredential, event string, priority notify.TNotifyPriority,
	loginInfo bool, kwargs *jsonutils.JSONDict, notifyAdmin bool,
) {
	meta, err := self.GetAllMetadata(ctx, userCred)
	if err != nil {
		return
	}
	if kwargs == nil {
		kwargs = jsonutils.NewDict()
	}

	kwargs.Add(jsonutils.NewString(self.Name), "name")
	kwargs.Add(jsonutils.NewString(self.Hypervisor), "hypervisor")
	host, _ := self.GetHost()
	if host != nil {
		brand := host.GetBrand()
		if brand == api.CLOUD_PROVIDER_ONECLOUD {
			brand = api.ComputeI18nTable.Lookup(ctx, brand)
		}
		kwargs.Add(jsonutils.NewString(brand), "brand")
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
	notifyclient.NotifyWithCtx(ctx, []string{userCred.GetUserId()}, false, priority, event, kwargs)
	if notifyAdmin {
		notifyclient.SystemNotifyWithCtx(ctx, priority, event, kwargs)
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
	notifyclient.SystemNotifyWithCtx(ctx, priority, event, kwargs)
}

func (self *SGuest) StartGuestStopTask(ctx context.Context, userCred mcclient.TokenCredential, isForce, stopCharging bool, parentTaskId string) error {
	if len(parentTaskId) == 0 {
		self.SetStatus(userCred, api.VM_START_STOP, "")
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewBool(isForce), "is_force")
	params.Add(jsonutils.NewBool(stopCharging), "stop_charging")
	if len(parentTaskId) > 0 {
		params.Add(jsonutils.JSONTrue, "subtask")
	}
	driver := self.GetDriver()
	shutdownMode := api.VM_SHUTDOWN_MODE_KEEP_CHARGING
	if stopCharging && driver.IsSupportShutdownMode() {
		shutdownMode = api.VM_SHUTDOWN_MODE_STOP_CHARGING
	}
	_, err := db.Update(self, func() error {
		self.ShutdownMode = shutdownMode
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return driver.StartGuestStopTask(self, ctx, userCred, params, parentTaskId)
}

func (self *SGuest) insertIso(imageId string, cdromOrdinal int64) bool {
	cdrom := self.getCdrom(true, cdromOrdinal)
	return cdrom.insertIso(imageId)
}

func (self *SGuest) InsertIsoSucc(cdromOrdinal int64, imageId string, path string, size int64, name string, bootIndex *int8) (*SGuestcdrom, bool) {
	cdrom := self.getCdrom(false, cdromOrdinal)
	return cdrom, cdrom.insertIsoSucc(imageId, path, size, name, bootIndex)
}

func (self *SGuest) GetDetailsIso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var cdromOrdinal int64 = 0
	if query.Contains("ordinal") {
		cdromOrdinal, _ = query.Int("ordinal")
	}
	cdrom := self.getCdrom(false, cdromOrdinal)
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
	return desc, nil
}

func (self *SGuest) PerformInsertiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	cdromOrdinal, _ := data.Int("cdrom_ordinal")
	if cdromOrdinal < 0 {
		return nil, httperrors.NewServerStatusError("invalid cdrom_ordinal: %d", cdromOrdinal)
	}
	cdrom := self.getCdrom(false, cdromOrdinal)
	if cdrom != nil && len(cdrom.ImageId) > 0 {
		return nil, httperrors.NewBadRequestError("CD-ROM not empty, please eject first")
	}
	imageId, _ := data.GetString("image_id")
	image, err := parseIsoInfo(ctx, userCred, imageId)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	var bootIndex *int8
	if data.Contains("boot_index") {
		bd, _ := data.Int("boot_index")
		bd8 := int8(bd)
		bootIndex = &bd8

		if isDup, err := self.isBootIndexDuplicated(bd8); err != nil {
			return nil, err
		} else if isDup {
			return nil, httperrors.NewInputParameterError("boot index %d is duplicated", bd8)
		}
	}

	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		err = self.StartInsertIsoTask(ctx, cdromOrdinal, image.Id, false, bootIndex, self.HostId, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Insert ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) PerformEjectiso(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	cdromOrdinal, _ := data.Int("cdrom_ordinal")
	if cdromOrdinal < 0 {
		return nil, httperrors.NewServerStatusError("invalid cdrom_ordinal: %d", cdromOrdinal)
	}
	cdrom := self.getCdrom(false, cdromOrdinal)
	if cdrom == nil || len(cdrom.ImageId) == 0 {
		return nil, httperrors.NewBadRequestError("No ISO to eject")
	}
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		err := self.StartEjectisoTask(ctx, cdromOrdinal, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Eject ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) StartEjectisoTask(ctx context.Context, cdromOrdinal int64, userCred mcclient.TokenCredential, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewInt(cdromOrdinal), "cdrom_ordinal")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestEjectISOTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartInsertIsoTask(ctx context.Context, cdromOrdinal int64, imageId string, boot bool, bootIndex *int8, hostId string, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.insertIso(imageId, cdromOrdinal)

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	data.Add(jsonutils.NewInt(cdromOrdinal), "cdrom_ordinal")
	data.Add(jsonutils.NewString(hostId), "host_id")
	if boot {
		data.Add(jsonutils.JSONTrue, "boot")
	}
	if bootIndex != nil {
		data.Add(jsonutils.NewInt(int64(*bootIndex)), "boot_index")
	}
	taskName := "GuestInsertIsoTask"
	if self.BackupHostId != "" {
		taskName = "HaGuestInsertIsoTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) insertVfd(imageId string, floppyOrdinal int64) bool {
	floppy := self.getFloppy(true, floppyOrdinal)
	return floppy.insertVfd(imageId)
}

func (self *SGuest) InsertVfdSucc(floppyOrdinal int64, imageId string, path string, size int64, name string) bool {
	floppy := self.getFloppy(true, floppyOrdinal)
	return floppy.insertVfdSucc(imageId, path, size, name)
}

func (self *SGuest) GetDetailsVfd(floppyOrdinal int64, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	floppy := self.getFloppy(false, floppyOrdinal)
	desc := jsonutils.NewDict()
	if len(floppy.ImageId) > 0 {
		desc.Set("image_id", jsonutils.NewString(floppy.ImageId))
		desc.Set("status", jsonutils.NewString("inserting"))
	}
	if len(floppy.Path) > 0 {
		desc.Set("name", jsonutils.NewString(floppy.Name))
		desc.Set("size", jsonutils.NewInt(int64(floppy.Size)))
		desc.Set("status", jsonutils.NewString("ready"))
	}
	return desc
}

func (self *SGuest) PerformInsertvfd(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data api.ServerInsertVfdInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if data.FloppyOrdinal < 0 {
		return nil, httperrors.NewServerStatusError("invalid floppy_ordinal: %d", data.FloppyOrdinal)
	}
	floppy := self.getFloppy(false, data.FloppyOrdinal)
	if floppy != nil && len(floppy.ImageId) > 0 {
		return nil, httperrors.NewBadRequestError("Floppy not empty, please eject first")
	}
	image, err := parseIsoInfo(ctx, userCred, data.ImageId)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		err = self.StartInsertVfdTask(ctx, data.FloppyOrdinal, image.Id, false, self.HostId, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Insert ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) PerformEjectvfd(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data api.ServerEjectVfdInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL}) {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if data.FloppyOrdinal < 0 {
		return nil, httperrors.NewServerStatusError("invalid floppy_ordinal: %d", data.FloppyOrdinal)
	}
	floppy := self.getFloppy(false, data.FloppyOrdinal)
	if floppy == nil || len(floppy.ImageId) == 0 {
		return nil, httperrors.NewBadRequestError("No VFD to eject")
	}
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		err := self.StartEjectvfdTask(ctx, userCred, "")
		return nil, err
	} else {
		return nil, httperrors.NewServerStatusError("Eject ISO not allowed in status %s", self.Status)
	}
}

func (self *SGuest) StartEjectvfdTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestEjectVFDTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartInsertVfdTask(ctx context.Context, floppyOrdinal int64, imageId string, boot bool, hostId string, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.insertVfd(imageId, floppyOrdinal)

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	data.Add(jsonutils.NewString(hostId), "host_id")
	if boot {
		data.Add(jsonutils.JSONTrue, "boot")
	}
	taskName := "GuestInsertVfdTask"
	if self.BackupHostId != "" {
		taskName = "HaGuestInsertVfdTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGueststartTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	data *jsonutils.JSONDict, parentTaskId string,
) error {
	if self.Hypervisor == api.HYPERVISOR_KVM && self.guestDisksStorageTypeIsShared() {
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
	taskName := "GuestStartTask"
	if self.BackupHostId != "" {
		taskName = "HAGuestStartTask"
	}
	task, err := taskman.TaskManager.NewTask(ctx, taskName, self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGuestCreateTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput, pendingUsage quotas.IQuota, parentTaskId string) error {
	if input.FakeCreate {
		self.fixFakeServerInfo(ctx, userCred)
		return nil
	}
	return self.GetDriver().StartGuestCreateTask(self, ctx, userCred, input.JSON(input), pendingUsage, parentTaskId)
}

func (self *SGuest) fixFakeServerInfo(ctx context.Context, userCred mcclient.TokenCredential) {
	status := []string{api.VM_READY, api.VM_RUNNING}
	rand.Seed(time.Now().Unix())
	db.Update(self, func() error {
		self.Status = status[rand.Intn(len(status))]
		self.PowerStates = api.VM_POWER_STATES_ON
		if self.Status == api.VM_READY {
			self.PowerStates = api.VM_POWER_STATES_OFF
		}
		return nil
	})
	disks, _ := self.GetDisks()
	for i := range disks {
		db.Update(&disks[i], func() error {
			disks[i].Status = api.DISK_READY
			return nil
		})
	}
	networks, _ := self.GetNetworks("")
	for i := range networks {
		if len(networks[i].IpAddr) > 0 {
			continue
		}
		network := networks[i].GetNetwork()
		if network != nil {
			db.Update(&networks[i], func() error {
				networks[i].IpAddr, _ = network.GetFreeIP(ctx, userCred, nil, nil, "", api.IPAllocationRadnom, false)
				return nil
			})
		}
	}
	if eip, _ := self.GetEipOrPublicIp(); eip != nil && len(eip.IpAddr) == 0 {
		db.Update(eip, func() error {
			if len(eip.NetworkId) > 0 {
				if network, _ := eip.GetNetwork(); network != nil {
					eip.IpAddr, _ = network.GetFreeIP(ctx, userCred, nil, nil, "", api.IPAllocationRadnom, false)
					return nil
				}
			}
			return nil
		})
	}
}

func (self *SGuest) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return self.GetDriver().StartGuestSyncstatusTask(self, ctx, userCred, parentTaskId)
}

func (self *SGuest) StartAutoDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	db.OpsLog.LogEvent(self, db.ACT_DELETE, "auto-delete after stop", userCred)
	opts := api.ServerDeleteInput{}
	return self.StartDeleteGuestTask(ctx, userCred, parentTaskId, opts)
}

func (self *SGuest) StartDeleteGuestTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string,
	opts api.ServerDeleteInput,
) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.Status), "guest_status")
	params.Update(jsonutils.Marshal(opts))
	self.SetStatus(userCred, api.VM_START_DELETE, "")
	return self.GetDriver().StartDeleteGuestTask(ctx, userCred, self, params, parentTaskId)
}

func (self *SGuest) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidatePurgeCondition(ctx)
	if err != nil {
		return nil, err
	}
	host, _ := self.GetHost()
	if host != nil && host.GetEnabled() {
		return nil, httperrors.NewInvalidStatusError("Cannot purge server on enabled host")
	}
	opts := api.ServerDeleteInput{
		Purge: true,
	}
	err = self.StartDeleteGuestTask(ctx, userCred, "", opts)
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

// 重装系统(更换系统镜像)
func (self *SGuest) PerformRebuildRoot(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.ServerRebuildRootInput,
) (*api.SGuest, error) {
	input, err := self.GetDriver().ValidateRebuildRoot(ctx, userCred, self, input)
	if err != nil {
		return nil, err
	}

	if len(input.ImageId) > 0 {
		img, err := CachedimageManager.getImageInfo(ctx, userCred, input.ImageId, false)
		if err != nil {
			return nil, httperrors.NewNotFoundError("failed to find %s", input.ImageId)
		}
		err = self.GetDriver().ValidateImage(ctx, img)
		if err != nil {
			return nil, err
		}

		// compare os arch
		if len(self.InstanceType) > 0 {
			provider := GetDriver(self.Hypervisor).GetProvider()
			sku, _ := ServerSkuManager.FetchSkuByNameAndProvider(self.InstanceType, provider, true)
			if sku != nil && len(sku.CpuArch) > 0 && len(img.Properties["os_arch"]) > 0 && !strings.Contains(img.Properties["os_arch"], sku.CpuArch) {
				return nil, httperrors.NewConflictError("root disk image(%s) and sku(%s) architecture mismatch", img.Properties["os_arch"], sku.CpuArch)
			}
		}

		diskCat := self.CategorizeDisks()
		if img.MinDiskMB == 0 || img.Status != imageapi.IMAGE_STATUS_ACTIVE {
			return nil, httperrors.NewInputParameterError("invlid image")
		}
		if img.MinDiskMB > diskCat.Root.DiskSize {
			return nil, httperrors.NewInputParameterError("image size exceeds root disk size")
		}
		osType, _ := img.Properties["os_type"]
		osName := self.GetMetadata(ctx, "os_name", userCred)
		if len(osName) == 0 && len(osType) == 0 && strings.ToLower(osType) != strings.ToLower(osName) {
			return nil, httperrors.NewBadRequestError("Cannot switch OS between %s-%s", osName, osType)
		}
		input.ImageId = img.Id
	}
	templateId := self.GetTemplateId()

	if templateId != input.ImageId && len(templateId) > 0 && len(input.ImageId) > 0 && !self.GetDriver().IsRebuildRootSupportChangeUEFI() {
		q := CachedimageManager.Query().In("id", []string{input.ImageId, templateId})
		images := []SCachedimage{}
		err := db.FetchModelObjects(CachedimageManager, q, &images)
		if err != nil {
			return nil, errors.Wrap(err, "FetchModelObjects")
		}
		if len(images) == 2 && images[0].UEFI != images[1].UEFI {
			return nil, httperrors.NewUnsupportOperationError("Can not rebuild root with with diff uefi image")
		}
	}

	rebuildStatus, err := self.GetDriver().GetRebuildRootStatus()
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}

	if !self.GetDriver().IsRebuildRootSupportChangeImage() && len(input.ImageId) > 0 {
		if len(templateId) == 0 {
			return nil, httperrors.NewBadRequestError("No template for root disk, cannot rebuild root")
		}
		if input.ImageId != templateId {
			return nil, httperrors.NewInputParameterError("%s not support rebuild root with a different image", self.GetDriver().GetHypervisor())
		}
	}

	if !utils.IsInStringArray(self.Status, rebuildStatus) {
		return nil, httperrors.NewInvalidStatusError("Cannot reset root in status %s", self.Status)
	}

	if self.Status == api.VM_READY && self.ShutdownMode == api.VM_SHUTDOWN_MODE_STOP_CHARGING {
		return nil, httperrors.NewInvalidStatusError("Cannot reset root with %s", self.ShutdownMode)
	}

	autoStart := false
	if input.AutoStart != nil {
		autoStart = *input.AutoStart
	}

	var needStop = false
	if self.Status == api.VM_RUNNING {
		needStop = true
	}
	resetPasswd := input.ResetPassword

	passwd := input.Password
	if len(passwd) > 0 {
		err = seclib2.ValidatePassword(passwd)
		if err != nil {
			return nil, errors.Wrap(err, "ValidatePassword")
		}
	}

	if len(input.KeypairId) > 0 {
		_, err := validators.ValidateModel(userCred, KeypairManager, &input.KeypairId)
		if err != nil {
			return nil, err
		}
		if self.KeypairId != input.KeypairId {
			err = self.setKeypairId(userCred, input.KeypairId)
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

	input.ResetPassword = resetPasswd

	return nil, self.StartRebuildRootTask(ctx, userCred, input.ImageId, needStop, autoStart, allDisks, &input.ServerDeployInputBase)
}

func (self *SGuest) GetTemplateId() string {
	gdc := self.CategorizeDisks()
	if gdc.Root != nil {
		return gdc.Root.GetTemplateId()
	}
	return ""
}

func (self *SGuest) StartRebuildRootTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, needStop, autoStart bool, allDisk bool, deployInput *api.ServerDeployInputBase) error {
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
	/*if resetPasswd {
		data.Set("reset_password", jsonutils.JSONTrue)
	} else {
		data.Set("reset_password", jsonutils.JSONFalse)
	}
	if len(passwd) > 0 {
		data.Set("password", jsonutils.NewString(passwd))
	}*/

	if allDisk {
		data.Set("all_disks", jsonutils.JSONTrue)
	} else {
		data.Set("all_disks", jsonutils.JSONFalse)
	}
	data.Set("deploy_params", jsonutils.Marshal(deployInput))

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
			return nil, httperrors.NewBadRequestError("%v", err)
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
	host, _ := self.GetHost()
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
		return nil, httperrors.NewOutOfQuotaError("%v", err)
	}

	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)

	err = self.CreateDisksOnHost(ctx, userCred, host, disksConf, pendingUsage, false, false, nil, nil, false)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CREATE, err.Error(), userCred, false)
		return nil, httperrors.NewBadRequestError("%v", err)
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

func (self *SGuest) PerformDetachdisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerDetachDiskInput) (jsonutils.JSONObject, error) {
	if len(input.DiskId) == 0 {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	diskObj, err := validators.ValidateModel(userCred, DiskManager, &input.DiskId)
	if err != nil {
		return nil, err
	}
	disk := diskObj.(*SDisk)
	attached, err := self.isAttach2Disk(disk)
	if err != nil {
		return nil, httperrors.NewInternalServerError("check isAttach2Disk fail %s", err)
	}
	if !attached {
		return nil, nil
	}
	detachDiskStatus, err := self.GetDriver().GetDetachDiskStatus()
	if err != nil {
		return nil, err
	}
	if input.KeepDisk && !self.GetDriver().CanKeepDetachDisk() {
		return nil, httperrors.NewInputParameterError("Cannot keep detached disk")
	}

	err = self.GetDriver().ValidateDetachDisk(ctx, userCred, self, disk)
	if err != nil {
		return nil, err
	}

	if utils.IsInStringArray(self.Status, detachDiskStatus) {
		self.SetStatus(userCred, api.VM_DETACH_DISK, "")
		err = self.StartGuestDetachdiskTask(ctx, userCred, disk, input.KeepDisk, "", false, false)
		return nil, err
	}
	return nil, httperrors.NewInvalidStatusError("Server in %s not able to detach disk", self.Status)
}

func (self *SGuest) StartGuestDetachdiskTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	disk *SDisk, keepDisk bool, parentTaskId string, purge bool, syncDescOnly bool,
) error {
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(disk.Id), "disk_id")
	taskData.Add(jsonutils.NewBool(keepDisk), "keep_disk")
	taskData.Add(jsonutils.NewBool(purge), "purge")

	// finally reuse fw_only option
	taskData.Add(jsonutils.NewBool(syncDescOnly), "sync_desc_only")
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

func (self *SGuest) PerformDetachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if !utils.IsInStringArray(self.GetStatus(), []string{api.VM_READY, api.VM_RUNNING}) {
		msg := fmt.Sprintf("Can't detach isolated device when guest is %s", self.GetStatus())
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
		err = self.startDetachIsolateDeviceWithoutNic(ctx, userCred, device)
		if err != nil {
			return nil, err
		}
	} else {
		devs, _ := self.GetIsolatedDevices()
		host, _ := self.GetHost()
		lockman.LockObject(ctx, host)
		defer lockman.ReleaseObject(ctx, host)
		for i := 0; i < len(devs); i++ {
			// check first
			dev := devs[i]
			if !utils.IsInStringArray(dev.DevType, api.VALID_ATTACH_TYPES) {
				if devModel, err := IsolatedDeviceModelManager.GetByDevType(dev.DevType); err != nil {
					msg := fmt.Sprintf("Can't separately detach dev type %s", dev.DevType)
					logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
					return nil, httperrors.NewBadRequestError(msg)
				} else {
					if !devModel.HotPluggable.Bool() && self.GetStatus() == api.VM_RUNNING {
						msg := fmt.Sprintf("dev type %s model %s unhotpluggable", dev.DevType, devModel.Model)
						logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
						return nil, httperrors.NewBadRequestError(msg)
					}
				}
			}
		}
		for i := 0; i < len(devs); i++ {
			err := self.detachIsolateDevice(ctx, userCred, &devs[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, self.startIsolatedDevicesSyncTask(ctx, userCred, jsonutils.QueryBoolean(data, "auto_start", false), "")
}

func (self *SGuest) startDetachIsolateDeviceWithoutNic(ctx context.Context, userCred mcclient.TokenCredential, device string) error {
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msgFmt := "Isolated device %s not found"
		msg := fmt.Sprintf(msgFmt, device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
		return httperrors.NewBadRequestError(msgFmt, device)
	}
	dev := iDev.(*SIsolatedDevice)
	if !utils.IsInStringArray(dev.DevType, api.VALID_ATTACH_TYPES) {
		if devModel, err := IsolatedDeviceModelManager.GetByDevType(dev.DevType); err != nil {
			msg := fmt.Sprintf("Can't separately detach dev type %s", dev.DevType)
			logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
			return httperrors.NewBadRequestError(msg)
		} else {
			if !devModel.HotPluggable.Bool() && self.GetStatus() == api.VM_RUNNING {
				msg := fmt.Sprintf("dev type %s model %s unhotpluggable", dev.DevType, devModel.Model)
				logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_DETACH_ISOLATED_DEVICE, msg, userCred, false)
				return httperrors.NewBadRequestError(msg)
			}
		}
	}

	host, _ := self.GetHost()
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
		dev.NetworkIndex = -1
		dev.DiskIndex = -1
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) PerformAttachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if !utils.IsInStringArray(self.GetStatus(), []string{api.VM_READY, api.VM_RUNNING}) {
		msg := fmt.Sprintf("Can't attach isolated device when guest is %s", self.GetStatus())
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return nil, httperrors.NewInvalidStatusError(msg)
	}
	var err error
	autoStart := jsonutils.QueryBoolean(data, "auto_start", false)
	if data.Contains("device") {
		device, _ := data.GetString("device")
		err = self.StartAttachIsolatedDeviceGpuOrUsb(ctx, userCred, device, autoStart)
	} else if data.Contains("model") {
		vmodel, _ := data.GetString("model")
		var count int64 = 1
		if data.Contains("count") {
			count, _ = data.Int("count")
		}
		if count < 1 {
			return nil, httperrors.NewBadRequestError("guest attach gpu count must > 0")
		}
		err = self.StartAttachIsolatedDevices(ctx, userCred, vmodel, int(count), autoStart)
	} else {
		return nil, httperrors.NewMissingParameterError("device||model")
	}

	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) StartAttachIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, devModel string, count int, autoStart bool) error {
	if err := self.startAttachIsolatedDevices(ctx, userCred, devModel, count); err != nil {
		return err
	}
	// perform post attach task
	return self.startIsolatedDevicesSyncTask(ctx, userCred, autoStart, "")
}

func (self *SGuest) startAttachIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, devModel string, count int) error {
	host, _ := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	devs, err := IsolatedDeviceManager.GetDevsOnHost(host.Id, devModel, count)
	if err != nil {
		return httperrors.NewInternalServerError("fetch gpu failed %s", err)
	}
	if len(devs) == 0 || len(devs) != count {
		return httperrors.NewBadRequestError("guest %s host %s isolated device not enough", self.GetName(), host.GetName())
	}
	dev := devs[0]
	if !utils.IsInStringArray(dev.DevType, api.VALID_ATTACH_TYPES) {
		if devModel, err := IsolatedDeviceModelManager.GetByDevType(dev.DevType); err != nil {
			return httperrors.NewBadRequestError("Can't separately attach dev type %s", dev.DevType)
		} else {
			if !devModel.HotPluggable.Bool() && self.GetStatus() == api.VM_RUNNING {
				return httperrors.NewBadRequestError("dev type %s model %s unhotpluggable", dev.DevType, devModel.Model)
			}
		}
	}

	if dev.DevType == api.LEGACY_VGPU_TYPE {
		devs, err := self.GetIsolatedDevices()
		if err != nil {
			return errors.Wrap(err, "get isolated devices")
		}
		for i := range devs {
			if devs[i].DevType == api.LEGACY_VGPU_TYPE {
				return httperrors.NewBadRequestError("Nvidia vgpu count exceed > 1")
			} else if utils.IsInStringArray(devs[i].DevType, api.VALID_GPU_TYPES) {
				return httperrors.NewBadRequestError("Nvidia vgpu can't passthrough with other gpus")
			}
		}
	}

	defer func() { go host.ClearSchedDescCache() }()
	for i := 0; i < len(devs); i++ {
		err = self.attachIsolatedDevice(ctx, userCred, &devs[i], nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) StartAttachIsolatedDeviceGpuOrUsb(ctx context.Context, userCred mcclient.TokenCredential, device string, autoStart bool) error {
	if err := self.startAttachIsolatedDevGeneral(ctx, userCred, device); err != nil {
		return err
	}
	// perform post attach task
	return self.startIsolatedDevicesSyncTask(ctx, userCred, autoStart, "")
}

func (self *SGuest) startAttachIsolatedDevGeneral(ctx context.Context, userCred mcclient.TokenCredential, device string) error {
	iDev, err := IsolatedDeviceManager.FetchByIdOrName(userCred, device)
	if err != nil {
		msgFmt := "Isolated device %s not found"
		msg := fmt.Sprintf(msgFmt, device)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, false)
		return httperrors.NewBadRequestError(msgFmt, device)
	}
	dev := iDev.(*SIsolatedDevice)
	if !utils.IsInStringArray(dev.DevType, api.VALID_ATTACH_TYPES) {
		if devModel, err := IsolatedDeviceModelManager.GetByDevType(dev.DevType); err != nil {
			return httperrors.NewBadRequestError("Can't separately attach dev type %s", dev.DevType)
		} else {
			if !devModel.HotPluggable.Bool() && self.GetStatus() == api.VM_RUNNING {
				return httperrors.NewBadRequestError("dev type %s model %s unhotpluggable", dev.DevType, devModel.Model)
			}
		}
	}
	if !utils.IsInStringArray(self.GetStatus(), []string{api.VM_READY, api.VM_RUNNING}) {
		return httperrors.NewInvalidStatusError("Can't attach GPU when status is %q", self.GetStatus())
	}

	if dev.DevType == api.LEGACY_VGPU_TYPE {
		devs, err := self.GetIsolatedDevices()
		if err != nil {
			return errors.Wrap(err, "get isolated devices")
		}
		for i := range devs {
			if devs[i].DevType == api.LEGACY_VGPU_TYPE {
				return httperrors.NewBadRequestError("Nvidia vgpu count exceed > 1")
			} else if utils.IsInStringArray(devs[i].DevType, api.VALID_GPU_TYPES) {
				return httperrors.NewBadRequestError("Nvidia vgpu can't passthrough with other gpus")
			}
		}
	}

	host, _ := self.GetHost()
	lockman.LockObject(ctx, host)
	defer lockman.ReleaseObject(ctx, host)
	err = self.attachIsolatedDevice(ctx, userCred, dev, nil, nil)
	var msg string
	if err != nil {
		msg = err.Error()
	} else {
		go host.ClearSchedDescCache()
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_GUEST_ATTACH_ISOLATED_DEVICE, msg, userCred, err == nil)
	return err
}

func (self *SGuest) attachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, dev *SIsolatedDevice, networkIndex, diskIndex *int8) error {
	if len(dev.GuestId) > 0 {
		return fmt.Errorf("Isolated device already attached to another guest: %s", dev.GuestId)
	}
	if dev.HostId !=
		self.HostId {
		return fmt.Errorf("Isolated device and guest are not located in the same host")
	}
	_, err := db.Update(dev, func() error {
		dev.GuestId = self.Id
		if networkIndex != nil {
			dev.NetworkIndex = *networkIndex
		} else {
			dev.NetworkIndex = -1
		}
		if diskIndex != nil {
			dev.DiskIndex = *diskIndex
		} else {
			dev.DiskIndex = -1
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_ATTACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) PerformSetIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewNotAcceptableError("Not allow for hypervisor %s", self.Hypervisor)
	}
	if !utils.IsInStringArray(self.GetStatus(), []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Can't set isolated device when guest is %s", self.GetStatus())
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
		err := self.startDetachIsolateDeviceWithoutNic(ctx, userCred, delDevs[i])
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(addDevs); i++ {
		err := self.startAttachIsolatedDevGeneral(ctx, userCred, addDevs[i])
		if err != nil {
			return nil, err
		}
	}
	return nil, self.startIsolatedDevicesSyncTask(ctx, userCred, jsonutils.QueryBoolean(data, "auto_start", false), "")
}

func (self *SGuest) startIsolatedDevicesSyncTask(ctx context.Context, userCred mcclient.TokenCredential, autoStart bool, parenetId string) error {
	if self.GetStatus() == api.VM_RUNNING {
		autoStart = false
	}
	data := jsonutils.Marshal(map[string]interface{}{
		"auto_start": autoStart,
	}).(*jsonutils.JSONDict)
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestIsolatedDeviceSyncTask", self, userCred, data, parenetId, "", nil); err != nil {
		return err
	} else {
		return task.ScheduleRun(nil)
	}
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

func (self *SGuest) getReuseAddr(gn *SGuestnetwork) string {
	if self.GetHypervisor() != api.HYPERVISOR_BAREMETAL {
		return ""
	}
	host, _ := self.GetHost()
	hostNics := host.GetNics()
	for _, hn := range hostNics {
		if hn.GetMac().String() == gn.MacAddr {
			return hn.IpAddr
		}
	}
	return ""
}

// Change IPaddress of a guestnetwork
// first detach the network, then attach a network with identity mac address but different IP configurations
// TODO change IP address of a teaming NIC may fail!!
func (self *SGuest) PerformChangeIpaddr(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != api.VM_READY && self.Status != api.VM_RUNNING && self.Status != api.VM_BLOCK_STREAM {
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
	conf, err = parseNetworkInfo(ctx, userCred, conf)
	if err != nil {
		return nil, err
	}
	err = isValidNetworkInfo(ctx, userCred, conf, self.getReuseAddr(gn))
	if err != nil {
		return nil, err
	}
	host, _ := self.GetHost()

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
			return nil, httperrors.NewBadRequestError("%v", err)
		}
		if _, err := db.Update(&ngn[0], func() error {
			ngn[0].EipId = gn.EipId
			return nil
		}); err != nil {
			return nil, err
		}

		return ngn, nil
	}()

	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CHANGE_NIC, err, userCred, false)
		return nil, err
	}

	//Get the detailed description of the NIC
	networkJsonDesc := ngn[0].getJsonDesc()
	newIpAddr := networkJsonDesc.Ip
	newMacAddr := networkJsonDesc.Mac
	newMaskLen := networkJsonDesc.Masklen
	newGateway := networkJsonDesc.Gateway
	ipMask := fmt.Sprintf("%s/%d", newIpAddr, newMaskLen)

	notes := jsonutils.NewDict()
	if gn != nil {
		notes.Add(jsonutils.NewString(gn.IpAddr), "prev_ip")
	}
	if ngn != nil {
		notes.Add(jsonutils.NewString(ngn[0].IpAddr), "ip")
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CHANGE_NIC, notes, userCred, true)

	restartNetwork, _ := data.Bool("restart_network")

	taskData := jsonutils.NewDict()
	if self.Hypervisor == api.HYPERVISOR_KVM && restartNetwork && (self.Status == api.VM_RUNNING || self.Status == api.VM_BLOCK_STREAM) {
		taskData.Set("restart_network", jsonutils.JSONTrue)
		taskData.Set("prev_ip", jsonutils.NewString(gn.IpAddr))
		taskData.Set("prev_mac", jsonutils.NewString(newMacAddr))
		net := ngn[0].GetNetwork()
		taskData.Set("is_vpc_network", jsonutils.NewBool(net.isOneCloudVpcNetwork()))
		taskData.Set("ip_mask", jsonutils.NewString(ipMask))
		taskData.Set("gateway", jsonutils.NewString(newGateway))
		if self.Status == api.VM_BLOCK_STREAM {
			taskData.Set("in_block_stream", jsonutils.JSONTrue)
		}
		self.SetStatus(userCred, api.VM_RESTART_NETWORK, "restart network")
	}
	return nil, self.startSyncTask(ctx, userCred, false, "", taskData)
}

func (self *SGuest) GetIfNameByMac(ctx context.Context, userCred mcclient.TokenCredential, mac string) (string, error) {
	//Find the network card according to the mac address, if it is empty, it means no network card is found
	ifnameData, err := self.PerformQgaGetNetwork(ctx, userCred, nil, nil)
	if err != nil {
		return "", err
	}
	//Get the name of the network card
	var parsedData []api.IfnameDetail
	if err := ifnameData.Unmarshal(&parsedData); err != nil {
		return "", err
	}
	var ifnameDevice string
	//Finding a network card by its mac address
	for _, detail := range parsedData {
		if detail.HardwareAddress == mac {
			ifnameDevice = detail.Name
		}
	}
	return ifnameDevice, nil
}

func (self *SGuest) PerformDetachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerDetachnetworkInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot detach network in status %s", self.Status)
	}

	err := self.GetDriver().ValidateDetachNetwork(ctx, userCred, self)
	if err != nil {
		return nil, err
	}

	var gns []SGuestnetwork
	if len(input.NetId) > 0 {
		netObj, err := validators.ValidateModel(userCred, NetworkManager, &input.NetId)
		if err != nil {
			return nil, err
		}
		gns, err = self.GetNetworks(netObj.GetId())
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	} else if len(input.IpAddr) > 0 {
		gn, err := self.GetGuestnetworkByIp(input.IpAddr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("ip %s not found", input.IpAddr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		gns = []SGuestnetwork{*gn}
	} else if len(input.Mac) > 0 {
		gn, err := self.GetGuestnetworkByMac(input.Mac)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("mac %s not found", input.Mac)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		gns = []SGuestnetwork{*gn}
	} else {
		return nil, httperrors.NewMissingParameterError("net_id")
	}

	if self.Status == api.VM_READY {
		return nil, self.detachNetworks(ctx, userCred, gns, input.Reserve, true)
	}
	err = self.detachNetworks(ctx, userCred, gns, input.Reserve, false)
	if err != nil {
		return nil, err
	}
	return nil, self.StartSyncTask(ctx, userCred, false, "")
}

func (self *SGuest) PerformAttachnetwork(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.AttachNetworkInput) (*api.SGuest, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewBadRequestError("Cannot attach network in status %s", self.Status)
	}
	count := len(input.Nets)
	if count == 0 {
		return nil, httperrors.NewMissingParameterError("nets")
	}
	var inicCnt, enicCnt, isolatedDevCount int
	for i := 0; i < count; i++ {
		err := isValidNetworkInfo(ctx, userCred, input.Nets[i], "")
		if err != nil {
			return nil, err
		}
		if IsExitNetworkInfo(userCred, input.Nets[i]) {
			enicCnt = count
			// ebw = input.BwLimit
		} else {
			inicCnt = count
			// ibw = input.BwLimit
		}
		if input.Nets[i].SriovDevice != nil {
			if self.BackupHostId != "" {
				return nil, httperrors.NewBadRequestError("Cannot create backup with isolated device")
			}
			devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, input.Nets[i].SriovDevice)
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
			}
			err = IsolatedDeviceManager.isValidNicDeviceInfo(devConfig)
			if err != nil {
				return nil, err
			}
			input.Nets[i].SriovDevice = devConfig
			input.Nets[i].Driver = api.NETWORK_DRIVER_VFIO
			isolatedDevCount += 1
		}
	}

	pendingUsage := &SRegionQuota{
		Port:  inicCnt,
		Eport: enicCnt,
		//Bw:    ibw,
		//Ebw:   ebw,
	}
	pendingUsageHost := &SQuota{IsolatedDevice: isolatedDevCount}
	keys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		return nil, err
	}
	quotakeys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	pendingUsageHost.SetKeys(quotakeys)
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("%v", err)
	}
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsageHost)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("%v", err)
	}
	host, _ := self.GetHost()
	defer host.ClearSchedDescCache()
	for i := 0; i < count; i++ {
		gns, err := self.attach2NetworkDesc(ctx, userCred, host, input.Nets[i], pendingUsage, nil)
		logclient.AddSimpleActionLog(self, logclient.ACT_ATTACH_NETWORK, input.Nets[i], userCred, err == nil)
		if err != nil {
			quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
			return nil, httperrors.NewBadRequestError("%v", err)
		}
		net := gns[0].GetNetwork()
		if input.Nets[i].SriovDevice != nil {
			input.Nets[i].SriovDevice.NetworkIndex = &gns[0].Index
			input.Nets[i].SriovDevice.WireId = net.WireId
			err = self.createIsolatedDeviceOnHost(ctx, userCred, host, input.Nets[i].SriovDevice, pendingUsageHost)
			if err != nil {
				quotas.CancelPendingUsage(ctx, userCred, pendingUsageHost, pendingUsageHost, false)
				return nil, errors.Wrap(err, "self.createIsolatedDeviceOnHost")
			}
		}
	}

	if self.Status == api.VM_READY {
		err = self.StartGuestDeployTask(ctx, userCred, nil, "deploy", "")
	} else {
		err = self.StartSyncTask(ctx, userCred, false, "")
	}
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, pendingUsage, pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, pendingUsageHost, pendingUsageHost, false)
	}
	return nil, err
}

func (self *SGuest) PerformChangeBandwidth(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewBadRequestError("Cannot change bandwidth in status %s", self.Status)
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

func (self *SGuest) PerformModifySrcCheck(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewBadRequestError("Cannot change setting in status %s", self.Status)
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

func (self *SGuest) PerformChangeConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerChangeConfigInput) (jsonutils.JSONObject, error) {
	if !self.GetDriver().AllowReconfigGuest() {
		return nil, httperrors.NewInvalidStatusError("Not allow to change config")
	}

	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Guest have backup not allow to change config")
	}

	changeStatus, err := self.GetDriver().GetChangeConfigStatus(self)
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	if !utils.IsInStringArray(self.Status, changeStatus) {
		return nil, httperrors.NewInvalidStatusError("Cannot change config in %s for %s, requires %s", self.Status, self.GetHypervisor(), changeStatus)
	}

	_, err = self.GetHost()
	if err != nil {
		return nil, errors.Wrapf(err, "GetHost")
	}

	var addCpu, addMem int
	var cpuChanged, memChanged bool

	confs := jsonutils.NewDict()
	confs.Add(jsonutils.Marshal(map[string]interface{}{
		"instance_type": self.InstanceType,
		"vcpu_count":    self.VcpuCount,
		"vmem_size":     self.VmemSize,
	}), "old")
	if len(input.InstanceType) > 0 {
		sku, err := ServerSkuManager.FetchSkuByNameAndProvider(input.InstanceType, self.GetDriver().GetProvider(), true)
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
		if input.VcpuCount != self.VcpuCount {
			cpuChanged = true
			addCpu = input.VcpuCount - self.VcpuCount
			confs.Add(jsonutils.NewInt(int64(input.VcpuCount)), "vcpu_count")
		}
		if !regutils.MatchSize(input.VmemSize) {
			return nil, httperrors.NewBadRequestError("Memory size %q must be number[+unit], like 256M, 1G or 256", input.VmemSize)
		}
		nVmem, err := fileutils.GetSizeMb(input.VmemSize, 'M', 1024)
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

	if self.Status == api.VM_RUNNING && (cpuChanged || memChanged) && self.GetDriver().NeedStopForChangeSpec(ctx, self, cpuChanged, memChanged) {
		return nil, httperrors.NewInvalidStatusError("cannot change CPU/Memory spec in status %s", self.Status)
	}

	if addCpu < 0 {
		addCpu = 0
	}
	if addMem < 0 {
		addMem = 0
	}

	disks, err := self.GetGuestDisks()
	if err != nil {
		return nil, err
	}
	var addDisk int
	var newDiskIdx = 0
	var newDisks = make([]*api.DiskConfig, 0)
	var resizeDisks = jsonutils.NewArray()

	var schedInputDisks = make([]*api.DiskConfig, 0)
	var diskIdx = 1
	for i := range input.Disks {
		disk := input.Disks[i]
		if len(disk.SnapshotId) > 0 {
			snapObj, err := SnapshotManager.FetchById(disk.SnapshotId)
			if err != nil {
				return nil, httperrors.NewResourceNotFoundError("snapshot %s not found", disk.SnapshotId)
			}
			snap := snapObj.(*SSnapshot)
			disk.Storage = snap.StorageId
		}
		if len(disk.Backend) == 0 && len(disk.Storage) == 0 {
			disk.Backend = self.getDefaultStorageType()
		}
		if disk.SizeMb > 0 {
			if diskIdx >= len(disks) {
				newDisks = append(newDisks, &disk)
				newDiskIdx += 1
				addDisk += disk.SizeMb
				schedInputDisks = append(schedInputDisks, &disk)
			} else {
				gDisk := disks[diskIdx].GetDisk()
				oldSize := gDisk.DiskSize
				if disk.SizeMb < oldSize {
					return nil, httperrors.NewInputParameterError("Cannot reduce disk size")
				}
				if disk.SizeMb > oldSize {
					arr := jsonutils.NewArray(jsonutils.NewString(disks[diskIdx].DiskId), jsonutils.NewInt(int64(disk.SizeMb)))
					resizeDisks.Add(arr)
					addDisk += disk.SizeMb - oldSize
					storage, _ := disks[diskIdx].GetDisk().GetStorage()
					schedInputDisks = append(schedInputDisks, &api.DiskConfig{
						SizeMb:  addDisk,
						Index:   disk.Index,
						Storage: storage.Id,
					})
				}
			}
		}
		diskIdx += 1
	}

	if resizeDisks.Length() > 0 {
		confs.Add(resizeDisks, "resize")
	}
	if self.Status != api.VM_RUNNING && input.AutoStart {
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
	schedDesc := self.changeConfToSchedDesc(addCpu, addMem, schedInputDisks)
	confs.Set("sched_desc", jsonutils.Marshal(schedDesc))
	s := auth.GetAdminSession(ctx, options.Options.Region)
	canChangeConf, res, err := scheduler.SchedManager.DoScheduleForecast(s, schedDesc, 1)
	if err != nil {
		return nil, err
	}
	if !canChangeConf {
		return nil, httperrors.NewInsufficientResourceError(res.String())
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
		confs.Add(jsonutils.Marshal(newDisks), "create")
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_CHANGE_CONFIG, confs, userCred, true)
	self.StartChangeConfigTask(ctx, userCred, confs, "", pendingUsage)
	return nil, nil
}

func (self *SGuest) changeConfToSchedDesc(addCpu, addMem int, schedInputDisks []*api.DiskConfig) *schedapi.ScheduleInput {
	devs, _ := self.GetIsolatedDevices()
	desc := &schedapi.ScheduleInput{
		ServerConfig: schedapi.ServerConfig{
			ServerConfigs: &api.ServerConfigs{
				Hypervisor: self.Hypervisor,
				PreferHost: self.HostId,
				Disks:      schedInputDisks,
			},
			Memory:  addMem,
			Ncpu:    addCpu,
			Project: self.ProjectId,
			Domain:  self.DomainId,
		},
		OsArch:            self.OsArch,
		ChangeConfig:      true,
		HasIsolatedDevice: len(devs) > 0,
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

func (self *SGuest) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	eip, _ := self.GetEipOrPublicIp()
	if eip != nil {
		eip.DoPendingDelete(ctx, userCred)
	}

	disks, _ := self.GetDisks()
	for i := range disks {
		if !disks[i].IsDetachable() {
			disks[i].DoPendingDelete(ctx, userCred)
		} else {
			log.Warningf("detachable disk on pending delete guests!!! should be removed earlier")
			self.DetachDisk(ctx, &disks[i], userCred)
		}
	}
	self.SVirtualResourceBase.DoPendingDelete(ctx, userCred)
}

func (self *SGuest) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted && !self.Deleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "DoCancelPendingDelete")
		}
		self.RecoverUsages(ctx, userCred)
	}
	return nil, nil
}

func (self *SGuest) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	eip, _ := self.GetEipOrPublicIp()
	if eip != nil {
		eip.DoCancelPendingDelete(ctx, userCred)
	}

	disks, _ := self.GetDisks()
	for _, disk := range disks {
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
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})
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

func (self *SGuest) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isHard := jsonutils.QueryBoolean(data, "is_hard", false)
	if self.Status == api.VM_RUNNING || self.Status == api.VM_STOP_FAILED {
		self.GetDriver().StartGuestResetTask(self, ctx, userCred, isHard, "")
		return nil, nil
	}
	return nil, httperrors.NewInvalidStatusError("Cannot reset VM in status %s", self.Status)
}

// 同步虚拟机状态
func (self *SGuest) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var openTask = true
	count, err := taskman.TaskManager.QueryTasksOfObject(self, time.Now().Add(-3*time.Minute), &openTask).CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, httperrors.NewBadRequestError("Guest has %d task active, can't sync status", count)
	}

	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SGuest) isNotRunningStatus(status string) bool {
	if status == api.VM_READY || status == api.VM_SUSPEND {
		return true
	}
	return false
}

func (self *SGuest) SetStatus(userCred mcclient.TokenCredential, status, reason string) error {
	if status == api.VM_RUNNING {
		if err := self.SetPowerStates(api.VM_POWER_STATES_ON); err != nil {
			return err
		}
	} else if status == api.VM_READY {
		if err := self.SetPowerStates(api.VM_POWER_STATES_OFF); err != nil {
			return err
		}
	}

	return self.SVirtualResourceBase.SetStatus(userCred, status, reason)
}

func (self *SGuest) SetPowerStates(powerStates string) error {
	if self.PowerStates == powerStates {
		return nil
	}
	_, err := db.Update(self, func() error {
		self.PowerStates = powerStates
		return nil
	})
	return errors.Wrap(err, "Update power states")
}

func (self *SGuest) SetBackupGuestStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if self.BackupGuestStatus == status {
		return nil
	}
	oldStatus := self.BackupGuestStatus
	_, err := db.Update(self, func() error {
		self.BackupGuestStatus = status
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update backup guest status")
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_BACKUP_GUEST_STATUS, notes, userCred)
		logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE_BACKUP_GUEST_STATUS, notes, userCred, true)
	}
	return nil
}

func (self *SGuest) PerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformStatusInput) (jsonutils.JSONObject, error) {
	if input.HostId != "" && self.BackupHostId != "" && input.HostId == self.BackupHostId {
		// perform status called from slave guest
		return nil, self.SetBackupGuestStatus(userCred, input.Status, input.Reason)
	}

	if input.HostId != "" && input.HostId != self.HostId {
		// perform status called from volatile host, eg: migrate dest host
		return nil, nil
	}

	if input.PowerStates != "" {
		if err := self.SetPowerStates(input.PowerStates); err != nil {
			return nil, errors.Wrap(err, "set power states")
		}
	}

	preStatus := self.Status
	_, err := self.SVirtualResourceBase.PerformStatus(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.PerformStatus")
	}

	if self.HasBackupGuest() {
		if input.Status == api.VM_READY {
			if err := self.ResetGuestQuorumChildIndex(ctx, userCred); err != nil {
				return nil, errors.Wrap(err, "reset guest quorum child index")
			}
		}
	}

	if ispId := self.GetMetadata(ctx, api.BASE_INSTANCE_SNAPSHOT_ID, userCred); len(ispId) > 0 {
		ispM, err := InstanceSnapshotManager.FetchById(ispId)
		if err == nil {
			isp := ispM.(*SInstanceSnapshot)
			isp.DecRefCount(ctx, userCred)
		}
		self.SetMetadata(ctx, api.BASE_INSTANCE_SNAPSHOT_ID, "", userCred)
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

func (self *SGuest) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	input api.ServerStopInput) (jsonutils.JSONObject, error) {
	// XXX if is force, force stop guest
	if input.IsForce || utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_STOP_FAILED}) {
		if err := self.ValidateEncryption(ctx, userCred); err != nil {
			return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
		}
		return nil, self.StartGuestStopTask(ctx, userCred, input.IsForce, input.StopCharging, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot stop server in status %s", self.Status)
}

func (self *SGuest) PerformFreeze(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformFreezeInput) (jsonutils.JSONObject, error) {
	if self.Freezed {
		return nil, httperrors.NewBadRequestError("virtual resource already freezed")
	}
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_STOP_FAILED}) {
		return nil, self.StartGuestStopAndFreezeTask(ctx, userCred)
	} else {
		return self.SVirtualResourceBase.PerformFreeze(ctx, userCred, query, input)
	}
}

func (self *SGuest) StartGuestStopAndFreezeTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(userCred, api.VM_START_STOP, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStopAndFreezeTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) PerformRestart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isForce := jsonutils.QueryBoolean(data, "is_force", false)
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_STOP_FAILED}) || (isForce && self.Status == api.VM_STOPPING) {
		return nil, self.GetDriver().StartGuestRestartTask(self, ctx, userCred, isForce, "")
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do restart server in status %s", self.Status)
	}
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
		return nil, httperrors.NewBadRequestError("%v", err)
	}
	cmd := fmt.Sprintf("sendkey %s", keys)
	duration, err := data.Int("duration")
	if err == nil {
		cmd = fmt.Sprintf("%s %d", cmd, duration)
	}
	_, err = self.SendMonitorCommand(ctx, userCred, &api.ServerMonitorInput{COMMAND: cmd})
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

func (self *SGuest) SendMonitorCommand(ctx context.Context, userCred mcclient.TokenCredential, cmd *api.ServerMonitorInput) (jsonutils.JSONObject, error) {
	host, _ := self.GetHost()
	url := fmt.Sprintf("%s/servers/%s/monitor", host.ManagerUri, self.Id)
	header := http.Header{}
	header.Add("X-Auth-Token", userCred.GetTokenString())
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(cmd.COMMAND), "cmd")
	body.Add(jsonutils.NewBool(cmd.QMP), "qmp")
	_, res, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return nil, err
	}
	ret := res.(*jsonutils.JSONDict)
	return ret, nil
}

func (self *SGuest) PerformAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerAssociateEipInput) (jsonutils.JSONObject, error) {
	err := self.IsEipAssociable()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	eipStr := input.EipId
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

	eip := eipObj.(*SElasticip)
	eipRegion, err := eip.GetRegion()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "eip.GetRegion"))
	}
	instRegion, _ := self.getRegion()

	if eip.Mode == api.EIP_MODE_INSTANCE_PUBLICIP {
		return nil, httperrors.NewUnsupportOperationError("fixed eip cannot be associated")
	}

	if eip.IsAssociated() {
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

	eipZone, _ := eip.GetZone()
	if eipZone != nil {
		insZone, _ := self.getZone()
		if eipZone.Id != insZone.Id {
			return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different zone")
		}
	}

	host, _ := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInputParameterError("server host is not found???")
	}

	if host.ManagerId != eip.ManagerId {
		return nil, httperrors.NewInputParameterError("cannot associate eip and instance in different provider")
	}

	self.SetStatus(userCred, api.INSTANCE_ASSOCIATE_EIP, "associate eip")

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.ExternalId), "instance_external_id")
	params.Add(jsonutils.NewString(self.Id), "instance_id")
	params.Add(jsonutils.NewString(api.EIP_ASSOCIATE_TYPE_SERVER), "instance_type")
	if len(input.IpAddr) > 0 {
		params.Add(jsonutils.NewString(input.IpAddr), "ip_addr")
	}

	err = eip.StartEipAssociateTask(ctx, userCred, params, "")

	return nil, err
}

func (self *SGuest) PerformDissociateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerDissociateEipInput) (jsonutils.JSONObject, error) {
	eip, err := self.GetElasticIp()
	if err != nil {
		log.Errorf("Fail to get Eip %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	if eip == nil {
		return nil, httperrors.NewInvalidStatusError("No eip to dissociate")
	}

	err = db.IsObjectRbacAllowed(ctx, eip, userCred, policy.PolicyActionGet)
	if err != nil {
		return nil, errors.Wrap(err, "eip is not accessible")
	}

	self.SetStatus(userCred, api.INSTANCE_DISSOCIATE_EIP, "associate eip")

	autoDelete := (input.AudoDelete != nil && *input.AudoDelete)

	err = eip.StartEipDissociateTask(ctx, userCred, autoDelete, "")
	if err != nil {
		log.Errorf("fail to start dissociate task %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

func (self *SGuest) PerformCreateEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerCreateEipInput) (jsonutils.JSONObject, error) {
	var (
		host, _      = self.GetHost()
		region, _    = host.GetRegion()
		regionDriver = region.GetDriver()

		bw            = input.Bandwidth
		chargeType    = input.ChargeType
		bgpType       = input.BgpType
		autoDellocate = (input.AutoDellocate != nil && *input.AutoDellocate)
	)

	err := self.IsEipAssociable()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	if chargeType == "" {
		chargeType = regionDriver.GetEipDefaultChargeType()
	}

	if chargeType == api.EIP_CHARGE_TYPE_BY_BANDWIDTH {
		if bw == 0 {
			return nil, httperrors.NewMissingParameterError("bandwidth")
		}
	}

	err = self.GetDriver().ValidateCreateEip(ctx, userCred, input)
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

	eip, err := ElasticipManager.NewEipForVMOnHost(ctx, userCred, &NewEipForVMOnHostArgs{
		Bandwidth:     int(bw),
		BgpType:       bgpType,
		ChargeType:    chargeType,
		AutoDellocate: autoDellocate,

		Guest:        self,
		Host:         host,
		PendingUsage: eipPendingUsage,
	})
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, eipPendingUsage, eipPendingUsage, false)
		return nil, httperrors.NewGeneralError(err)
	}

	opts := api.ElasticipAssociateInput{
		InstanceId:         self.Id,
		InstanceExternalId: self.ExternalId,
		InstanceType:       api.EIP_ASSOCIATE_TYPE_SERVER,
	}

	err = eip.AllocateAndAssociateInstance(ctx, userCred, self, opts, "")
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (self *SGuest) setUserData(ctx context.Context, userCred mcclient.TokenCredential, data string) error {
	if err := userdata.ValidateUserdata(data, self.OsType); err != nil {
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

func (self *SGuest) GetUserData(ctx context.Context, userCred mcclient.TokenCredential) string {
	userData := self.GetMetadata(ctx, "user_data", userCred)
	if len(userData) == 0 {
		return userData
	}
	decodeData, _ := userdata.Decode(userData)
	return decodeData
}

func (self *SGuest) PerformUserData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerUserDataInput) (jsonutils.JSONObject, error) {
	if len(input.UserData) == 0 {
		return nil, httperrors.NewMissingParameterError("user_data")
	}
	err := self.setUserData(ctx, userCred, input.UserData)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(self.HostId) > 0 {
		return nil, self.StartSyncTask(ctx, userCred, false, "")
	}
	return nil, nil
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
	usbKbd, err := data.GetString("disable_usb_kbd")
	if err == nil {
		err = self.SetMetadata(ctx, "disable_usb_kbd", usbKbd, userCred)
		if err != nil {
			return nil, err
		}
	}
	usbContType, err := data.GetString("usb_controller_type")
	if err == nil {
		err = self.SetMetadata(ctx, "usb_controller_type", usbContType, userCred)
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

func (self *SGuest) PerformSwitchToBackup(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if self.Status == api.VM_BLOCK_STREAM {
		return nil, httperrors.NewBadRequestError("Cannot swith to backup when guest in status %s", self.Status)
	}
	if len(self.BackupHostId) == 0 {
		return nil, httperrors.NewBadRequestError("Guest no backup host")
	}
	backupHost := HostManager.FetchHostById(self.BackupHostId)
	if backupHost.HostStatus != api.HOST_ONLINE {
		return nil, httperrors.NewBadRequestError("Can't switch to backup host on host status %s", backupHost.HostStatus)
	}

	if !self.IsGuestBackupMirrorJobReady(ctx, userCred) {
		return nil, httperrors.NewBadRequestError("Guest can't switch to backup, mirror job not ready")
	}
	if !utils.IsInStringArray(self.BackupGuestStatus, []string{api.VM_RUNNING, api.VM_READY, api.VM_UNKNOWN}) {
		return nil, httperrors.NewInvalidStatusError("Guest can't switch to backup with backup status %s", self.BackupGuestStatus)
	}

	oldStatus := self.Status
	taskData := jsonutils.NewDict()
	taskData.Set("old_status", jsonutils.NewString(oldStatus))
	taskData.Set("auto_start", jsonutils.NewBool(jsonutils.QueryBoolean(data, "auto_start", false)))
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSwitchToBackupTask", self, userCred, taskData, "", "", nil); err != nil {
		log.Errorln(err)
		return nil, err
	} else {
		self.SetStatus(userCred, api.VM_SWITCH_TO_BACKUP, "Switch to backup")
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
	q2 = manager.FilterByOwner(q2, manager, userCred, userCred, manager.NamespaceScope())
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
		dictStore[db.USER_TAG_PREFIX+k], _ = v.GetString()
	}
	return dictStore, nil
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

func (self *SGuest) PerformBlockStreamFailed(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		if err := self.SetGuestBackupMirrorJobFailed(ctx, userCred); err != nil {
			return nil, errors.Wrap(err, "set guest backup mirror job failed")
		}
	}
	if self.Status == api.VM_BLOCK_STREAM || self.Status == api.VM_RUNNING {
		reason, _ := data.GetString("reason")
		logclient.AddSimpleActionLog(self, logclient.ACT_VM_BLOCK_STREAM, reason, userCred, false)
		return nil, self.SetStatus(userCred, api.VM_BLOCK_STREAM_FAIL, reason)
	}
	return nil, nil
}

func (self *SGuest) PerformSlaveBlockStreamReady(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		if err := self.TrySetGuestBackupMirrorJobReady(ctx, userCred); err != nil {
			return nil, errors.Wrap(err, "set guest backup mirror job status ready")
		}
		self.SetBackupGuestStatus(userCred, api.VM_RUNNING, "perform slave block stream ready")
	}
	return nil, nil
}

func (self *SGuest) PerformBlockMirrorReady(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status == api.VM_BLOCK_STREAM || self.Status == api.VM_RUNNING {
		diskId, err := data.GetString("disk_id")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("disk_id")
		}
		log.Infof("disk_id %s", diskId)
		disk := DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, httperrors.NewNotFoundError("disk %s not found", diskId)
		}
		if taskId := disk.GetMetadata(ctx, api.DISK_CLONE_TASK_ID, userCred); len(taskId) > 0 {
			log.Infof("task_id %s", taskId)
			if err := self.startSwitchToClonedDisk(ctx, userCred, taskId); err != nil {
				return nil, errors.Wrap(err, "startSwitchToClonedDisk")
			}
		}
	}
	return nil, nil
}

func (self *SGuest) StartMirrorJob(ctx context.Context, userCred mcclient.TokenCredential, nbdServerPort int64, parentTaskId string) error {
	taskData := jsonutils.NewDict()
	taskData.Set("nbd_server_port", jsonutils.NewInt(nbdServerPort))
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestReSyncToBackup", self, userCred, taskData, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
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

func (self *SGuest) guestDisksStorageTypeIsLocal() bool {
	disks, _ := self.GetDisks()
	for _, disk := range disks {
		storage, _ := disk.GetStorage()
		if storage.StorageType != api.STORAGE_LOCAL {
			return false
		}
	}
	return true
}

func (self *SGuest) guestDisksStorageTypeIsShared() bool {
	disks, _ := self.GetDisks()
	for _, disk := range disks {
		storage, _ := disk.GetStorage()
		if storage.StorageType == api.STORAGE_LOCAL {
			return false
		}
	}
	return true
}

func (self *SGuest) PerformCreateBackup(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if len(self.BackupHostId) > 0 {
		return nil, httperrors.NewBadRequestError("Already have backup server")
	}
	if self.Status != api.VM_READY {
		return nil, httperrors.NewBadRequestError("Can't create backup in guest status %s", self.Status)
	}
	if !self.guestDisksStorageTypeIsLocal() {
		return nil, httperrors.NewBadRequestError("Cannot create backup with shared storage")
	}
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Backup only support hypervisor kvm")
	}
	devs, _ := self.GetIsolatedDevices()
	if len(devs) > 0 {
		return nil, httperrors.NewBadRequestError("Cannot create backup with isolated devices")
	}
	hasSnapshot, err := self.GuestDisksHasSnapshot()
	if err != nil {
		return nil, httperrors.NewInternalServerError("GuestDisksHasSnapshot fail %s", err)
	}
	if hasSnapshot {
		return nil, httperrors.NewBadRequestError("Cannot create backup with snapshot")
	}
	return self.StartGuestCreateBackupTask(ctx, userCred, "", data)
}

func (self *SGuest) StartGuestCreateBackupTask(
	ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	req := self.getGuestBackupResourceRequirements(ctx, userCred)
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, err
	}
	req.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &req)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError("%v", err)
	}

	params := data.(*jsonutils.JSONDict)
	params.Set("guest_status", jsonutils.NewString(self.Status))
	self.SetStatus(userCred, api.VM_BACKUP_CREATING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCreateBackupTask", self, userCred, params, parentTaskId, "", &req)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &req, &req, false)
		log.Errorln(err)
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
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
	taskData.Set("create", jsonutils.NewBool(jsonutils.QueryBoolean(data, "create", false)))
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

func (self *SGuest) PerformStartBackup(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if !self.HasBackupGuest() {
		return nil, httperrors.NewBadRequestError("guest has no backup guest")
	}
	if host := HostManager.FetchHostById(self.BackupHostId); host.HostStatus != api.HOST_ONLINE {
		return nil, httperrors.NewBadRequestError("can't start backup guest on host status %s", host.HostStatus)
	}
	if self.Status != api.VM_RUNNING || self.BackupGuestStatus == api.VM_RUNNING {
		return nil, httperrors.NewBadRequestError("can't start backup guest on backup guest status %s", self.BackupGuestStatus)
	}
	return nil, self.GuestStartAndSyncToBackup(ctx, userCred, "", self.Status)
}

func (self *SGuest) PerformSetExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerSetExtraOptionInput) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "input.Validate")
	}
	extraOptions := self.GetExtraOptions(ctx, userCred)
	optVal := make([]string, 0)
	extraOptions.Unmarshal(&optVal, input.Key)
	if !utils.IsInStringArray(input.Value, optVal) {
		optVal = append(optVal, input.Value)
	}
	extraOptions.Set(input.Key, jsonutils.Marshal(optVal))
	return nil, self.SetExtraOptions(ctx, userCred, extraOptions)
}

func (self *SGuest) GetExtraOptions(ctx context.Context, userCred mcclient.TokenCredential) *jsonutils.JSONDict {
	options := self.GetMetadataJson(ctx, "extra_options", userCred)
	o, ok := options.(*jsonutils.JSONDict)
	if ok {
		return o
	}
	return jsonutils.NewDict()
}

func (self *SGuest) SetExtraOptions(ctx context.Context, userCred mcclient.TokenCredential, extraOptions *jsonutils.JSONDict) error {
	return self.SetMetadata(ctx, "extra_options", extraOptions, userCred)
}

func (self *SGuest) PerformDelExtraOption(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerDelExtraOptionInput) (jsonutils.JSONObject, error) {
	err := input.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "input.Validate")
	}
	extraOptions := self.GetExtraOptions(ctx, userCred)
	var newOpt []string
	if len(input.Value) > 0 {
		optVal := make([]string, 0)
		extraOptions.Unmarshal(&optVal, input.Key)
		for _, v := range optVal {
			if v != input.Value {
				newOpt = append(newOpt, v)
			}
		}
	}
	if len(newOpt) > 0 {
		extraOptions.Set(input.Key, jsonutils.Marshal(newOpt))
	} else if extraOptions.Contains(input.Key) {
		extraOptions.Remove(input.Key)
	}
	return nil, self.SetExtraOptions(ctx, userCred, extraOptions)
}

func (self *SGuest) PerformCancelExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("guest billing type %s not support cancel expire", self.BillingType)
	}
	if err := self.GetDriver().CancelExpireTime(ctx, userCred, self); err != nil {
		return nil, err
	}
	disks, err := self.GetDisks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(disks); i += 1 {
		if err := disks[i].CancelExpireTime(ctx, userCred); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (self *SGuest) PerformPostpaidExpire(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PostpaidExpireInput) (jsonutils.JSONObject, error) {
	if self.BillingType != billing_api.BILLING_TYPE_POSTPAID {
		return nil, httperrors.NewBadRequestError("guest billing type is %s", self.BillingType)
	}

	if !self.GetDriver().IsSupportPostpaidExpire() {
		return nil, httperrors.NewBadRequestError("guest %s unsupport postpaid expire", self.Hypervisor)
	}

	bc, err := ParseBillingCycleInput(&self.SBillingResourceBase, input)
	if err != nil {
		return nil, err
	}

	err = self.SaveRenewInfo(ctx, userCred, bc, nil, billing_api.BILLING_TYPE_POSTPAID)
	if err != nil {
		return nil, err
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_SET_EXPIRED_TIME, input, userCred, true)
	return nil, nil
}

func (self *SGuest) PerformRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	durationStr, _ := data.GetString("duration")
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

func (self *SGuest) GetStorages() []*SStorage {
	disks, _ := self.GetDisks()
	storageMap := make(map[string]*SStorage)
	for i := range disks {
		storage, _ := disks[i].GetStorage()
		if _, ok := storageMap[storage.GetId()]; !ok {
			storageMap[storage.GetId()] = storage
		}
	}
	ret := make([]*SStorage, 0, len(storageMap))
	for _, s := range storageMap {
		ret = append(ret, s)
	}
	return ret
}

func (self *SGuest) SyncCapacityUsedForStorage(ctx context.Context, storageIds []string) error {
	if self.Hypervisor != api.HYPERVISOR_ESXI {
		return nil
	}
	var storages []*SStorage
	if len(storageIds) == 0 {
		storages = self.GetStorages()
	} else {
		q := StorageManager.Query()
		if len(storageIds) == 1 {
			q = q.Equals("id", storageIds[0])
		} else {
			q = q.In("id", storageIds[0])
		}
		ss := make([]SStorage, 0, len(storageIds))
		err := db.FetchModelObjects(StorageManager, q, &ss)
		if err != nil {
			return errors.Wrap(err, "FetchModelObjects")
		}
		storages = make([]*SStorage, len(ss))
		for i := range ss {
			storages[i] = &ss[i]
		}
	}
	for _, s := range storages {
		err := s.SyncCapacityUsed(ctx)
		return errors.Wrapf(err, "unable to SyncCapacityUsed for storage %q", s.GetId())
	}
	return nil
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
	disks, err := self.GetDisks()
	if err != nil {
		return err
	}
	for i := 0; i < len(disks); i += 1 {
		if disks[i].AutoDelete {
			err = disks[i].SaveRenewInfo(ctx, userCred, bc, expireAt, billingType)
			if err != nil {
				return err
			}
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
		} else if bc != nil {
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

func (self *SGuest) PerformStreamDisksComplete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	disks, err := self.GetDisks()
	if err != nil {
		return nil, err
	}
	for _, disk := range disks {
		if len(disk.SnapshotId) > 0 && disk.GetMetadata(ctx, "merge_snapshot", userCred) == "true" {
			SnapshotManager.AddRefCount(disk.SnapshotId, -1)
			disk.SetMetadata(ctx, "merge_snapshot", jsonutils.JSONFalse, userCred)
		}
		if len(disk.GetMetadata(ctx, api.DISK_META_ESXI_FLAT_FILE_PATH, nil)) > 0 {
			disk.SetMetadata(ctx, api.DISK_META_ESXI_FLAT_FILE_PATH, "", userCred)
		}
	}
	return nil, nil
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
	if err := db.NewNameValidator(man, userCred, desc.Name, nil); err != nil {
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
		return gst, err
	}
	// 3. import disks
	if err := gst.importDisks(ctx, userCred, desc.Disks); err != nil {
		return gst, err
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
	err = man.TableSpec().Insert(ctx, gst)
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
		host, _ := self.GetHost()
		_, err = self.attach2NetworkDesc(ctx, userCred, host, ToNetConfig(&nic, net), nil, nil)
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
	host, _ := self.GetHost()
	for _, disk := range disks {
		disk, err := self.createDiskOnHost(ctx, userCred, host, ToDiskConfig(&disk), nil, true, true, nil, nil, true)
		if err != nil {
			return err
		}
		disk.SetStatus(userCred, api.DISK_READY, "")
	}
	return nil
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
	sHost, err := HostManager.GetHostByIp("", api.HOST_TYPE_HYPERVISOR, host.HostIp)
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

	host, _ := self.GetHost()
	if utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_BLOCK_STREAM}) {
		vncInfo, err := self.GetDriver().GetGuestVncInfo(ctx, userCred, self, host, nil)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		vdiProtocol = vncInfo.Protocol
		vdiListenPort = int64(vncInfo.Port)
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

	host, _ := self.GetHost()

	// disks
	guestDisks, err := self.GetGuestDisks()
	if err != nil {
		return "", errors.Wrapf(err, "GetGuestDisks")
	}
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
	isolatedDevices, _ := self.GetIsolatedDevices()
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

func (self *SGuest) PerformConvert(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *api.ConvertEsxiToKvmInput,
) (jsonutils.JSONObject, error) {
	switch data.TargetHypervisor {
	case api.HYPERVISOR_KVM:
		return self.PerformConvertToKvm(ctx, userCred, query, data)
	default:
		return nil, httperrors.NewBadRequestError("not support hypervisor %s", data.TargetHypervisor)
	}
}

func (self *SGuest) PerformConvertToKvm(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *api.ConvertEsxiToKvmInput,
) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_ESXI {
		return nil, httperrors.NewBadRequestError("not support %s", self.Hypervisor)
	}
	if len(self.GetMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, userCred)) > 0 {
		return nil, httperrors.NewBadRequestError("guest has been converted")
	}
	preferHost := data.PreferHost
	if len(preferHost) > 0 {
		iHost, err := HostManager.FetchByIdOrName(userCred, preferHost)
		if err != nil {
			return nil, err
		}
		host := iHost.(*SHost)
		if host.HostType != api.HOST_TYPE_HYPERVISOR {
			return nil, httperrors.NewBadRequestError("host %s is not kvm host", preferHost)
		}
		preferHost = host.GetId()
	}

	if self.Status != api.VM_READY {
		return nil, httperrors.NewBadRequestError("guest status must be ready")
	}

	nets, err := self.GetNetworks("")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	if len(nets) == 0 {
		syncIps := self.GetMetadata(ctx, "sync_ips", userCred)
		if len(syncIps) > 0 {
			return nil, errors.Wrap(httperrors.ErrInvalidStatus, "VMware network not configured properly")
		}
	}

	newGuest, createInput, err := self.createConvertedServer(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "create converted server")
	}
	return nil, self.StartConvertEsxiToKvmTask(ctx, userCred, preferHost, newGuest, createInput)
}

func (self *SGuest) StartConvertEsxiToKvmTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	preferHostId string, newGuest *SGuest, createInput *api.ServerCreateInput,
) error {
	params := jsonutils.NewDict()
	if len(preferHostId) > 0 {
		params.Set("prefer_host_id", jsonutils.NewString(preferHostId))
	}
	params.Set("target_guest_id", jsonutils.NewString(newGuest.Id))
	params.Set("input", jsonutils.Marshal(createInput))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestConvertEsxiToKvmTask", self, userCred,
		params, "", "", nil)
	if err != nil {
		return err
	} else {
		self.SetStatus(userCred, api.VM_CONVERTING, "esxi guest convert to kvm")
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SGuest) createConvertedServer(
	ctx context.Context, userCred mcclient.TokenCredential,
) (*SGuest, *api.ServerCreateInput, error) {
	// set guest pending usage
	pendingUsage, pendingRegionUsage, err := self.getGuestUsage(1)
	keys, err := self.GetQuotaKeys()
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetQuotaKeys")
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage)
	if err != nil {
		return nil, nil, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}
	regionKeys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, nil, errors.Wrap(err, "GetRegionalQuotaKeys")
	}
	pendingRegionUsage.SetKeys(regionKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingRegionUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, nil, errors.Wrap(err, "CheckSetPendingQuota")
	}
	// generate guest create params
	createInput := self.ToCreateInput(ctx, userCred)
	createInput.Hypervisor = api.HYPERVISOR_KVM
	createInput.PreferHost = ""
	createInput.Vdi = api.VM_VDI_PROTOCOL_VNC
	createInput.GenerateName = fmt.Sprintf("%s-%s", self.Name, api.HYPERVISOR_KVM)
	// change drivers so as to bootable in KVM
	for i := range createInput.Disks {
		if createInput.Disks[i].Driver != "ide" {
			createInput.Disks[i].Driver = "ide"
		}
		createInput.Disks[i].Format = ""
		createInput.Disks[i].Backend = ""
		createInput.Disks[i].Medium = ""
	}
	for i := range createInput.Networks {
		if createInput.Networks[i].Driver != "e1000" && createInput.Networks[i].Driver != "vmxnet3" {
			createInput.Networks[i].Driver = "e1000"
		}
	}

	lockman.LockClass(ctx, GuestManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, GuestManager, userCred.GetProjectId())
	newGuest, err := db.DoCreate(GuestManager, ctx, userCred, nil,
		jsonutils.Marshal(createInput), self.GetOwnerId())
	quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, true)
	if err != nil {
		return nil, nil, errors.Wrap(err, "db.DoCreate")
	}
	return newGuest.(*SGuest), createInput, nil
}

func (self *SGuest) PerformSyncFixNics(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestSyncFixNicsInput) (jsonutils.JSONObject, error) {
	iVM, err := self.GetIVM(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	vnics, err := iVM.GetINics()
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	host, _ := self.GetHost()
	if host == nil {
		return nil, httperrors.NewInternalServerError("host not found???")
	}
	iplist := input.Ip
	// validate iplist
	if len(iplist) == 0 {
		return nil, httperrors.NewInputParameterError("empty ip list")
	}
	for _, ip := range iplist {
		// ip is reachable on host
		net, err := host.getNetworkOfIPOnHost(ip)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Unreachable IP %s: %s", ip, err)
		}
		// check ip is reserved or free
		rip := ReservedipManager.GetReservedIP(net, ip)
		if rip == nil {
			// check ip is free
			nip, err := net.GetFreeIPWithLock(ctx, userCred, nil, nil, ip, "", false)
			if err != nil {
				return nil, httperrors.NewInputParameterError("Unavailable IP %s: occupied", ip)
			}
			if nip != ip {
				return nil, httperrors.NewInputParameterError("Unavailable IP %s: occupied", ip)
			}
		}
	}
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
		return nil, httperrors.NewInvalidStatusError("%v", errors.NewAggregate(errs))
	}
	result := self.SyncVMNics(ctx, userCred, host, vnics, iplist)
	if result.IsError() {
		return nil, httperrors.NewInternalServerError("%s", result.Result())
	}
	return nil, nil
}

func (guest *SGuest) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	disks, err := guest.GetDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	for i := range disks {
		disk := disks[i]
		_, err := disk.PerformChangeOwner(ctx, userCred, query, input)
		if err != nil {
			return nil, err
		}
	}

	if eip, _ := guest.GetEipOrPublicIp(); eip != nil {
		_, err := eip.PerformChangeOwner(ctx, userCred, query, input)
		if err != nil {
			return nil, err
		}
	}
	// change owner for instance snapshot
	isps, err := guest.GetInstanceSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "unable to GetInstanceSnapshots")
	}
	for i := range isps {
		_, err := isps[i].PerformChangeOwner(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to change owner for instance snapshot %s", isps[i].GetId())
		}
	}
	changOwner, err := guest.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	err = guest.StartSyncTask(ctx, userCred, false, "")
	if err != nil {
		return nil, errors.Wrap(err, "PerformChangeOwner StartSyncTask err")
	}
	return changOwner, nil
}

func (guest *SGuest) PerformResizeDisk(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerResizeDiskInput) (jsonutils.JSONObject, error) {
	if guest.Hypervisor == api.HYPERVISOR_ESXI {
		c, err := guest.GetInstanceSnapshotCount()
		if err != nil {
			return nil, errors.Wrapf(err, "unable to GetInstanceSnapshotCount for guest %s", guest.GetName())
		}
		if c > 0 {
			return nil, httperrors.NewUnsupportOperationError("the disk of a esxi virtual machine with instance snapshots does not support resizing")
		}
	}
	if len(input.DiskId) == 0 {
		return nil, httperrors.NewMissingParameterError("disk_id")
	}
	diskObj, err := validators.ValidateModel(userCred, DiskManager, &input.DiskId)
	if err != nil {
		return nil, err
	}
	guestdisk := guest.GetGuestDisk(input.DiskId)
	if guestdisk == nil {
		return nil, httperrors.NewInvalidStatusError("disk %s not attached to server", input.DiskId)
	}
	disk := diskObj.(*SDisk)
	sizeMb, err := input.SizeMb()
	if err != nil {
		return nil, err
	}
	err = disk.doResize(ctx, userCred, sizeMb, guest)
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

func (self *SGuest) PerformIoThrottle(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ServerSetDiskIoThrottleInput) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_KVM {
		return nil, httperrors.NewBadRequestError("Hypervisor %s can't do io throttle", self.Hypervisor)
	}
	if self.Status != api.VM_RUNNING && self.Status != api.VM_READY {
		return nil, httperrors.NewServerStatusError("Cannot do io throttle in status %s", self.Status)
	}

	for diskId, bpsMb := range input.Bps {
		if bpsMb < 0 {
			return nil, httperrors.NewInputParameterError("disk %s bps must > 0", diskId)
		}
		disk := DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, httperrors.NewNotFoundError("disk %s not found", diskId)
		}
	}

	for diskId, iops := range input.IOPS {
		if iops < 0 {
			return nil, httperrors.NewInputParameterError("disk %s iops must > 0", diskId)
		}
		disk := DiskManager.FetchDiskById(diskId)
		if disk == nil {
			return nil, httperrors.NewNotFoundError("disk %s not found", diskId)
		}
	}

	if err := self.UpdateIoThrottle(input); err != nil {
		return nil, errors.Wrap(err, "update io throttles")
	}
	return nil, self.StartBlockIoThrottleTask(ctx, userCred, input)
}

func (self *SGuest) UpdateIoThrottle(input *api.ServerSetDiskIoThrottleInput) error {
	gds, err := self.GetGuestDisks()
	if err != nil {
		return err
	}
	for i := 0; i < len(gds); i++ {
		_, err := db.Update(&gds[i], func() error {
			if bps, ok := input.Bps[gds[i].DiskId]; ok {
				gds[i].Bps = bps
			}
			if iops, ok := input.IOPS[gds[i].DiskId]; ok {
				gds[i].Iops = iops
			}
			return nil
		})
		if err != nil {
			return nil
		}
	}
	return nil
}

func (self *SGuest) StartBlockIoThrottleTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerSetDiskIoThrottleInput) error {
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
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

func (self *SGuest) validateForBatchMigrate(ctx context.Context, rescueMode bool) (*SGuest, error) {
	guest := GuestManager.FetchGuestById(self.Id)
	if guest.Hypervisor != api.HYPERVISOR_KVM {
		return guest, httperrors.NewBadRequestError("guest %s hypervisor %s can't migrate",
			guest.Name, guest.Hypervisor)
	}
	if len(guest.BackupHostId) > 0 {
		return guest, httperrors.NewBadRequestError("guest %s has backup, can't migrate", guest.Name)
	}
	devs, _ := guest.GetIsolatedDevices()
	if len(devs) > 0 {
		return guest, httperrors.NewBadRequestError("guest %s has isolated device, can't migrate", guest.Name)
	}
	if rescueMode {
		if !guest.guestDisksStorageTypeIsShared() {
			return guest, httperrors.NewBadRequestError("can't rescue geust %s with local storage", guest.Name)
		}
		return guest, nil
	}
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY, api.VM_UNKNOWN}) {
		return guest, httperrors.NewBadRequestError("guest %s status %s can't migrate", guest.Name, guest.Status)
	}
	if guest.Status == api.VM_RUNNING {
		cdroms, _ := guest.getCdroms()
		for _, cdrom := range cdroms {
			if len(cdrom.ImageId) > 0 {
				return guest, httperrors.NewBadRequestError("cannot migrate with cdrom")
			}
		}
		floppys, _ := guest.getFloppys()
		for _, floppy := range floppys {
			if len(floppy.ImageId) > 0 {
				return guest, httperrors.NewBadRequestError("cannot migrate with floppy")
			}
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
	if len(params.PreferHostId) > 0 {
		iHost, _ := HostManager.FetchByIdOrName(userCred, params.PreferHostId)
		if iHost == nil {
			return nil, httperrors.NewBadRequestError("Host %s not found", params.PreferHostId)
		}
		host := iHost.(*SHost)
		preferHostId = host.Id

		err := host.IsAssignable(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "IsAssignable")
		}
	}

	if params.EnableTLS == nil {
		params.EnableTLS = &options.Options.EnableTlsMigration
	}

	guests := make([]SGuest, 0)
	q := GuestManager.Query().In("id", params.GuestIds)
	err = db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		return nil, httperrors.NewInternalServerError("%v", err)
	}
	if len(guests) != len(params.GuestIds) {
		return nil, httperrors.NewBadRequestError("Check input guests is exist")
	}
	errs := []error{}
	for i := 0; i < len(guests); i++ {
		func() {
			lockman.LockObject(ctx, &guests[i])
			defer lockman.ReleaseObject(ctx, &guests[i])
			_, err = guests[i].validateForBatchMigrate(ctx, false)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Guest %s", guests[i].Name))
				return
			}
			if guests[i].Status == api.VM_RUNNING {
				err = guests[i].StartGuestLiveMigrateTask(ctx, userCred,
					guests[i].Status, preferHostId, &params.SkipCpuCheck, &params.SkipKernelCheck,
					params.EnableTLS, params.QuciklyFinish, params.MaxBandwidthMb, nil, "",
				)
			} else {
				err = guests[i].StartMigrateTask(ctx, userCred, guests[i].Status == api.VM_UNKNOWN,
					false, guests[i].Status, preferHostId, "")
			}
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Guest %s", guests[i].Name))
			}
		}()

	}
	return nil, errors.NewAggregate(errs)
}

func (manager *SGuestManager) StartHostGuestsMigrateTask(
	ctx context.Context, userCred mcclient.TokenCredential,
	guests []*SGuest, kwargs *jsonutils.JSONDict, parentTaskId string,
) error {
	taskItems := make([]db.IStandaloneModel, len(guests))
	for i := range guests {
		taskItems[i] = guests[i]
	}
	task, err := taskman.TaskManager.NewParallelTask(ctx, "HostGuestsMigrateTask", taskItems, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		log.Errorln(err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

var supportInstanceSnapshotHypervisors = []string{
	api.HYPERVISOR_KVM,
	api.HYPERVISOR_ESXI,
}

func (self *SGuest) validateCreateInstanceSnapshot(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ServerCreateSnapshotParams,
) (*SRegionQuota, api.ServerCreateSnapshotParams, error) {

	if !utils.IsInStringArray(self.Hypervisor, supportInstanceSnapshotHypervisors) {
		return nil, input, httperrors.NewBadRequestError("guest hypervisor %s can't create instance snapshot", self.Hypervisor)
	}

	if len(self.BackupHostId) > 0 {
		return nil, input, httperrors.NewBadRequestError("Can't do instance snapshot with backup guest")
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return nil, input, httperrors.NewInvalidStatusError("guest can't do snapshot in status %s", self.Status)
	}

	ownerId := self.GetOwnerId()
	// dataDict := data.(*jsonutils.JSONDict)
	// nameHint, err := dataDict.GetString("generate_name")
	if len(input.GenerateName) > 0 {
		name, err := db.GenerateName(ctx, InstanceSnapshotManager, ownerId, input.GenerateName)
		if err != nil {
			return nil, input, errors.Wrap(err, "GenerateName")
		}
		input.Name = name
	} else if len(input.Name) == 0 {
		return nil, input, httperrors.NewMissingParameterError("name")
	}

	err := db.NewNameValidator(InstanceSnapshotManager, ownerId, input.Name, nil)
	if err != nil {
		return nil, input, errors.Wrap(err, "NewNameValidator")
	}

	// construct Quota
	pendingUsage := &SRegionQuota{InstanceSnapshot: 1}
	host, _ := self.GetHost()
	provider := host.GetProviderName()
	if utils.IsInStringArray(provider, ProviderHasSubSnapshot) {
		disks, err := self.GetDisks()
		if err != nil {
			return nil, input, errors.Wrapf(err, "GetDisks")
		}
		for i := 0; i < len(disks); i++ {
			if storage, _ := disks[i].GetStorage(); utils.IsInStringArray(storage.StorageType, api.FIEL_STORAGE) {
				count, err := SnapshotManager.GetDiskManualSnapshotCount(disks[i].Id)
				if err != nil {
					return nil, input, httperrors.NewInternalServerError("%v", err)
				}
				if count >= options.Options.DefaultMaxManualSnapshotCount {
					return nil, input, httperrors.NewBadRequestError("guests disk %d snapshot full, can't take anymore", i)
				}
			}
		}
		pendingUsage.Snapshot = len(disks)
	}
	keys, err := self.GetRegionalQuotaKeys()
	if err != nil {
		return nil, input, errors.Wrap(err, "GetRegionalQuotaKeys")
	}
	pendingUsage.SetKeys(keys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, pendingUsage)
	if err != nil {
		return nil, input, httperrors.NewOutOfQuotaError("Check set pending quota error %s", err)
	}
	return pendingUsage, input, nil
}

func (self *SGuest) validateCreateInstanceBackup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	if !utils.IsInStringArray(self.Hypervisor, []string{api.HYPERVISOR_KVM}) {
		return httperrors.NewBadRequestError("guest hypervisor %s can't create instance snapshot", self.Hypervisor)
	}

	if len(self.BackupHostId) > 0 {
		return httperrors.NewBadRequestError("Can't do instance snapshot with backup guest")
	}

	if !utils.IsInStringArray(self.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return httperrors.NewInvalidStatusError("guest can't do snapshot in status %s", self.Status)
	}

	var name string
	ownerId := self.GetOwnerId()
	dataDict := data.(*jsonutils.JSONDict)
	nameHint, err := dataDict.GetString("generate_name")
	if err == nil {
		name, err = db.GenerateName(ctx, InstanceBackupManager, ownerId, nameHint)
		if err != nil {
			return err
		}
		dataDict.Set("name", jsonutils.NewString(name))
	} else if name, err = dataDict.GetString("name"); err != nil {
		return httperrors.NewMissingParameterError("name")
	}

	err = db.NewNameValidator(InstanceBackupManager, ownerId, name, nil)
	if err != nil {
		return err
	}
	return nil
}

// 1. validate guest status, guest hypervisor
// 2. validate every disk manual snapshot count
// 3. validate snapshot quota with disk count
func (self *SGuest) PerformInstanceSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerInstanceSnapshot,
) (jsonutils.JSONObject, error) {
	if err := self.ValidateEncryption(ctx, userCred); err != nil {
		return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
	}

	lockman.LockClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	pendingUsage, params, err := self.validateCreateInstanceSnapshot(ctx, userCred, query, input.ServerCreateSnapshotParams)
	if err != nil {
		return nil, errors.Wrap(err, "validateCreateInstanceSnapshot")
	}
	input.ServerCreateSnapshotParams = params
	if input.WithMemory {
		if self.Status != api.VM_RUNNING {
			return nil, httperrors.NewUnsupportOperationError("Can't save memory state when guest status is %q", self.Status)
		}
	}
	instanceSnapshot, err := InstanceSnapshotManager.CreateInstanceSnapshot(ctx, userCred, self, input.Name, false, input.WithMemory)
	if err != nil {
		quotas.CancelPendingUsage(
			ctx, userCred, pendingUsage, pendingUsage, false)
		return nil, httperrors.NewInternalServerError("create instance snapshot failed: %s", err)
	}
	err = self.InheritTo(ctx, instanceSnapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to inherit from guest %s to instance snapshot %s", self.GetId(), instanceSnapshot.GetId())
	}
	err = self.InstaceCreateSnapshot(ctx, userCred, instanceSnapshot, pendingUsage)
	if err != nil {
		quotas.CancelPendingUsage(
			ctx, userCred, pendingUsage, pendingUsage, false)
		return nil, httperrors.NewInternalServerError("start create snapshot task failed: %s", err)
	}
	return nil, nil
}

func (self *SGuest) PerformInstanceBackup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if err := self.ValidateEncryption(ctx, userCred); err != nil {
		return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
	}
	lockman.LockClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, InstanceSnapshotManager, userCred.GetProjectId())
	err := self.validateCreateInstanceBackup(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	name, _ := data.GetString("name")
	backupStorageId, _ := data.GetString("backup_storage_id")
	if backupStorageId == "" {
		return nil, httperrors.NewMissingParameterError("backup_storage_id")
	}
	ibs, err := BackupStorageManager.FetchByIdOrName(userCred, backupStorageId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(BackupStorageManager.Keyword(), backupStorageId)
		}
		if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
			return nil, httperrors.NewDuplicateResourceError(BackupStorageManager.Keyword(), backupStorageId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	bs := ibs.(*SBackupStorage)
	if bs.Status != api.BACKUPSTORAGE_STATUS_ONLINE {
		return nil, httperrors.NewForbiddenError("can't backup guest to backup storage with status %s", bs.Status)
	}
	instanceBackup, err := InstanceBackupManager.CreateInstanceBackup(ctx, userCred, self, name, backupStorageId)
	if err != nil {
		return nil, httperrors.NewInternalServerError("create instance backup failed: %s", err)
	}
	err = self.InheritTo(ctx, instanceBackup)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to inherit from guest %s to instance backup %s", self.GetId(), instanceBackup.GetId())
	}
	err = self.InstanceCreateBackup(ctx, userCred, instanceBackup)
	if err != nil {
		return nil, httperrors.NewInternalServerError("start create backup task failed: %s", err)
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

func (self *SGuest) InstanceCreateBackup(ctx context.Context, userCred mcclient.TokenCredential, instanceBackup *SInstanceBackup) error {
	self.SetStatus(userCred, api.VM_START_INSTANCE_BACKUP, "instance backup")
	return instanceBackup.StartCreateInstanceBackupTask(ctx, userCred, "")
}

func (self *SGuest) PerformInstanceSnapshotReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerResetInput) (jsonutils.JSONObject, error) {
	if err := self.ValidateEncryption(ctx, userCred); err != nil {
		return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
	}

	if self.Status != api.VM_READY {
		return nil, httperrors.NewInvalidStatusError("guest can't do snapshot in status %s", self.Status)
	}

	obj, err := InstanceSnapshotManager.FetchByIdOrName(userCred, input.InstanceSnapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch instance snapshot %q", input.InstanceSnapshot)
	}

	instanceSnapshot := obj.(*SInstanceSnapshot)

	if instanceSnapshot.GuestId != self.GetId() {
		return nil, httperrors.NewBadRequestError("instance snapshot %q not belong server %q", instanceSnapshot.GetName(), self.GetId())
	}

	if instanceSnapshot.Status != api.INSTANCE_SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Instance snapshot not ready")
	}

	if input.WithMemory && !instanceSnapshot.WithMemory {
		return nil, httperrors.NewBadRequestError("Instance snapshot not with memory statefile")
	}

	err = self.StartSnapshotResetTask(ctx, userCred, instanceSnapshot, input.AutoStart, input.WithMemory)
	if err != nil {
		return nil, httperrors.NewInternalServerError("start snapshot reset failed %s", err)
	}

	return nil, nil
}

func (self *SGuest) StartSnapshotResetTask(ctx context.Context, userCred mcclient.TokenCredential, instanceSnapshot *SInstanceSnapshot, autoStart *bool, withMemory bool) error {
	data := jsonutils.NewDict()
	if autoStart != nil && *autoStart {
		data.Set("auto_start", jsonutils.JSONTrue)
	}
	data.Add(jsonutils.NewBool(withMemory), "with_memory")
	self.SetStatus(userCred, api.VM_START_SNAPSHOT_RESET, "start snapshot reset task")
	instanceSnapshot.SetStatus(userCred, api.INSTANCE_SNAPSHOT_RESET, "start snapshot reset task")
	log.Errorf("====data: %s", data)
	if task, err := taskman.TaskManager.NewTask(
		ctx, "InstanceSnapshotResetTask", instanceSnapshot, userCred, data, "", "", nil,
	); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) PerformSnapshotAndClone(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ServerSnapshotAndCloneInput,
) (jsonutils.JSONObject, error) {
	if err := self.ValidateEncryption(ctx, userCred); err != nil {
		return nil, errors.Wrap(httperrors.ErrForbidden, "encryption key not accessible")
	}

	newlyGuestName := input.Name
	if len(input.Name) == 0 {
		return nil, httperrors.NewMissingParameterError("name")
	}
	count := 1
	if input.Count != nil {
		count = *input.Count
		if count <= 0 {
			return nil, httperrors.NewInputParameterError("count must > 0")
		}
	}

	lockman.LockRawObject(ctx, InstanceSnapshotManager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, InstanceSnapshotManager.Keyword(), "name")

	// validate create instance snapshot and set snapshot pending usage
	snapshotUsage, params, err := self.validateCreateInstanceSnapshot(ctx, userCred, query, input.ServerCreateSnapshotParams)
	if err != nil {
		return nil, errors.Wrap(err, "validateCreateInstanceSnapshot")
	}
	input.ServerCreateSnapshotParams = params
	// set guest pending usage
	pendingUsage, pendingRegionUsage, err := self.getGuestUsage(count)
	keys, err := self.GetQuotaKeys()
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		return nil, errors.Wrap(err, "GetQuotaKeys")
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
		return nil, errors.Wrap(err, "GetRegionalQuotaKeys")
	}
	pendingRegionUsage.SetKeys(regionKeys)
	err = quotas.CheckSetPendingQuota(ctx, userCred, &pendingRegionUsage)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, snapshotUsage, snapshotUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		return nil, errors.Wrap(err, "CheckSetPendingQuota")
	}
	// migrate snapshotUsage into regionUsage, then discard snapshotUsage
	pendingRegionUsage.Snapshot = snapshotUsage.Snapshot

	instanceSnapshotName, err := db.GenerateName(ctx, InstanceSnapshotManager, self.GetOwnerId(),
		fmt.Sprintf("%s-%s", input.Name, rand.String(8)))
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, false)
		return nil, httperrors.NewInternalServerError("Generate snapshot name failed %s", err)
	}
	instanceSnapshot, err := InstanceSnapshotManager.CreateInstanceSnapshot(
		ctx, userCred, self, instanceSnapshotName,
		input.AutoDeleteInstanceSnapshot != nil && *input.AutoDeleteInstanceSnapshot, false)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, false)
		return nil, httperrors.NewInternalServerError("create instance snapshot failed: %s", err)
	} else {
		cancelRegionUsage := &SRegionQuota{Snapshot: snapshotUsage.Snapshot}
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, cancelRegionUsage, true)
	}

	err = self.StartInstanceSnapshotAndCloneTask(
		ctx, userCred, newlyGuestName, &pendingUsage, &pendingRegionUsage, instanceSnapshot, input)
	if err != nil {
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &pendingUsage, false)
		quotas.CancelPendingUsage(ctx, userCred, &pendingRegionUsage, &pendingRegionUsage, false)
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) StartInstanceSnapshotAndCloneTask(
	ctx context.Context, userCred mcclient.TokenCredential, newlyGuestName string,
	pendingUsage *SQuota, pendingRegionUsage *SRegionQuota, instanceSnapshot *SInstanceSnapshot,
	input api.ServerSnapshotAndCloneInput,
) error {
	params := jsonutils.NewDict()
	params.Set("guest_params", jsonutils.Marshal(input))
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
	ctx context.Context, userCred mcclient.TokenCredential, input api.ServerSnapshotAndCloneInput, isp *SInstanceSnapshot,
) (*SGuest, *jsonutils.JSONDict, error) {
	lockman.LockRawObject(ctx, manager.Keyword(), "name")
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "name")

	if guestName, err := db.GenerateName(ctx, manager, isp.GetOwnerId(), input.Name); err != nil {
		return nil, nil, errors.Wrap(err, "db.GenerateName")
	} else {
		input.Name = guestName
	}

	input.InstanceSnapshotId = isp.Id

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)

	iGuest, err := db.DoCreate(manager, ctx, userCred, nil, params, isp.GetOwnerId())
	if err != nil {
		return nil, nil, errors.Wrap(err, "db.DoCreate")
	}
	guest := iGuest.(*SGuest)
	notes := map[string]string{
		"instance_snapshot_id": isp.Id,
		"guest_id":             isp.GuestId,
	}

	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_SNAPSHOT_AND_CLONE, notes, userCred, true)
	func() {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		guest.PostCreate(ctx, userCred, guest.GetOwnerId(), nil, params)
	}()

	return guest, params, nil
}

func (self *SGuest) GetDetailsJnlp(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Hypervisor != api.HYPERVISOR_BAREMETAL {
		return nil, httperrors.NewInvalidStatusError("not a baremetal server")
	}
	host, _ := self.GetHost()
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

func (self *SGuest) PerformResetNicTrafficLimit(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input *api.ServerNicTrafficLimit) (jsonutils.JSONObject, error) {

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewUnsupportOperationError("The guest status need be %s or %s, current is %s", api.VM_READY, api.VM_RUNNING, self.Status)
	}
	input.Mac = strings.ToLower(input.Mac)
	_, err := self.GetGuestnetworkByMac(input.Mac)
	if err != nil {
		return nil, errors.Wrap(err, "get guest network by mac")
	}

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	params.Set("old_status", jsonutils.NewString(self.Status))
	self.SetStatus(userCred, api.VM_SYNC_TRAFFIC_LIMIT, "PerformResetNicTrafficLimit")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestResetNicTrafficsTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SGuest) PerformSetNicTrafficLimit(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input *api.ServerNicTrafficLimit) (jsonutils.JSONObject, error) {

	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewUnsupportOperationError("The guest status need be %s or %s, current is %s", api.VM_READY, api.VM_RUNNING, self.Status)
	}
	if input.RxTrafficLimit == nil && input.TxTrafficLimit == nil {
		return nil, httperrors.NewBadRequestError("rx/tx traffic not provider")
	}
	input.Mac = strings.ToLower(input.Mac)
	_, err := self.GetGuestnetworkByMac(input.Mac)
	if err != nil {
		return nil, errors.Wrap(err, "get guest network by mac")
	}
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	params.Set("old_status", jsonutils.NewString(self.Status))
	self.SetStatus(userCred, api.VM_SYNC_TRAFFIC_LIMIT, "GuestSetNicTrafficsTask")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSetNicTrafficsTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
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

// 公网Ip转Eip
// 要求虚拟机有公网IP,并且虚拟机状态为running 或 ready
// 目前仅支持阿里云和腾讯云
func (self *SGuest) PerformPublicipToEip(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.GuestPublicipToEipInput) (jsonutils.JSONObject, error) {
	publicip, err := self.GetPublicIp()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetPublicIp"))
	}
	if publicip == nil {
		return nil, httperrors.NewInputParameterError("The guest %s does not have any public IP", self.Name)
	}
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewUnsupportOperationError("The guest status need be %s or %s, current is %s", api.VM_READY, api.VM_RUNNING, self.Status)
	}
	if !self.GetDriver().IsSupportPublicipToEip() {
		return nil, httperrors.NewUnsupportOperationError("The %s guest not support public ip to eip operation", self.Hypervisor)
	}
	return nil, self.StartPublicipToEipTask(ctx, userCred, input.AutoStart, "")
}

func (self *SGuest) StartPublicipToEipTask(ctx context.Context, userCred mcclient.TokenCredential, autoStart bool, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("auto_start", jsonutils.NewBool(autoStart))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestPublicipToEipTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.VM_START_EIP_CONVERT, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) SetAutoRenew(autoRenew bool) error {
	_, err := db.Update(self, func() error {
		self.AutoRenew = autoRenew
		return nil
	})
	return err
}

// 设置自动续费
// 要求虚拟机状态为running 或 ready
// 要求虚拟机计费类型为包年包月(预付费)
func (self *SGuest) PerformSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.GuestAutoRenewInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return nil, httperrors.NewUnsupportOperationError("The guest status need be %s or %s, current is %s", api.VM_READY, api.VM_RUNNING, self.Status)
	}

	if self.BillingType != billing_api.BILLING_TYPE_PREPAID {
		return nil, httperrors.NewUnsupportOperationError("Only %s guest support this operation", billing_api.BILLING_TYPE_PREPAID)
	}

	if len(input.Duration) == 0 {
		input.Duration = "1M"
	}

	_, err := billing.ParseBillingCycle(input.Duration)
	if err != nil {
		return nil, httperrors.NewInputParameterError("invalid duration %s: %s", input.Duration, err)
	}

	if self.AutoRenew == input.AutoRenew {
		return nil, nil
	}

	if !self.GetDriver().IsSupportSetAutoRenew() {
		err := self.SetAutoRenew(input.AutoRenew)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}

		logclient.AddSimpleActionLog(self, logclient.ACT_SET_AUTO_RENEW, jsonutils.Marshal(input), userCred, true)
		return nil, nil
	}

	return nil, self.StartSetAutoRenewTask(ctx, userCred, input, "")
}

func (self *SGuest) StartSetAutoRenewTask(ctx context.Context, userCred mcclient.TokenCredential, input api.GuestAutoRenewInput, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Set("auto_renew", jsonutils.NewBool(input.AutoRenew))
	data.Set("duration", jsonutils.NewString(input.Duration))
	task, err := taskman.TaskManager.NewTask(ctx, "GuestSetAutoRenewTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.VM_SET_AUTO_RENEW, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) PerformRemoteUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ServerRemoteUpdateInput) (jsonutils.JSONObject, error) {
	err := self.StartRemoteUpdateTask(ctx, userCred, (input.ReplaceTags != nil && *input.ReplaceTags), "")
	if err != nil {
		return nil, errors.Wrap(err, "StartRemoteUpdateTask")
	}
	return nil, nil
}

func (self *SGuest) PerformOpenForward(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	req, err := guestdriver_types.NewOpenForwardRequestFromJSON(data)
	if err != nil {
		return nil, err
	}
	for _, nicDesc := range self.fetchNICShortDesc(ctx) {
		if nicDesc.VpcId != api.DEFAULT_VPC_ID {
			if req.Addr == "" {
				req.Addr = nicDesc.IpAddr
			}
			req.NetworkId = nicDesc.NetworkId
		}
	}
	if req.NetworkId == "" {
		return nil, httperrors.NewInputParameterError("guest has no vpc ip")
	}

	resp, err := self.GetDriver().RequestOpenForward(ctx, userCred, self, req)
	if err != nil {
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_WEBSSH, err, userCred, false)
		return nil, httperrors.NewGeneralError(err)
	}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_WEBSSH, resp, userCred, true)
	return resp.JSON(), nil
}

func (self *SGuest) PerformCloseForward(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	req, err := guestdriver_types.NewCloseForwardRequestFromJSON(data)
	if err != nil {
		return nil, err
	}
	for _, nicDesc := range self.fetchNICShortDesc(ctx) {
		if nicDesc.VpcId != api.DEFAULT_VPC_ID {
			req.NetworkId = nicDesc.NetworkId
		}
	}
	if req.NetworkId == "" {
		return nil, httperrors.NewInputParameterError("guest has no vpc ip")
	}

	resp, err := self.GetDriver().RequestCloseForward(ctx, userCred, self, req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return resp.JSON(), nil
}

func (self *SGuest) PerformListForward(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	req, err := guestdriver_types.NewListForwardRequestFromJSON(data)
	if err != nil {
		return nil, err
	}
	for _, nicDesc := range self.fetchNICShortDesc(ctx) {
		if nicDesc.VpcId != api.DEFAULT_VPC_ID {
			if req.Addr == "" {
				req.Addr = nicDesc.IpAddr
			}
			req.NetworkId = nicDesc.NetworkId
		}
	}
	if req.NetworkId == "" {
		return nil, httperrors.NewInputParameterError("guest has no vpc ip")
	}

	resp, err := self.GetDriver().RequestListForward(ctx, userCred, self, req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return resp.JSON(), nil
}

func (self *SGuest) PerformChangeStorage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ServerChangeStorageInput) (*api.ServerChangeStorageInput, error) {
	// validate input
	if input.TargetStorageId == "" {
		return nil, httperrors.NewNotEmptyError("Storage id is empty")
	}

	// validate storage
	storageObj, err := StorageManager.FetchByIdOrName(userCred, input.TargetStorageId)
	if err != nil {
		return nil, errors.Wrapf(err, "Found storage by %s", input.TargetStorageId)
	}
	storage := storageObj.(*SStorage)
	input.TargetStorageId = storage.GetId()

	// validate disk
	disks, err := self.GetDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %s disks", self.GetName())
	}
	var changeDisks = []string{}
	for _, disk := range disks {
		if disk.StorageId != input.TargetStorageId {
			changeDisks = append(changeDisks, disk.Id)
		}
	}

	// driver validate
	drv := self.GetDriver()
	if err := drv.ValidateChangeDiskStorage(ctx, userCred, self, input.TargetStorageId); err != nil {
		return nil, err
	}
	return nil, self.StartGuestChangeStorageTask(ctx, userCred, input, changeDisks)
}

func (self *SGuest) StartGuestChangeStorageTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerChangeStorageInput, disks []string) error {
	params := api.ServerChangeStorageInternalInput{
		ServerChangeStorageInput: *input,
		Disks:                    disks,
		DiskCount:                len(disks),
		GuestRunning:             self.Status == api.VM_RUNNING,
	}
	reason := fmt.Sprintf("Change guest disks storage to %s", input.TargetStorageId)
	self.SetStatus(userCred, api.VM_DISK_CHANGE_STORAGE, reason)
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestChangeDisksStorageTask", self, userCred, jsonutils.Marshal(params).(*jsonutils.JSONDict),
		"", "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SGuest) PerformChangeDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ServerChangeDiskStorageInput) (*api.ServerChangeDiskStorageInput, error) {
	// validate input
	if input.DiskId == "" {
		return nil, httperrors.NewNotEmptyError("Disk id is empty")
	}
	if input.TargetStorageId == "" {
		return nil, httperrors.NewNotEmptyError("Storage id is empty")
	}

	// validate disk
	disks, err := self.GetDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %s disks", self.GetName())
	}
	var srcDisk *SDisk
	for _, disk := range disks {
		if input.DiskId == disk.GetId() || input.DiskId == disk.GetName() {
			srcDisk = &disk
			input.DiskId = disk.GetId()
			break
		}
	}
	if srcDisk == nil {
		return nil, httperrors.NewNotFoundError("Disk %s not found on server %s", input.DiskId, self.GetName())
	}

	// validate storage
	storageObj, err := StorageManager.FetchByIdOrName(userCred, input.TargetStorageId)
	if err != nil {
		return nil, errors.Wrapf(err, "Found storage by %s", input.TargetStorageId)
	}
	storage := storageObj.(*SStorage)
	input.TargetStorageId = storage.GetId()

	// driver validate
	drv := self.GetDriver()
	if err := drv.ValidateChangeDiskStorage(ctx, userCred, self, input.TargetStorageId); err != nil {
		return nil, err
	}

	// create a disk on target storage from source disk
	diskConf := &api.DiskConfig{
		Index:    -1,
		ImageId:  srcDisk.TemplateId,
		SizeMb:   srcDisk.DiskSize,
		Fs:       srcDisk.FsFormat,
		DiskType: srcDisk.DiskType,
	}

	targetDisk, err := self.CreateDiskOnStorage(ctx, userCred, storage, diskConf, nil, true, true)
	if err != nil {
		return nil, errors.Wrapf(err, "Create target disk on storage %s", storage.GetName())
	}

	internalInput := &api.ServerChangeDiskStorageInternalInput{
		ServerChangeDiskStorageInput: *input,
		StorageId:                    srcDisk.StorageId,
		TargetDiskId:                 targetDisk.GetId(),
		GuestRunning:                 self.Status == api.VM_RUNNING,
	}

	return nil, self.StartChangeDiskStorageTask(ctx, userCred, internalInput, "")
}

func (self *SGuest) StartChangeDiskStorageTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerChangeDiskStorageInternalInput, parentTaskId string) error {
	reason := fmt.Sprintf("Change disk %s to storage %s", input.DiskId, input.TargetStorageId)
	self.SetStatus(userCred, api.VM_DISK_CHANGE_STORAGE, reason)
	return self.GetDriver().StartChangeDiskStorageTask(self, ctx, userCred, input, parentTaskId)
}

func (self *SGuest) startSwitchToClonedDisk(ctx context.Context, userCred mcclient.TokenCredential, taskId string) error {
	task := taskman.TaskManager.FetchTaskById(taskId)
	if task == nil {
		return errors.Errorf("no task %s found", taskId)
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		log.Infof("guest %s start switch to cloned disk task", self.Id)
		params := jsonutils.NewDict()
		params.Set("block_jobs_ready", jsonutils.JSONTrue)
		return params, nil
	})
	return nil
}

func (self *SGuest) PerformSetBootIndex(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ServerSetBootIndexInput) (jsonutils.JSONObject, error) {
	gds, err := self.GetGuestDisks()
	if err != nil {
		return nil, err
	}
	diskBootIndexes := map[int8]int8{}
	for i := 0; i < len(gds); i++ {
		diskBootIndexes[gds[i].Index] = gds[i].BootIndex
	}

	gcs, err := self.getCdroms()
	if err != nil {
		return nil, err
	}
	cdromBootIndexes := map[int]int8{}
	for i := 0; i < len(gcs); i++ {
		cdromBootIndexes[gcs[i].Ordinal] = gcs[i].BootIndex
	}

	for sDiskIndex, bootIndex := range input.Disks {
		iDiskIndex, err := strconv.Atoi(sDiskIndex)
		if err != nil {
			return nil, httperrors.NewInputParameterError("failed parse disk index %s", sDiskIndex)
		} else if iDiskIndex > 127 {
			return nil, httperrors.NewInputParameterError("disk inex %s is exceed 127", sDiskIndex)
		}
		diskIndex := int8(iDiskIndex)
		if _, ok := diskBootIndexes[diskIndex]; !ok {
			return nil, httperrors.NewBadRequestError("disk has no index %d", diskIndex)
		}
		diskBootIndexes[diskIndex] = bootIndex
	}
	for sCdromOrdinal, bootIndex := range input.Cdroms {
		cdromOrdinal, err := strconv.Atoi(sCdromOrdinal)
		if err != nil {
			return nil, httperrors.NewInputParameterError("failed parse cdrom ordinal %s", sCdromOrdinal)
		}
		if _, ok := cdromBootIndexes[cdromOrdinal]; !ok {
			return nil, httperrors.NewBadRequestError("cdrom has no ordinal %d", cdromOrdinal)
		}
		cdromBootIndexes[cdromOrdinal] = bootIndex
	}
	bm := bitmap.NewBitMap(128)
	for diskIndex, bootIndex := range diskBootIndexes {
		if bootIndex < 0 {
			continue
		}
		if bm.Has(int64(bootIndex)) {
			return nil, httperrors.NewBadRequestError("disk index %d boot index %d is duplicated", diskIndex, bootIndex)
		} else {
			bm.Set(int64(bootIndex))
		}
	}
	for cdromOrdinal, bootIndex := range cdromBootIndexes {
		if bootIndex < 0 {
			continue
		}
		if bm.Has(int64(bootIndex)) {
			return nil, httperrors.NewBadRequestError("cdrom ordianl %d boot index %d is duplicated", cdromOrdinal, bootIndex)
		} else {
			bm.Set(int64(bootIndex))
		}
	}

	for i := 0; i < len(gds); i++ {
		if gds[i].BootIndex != diskBootIndexes[gds[i].Index] {
			if err := gds[i].SetBootIndex(diskBootIndexes[gds[i].Index]); err != nil {
				log.Errorf("gds[i].SetBootIndex: %s", err)
				return nil, err
			}
		}
	}

	for i := 0; i < len(gcs); i++ {
		if gcs[i].BootIndex != cdromBootIndexes[gcs[i].Ordinal] {
			if err := gcs[i].SetBootIndex(cdromBootIndexes[gcs[i].Ordinal]); err != nil {
				log.Errorf("gcs[i].SetBootIndex: %s", err)
				return nil, err
			}
		}
	}

	return nil, nil
}

func (self *SGuest) PerformProbeIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, errors.Wrap(err, "GetHost")
	}
	hostDevs, err := host.GetHostDriver().RequestProbeIsolatedDevices(ctx, userCred, host, data)
	if err != nil {
		return nil, errors.Wrap(err, "RequestProbeIsolatedDevices")
	}
	objs, err := hostDevs.GetArray()
	if err != nil {
		return nil, errors.Wrapf(err, "GetArray from %q", hostDevs)
	}
	devs := make([]*SIsolatedDevice, 0)
	for _, obj := range objs {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "device %s", obj)
		}
		devObj, err := IsolatedDeviceManager.FetchById(id)
		if err != nil {
			return nil, errors.Wrapf(err, "FetchById %q", id)
		}
		dev := devObj.(*SIsolatedDevice)
		if dev.GuestId == "" {
			devs = append(devs, dev)
		}
	}
	return jsonutils.Marshal(devs), nil
}

func (self *SGuest) PerformCpuset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.ServerCPUSetInput) (jsonutils.JSONObject, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, errors.Wrap(err, "get host model")
	}
	allCores, err := host.getHostLogicalCores()
	if err != nil {
		return nil, err
	}

	if !sets.NewInt(allCores...).HasAll(data.CPUS...) {
		return nil, httperrors.NewInputParameterError("Host cores %v not contains input %v", allCores, data.CPUS)
	}

	pinnedMap, err := host.GetPinnedCpusetCores(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "Get host pinned cpu cores")
	}

	pinnedSets := sets.NewInt()
	for key, pinned := range pinnedMap {
		if key == self.GetId() {
			continue
		}
		pinnedSets.Insert(pinned...)
	}

	if pinnedSets.HasAny(data.CPUS...) {
		return nil, httperrors.NewInputParameterError("More than one of input cores %v already set in host %v", data.CPUS, pinnedSets.List())
	}

	if err := self.SetMetadata(ctx, api.VM_METADATA_CGROUP_CPUSET, data, userCred); err != nil {
		return nil, errors.Wrap(err, "set metadata")
	}

	return nil, self.StartGuestCPUSetTask(ctx, userCred, data)
}

func (self *SGuest) StartGuestCPUSetTask(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCPUSetInput) error {
	task, err := taskman.TaskManager.NewTask(ctx, "GuestCPUSetTask", self, userCred, jsonutils.Marshal(input).(*jsonutils.JSONDict), "", "")
	if err != nil {
		return errors.Wrap(err, "New GuestCPUSetTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SGuest) PerformCpusetRemove(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.ServerCPUSetRemoveInput) (*api.ServerCPUSetRemoveResp, error) {
	if err := self.RemoveMetadata(ctx, api.VM_METADATA_CGROUP_CPUSET, userCred); err != nil {
		return nil, errors.Wrapf(err, "remove metadata %q", api.VM_METADATA_CGROUP_CPUSET)
	}
	host, err := self.GetHost()
	if err != nil {
		return nil, errors.Wrap(err, "get host model")
	}

	// TODO: maybe change to async task
	db.OpsLog.LogEvent(self, db.ACT_GUEST_CPUSET_REMOVE, nil, userCred)
	resp := new(api.ServerCPUSetRemoveResp)
	if err := self.GetDriver().RequestCPUSetRemove(ctx, userCred, host, self, data); err != nil {
		db.OpsLog.LogEvent(self, db.ACT_GUEST_CPUSET_REMOVE_FAIL, err, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CPUSET_REMOVE, data, userCred, false)
		resp.Error = err.Error()
	} else {
		db.OpsLog.LogEvent(self, db.ACT_GUEST_CPUSET_REMOVE, nil, userCred)
		logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_CPUSET_REMOVE, data, userCred, true)
		resp.Done = true
	}
	return resp, nil
}

func (self *SGuest) getPinnedCpusetCores(ctx context.Context, userCred mcclient.TokenCredential) ([]int, error) {
	obj := self.GetMetadataJson(ctx, api.VM_METADATA_CGROUP_CPUSET, userCred)
	if obj == nil {
		return nil, nil
	}
	pinnedInput := new(api.ServerCPUSetInput)
	if err := obj.Unmarshal(pinnedInput); err != nil {
		return nil, errors.Wrap(err, "Unmarshal to ServerCPUSetInput")
	}
	return pinnedInput.CPUS, nil
}

func (self *SGuest) GetDetailsCpusetCores(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerGetCPUSetCoresInput) (*api.ServerGetCPUSetCoresResp, error) {
	host, err := self.GetHost()
	if err != nil {
		return nil, err
	}

	allCores, err := host.getHostLogicalCores()
	if err != nil {
		return nil, err
	}

	usedMap, err := host.GetPinnedCpusetCores(ctx, userCred)
	if err != nil {
		return nil, err
	}
	usedSets := sets.NewInt()
	for _, used := range usedMap {
		usedSets.Insert(used...)
	}

	resp := &api.ServerGetCPUSetCoresResp{
		HostCores:     allCores,
		HostUsedCores: usedSets.List(),
	}

	// fetch cpuset pinned
	pinned, err := self.getPinnedCpusetCores(ctx, userCred)
	if err != nil {
		log.Errorf("getPinnedCpusetCores error: %v", err)
	} else {
		resp.PinnedCores = pinned
	}

	return resp, nil
}

func (self *SGuest) PerformCalculateRecordChecksum(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	checksum, err := db.CalculateModelChecksum(self)
	if err != nil {
		return nil, errors.Wrap(err, "CalculateModelChecksum")
	}
	return jsonutils.Marshal(map[string]string{
		"checksum": checksum,
	}), nil
}

func (self *SGuest) PerformEnableMemclean(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, self.SetMetadata(ctx, api.VM_METADATA_ENABLE_MEMCLEAN, "true", userCred)
}
