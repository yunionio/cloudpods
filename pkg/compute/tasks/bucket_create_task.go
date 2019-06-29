package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BucketCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BucketCreateTask{})
}

func (task *BucketCreateTask) taskFailed(ctx context.Context, bucket *models.SBucket, err error) {
	bucket.SetStatus(task.UserCred, api.BUCKET_STATUS_CREATE_FAIL, err.Error())
	db.OpsLog.LogEvent(bucket, db.ACT_ALLOCATE_FAIL, err.Error(), task.UserCred)
	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_ALLOCATE, err.Error(), task.UserCred, false)
	task.SetStageFailed(ctx, err.Error())
}

func (task *BucketCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	bucket := obj.(*models.SBucket)

	bucket.SetStatus(task.UserCred, api.BUCKET_STATUS_CREATING, "StartBucketCreateTask")

	err := bucket.RemoteCreate(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, bucket, err)
		return
	}

	bucket.SetStatus(task.UserCred, api.BUCKET_STATUS_READY, "BucketCreateTask")
	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_ALLOCATE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
