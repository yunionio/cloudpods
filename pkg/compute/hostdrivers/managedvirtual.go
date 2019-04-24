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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type SManagedVirtualizationHostDriver struct {
	SVirtualizationHostDriver
}

func (self *SManagedVirtualizationHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}

	osArch, _ := params.GetString("os_arch")
	osType, _ := params.GetString("os_type")
	osDist, _ := params.GetString("os_distribution")
	var osVersion string
	providerName := storageCache.GetProviderName()
	if providerName == api.CLOUD_PROVIDER_HUAWEI {
		osVersion, _ = params.GetString("os_full_version")
	} else {
		osVersion, _ = params.GetString("os_version")
	}

	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		lockman.LockRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))
		defer lockman.ReleaseRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), storageCache.Id, imageId, "")

		cachedImage := scimg.GetCachedimage()
		if cachedImage == nil {
			return nil, fmt.Errorf("cached image not found???")
		}

		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, err
		}

		var extImgId string
		if cachedImage.ImageType == cloudprovider.CachedImageTypeCustomized {
			extImgId, err = iStorageCache.UploadImage(ctx, userCred, imageId, osArch, osType, osDist, osVersion, scimg.ExternalId, isForce)
		} else {
			_, err = iStorageCache.GetIImageById(cachedImage.ExternalId)
			if err != nil {
				log.Errorf("remote image fetch error %s", err)
				return nil, err
			}
			extImgId = cachedImage.ExternalId
		}

		if err != nil {
			return nil, err
		}

		// scimg.SetExternalId(extImgId)

		ret := jsonutils.NewDict()
		ret.Add(jsonutils.NewString(extImgId), "image_id")
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

func (self *SManagedVirtualizationHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, guest *models.SGuest, sizeMb int64, task taskman.ITask) error {
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
		err = disk.SetExternalId(task.GetUserCred(), iDisk.GetGlobalId())
		if err != nil {
			log.Errorf("Update disk externalId err: %v", err)
			return nil, err
		}

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
	if quota > 0 && len(cachedImages) >= quota {
		return true
	}
	return false
}
