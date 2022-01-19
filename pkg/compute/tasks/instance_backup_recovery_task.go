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

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	compute_modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceBackupRecoveryTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupRecoveryTask{})
}

func (self *InstanceBackupRecoveryTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	ib.SetStatus(self.UserCred, compute.INSTANCE_BACKUP_STATUS_CREATE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_RECOVERY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupRecoveryTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup) {
	ib.SetStatus(self.UserCred, compute.INSTANCE_BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_RECOVERY, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupRecoveryTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	serverName, _ := self.Params.GetString("server_name")
	if serverName == "" {
		serverName, _ = ib.ServerConfig.GetString("name")
	}
	sourceInput := &compute.ServerCreateInput{}
	sourceInput.ServerConfigs = &compute.ServerConfigs{}
	sourceInput.GenerateName = serverName
	sourceInput.Description = fmt.Sprintf("recovery from instance backup %s", ib.GetName())
	sourceInput.InstanceBackupId = ib.GetId()
	sourceInput.Hypervisor = compute.HYPERVISOR_KVM
	taskHeader := self.GetTaskRequestHeader()
	session := auth.GetSession(ctx, self.UserCred, "", "")
	session.Header.Set(mcclient.TASK_NOTIFY_URL, taskHeader.Get(mcclient.TASK_NOTIFY_URL))
	session.Header.Set(mcclient.TASK_ID, taskHeader.Get(mcclient.TASK_ID))
	serverData, err := compute_modules.Servers.Create(session, jsonutils.Marshal(sourceInput))
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	guestId, _ := serverData.GetString("id")
	params := jsonutils.NewDict()
	params.Set("guest_id", jsonutils.NewString(guestId))
	self.SetStage("OnCreateGuest", params)
}

func (self *InstanceBackupRecoveryTask) OnCreateGuest(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	guestId, _ := self.Params.GetString("guest_id")
	guest := models.GuestManager.FetchGuestById(guestId)
	if guest == nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(fmt.Sprintf("no such guest %s", guest.GetId())))
		return
	}
	disks, err := guest.GetDisks()
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	if len(disks) == 0 {
		self.taskFailed(ctx, ib, jsonutils.NewString(fmt.Sprintf("no disks in guest %s?", guestId)))
		return
	}
	sysDisk := &disks[0]
	backups, err := ib.GetBackups()
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(fmt.Sprintf(err.Error())))
		return
	}
	db.Update(sysDisk, func() error {
		sysDisk.TemplateId, _ = backups[0].DiskConfig.GetString("image_id")
		sysDisk.SnapshotId, _ = backups[0].DiskConfig.GetString("snapshot_id")
		return nil
	})
	self.taskSuccess(ctx, ib)
}

func (self *InstanceBackupRecoveryTask) OnCreateGuestFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}
