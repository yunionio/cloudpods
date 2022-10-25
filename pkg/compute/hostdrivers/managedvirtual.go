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
	input := api.CacheImageInput{}
	task.GetParams().Unmarshal(&input)
	image := &cloudprovider.SImageCreateOption{}
	task.GetParams().Unmarshal(&image)

	if len(image.ImageId) == 0 {
		return fmt.Errorf("no image_id params")
	}

	providerName := storageCache.GetProviderName()
	if utils.IsInStringArray(providerName, []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS, api.CLOUD_PROVIDER_UCLOUD}) {
		image.OsVersion = input.OsFullVersion
	}

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

		iStorageCache, err := storageCache.GetIStorageCache(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "storageCache.GetIStorageCache")
		}

		image.ExternalId = scimg.ExternalId
		if cloudprovider.TImageType(cachedImage.ImageType) == cloudprovider.ImageTypeCustomized {
			var guest *models.SGuest
			if len(input.ServerId) > 0 {
				server, _ := models.GuestManager.FetchById(input.ServerId)
				if server != nil {
					guest = server.(*models.SGuest)
				}
			}

			callback := func(progress float32) {
				guestInfo := ""
				if guest != nil {
					guest.SetProgress(progress)
					guestInfo = fmt.Sprintf(" for server %s ", guest.Name)
				}
				log.Infof("Upload image %s from storagecache %s%s status: %.2f%%", image.ImageName, storageCache.Name, guestInfo, progress)
			}

			image.ExternalId, err = func() (string, error) {
				if len(image.ExternalId) > 0 {
					log.Debugf("UploadImage: Image external ID exists %s", image.ExternalId)
					iImg, err := iStorageCache.GetIImageById(image.ExternalId)
					if err != nil {
						if errors.Cause(err) != cloudprovider.ErrNotFound {
							return "", errors.Wrapf(err, "GetIImageById(%s)", image.ExternalId)
						}
						return iStorageCache.UploadImage(ctx, userCred, image, callback)
					}
					if iImg.GetImageStatus() == cloudprovider.IMAGE_STATUS_ACTIVE && !input.IsForce {
						return image.ExternalId, nil
					}
					log.Debugf("UploadImage: %s status: %s is_force: %v", image.ExternalId, iImg.GetStatus(), input.IsForce)
					err = iImg.Delete(ctx)
					if err != nil {
						log.Warningf("delete image %s(%s) error: %v", iImg.GetName(), iImg.GetGlobalId(), err)
					}
					return iStorageCache.UploadImage(ctx, userCred, image, callback)
				}
				log.Debugf("UploadImage: no external ID")
				return iStorageCache.UploadImage(ctx, userCred, image, callback)
			}()
		} else {
			_, err = iStorageCache.GetIImageById(cachedImage.ExternalId)
			if err != nil {
				return nil, errors.Wrapf(err, "iStorageCache.GetIImageById(%s) for %s", cachedImage.ExternalId, iStorageCache.GetGlobalId())
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

		iStorageCache, err := storageCache.GetIStorageCache(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIStorageCache")
		}

		iImage, err := iStorageCache.GetIImageById(scimg.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iStorageCache.GetIImageById")
		}

		err = iImage.Delete(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "iImage.Delete")
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
	iDisk, err := disk.GetIDisk(ctx)
	if err != nil {
		return err
	}
	iStorage, err := disk.GetIStorage(ctx)
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
		if scimg.Status != api.CACHED_IMAGE_STATUS_ACTIVE {
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
		scimg.SetStatus(task.GetUserCred(), api.CACHED_IMAGE_STATUS_ACTIVE, "")
		return result, nil
	})
	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	iDisk, err := disk.GetIDisk(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetIDisk")
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err = iDisk.Resize(ctx, sizeMb)
		if err != nil {
			return nil, errors.Wrapf(err, "iDisk.Resize")
		}

		err = cloudprovider.WaitStatus(iDisk, api.DISK_READY, time.Second*5, time.Minute*3)
		if err != nil {
			return nil, errors.Wrapf(err, "Wait disk ready")
		}

		return jsonutils.Marshal(map[string]int64{"disk_size": sizeMb}), nil
	})

	return nil
}

func (self *SManagedVirtualizationHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	iCloudStorage, err := storage.GetIStorage(ctx)
	if err != nil {
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_cloudprovider := storage.GetCloudprovider()
		if _cloudprovider == nil {
			return nil, fmt.Errorf("invalid cloudprovider for storage %s(%s)", storage.Name, storage.Id)
		}
		projectId, err := _cloudprovider.SyncProject(ctx, userCred, disk.ProjectId)
		if err != nil {
			log.Errorf("failed to sync project for create %s disk %s error: %v", _cloudprovider.Provider, disk.GetName(), err)
		}
		conf := cloudprovider.DiskCreateConfig{
			Name:      disk.GetName(),
			SizeGb:    input.DiskSizeMb >> 10,
			ProjectId: projectId,
		}
		iDisk, err := iCloudStorage.CreateIDisk(&conf)
		if err != nil {
			return nil, err
		}
		err = db.SetExternalId(disk, task.GetUserCred(), iDisk.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		cloudprovider.WaitStatus(iDisk, api.DISK_READY, time.Second*5, time.Minute*5)

		models.SyncVirtualResourceMetadata(ctx, task.GetUserCred(), disk, iDisk)

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

	iCloudStorage, err := storage.GetIStorage(ctx)
	if err != nil {
		return err
	}

	iDisk, err := iCloudStorage.GetIDiskById(disk.GetExternalId())
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
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
	iDisk, err := disk.GetIDisk(ctx)
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

func (self *SManagedVirtualizationHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	iDisk, err := disk.GetIDisk(ctx)
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
