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

type BucketDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BucketDeleteTask{})
}

func (task *BucketDeleteTask) taskFailed(ctx context.Context, bucket *models.SBucket, err error) {
	bucket.SetStatus(task.UserCred, api.VPC_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(bucket, db.ACT_DELOCATE_FAIL, err.Error(), task.UserCred)
	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_DELETE, err.Error(), task.UserCred, false)
	task.SetStageFailed(ctx, err.Error())
}

func (task *BucketDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	bucket := obj.(*models.SBucket)

	bucket.SetStatus(task.UserCred, api.BUCKET_STATUS_DELETING, "StartBucketDeleteTask")

	err := bucket.RemoteDelete(ctx, task.UserCred)
	if err != nil {
		task.taskFailed(ctx, bucket, err)
		return
	}

	logclient.AddActionLogWithStartable(task, bucket, logclient.ACT_DELETE, nil, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
