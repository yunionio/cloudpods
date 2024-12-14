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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskSyncstatusTask{})
}

func (self *DiskSyncstatusTask) taskFailed(ctx context.Context, disk *models.SDisk, err error) {
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	db.OpsLog.LogEvent(disk, db.ACT_SYNC_STATUS, disk.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, disk, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
}

func (self *DiskSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	storage, _ := disk.GetStorage()
	if storage == nil {
		self.taskFailed(ctx, disk, fmt.Errorf("failed to found storage for disk %s", disk.Name))
		return
	}
	region, _ := storage.GetRegion()
	if region == nil {
		self.taskFailed(ctx, disk, fmt.Errorf("failed to found cloudregion for disk storage %s(%s)", disk.Name, disk.Id))
		return
	}

	self.SetStage("OnDiskSyncStatusComplete", nil)
	err := region.GetDriver().RequestSyncDiskStatus(ctx, self.GetUserCred(), disk, self)
	if err != nil {
		self.taskFailed(ctx, disk, errors.Wrap(err, "RequestSyncDiskStatus"))
		return
	}
}

func (self *DiskSyncstatusTask) OnDiskSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, obj, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskSyncstatusTask) OnDiskSyncStatusCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.taskFailed(ctx, disk, fmt.Errorf(data.String()))
}
