package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type StorageCacheImageTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(StorageCacheImageTask{})
}

func (self *StorageCacheImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)

	storageCache := obj.(*models.SStoragecache)
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)
	if scimg.Status != models.CACHED_IMAGE_STATUS_READY {
		scimg.SetStatus(self.UserCred, models.CACHED_IMAGE_STATUS_CACHING, "storage_cache_image_task")
	}

	db.OpsLog.LogEvent(storageCache, db.ACT_CACHING_IMAGE, imageId, self.UserCred)

	self.SetStage("on_image_cache_complete", nil)

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, err
		}

		extImgId, err := iStorageCache.UploadImage(self.UserCred, imageId, scimg.ExternalId, isForce)

		if err != nil {
			return nil, err
		} else {
			ret := jsonutils.NewDict()
			ret.Add(jsonutils.NewString(extImgId), "image_id")
			return ret, nil
		}
	})
}

func (self *StorageCacheImageTask) OnImageCacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)
	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)
	extImgId, _ := data.GetString("image_id")
	self.OnCacheSucc(ctx, storageCache, imageId, scimg, extImgId)
}

func (self *StorageCacheImageTask) OnImageCacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)
	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId)
	err := fmt.Errorf(data.String())
	self.OnCacheFailed(ctx, storageCache, imageId, scimg, err)
}

func (self *StorageCacheImageTask) OnCacheFailed(ctx context.Context, cache *models.SStoragecache, imageId string, scimg *models.SStoragecachedimage, err error) {
	scimg.SetStatus(self.UserCred, models.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
	/* if len(extImgId) > 0 && scimg.ExternalId != extImgId {
		scimg.SetExternalId(extImgId)
	}*/
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(err.Error()), "reason")
	body.Add(jsonutils.NewString(imageId), "image_id")
	db.OpsLog.LogEvent(cache, db.ACT_CACHE_IMAGE_FAIL, body, self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *StorageCacheImageTask) OnCacheSucc(ctx context.Context, cache *models.SStoragecache, imageId string, scimg *models.SStoragecachedimage, extImgId string) {
	scimg.SetStatus(self.UserCred, models.CACHED_IMAGE_STATUS_READY, "cached")
	if len(extImgId) > 0 && scimg.ExternalId != extImgId {
		scimg.SetExternalId(extImgId)
	}
	models.CachedimageManager.ImageAddRefCount(imageId)
	db.OpsLog.LogEvent(cache, db.ACT_CACHED_IMAGE, imageId, self.UserCred)
	self.SetStageComplete(ctx, nil)
}
