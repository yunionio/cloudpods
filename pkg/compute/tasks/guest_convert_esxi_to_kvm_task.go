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
	"yunion.io/x/pkg/errors"
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

func (task *GuestConvertEsxiToKvmTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, []db.IStandaloneModel{obj})
}

func (task *GuestConvertEsxiToKvmTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)

	input := new(api.ServerCreateInput)
	err := task.Params.Unmarshal(input, "input")
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshal create input")
	}

	schedDesc := guest.ToSchedDesc()
	if task.Params.Contains("prefer_host_id") {
		preferHostId, _ := task.Params.GetString("prefer_host_id")
		schedDesc.ServerConfig.PreferHost = preferHostId
	}
	for i := range schedDesc.Disks {
		schedDesc.Disks[i].Backend = ""
		schedDesc.Disks[i].Medium = ""
		schedDesc.Disks[i].Storage = ""
	}

	schedDesc.Networks = input.Networks
	schedDesc.Hypervisor = api.HYPERVISOR_KVM
	return schedDesc, nil
}

func (task *GuestConvertEsxiToKvmTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(context.Background(), task.UserCred, api.VM_CONVERTING, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERTING, "", task.UserCred)
}

func (task *GuestConvertEsxiToKvmTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	guest := task.GetObject().(*models.SGuest)
	task.taskFailed(ctx, guest, reason)
}

func (task *GuestConvertEsxiToKvmTask) taskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_CONVERT_FAILED, reason.String())
	targetGuest := task.getTargetGuest()
	targetGuest.SetStatus(ctx, task.UserCred, api.VM_CONVERT_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERT_FAIL, reason, task.UserCred)
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_CONVERT, reason, task.UserCred, false)
	logclient.AddSimpleActionLog(targetGuest, logclient.ACT_VM_CONVERT, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

func (task *GuestConvertEsxiToKvmTask) GenerateEsxiAcceessInfo(guest *models.SGuest) (*jsonutils.JSONDict, error) {
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

func (task *GuestConvertEsxiToKvmTask) getTargetGuest() *models.SGuest {
	guestId, _ := task.Params.GetString("target_guest_id")
	return models.GuestManager.FetchGuestById(guestId)
}

// update database for convert esxi to kvm in the part of guest, guestdisks, guestnetworks
func (task *GuestConvertEsxiToKvmTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource, index int) {
	guest := obj.(*models.SGuest)
	targetGuest := task.getTargetGuest()
	esxiAccessInfo, err := task.GenerateEsxiAcceessInfo(guest)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("generate esxi access info %s", err)))
		return
	}
	err = targetGuest.SetHostId(task.UserCred, target.HostId)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("update guest %s", err)))
		return
	}
	err = targetGuest.SetMetadata(ctx, api.SERVER_META_CONVERT_FROM_ESXI, guest.Id, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest set metadata %s", err)))
		return
	}
	host, _ := targetGuest.GetHost()

	//pendingUsage := models.SQuota{}
	input := new(api.ServerCreateInput)

	err = task.Params.Unmarshal(input, "input")
	if err != nil {
		log.Errorf("fail to unmarshal params input")
		input = guest.ToCreateInput(ctx, task.UserCred)
	}

	err = task.Params.Unmarshal(input, "input")
	if err != nil {
		log.Errorf("fail to unmarshal params input")
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("umarshal create input %s", err)))
		return
	}

	err = targetGuest.CreateNetworksOnHost(ctx, task.UserCred, host, input.Networks, nil, nil, target.Nets)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest create networks %s", err)))
		return
	}
	gns, _ := targetGuest.GetNetworks("")
	for i := range gns {
		_, err := db.Update(&gns[i], func() error {
			gns[i].MacAddr = input.Networks[i].Mac
			return nil
		})
		if err != nil {
			task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("update guest networks macaddr %s", err)))
			return
		}
	}

	//pendingUsage.Storage = guest.GetDisksSize()
	err = targetGuest.CreateDisksOnHost(ctx, task.UserCred, host, input.Disks, nil,
		true, true, target.Disks, nil, true)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest create disks %s", err)))
		return
	}

	task.SetStage("OnHostCreateGuest", nil)
	if err = task.RequestHostCreateGuestFromEsxi(ctx, targetGuest, esxiAccessInfo); err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	host.ClearSchedDescCache()
}

func (task *GuestConvertEsxiToKvmTask) RequestHostCreateGuestFromEsxi(
	ctx context.Context, guest *models.SGuest, esxiAccessInfo *jsonutils.JSONDict,
) error {
	host, _ := guest.GetHost()
	params := jsonutils.NewDict()
	desc := guest.GetJsonDescAtHypervisor(ctx, host)
	params.Set("desc", jsonutils.Marshal(desc))
	params.Set("esxi_access_info", esxiAccessInfo)
	url := fmt.Sprintf("%s/servers/%s/create-form-esxi", host.ManagerUri, guest.Id)
	header := task.GetTaskRequestHeader()
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, params, false)
	if err != nil {
		return err
	}
	return nil
}

func (task *GuestConvertEsxiToKvmTask) OnHostCreateGuest(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	targetGuest := task.getTargetGuest()
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
		err = disk.SetMetadata(ctx, api.DISK_META_REMOTE_ACCESS_PATH, esxiFlatFilePath, task.UserCred)
		if err != nil {
			log.Errorf("disk set metadata failed %s", err)
			task.taskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), task.UserCred)
	}
	task.SetStage("OnGuestConvertDoDeployGuest", nil)

	input := new(api.ServerCreateInput)
	err := task.Params.Unmarshal(input, "input")
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed unmarshal input: %s", err)))
		return
	}

	deployParams := jsonutils.NewDict()
	deployParams.Set("reset_password", jsonutils.JSONFalse)
	if jsonutils.QueryBoolean(task.Params, "deploy_telegraf", false) {
		deployParams.Set("deploy_telegraf", jsonutils.JSONTrue)
	}
	targetGuest.StartGuestDeployTask(ctx, task.UserCred, deployParams, "deploy", task.GetTaskId())
}

func (task *GuestConvertEsxiToKvmTask) OnHostCreateGuestFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	task.taskFailed(ctx, guest, data)
}

func (task *GuestConvertEsxiToKvmTask) OnGuestConvertDoDeployGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetGuest := task.getTargetGuest()
	task.TaskComplete(ctx, guest, targetGuest)
}

func (task *GuestConvertEsxiToKvmTask) OnGuestConvertDoDeployGuestFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.taskFailed(ctx, guest, data)
}

func (task *GuestConvertEsxiToKvmTask) TaskComplete(ctx context.Context, guest, targetGuest *models.SGuest) {
	guest.SetMetadata(ctx, api.SERVER_META_CONVERTED_SERVER, targetGuest.Id, task.UserCred)
	guest.SetStatus(ctx, task.UserCred, api.VM_CONVERTED, "")
	if osProfile := guest.GetMetadata(ctx, "__os_profile__", task.UserCred); len(osProfile) > 0 {
		targetGuest.SetMetadata(ctx, "__os_profile__", osProfile, task.UserCred)
	}
	if account := guest.GetMetadata(ctx, api.VM_METADATA_LOGIN_ACCOUNT, task.UserCred); len(account) > 0 {
		targetGuest.SetMetadata(ctx, api.VM_METADATA_LOGIN_ACCOUNT, account, task.UserCred)
	}
	if loginKey := guest.GetMetadata(ctx, api.VM_METADATA_LOGIN_KEY, task.UserCred); len(loginKey) > 0 {
		passwd, _ := utils.DescryptAESBase64(guest.Id, loginKey)
		if len(passwd) > 0 {
			secret, err := utils.EncryptAESBase64(targetGuest.Id, passwd)
			if err == nil {
				targetGuest.SetMetadata(ctx, api.VM_METADATA_LOGIN_KEY, secret, task.UserCred)
			}
		}
	}
	for _, k := range []string{api.VM_METADATA_OS_ARCH, api.VM_METADATA_OS_DISTRO, api.VM_METADATA_OS_NAME, api.VM_METADATA_OS_VERSION} {
		if v := guest.GetMetadata(ctx, k, task.UserCred); len(v) > 0 {
			targetGuest.SetMetadata(ctx, k, v, task.UserCred)
		}
	}

	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERT, "", task.UserCred)
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_CONVERT, "", task.UserCred, true)
	task.SetStageComplete(ctx, nil)
	targetGuest.StartGueststartTask(ctx, task.UserCred, nil, "")
}
