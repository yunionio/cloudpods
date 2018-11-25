package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"

	"yunion.io/x/onecloud/pkg/util/logclient"
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

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHING_IMAGE, imageId, self.UserCred)

	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)

	if scimg == nil || len(scimg.Path) == 0 {
		// "image is not cached on this storage"
		self.OnImageUncacheComplete(ctx, storageCache, nil)
	}

	if isForce {
		self.OnImageUncacheComplete(ctx, obj, data)
		return
	}

	host, err := storageCache.GetHost()
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, fmt.Sprintf("fail to get host %s", err))
		return
	}

	if host == nil {
		self.OnImageUncacheComplete(ctx, obj, data)
		return
	}

	self.SetStage("OnImageUncacheComplete", nil)

	err = host.GetHostDriver().RequestUncacheImage(ctx, host, storageCache, self)

	if err != nil {
		self.OnTaskFailed(ctx, storageCache, fmt.Sprintf("fail to uncache image %s", err))
	}
}

func (self *StorageUncacheImageTask) OnTaskFailed(ctx context.Context, storageCache *models.SStoragecache, reason string) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(reason), "reason")
	imageId, _ := self.Params.GetString("image_id")
	body.Add(jsonutils.NewString(imageId), "image_id")

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHE_IMAGE_FAIL, body, self.UserCred)

	logclient.AddActionLog(storageCache, logclient.ACT_UNCACHED_IMAGE, body, self.UserCred, false)

	self.SetStageFailed(ctx, reason)
}

func (self *StorageUncacheImageTask) OnImageUncacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)

	self.OnTaskFailed(ctx, storageCache, data.String())
}

func (self *StorageUncacheImageTask) OnImageUncacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("Uncached image task success: %s", data)
	storageCache := obj.(*models.SStoragecache)

	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)
	if scimg != nil {
		scimg.Detach(ctx, self.UserCred)
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(imageId), "image_id")
	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHED_IMAGE, body, self.UserCred)

	logclient.AddActionLog(storageCache, db.ACT_UNCACHED_IMAGE, body, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
