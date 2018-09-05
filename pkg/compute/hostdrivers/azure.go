package hostdrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SAzureHostDriver struct {
}

func init() {
	driver := SAzureHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAzureHostDriver) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SAzureHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, scimg *models.SStoragecachedimage, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iStorageCache, err := storageCache.GetIStorageCache()
		if err != nil {
			return nil, err
		}

		extImgId, err := iStorageCache.UploadImage(userCred, imageId, scimg.ExternalId, isForce)

		if err != nil {
			return nil, err
		} else {
			ret := jsonutils.NewDict()
			ret.Add(jsonutils.NewString(extImgId), "image_id")
			return ret, nil
		}
	})
	return nil
}

func (self *SAzureHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else {
		if size, err := content.Int("size"); err != nil {
			return err
		} else {
			size = size >> 10
			if iDisk, err := iCloudStorage.CreateIDisk(disk.GetName(), int(size), ""); err != nil {
				return err
			} else {
				if _, err := disk.GetModelManager().TableSpec().Update(disk, func() error {
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
				}); err != nil {
					log.Errorf("Update disk externalId err: %v", err)
					return err
				}
				data := jsonutils.NewDict()
				data.Add(jsonutils.NewInt(int64(iDisk.GetDiskSizeMB())), "disk_size")
				data.Add(jsonutils.NewString(iDisk.GetDiskFormat()), "disk_format")
				task.ScheduleRun(data)
			}
		}
	}
	return nil
}

func (self *SAzureHostDriver) RequestDeallocateDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else if iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId()); err != nil {
		return err
	} else if err := iDisk.Delete(); err != nil {
		return err
	} else {
		data := jsonutils.NewDict()
		task.ScheduleRun(data)
	}
	return nil
}

func (self *SAzureHostDriver) RequestResizeDiskOnHostOnline(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	return self.RequestResizeDiskOnHost(host, storage, disk, size, task)
}

func (self *SAzureHostDriver) RequestResizeDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	if iCloudStorage, err := storage.GetIStorage(); err != nil {
		return err
	} else if iDisk, err := iCloudStorage.GetIDisk(disk.GetExternalId()); err != nil {
		return err
	} else if err := iDisk.Resize(size >> 10); err != nil {
		return err
	} else {
		task.ScheduleRun(jsonutils.Marshal(map[string]int64{"disk_size": size}))
	}
	return nil
}

func (self *SAzureHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SAzureHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	return httperrors.NewNotImplementedError("not implement")
}
