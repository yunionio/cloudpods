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

func (self *StorageUncacheImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	// isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)
	isPurge := jsonutils.QueryBoolean(self.Params, "is_purge", false)

	storageCache := obj.(*models.SStoragecache)

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHING_IMAGE, imageId, self.UserCred)

	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId, "")

	if scimg == nil || (len(scimg.Path) == 0 && len(scimg.ExternalId) == 0) {
		// "image is not cached on this storage"
		self.OnImageUncacheComplete(ctx, storageCache, nil)
	}

	if isPurge {
		self.OnImageUncacheComplete(ctx, obj, data)
		return
	}

	storage, err := models.StorageManager.GetStorageByStoragecache(storageCache.Id)
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "fail to get storage by storagecache"))
		return
	}
	if storage.IsNeedDeactivateOnAllHost() {
		self.RequestUncacheDeactivateImage(ctx, storageCache)
		return
	}
	self.RequestUncacheRemoveImage(ctx, storageCache)
}

func (self *StorageUncacheImageTask) RequestUncacheDeactivateImage(ctx context.Context, storageCache *models.SStoragecache) {
	hosts, err := storageCache.GetHosts()
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "fail to get hosts"))
		return
	}
	for i := range hosts {
		if !hosts[i].Enabled.IsTrue() || hosts[i].HostStatus != compute.HOST_ONLINE {
			continue
		}
		driver, err := hosts[i].GetHostDriver()
		if err != nil {
			self.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetHostDriver"))
			return
		}

		err = driver.RequestUncacheImage(ctx, &hosts[i], storageCache, self, true)
		if err != nil {
			self.OnTaskFailed(ctx, storageCache, errors.Wrap(err, "RequestUncacheImage"))
			return
		}
	}

	self.RequestUncacheRemoveImage(ctx, storageCache)
}

func (self *StorageUncacheImageTask) RequestUncacheRemoveImage(ctx context.Context, storageCache *models.SStoragecache) {
	host, err := storageCache.GetMasterHost()
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetMasterHost"))
		return
	}

	driver, err := host.GetHostDriver()
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "GetHostDriver"))
		return
	}

	self.SetStage("OnImageUncacheComplete", nil)
	err = driver.RequestUncacheImage(ctx, host, storageCache, self, false)
	if err != nil {
		self.OnTaskFailed(ctx, storageCache, errors.Wrapf(err, "RequestUncacheImage"))
	}
}

func (self *StorageUncacheImageTask) OnTaskFailed(ctx context.Context, storageCache *models.SStoragecache, reason error) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(reason.Error()), "reason")
	imageId, _ := self.Params.GetString("image_id")
	body.Add(jsonutils.NewString(imageId), "image_id")

	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHE_IMAGE_FAIL, body, self.UserCred)

	logclient.AddActionLogWithStartable(self, storageCache, logclient.ACT_UNCACHED_IMAGE, body, self.UserCred, false)

	self.SetStageFailed(ctx, body)
}

func (self *StorageUncacheImageTask) OnImageUncacheCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storageCache := obj.(*models.SStoragecache)

	self.OnTaskFailed(ctx, storageCache, errors.Errorf(data.String()))
}

func (self *StorageUncacheImageTask) OnImageUncacheComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("Uncached image task success: %s", data)
	storageCache := obj.(*models.SStoragecache)

	imageId, _ := self.Params.GetString("image_id")
	scimg := models.StoragecachedimageManager.Register(ctx, self.UserCred, storageCache.Id, imageId, "")
	if scimg != nil {
		scimg.Detach(ctx, self.UserCred)
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(imageId), "image_id")
	db.OpsLog.LogEvent(storageCache, db.ACT_UNCACHED_IMAGE, body, self.UserCred)

	logclient.AddActionLogWithStartable(self, storageCache, db.ACT_UNCACHED_IMAGE, body, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
