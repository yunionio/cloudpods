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

package storage

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
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

func (uncacheTask *StorageUncacheImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := uncacheTask.Params.GetString("image_id")
	// isForce := jsonutils.QueryBoolean(uncacheTask.Params, "is_force", false)
	isPurge := jsonutils.QueryBoolean(uncacheTask.Params, "is_purge", false)

	storageCache := obj.(*models.SStoragecache)

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHING_IMAGE, imageId, uncacheTask.UserCred)

	scimg := models.StoragecachedimageManager.Register(ctx, uncacheTask.UserCred, storageCache.Id, imageId, "")

	if scimg == nil || (len(scimg.Path) == 0 && len(scimg.ExternalId) == 0) {
		// "image is not cached on this storage"
		uncacheTask.OnImageUncacheComplete(ctx, storageCache, nil)
		return
	}

	if isPurge {
		uncacheTask.OnImageUncacheComplete(ctx, obj, data)
		return
	}

	storage, err := models.StorageManager.GetStorageByStoragecache(storageCache.Id)
	if err != nil {
		uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "fail to get storage by storagecache"))
		return
	}
	if storage.IsNeedDeactivateOnAllHost() {
		uncacheTask.RequestUncacheDeactivateImage(ctx, storageCache)
	} else {
		uncacheTask.RequestUncacheRemoveImage(ctx, storageCache)
	}
}

func (uncacheTask *StorageUncacheImageTask) RequestUncacheDeactivateImage(ctx context.Context, storageCache *models.SStoragecache) {
	hosts, err := storageCache.GetHosts()
	if err != nil {
		uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "fail to get hosts"))
		return
	}
	for i := range hosts {
		if !hosts[i].Enabled.IsTrue() || hosts[i].HostStatus != compute.HOST_ONLINE {
			continue
		}
		driver, err := hosts[i].GetHostDriver()
		if err != nil {
			uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetHostDriver"))
			return
		}

		err = driver.RequestUncacheImage(ctx, &hosts[i], storageCache, uncacheTask, true)
		if err != nil {
			uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "RequestUncacheImage"))
			return
		}
	}

	uncacheTask.RequestUncacheRemoveImage(ctx, storageCache)
}

func (uncacheTask *StorageUncacheImageTask) RequestUncacheRemoveImage(ctx context.Context, storageCache *models.SStoragecache) {
	host, err := storageCache.GetMasterHost()
	if err != nil {
		uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetMasterHost"))
		return
	}

	driver, err := host.GetHostDriver()
	if err != nil {
		uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetHostDriver"))
		return
	}

	uncacheTask.SetStage("OnImageUncacheComplete", nil)
	err = driver.RequestUncacheImage(ctx, host, storageCache, uncacheTask, false)
	if err != nil {
		uncacheTask.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "RequestUncacheImage"))
		return
	}
}

func (uncacheTask *StorageUncacheImageTask) OnTaskFailed(ctx context.Context, storageCache *models.SStoragecache, reason error) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(reason.Error()), "reason")
	imageId, _ := uncacheTask.Params.GetString("image_id")
	body.Add(jsonutils.NewString(imageId), "image_id")
	body.Add(jsonutils.NewString(storageCache.Id), "storagecache_id")

	scimg := models.StoragecachedimageManager.Register(ctx, uncacheTask.UserCred, storageCache.Id, imageId, "")
	if scimg != nil {
		scimg.SetStatus(ctx, uncacheTask.UserCred, compute.CACHED_IMAGE_STATUS_DELETE_FAILED, reason.Error())
	}
	if cachedImage, _ := models.CachedimageManager.FetchById(imageId); cachedImage != nil {
		db.OpsLog.LogEvent(cachedImage, db.ACT_UNCACHE_IMAGE_FAIL, body, uncacheTask.UserCred)
		logclient.AddActionLogWithStartable(uncacheTask, cachedImage, logclient.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred, false)
	}
	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHE_IMAGE_FAIL, body, uncacheTask.UserCred)
	logclient.AddActionLogWithStartable(uncacheTask, storageCache, logclient.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred, false)

	uncacheTask.SetStageFailed(ctx, body)
}

func (uncacheTask *StorageUncacheImageTask) OnImageUncacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)

	uncacheTask.OnTaskFailed(ctx, storageCache, errors.Errorf("%s", data.String()))
}

func (uncacheTask *StorageUncacheImageTask) OnImageUncacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("Uncached image task success: %s", data)
	storageCache := obj.(*models.SStoragecache)

	imageId, _ := uncacheTask.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, uncacheTask.UserCred, storageCache.Id, imageId, "")
	if scimg != nil {
		scimg.Detach(ctx, uncacheTask.UserCred)
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(imageId), "image_id")
	body.Add(jsonutils.NewString(storageCache.Id), "storagecache_id")

	if cachedImage, _ := models.CachedimageManager.FetchById(imageId); cachedImage != nil {
		db.OpsLog.LogEvent(cachedImage, db.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred)
		logclient.AddActionLogWithStartable(uncacheTask, cachedImage, db.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred, true)
	}
	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred)
	logclient.AddActionLogWithStartable(uncacheTask, storageCache, db.ACT_UNCACHED_IMAGE, body, uncacheTask.UserCred, true)

	uncacheTask.SetStageComplete(ctx, nil)
}
