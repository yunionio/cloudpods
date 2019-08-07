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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type StorageCacheImageTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(StorageCacheImageTask{})
}

func (self *StorageCacheImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	// isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)

	storageCache := obj.(*models.SStoragecache)

	// first check if the storageCache reach cache limit
	if storageCache.IsReachCapacityLimit(imageId) {
		self.SetStage("OnRelinquishLeastUsedCachedImageComplete", nil)
		storageCache.StartRelinquishLeastUsedCachedImageTask(ctx, self.UserCred, imageId, self.GetTaskId())
	} else {
		self.OnRelinquishLeastUsedCachedImageComplete(ctx, obj, data)
	}
}

func (self *StorageCacheImageTask) OnRelinquishLeastUsedCachedImageComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	// isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)

	storageCache := obj.(*models.SStoragecache)

	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId, "")
	if scimg.Status != api.CACHED_IMAGE_STATUS_READY {
		scimg.SetStatus(self.UserCred, api.CACHED_IMAGE_STATUS_CACHING, "storage_cache_image_task")
	}

	db.OpsLog.LogEvent(storageCache, db.ACT_CACHING_IMAGE, imageId, self.UserCred)

	self.SetStage("on_image_cache_complete", nil)

	host, _ := storageCache.GetHost()
	err := host.GetHostDriver().CheckAndSetCacheImage(ctx, host, storageCache, self)
	if err != nil {
		errData := taskman.Error2TaskData(err)
		self.OnImageCacheCompleteFailed(ctx, storageCache, errData)
	}
}

func (self *StorageCacheImageTask) OnImageCacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)
	self.OnCacheSucc(ctx, storageCache, data.(*jsonutils.JSONDict))
}

func (self *StorageCacheImageTask) OnImageCacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)
	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId, "")
	err := fmt.Errorf(data.String())
	extImgId, _ := data.GetString("image_id")
	self.OnCacheFailed(ctx, storageCache, imageId, scimg, err, extImgId)
}

func (self *StorageCacheImageTask) OnCacheFailed(ctx context.Context, cache *models.SStoragecache, imageId string, scimg *models.SStoragecachedimage, err error, extImgId string) {
	scimg.SetStatus(self.UserCred, api.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
	if len(extImgId) > 0 && scimg.ExternalId != extImgId {
		scimg.SetExternalId(extImgId)
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(err.Error()), "reason")
	body.Add(jsonutils.NewString(imageId), "image_id")
	db.OpsLog.LogEvent(cache, db.ACT_CACHE_IMAGE_FAIL, body, self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *StorageCacheImageTask) OnCacheSucc(ctx context.Context, cache *models.SStoragecache, data *jsonutils.JSONDict) {
	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, cache.Id, imageId, "")
	extImgId, _ := data.GetString("image_id")

	scimg.SetStatus(self.UserCred, api.CACHED_IMAGE_STATUS_READY, "cached")
	if len(extImgId) > 0 && scimg.ExternalId != extImgId {
		scimg.SetExternalId(extImgId)
	}
	models.CachedimageManager.ImageAddRefCount(imageId)
	db.OpsLog.LogEvent(cache, db.ACT_CACHED_IMAGE, imageId, self.UserCred)
	self.SetStageComplete(ctx, data)
}
