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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskResizeTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskResizeTask{})
}

func (self *DiskResizeTask) SetDiskReady(ctx context.Context, disk *models.SDisk, userCred mcclient.TokenCredential, reason string) {
	// 此函数主要避免虚机更改配置时，虚机可能出现中间状态
	// 若是子任务，磁盘关联的虚拟机状态由父任务恢复，仅恢复磁盘自身状态即可
	disk.SetStatus(userCred, api.DISK_READY, reason)
	// 若不是子任务，由于扩容时设置了关联的虚机状态，虚机的状态也由自己恢复
}

func (self *DiskResizeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	var host *models.SHost
	storage := disk.GetStorage()
	guest := disk.GetGuest()
	if guest != nil {
		host = guest.GetHost()
	} else {
		host = storage.GetMasterHost()
	}

	reason := "Cannot find host for disk"
	if host == nil || host.HostStatus != api.HOST_ONLINE {
		self.SetDiskReady(ctx, disk, self.GetUserCred(), reason)
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, reason, self.UserCred, false)
		return
	}

	disk.SetStatus(self.GetUserCred(), api.DISK_START_RESIZE, "")
	self.StartResizeDisk(ctx, host, storage, disk)
}

func (self *DiskResizeTask) StartResizeDisk(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk) {
	log.Infof("Resizing disk on host %s ...", host.GetName())
	self.SetStage("OnDiskResizeComplete", nil)
	sizeMb, _ := self.GetParams().Int("size")
	if err := host.GetHostDriver().RequestResizeDiskOnHost(ctx, host, storage, disk, sizeMb, self); err != nil {
		log.Errorf("request_resize_disk_on_host: %v", err)
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	self.OnStartResizeDiskSucc(ctx, disk)
}

func (self *DiskResizeTask) OnStartResizeDiskSucc(ctx context.Context, disk *models.SDisk) {
	disk.SetStatus(self.GetUserCred(), api.DISK_RESIZING, "")
}

func (self *DiskResizeTask) OnStartResizeDiskFailed(ctx context.Context, disk *models.SDisk, reason error) {
	self.SetDiskReady(ctx, disk, self.GetUserCred(), reason.Error())
	self.SetStageFailed(ctx, reason.Error())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, reason.Error(), self.UserCred, false)
}

func (self *DiskResizeTask) OnDiskResizeComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	jSize, err := data.Get("disk_size")
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	sizeMb, err := jSize.Int()
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	oldStatus := disk.Status
	_, err = db.Update(disk, func() error {
		disk.Status = api.DISK_READY
		disk.DiskSize = int(sizeMb)
		return nil
	})
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	self.SetDiskReady(ctx, disk, self.GetUserCred(), "")
	notes := fmt.Sprintf("%s=>%s", oldStatus, disk.Status)
	db.OpsLog.LogEvent(disk, db.ACT_UPDATE_STATUS, notes, self.UserCred)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE, disk.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, nil, self.UserCred, true)
	self.OnDiskResized(ctx, disk)
}

func (self *DiskResizeTask) OnDiskResized(ctx context.Context, disk *models.SDisk) {
	self.SetStageComplete(ctx, disk.GetShortDesc(ctx))
	self.finalReleasePendingUsage(ctx)
}

func (self *DiskResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetDiskReady(ctx, disk, self.GetUserCred(), data.String())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, disk.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, data.String(), self.UserCred, false)
	self.SetStageFailed(ctx, data.String())
}
