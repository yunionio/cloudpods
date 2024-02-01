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

type BucketSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BucketSyncstatusTask{})
}

func (self *BucketSyncstatusTask) taskFailed(ctx context.Context, bucket *models.SBucket, err error) {
	bucket.SetStatus(ctx, self.GetUserCred(), api.BUCKET_STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	db.OpsLog.LogEvent(bucket, db.ACT_SYNC_STATUS, bucket.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, bucket, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
}

func (self *BucketSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	bucket := obj.(*models.SBucket)

	region, err := bucket.GetRegion()
	if err != nil {
		self.taskFailed(ctx, bucket, errors.Wrap(err, "bucket.GetRegion"))
		return
	}

	self.SetStage("OnBucketSyncStatusComplete", nil)
	err = region.GetDriver().RequestSyncBucketStatus(ctx, self.GetUserCred(), bucket, self)
	if err != nil {
		self.taskFailed(ctx, bucket, errors.Wrap(err, "RequestSyncBucketStatus"))
		return
	}
}

func (self *BucketSyncstatusTask) OnBucketSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *BucketSyncstatusTask) OnBucketSyncStatusCompleteFailed(ctx context.Context, bucket *models.SBucket, data jsonutils.JSONObject) {
	self.taskFailed(ctx, bucket, fmt.Errorf(data.String()))
}
