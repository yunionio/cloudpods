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

package hostdrivers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SManagedVirtualizationHostDriver struct {
	SVirtualizationHostDriver
}

func (self *SManagedVirtualizationHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	image := &cloudprovider.SImageCreateOption{}
	err := params.Unmarshal(image)
	if err != nil {
		return err
	}

	if len(image.ImageId) == 0 {
		return fmt.Errorf("no image_id params")
	}

	providerName := storageCache.GetProviderName()
	if utils.IsInStringArray(providerName, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_UCLOUD}) {
		image.OsVersion, _ = params.GetString("os_full_version")
	}

	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		lockman.LockRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, image.ImageId))
		defer lockman.ReleaseRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, image.ImageId))

		log.Debugf("XXX Hold lockman key %p cachedimages %s-%s", ctx, storageCache.Id, image.ImageId)

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), storageCache.Id, image.ImageId, "")

		cachedImage := scimg.GetCachedimage()
		if cachedImage == nil {
			return nil, errors.Wrap(httperrors.ErrImageNotFound, "cached image not found???")
		}

		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, errors.Wrap(err, "storageCache.GetIStorageCache")
		}

		image.ExternalId = scimg.ExternalId
		if cachedImage.ImageType == cloudprovider.CachedImageTypeCustomized {
			image.ExternalId, err = iStorageCache.UploadImage(ctx, userCred, image, isForce)
			if err != nil {
				return nil, errors.Wrap(err, "iStorageCache.UploadImage")
			}
		} else {
			_, err = iStorageCache.GetIImageById(cachedImage.ExternalId)
			if err != nil {
				log.Errorf("remote image fetch error %s", err)
				return nil, errors.Wrap(err, "iStorageCache.GetIImageById")
			}
			image.ExternalId = cachedImage.ExternalId
		}

		// should record the externalId immediately
		// so the waiting goroutine could pick the new externalId
		// and avoid duplicate uploading
		scimg.SetExternalId(image.ExternalId)

		ret := jsonutils.NewDict()
		ret.Add(jsonutils.NewString(image.ExternalId), "image_id")
		return ret, nil
	})
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestUncacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}

	scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), storageCache.Id, imageId, "")
	if scimg == nil {
		task.ScheduleRun(nil)
		return nil
	}

	if len(scimg.ExternalId) == 0 {
		log.Errorf("cached image has not external ID???")
		task.ScheduleRun(nil)
		return nil
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lockman.LockRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))
		defer lockman.ReleaseRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))

		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			log.Errorf("GetIStorageCache fail %s", err)
			return nil, err
		}

		iImage, err := iStorageCache.GetIImageById(scimg.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			log.Errorf("GetIImageById fail %s", err)
			return nil, err
		}

		err = iImage.Delete(ctx)
		if err != nil {
			log.Errorf("iImage Delete fail %s", err)
			return nil, err
		}

		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	iDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	iStorage, err := disk.GetIStorage()
	if err != nil {
		return err
	}
	iStoragecache := iStorage.GetIStoragecache()
	if iStoragecache == nil {
		return fmt.Errorf("fail to find iStoragecache for storage: %s", iStorage.GetName())
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		snapshot, err := iDisk.CreateISnapshot(ctx, fmt.Sprintf("Snapshot-%s", imageId), "PrepareSaveImage")
		if err != nil {
			return nil, err
		}
		params := task.GetParams()
		osType, _ := params.GetString("properties", "os_type")

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), iStoragecache.GetId(), imageId, "")
		if scimg.Status != api.CACHED_IMAGE_STATUS_READY {
			scimg.SetStatus(task.GetUserCred(), api.CACHED_IMAGE_STATUS_CACHING, "request_prepare_save_disk_on_host")
		}
		iImage, err := iStoragecache.CreateIImage(snapshot.GetId(), fmt.Sprintf("Image-%s", imageId), osType, "")
		if err != nil {
			log.Errorf("fail to create iImage: %v", err)
			scimg.SetStatus(task.GetUserCred(), api.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
			return nil, err
		}
		scimg.SetExternalId(iImage.GetId())
		if _, err := os.Stat(options.Options.TempPath); os.IsNotExist(err) {
			if err = os.MkdirAll(options.Options.TempPath, 0755); err != nil {
				return nil, err
			}
		}
		result, err := iStoragecache.DownloadImage(task.GetUserCred(), imageId, iImage.GetId(), options.Options.TempPath)
		if err != nil {
			scimg.SetStatus(task.GetUserCred(), api.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
			return nil, err
		}
		if err := iImage.Delete(ctx); err != nil {
			log.Errorf("Delete iImage %s failed: %v", iImage.GetId(), err)
		}
		if err := snapshot.Delete(); err != nil {
			log.Errorf("Delete snapshot %s failed: %v", snapshot.GetId(), err)
		}
		scimg.SetStatus(task.GetUserCred(), api.CACHED_IMAGE_STATUS_READY, "")
		return result, nil
	})
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	iCloudStorage, err := storage.GetIStorage()
	if err != nil {
		log.Errorf("storage.GetIStorage fail %s", err)
		return err
	}

	iDisk, err := iCloudStorage.GetIDiskById(disk.GetExternalId())
	if err != nil {
		log.Errorf("iCloudStorage.GetIDisk fail %s", err)
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err = iDisk.Resize(ctx, sizeMb)
		if err != nil {
			log.Errorf("iDisk.Resize fail %s", err)
			return nil, err
		}
		return jsonutils.Marshal(map[string]int64{"disk_size": sizeMb}), nil
	})

	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	iCloudStorage, err := storage.GetIStorage()
	if err != nil {
		return err
	}
	size, err := content.Int("size")
	if err != nil {
		return err
	}
	size = size >> 10

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iDisk, err := iCloudStorage.CreateIDisk(disk.GetName(), int(size), "")
		if err != nil {
			return nil, err
		}
		err = db.SetExternalId(disk, task.GetUserCred(), iDisk.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		cloudprovider.WaitStatus(iDisk, api.DISK_READY, time.Second*5, time.Minute*5)

		models.SyncMetadata(ctx, task.GetUserCred(), disk, iDisk)

		data := jsonutils.NewDict()
		data.Add(jsonutils.NewInt(int64(iDisk.GetDiskSizeMB())), "disk_size")
		data.Add(jsonutils.NewString(iDisk.GetDiskFormat()), "disk_format")
		data.Add(jsonutils.NewString(iDisk.GetAccessPath()), "disk_path")

		return data, nil
	})

	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	data := jsonutils.NewDict()

	iCloudStorage, err := storage.GetIStorage()
	if err != nil {
		return err
	}

	iDisk, err := iCloudStorage.GetIDiskById(disk.GetExternalId())
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			task.ScheduleRun(data)
			return nil
		}
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err := iDisk.Delete(ctx)
		return nil, err
	})

	return nil
}

func (self *SManagedVirtualizationHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	iDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	snapshotId, err := params.GetString("snapshot_id")
	if err != nil {
		return err
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		exteranlDiskId, err := iDisk.Reset(ctx, snapshotId)
		data := jsonutils.NewDict()
		data.Set("exteranl_disk_id", jsonutils.NewString(exteranlDiskId))
		return data, err
	})
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	iDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err := iDisk.Rebuild(ctx)

		if err != nil {
			return nil, err
		}

		data := jsonutils.NewDict()
		data.Add(jsonutils.NewInt(int64(iDisk.GetDiskSizeMB())), "disk_size")
		data.Add(jsonutils.NewString(iDisk.GetDiskFormat()), "disk_format")
		data.Add(jsonutils.NewString(iDisk.GetAccessPath()), "disk_path")

		return data, nil
	})
	return nil
}

func (driver *SManagedVirtualizationHostDriver) IsReachStoragecacheCapacityLimit(host *models.SHost, cachedImages []models.SCachedimage) bool {
	quota := host.GetHostDriver().GetStoragecacheQuota(host)
	log.Debugf("Cached image total: %d quota: %d", len(cachedImages), quota)
	if quota > 0 && len(cachedImages) >= quota {
		return true
	}
	return false
}
