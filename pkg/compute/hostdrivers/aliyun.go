package hostdrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAliyunHostDriver struct {
}

func init() {
	driver := SAliyunHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAliyunHostDriver) GetHostType() string {
	return models.HOST_TYPE_ALIYUN
}

func (self *SAliyunHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, scimg *models.SStoragecachedimage, task taskman.ITask) error {
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

func (self *SAliyunHostDriver) RequestAllocateDiskOnStorage(host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
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

func (self *SAliyunHostDriver) RequestDeallocateDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
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

func (self *SAliyunHostDriver) RequestResizeDiskOnHostOnline(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
	return self.RequestResizeDiskOnHost(host, storage, disk, size, task)
}

func (self *SAliyunHostDriver) RequestResizeDiskOnHost(host *models.SHost, storage *models.SStorage, disk *models.SDisk, size int64, task taskman.ITask) error {
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
