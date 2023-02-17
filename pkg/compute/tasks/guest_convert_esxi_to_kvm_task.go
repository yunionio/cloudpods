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
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestConvertEsxiToKvmTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestConvertEsxiToKvmTask{})
}

func (self *GuestConvertEsxiToKvmTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, []db.IStandaloneModel{obj})
}

func (self *GuestConvertEsxiToKvmTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	schedDesc := guest.ToSchedDesc()
	if self.Params.Contains("prefer_host_id") {
		preferHostId, _ := self.Params.GetString("prefer_host_id")
		schedDesc.ServerConfig.PreferHost = preferHostId
	}
	for i := range schedDesc.Disks {
		schedDesc.Disks[i].Backend = ""
		schedDesc.Disks[i].Medium = ""
		schedDesc.Disks[i].Storage = ""
	}
	schedDesc.Hypervisor = api.HYPERVISOR_KVM
	return schedDesc, nil
}

func (self *GuestConvertEsxiToKvmTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_CONVERTING, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERTING, "", self.UserCred)
}

func (self *GuestConvertEsxiToKvmTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	guest := self.GetObject().(*models.SGuest)
	self.taskFailed(ctx, guest, reason)
}

func (self *GuestConvertEsxiToKvmTask) taskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_CONVERT_FAILED, reason.String())
	targetGuest := self.getTargetGuest()
	targetGuest.SetStatus(self.UserCred, api.VM_CONVERT_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERT_FAIL, reason, self.UserCred)
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_CONVERT, reason, self.UserCred, false)
	logclient.AddSimpleActionLog(targetGuest, logclient.ACT_VM_CONVERT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestConvertEsxiToKvmTask) GenerateEsxiAcceessInfo(guest *models.SGuest) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	host, _ := guest.GetHost()
	accessInfo, err := host.GetCloudaccount().GetVCenterAccessInfo("")
	if err != nil {
		return nil, err
	}
	ret.Set("datastore", jsonutils.Marshal(accessInfo))
	ret.Set("host_ip", jsonutils.NewString(host.AccessIp))
	ret.Set("guest_ext_id", jsonutils.NewString(guest.ExternalId))
	return ret, nil
}

func (self *GuestConvertEsxiToKvmTask) getTargetGuest() *models.SGuest {
	guestId, _ := self.Params.GetString("target_guest_id")
	return models.GuestManager.FetchGuestById(guestId)
}

// update database for convert esxi to kvm in the part of guest, guestdisks, guestnetworks
func (self *GuestConvertEsxiToKvmTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource) {
	guest := obj.(*models.SGuest)
	targetGuest := self.getTargetGuest()
	esxiAccessInfo, err := self.GenerateEsxiAcceessInfo(guest)
	if err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("generate esxi access info %s", err)))
		return
	}
	err = targetGuest.SetHostId(self.UserCred, target.HostId)
	if err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("update guest %s", err)))
		return
	}
	err = targetGuest.SetMetadata(ctx, api.SERVER_META_CONVERT_FROM_ESXI, guest.Id, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest set metadata %s", err)))
		return
	}
	host, _ := targetGuest.GetHost()

	//pendingUsage := models.SQuota{}
	input := new(api.ServerCreateInput)

	err = self.Params.Unmarshal(input, "input")
	if err != nil {
		log.Errorf("fail to unmarshal params input")
		input = guest.ToCreateInput(ctx, self.UserCred)
	}

	//pendingUsage.Storage = guest.GetDisksSize()
	err = targetGuest.CreateDisksOnHost(ctx, self.UserCred, host, input.Disks, nil,
		true, true, target.Disks, nil, true)
	if err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest create disks %s", err)))
		return
	}

	self.SetStage("OnHostCreateGuest", nil)
	if err = self.RequestHostCreateGuestFromEsxi(ctx, targetGuest, esxiAccessInfo); err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	host.ClearSchedDescCache()
}

func (self *GuestConvertEsxiToKvmTask) RequestHostCreateGuestFromEsxi(
	ctx context.Context, guest *models.SGuest, esxiAccessInfo *jsonutils.JSONDict,
) error {
	host, _ := guest.GetHost()
	params := jsonutils.NewDict()
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	params.Set("desc", jsonutils.Marshal(desc))
	params.Set("esxi_access_info", esxiAccessInfo)
	url := fmt.Sprintf("%s/servers/%s/create-form-esxi", host.ManagerUri, guest.Id)
	header := self.GetTaskRequestHeader()
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *GuestConvertEsxiToKvmTask) OnHostCreateGuest(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	targetGuest := self.getTargetGuest()
	guestDisks, _ := targetGuest.GetGuestDisks()
	for i := 0; i < len(guestDisks); i++ {
		disk := guestDisks[i].GetDisk()
		esxiFlatFilePath, _ := data.GetString(disk.Id, "esxi_flat_filepath")
		diskPath, _ := data.GetString(disk.Id, "disk_path")
		_, err := db.Update(disk, func() error {
			disk.AccessPath = diskPath
			disk.Status = api.DISK_READY
			return nil
		})
		// TODO: update flat file path on guest start
		err = disk.SetMetadata(ctx, api.DISK_META_ESXI_FLAT_FILE_PATH, esxiFlatFilePath, self.UserCred)
		if err != nil {
			log.Errorf("disk set metadata failed %s", err)
			self.taskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), self.UserCred)
	}
	if err := guest.ConvertNetworks(targetGuest); err != nil {
		self.taskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	self.TaskComplete(ctx, guest, targetGuest)
}

func (self *GuestConvertEsxiToKvmTask) OnHostCreateGuestFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.taskFailed(ctx, guest, data)
}

func (self *GuestConvertEsxiToKvmTask) TaskComplete(ctx context.Context, guest, targetGuest *models.SGuest) {
	guest.SetMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, targetGuest.Id, self.UserCred)
	guest.SetStatus(self.UserCred, api.VM_CONVERTED, "")
	if osProfile := guest.GetMetadata(ctx, "__os_profile__", self.UserCred); len(osProfile) > 0 {
		targetGuest.SetMetadata(ctx, "__os_profile__", osProfile, self.UserCred)
	}
	if account := guest.GetMetadata(ctx, api.VM_METADATA_LOGIN_ACCOUNT, self.UserCred); len(account) > 0 {
		targetGuest.SetMetadata(ctx, api.VM_METADATA_LOGIN_ACCOUNT, account, self.UserCred)
	}
	if loginKey := guest.GetMetadata(ctx, api.VM_METADATA_LOGIN_KEY, self.UserCred); len(loginKey) > 0 {
		passwd, _ := utils.DescryptAESBase64(guest.Id, loginKey)
		if len(passwd) > 0 {
			secret, err := utils.EncryptAESBase64(targetGuest.Id, passwd)
			if err == nil {
				targetGuest.SetMetadata(ctx, api.VM_METADATA_LOGIN_KEY, secret, self.UserCred)
			}
		}
	}
	for _, k := range []string{api.VM_METADATA_OS_ARCH, api.VM_METADATA_OS_DISTRO, api.VM_METADATA_OS_NAME, api.VM_METADATA_OS_VERSION} {
		if v := guest.GetMetadata(ctx, k, self.UserCred); len(v) > 0 {
			targetGuest.SetMetadata(ctx, k, v, self.UserCred)
		}
	}

	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERT, "", self.UserCred)
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_CONVERT, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
	targetGuest.StartGueststartTask(ctx, self.UserCred, nil, "")
}
