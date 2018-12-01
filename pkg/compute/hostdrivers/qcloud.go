package hostdrivers

import (
	"context"
	"fmt"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SQcloudHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SQcloudHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SQcloudHostDriver) GetHostType() string {
	return models.HOST_TYPE_QCLOUD
}

func (self *SQcloudHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	if sizeGb%10 != 0 {
		return fmt.Errorf("The disk size must be a multiple of 10Gb")
	}
	if storage.StorageType == models.STORAGE_CLOUD_BASIC {
		if sizeGb < 10 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 10 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_CLOUD_PREMIUM {
		if sizeGb < 50 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 50 ~ 16000GB", storage.StorageType)
		}
	} else if storage.StorageType == models.STORAGE_CLOUD_SSD {
		if sizeGb < 100 || sizeGb > 16000 {
			return fmt.Errorf("The %s disk size must be in the range of 100 ~ 16000GB", storage.StorageType)
		}
	} else {
		return fmt.Errorf("Not support create %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}

	osArch, _ := params.GetString("os_arch")
	osType, _ := params.GetString("os_type")
	osDist, _ := params.GetString("os_distribution")

	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lockman.LockRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))
		defer lockman.ReleaseRawObject(ctx, "cachedimages", fmt.Sprintf("%s-%s", storageCache.Id, imageId))

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), storageCache.Id, imageId)
		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, err
		}

		extImgId, err := iStorageCache.UploadImage(userCred, imageId, osArch, osType, osDist, scimg.ExternalId, isForce)
		if err != nil {
			return nil, err
		}
		scimg.SetExternalId(extImgId)

		ret := jsonutils.NewDict()
		ret.Add(jsonutils.NewString(extImgId), "image_id")
		return ret, nil
	})
	return nil
}

func (self *SQcloudHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	iCloudStorage, err := storage.GetIStorage()
	if err != nil {
		return err
	}
	size, err := content.Int("size")
	if err != nil {
		return err
	}
	size = size >> 10
	iDisk, err := iCloudStorage.CreateIDisk(disk.GetName(), int(size), "")
	if err != nil {
		return err
	}
	_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
		disk.ExternalId = iDisk.GetGlobalId()

		if metaData := iDisk.GetMetadata(); metaData != nil {
			meta := make(map[string]string)
			if err := metaData.Unmarshal(meta); err != nil {
				log.Errorf("Get disk %s Metadata error: %v", disk.Name, err)
			} else {
				for key, value := range meta {
					if err := disk.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
						log.Errorf("set disk %s mata %s => %s error: %v", disk.Name, key, value, err)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Errorf("Update disk externalId err: %v", err)
		return err
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewInt(int64(iDisk.GetDiskSizeMB())), "disk_size")
	data.Add(jsonutils.NewString(iDisk.GetDiskFormat()), "disk_format")
	task.ScheduleRun(data)
	return nil
}

func (self *SQcloudHostDriver) RequestDeallocateDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	data := jsonutils.NewDict()
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else if iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId()); err != nil {
		if err == cloudprovider.ErrNotFound {
			task.ScheduleRun(data)
			return nil
		}
		return err
	} else if err := iDisk.Delete(); err != nil {
		return err
	}
	task.ScheduleRun(data)
	return nil
}

func (self *SQcloudHostDriver) RequestResizeDiskOnHostOnline(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	return self.RequestResizeDiskOnHost(host, storage, disk, size, task)
}

func (self *SQcloudHostDriver) RequestResizeDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	iCloudStorage, err := storage.GetIStorage()
	if err != nil {
		return err
	}
	iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId())
	if err != nil {
		return err
	}
	err = iDisk.Resize(size >> 10)
	if err != nil {
		return err
	}
	task.ScheduleRun(jsonutils.Marshal(map[string]int64{"disk_size": size}))
	return nil
}

func (self *SQcloudHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SQcloudHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
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
		return httperrors.NewResourceNotFoundError("fail to find iStoragecache for storage: %s", iStorage.GetName())
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		snapshot, err := iDisk.CreateISnapshot(fmt.Sprintf("Snapshot-%s", imageId), "PrepareSaveImage")
		if err != nil {
			return nil, err
		}
		params := task.GetParams()
		osType, _ := params.GetString("properties", "os_type")

		scimg := models.StoragecachedimageManager.Register(ctx, task.GetUserCred(), iStoragecache.GetId(), imageId)
		if scimg.Status != models.CACHED_IMAGE_STATUS_READY {
			scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHING, "request_prepare_save_disk_on_host")
		}
		iImage, err := iStoragecache.CreateIImage(snapshot.GetId(), fmt.Sprintf("Image-%s", imageId), osType, "")
		if err != nil {
			log.Errorf("fail to create iImage: %v", err)
			scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
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
			scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_CACHE_FAILED, err.Error())
			return nil, err
		}
		if err := iImage.Delete(); err != nil {
			log.Errorf("Delete iImage %s failed: %v", iImage.GetId(), err)
		}
		if err := snapshot.Delete(); err != nil {
			log.Errorf("Delete snapshot %s failed: %v", snapshot.GetId(), err)
		}
		scimg.SetStatus(task.GetUserCred(), models.CACHED_IMAGE_STATUS_READY, "")
		return result, nil
	})
	return nil
}

func (self *SQcloudHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}
