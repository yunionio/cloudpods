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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskChangeStorageTypeTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskChangeStorageTypeTask{})
}

func (self *DiskChangeStorageTypeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	idisk, err := disk.GetIDisk(ctx)
	if err != nil {
		self.taskFail(ctx, disk, errors.Wrapf(err, "GetIDisk"))
		return
	}

	storageId, err := self.GetParams().GetString("storage_id")
	if err != nil {
		self.taskFail(ctx, disk, errors.Wrapf(err, "get storage_id"))
		return
	}

	storageObj, err := models.StorageManager.FetchById(storageId)
	if err != nil {
		self.taskFail(ctx, disk, errors.Wrapf(err, "fetch storage by id"))
		return
	}
	storage := storageObj.(*models.SStorage)

	opts := &cloudprovider.ChangeStorageOptions{
		DiskId:      disk.ExternalId,
		StorageType: storage.StorageType,
	}

	err = idisk.ChangeStorage(ctx, opts)
	if err != nil {
		self.taskFail(ctx, disk, errors.Wrapf(err, "ChangeStorage"))
		return
	}

	db.Update(disk, func() error {
		disk.StorageId = storage.Id
		disk.Status = api.DISK_READY
		return nil
	})

	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_MIGRATE, opts.StorageType, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskChangeStorageTypeTask) taskFail(ctx context.Context, disk *models.SDisk, err error) {
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_MIGRATE_FAIL, "")
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_MIGRATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
