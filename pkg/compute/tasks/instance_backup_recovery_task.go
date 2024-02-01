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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_CREATE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_RECOVERY, reason, self.UserCred, false)
	db.OpsLog.LogEvent(ib, db.ACT_RECOVERY_FAIL, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupRecoveryTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup) {
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_RECOVERY, nil, self.UserCred, true)
	db.OpsLog.LogEvent(ib, db.ACT_RECOVERY, ib.GetShortDesc(ctx), self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupRecoveryTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)

	sourceInput := compute.ServerCreateInput{}

	serverName, _ := self.Params.GetString("server_name")
	if serverName == "" {
		serverName, _ = ib.ServerConfig.GetString("name")
	}
	sourceInput.GenerateName = serverName
	sourceInput.Description = fmt.Sprintf("recovered from instance backup %s", ib.GetName())
	sourceInput.InstanceBackupId = ib.GetId()

	// PANIC ?????
	// sourceInput.Hypervisor = compute.HYPERVISOR_KVM

	ownerId := ib.GetOwnerId()

	projectId, _ := self.Params.GetString("project_id")
	if projectId == "" {
		projectId, _ = ib.ServerConfig.GetString("tenant_id")
	}
	if projectId != "" {
		tenant, err := db.TenantCacheManager.FetchTenantByIdOrNameInDomain(ctx, projectId, "")
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
			return
		}
		if tenant != nil {
			sourceInput.ProjectId = tenant.Id
			sourceInput.ProjectDomainId = tenant.DomainId
			ownerId = &db.SOwnerId{DomainId: tenant.DomainId, ProjectId: tenant.Id}
		}
	}

	params := sourceInput.JSON(sourceInput)
	guestObj, err := db.DoCreate(models.GuestManager, ctx, self.UserCred, nil, params, ownerId)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	guest := guestObj.(*models.SGuest)

	func() {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		guest.PostCreate(ctx, self.UserCred, ownerId, nil, params)
	}()

	params.Set("guest_id", jsonutils.NewString(guest.Id))
	self.SetStage("OnCreateGuest", params)
	params.Set("parent_task_id", jsonutils.NewString(self.GetTaskId()))
	models.GuestManager.OnCreateComplete(ctx, []db.IModel{guest}, self.UserCred, ownerId, nil, []jsonutils.JSONObject{params})
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
		sysDisk.TemplateId = backups[0].DiskConfig.ImageId
		sysDisk.SnapshotId = backups[0].DiskConfig.SnapshotId
		return nil
	})
	self.taskSuccess(ctx, ib)
}

func (self *InstanceBackupRecoveryTask) OnCreateGuestFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}
