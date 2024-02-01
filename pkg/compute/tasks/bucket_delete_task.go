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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BucketDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BucketDeleteTask{})
}

func (task *BucketDeleteTask) taskFailed(ctx context.Context, bucket *models.SBucket, err error) {
	bucket.SetStatus(ctx, task.UserCred, api.VPC_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(bucket, db.ACT_DELOCATE_FAIL, err.Error(), task.UserCred)
	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_DELETE, err.Error(), task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *BucketDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	bucket := obj.(*models.SBucket)

	bucket.SetStatus(ctx, task.UserCred, api.BUCKET_STATUS_DELETING, "StartBucketDeleteTask")

	err := bucket.RemoteDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, bucket, err)
		return
	}

	notifyclient.EventNotify(ctx, task.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    bucket,
		Action: notifyclient.ActionDelete,
	})
	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_DELETE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
