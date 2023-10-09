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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestConvertCloudpodsToKvmTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestConvertCloudpodsToKvmTask{})
}

func (task *GuestConvertCloudpodsToKvmTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
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
	schedDesc.Hypervisor = api.HYPERVISOR_KVM
	return schedDesc, nil
}

func (task *GuestConvertCloudpodsToKvmTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, []db.IStandaloneModel{obj})
}

func (task *GuestConvertCloudpodsToKvmTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(task.UserCred, api.VM_CONVERTING, "")
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERTING, "", task.UserCred)
}

func (task *GuestConvertCloudpodsToKvmTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	guest := task.GetObject().(*models.SGuest)
	task.taskFailed(ctx, guest, reason)
}

func (task *GuestConvertCloudpodsToKvmTask) taskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(task.UserCred, api.VM_CONVERT_FAILED, reason.String())
	targetGuest := task.getTargetGuest()
	targetGuest.SetStatus(task.UserCred, api.VM_CONVERT_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_VM_CONVERT_FAIL, reason, task.UserCred)
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_CONVERT, reason, task.UserCred, false)
	logclient.AddSimpleActionLog(targetGuest, logclient.ACT_VM_CONVERT, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

func (task *GuestConvertCloudpodsToKvmTask) getTargetGuest() *models.SGuest {
	guestId, _ := task.Params.GetString("target_guest_id")
	return models.GuestManager.FetchGuestById(guestId)
}

func (task *GuestConvertCloudpodsToKvmTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource, index int) {
	guest := obj.(*models.SGuest)
	targetGuest := task.getTargetGuest()

	err := targetGuest.SetHostId(task.UserCred, target.HostId)
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
	input := new(api.ServerCreateInput)

	err = task.Params.Unmarshal(input, "input")
	if err != nil {
		log.Errorf("fail to unmarshal params input")
		input = guest.ToCreateInput(ctx, task.UserCred)
	}

	//pendingUsage.Storage = guest.GetDisksSize()
	err = targetGuest.CreateDisksOnHost(ctx, task.UserCred, host, input.Disks, nil,
		true, true, target.Disks, nil, true)
	if err != nil {
		task.taskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest create disks %s", err)))
		return
	}
}
