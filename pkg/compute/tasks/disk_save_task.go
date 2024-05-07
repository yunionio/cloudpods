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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

type DiskSaveTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskSaveTask{})
}

func (self *DiskSaveTask) GetMasterHost(disk *models.SDisk) *models.SHost {
	if guests := disk.GetGuests(); len(guests) == 1 {
		if host, _ := guests[0].GetHost(); host == nil {
			if storage, _ := disk.GetStorage(); storage != nil {
				host, _ = storage.GetMasterHost()
				return host
			}
		} else {
			return host
		}
	}
	return nil
}

func (self *DiskSaveTask) taskFailed(ctx context.Context, disk *models.SDisk, err error) {
	disk.SetDiskReady(ctx, self.GetUserCred(), err.Error())
	db.OpsLog.LogEvent(disk, db.ACT_SAVE_FAIL, err.Error(), self.GetUserCred())
	if imageId, _ := self.GetParams().GetString("image_id"); len(imageId) > 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		image.Images.PerformAction(s, imageId, "status", jsonutils.Marshal(map[string]string{"status": "killed", "reason": err.Error()}))
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DiskSaveTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	host := self.GetMasterHost(disk)
	if host == nil {
		self.taskFailed(ctx, disk, fmt.Errorf("Cannot find host for disk"))
		return
	}
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_START_SAVE, "")
	for _, guest := range disk.GetGuests() {
		guest.SetStatus(ctx, self.GetUserCred(), api.VM_SAVE_DISK, "")
	}
	self.StartBackupDisk(ctx, disk, host)
}

func (self *DiskSaveTask) StartBackupDisk(ctx context.Context, disk *models.SDisk, host *models.SHost) {
	self.SetStage("OnDiskBackupComplete", nil)
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_SAVING, "")
	imageId, _ := self.GetParams().GetString("image_id")
	driver, err := host.GetHostDriver()
	if err != nil {
		self.taskFailed(ctx, disk, errors.Wrapf(err, "GetHostDriver"))
		return
	}
	err = driver.RequestPrepareSaveDiskOnHost(ctx, host, disk, imageId, self)
	if err != nil {
		self.taskFailed(ctx, disk, errors.Wrapf(err, "RequestPrepareSaveDiskOnHost"))
		return
	}
}

func (self *DiskSaveTask) OnDiskBackupCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetDiskReady(ctx, self.GetUserCred(), data.String())
	db.OpsLog.LogEvent(disk, db.ACT_SAVE_FAIL, data, self.GetUserCred())
	self.SetStageFailed(ctx, data)
}

func (self *DiskSaveTask) OnDiskBackupComplete(ctx context.Context, disk *models.SDisk, data *jsonutils.JSONDict) {
	disk.SetDiskReady(ctx, self.GetUserCred(), "")
	db.OpsLog.LogEvent(disk, db.ACT_SAVE, disk.GetShortDesc(ctx), self.GetUserCred())
	imageId, _ := self.GetParams().GetString("image_id")
	host := self.GetMasterHost(disk)
	if host == nil {
		self.taskFailed(ctx, disk, fmt.Errorf("Saved disk Host mast not be nil"))
		return
	}
	if self.Params.Contains("format") {
		format, _ := self.Params.Get("format")
		data.Add(format, "format")
	}
	err := self.UploadDisk(ctx, host, disk, imageId, data)
	if err != nil {
		self.taskFailed(ctx, disk, errors.Wrapf(err, "UploadDisk"))
		return
	} else {
		// notify guest save image task to resume guest
		// disk save task waiting for image uploaded to refresh imagecache
		self.NotifyParentTaskComplete(ctx, jsonutils.NewDict(), false)
	}
}

func (self *DiskSaveTask) RefreshImageCache(ctx context.Context, imageId string) {
	models.CachedimageManager.GetImageById(ctx, self.GetUserCred(), imageId, true)
}

func (self *DiskSaveTask) UploadDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, data *jsonutils.JSONDict) error {
	driver, err := host.GetHostDriver()
	if err != nil {
		return errors.Wrapf(err, "GetHostDriver")
	}
	self.SetStage("OnUploadDiskComplete", nil)
	return driver.RequestSaveUploadImageOnHost(ctx, host, disk, imageId, self, jsonutils.Marshal(data))
}

func (self *DiskSaveTask) OnUploadDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	imageId, _ := self.GetParams().GetString("image_id")
	self.RefreshImageCache(ctx, imageId)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskSaveTask) OnUploadDiskCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.taskFailed(ctx, disk, fmt.Errorf(data.String()))
}
