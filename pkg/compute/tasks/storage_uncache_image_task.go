package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type StorageUncacheImageTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(StorageUncacheImageTask{})
}

func (self *StorageUncacheImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)

	storageCache := obj.(*models.SStoragecache)

	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHED_IMAGE, imageId, self.UserCred)

	if isForce {
		scimg.Detach(ctx, self.UserCred)
		self.SetStageComplete(ctx, nil)
		return
	}

	// TODO
}
