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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudregionSyncImagesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudregionSyncImagesTask{})
}

func (self *CloudregionSyncImagesTask) taskFailed(ctx context.Context, region *models.SCloudregion, err error) {
	db.OpsLog.LogEvent(region, db.ACT_SYNC_CLOUD_IMAGES, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, region, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudregionSyncImagesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	region := obj.(*models.SCloudregion)

	err := region.SyncCloudImages(ctx, self.GetUserCred(), true, false)
	if err != nil {
		self.taskFailed(ctx, region, errors.Wrapf(err, "SyncCloudImages"))
		return
	}

	storagecaches, err := region.GetStoragecaches()
	if err != nil {
		self.taskFailed(ctx, region, errors.Wrapf(err, "GetStoragecaches"))
		return
	}

	for i := range storagecaches {
		err = storagecaches[i].CheckCloudimages(ctx, self.GetUserCred(), region.Name, region.Id)
		if err != nil {
			log.Errorf("SyncSystemImages for region %s(%s) storagecache %s error: %v", region.Name, region.Id, storagecaches[i].Name, err)
		}
	}

	self.SetStageComplete(ctx, nil)
}
