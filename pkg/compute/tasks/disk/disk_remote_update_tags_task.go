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

package disk

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DiskRemoteUpdateTask{})
}

func (self *DiskRemoteUpdateTask) taskFail(ctx context.Context, disk *models.SDisk, err error) {
	disk.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DiskRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	var host *models.SHost
	storage, _ := disk.GetStorage()
	guest := disk.GetGuest()

	if guest != nil {
		host, _ = guest.GetHost()
	} else {
		host, _ = storage.GetMasterHost()
	}

	reason := "Cannot find host for disk"
	if host == nil || host.HostStatus != api.HOST_ONLINE {
		disk.SetStatus(ctx, self.GetUserCred(), api.DISK_READY, reason)
		self.SetStageFailed(ctx, jsonutils.NewString(reason))
		logclient.AddActionLogWithStartable(self, disk, logclient.ACT_UPDATE_TAGS, reason, self.UserCred, false)
		return
	}

	self.StartRemoteUpdateTask(ctx, host, storage, disk)
}

func (self *DiskRemoteUpdateTask) StartRemoteUpdateTask(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk) {
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		driver, err := host.GetHostDriver()
		if err != nil {
			return nil, errors.Wrap(err, "GetHostDriver")
		}
		err = driver.RequestRemoteUpdateDisk(ctx, self.GetUserCred(), storage, disk, replaceTags)
		if err != nil {
			return nil, errors.Wrap(err, "RequestRemoteUpdateDisk")
		}
		return nil, nil
	})
}

func (self *DiskRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	disk.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *DiskRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.taskFail(ctx, disk, errors.Errorf(data.String()))
}

func (self *DiskRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *DiskRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
