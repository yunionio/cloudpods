package hostdrivers

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
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

func (self *SKVMHostDriver) CheckAndSetCacheImage(ctx context.Context, host *SHost, storageCache *SStoragecache, scimg *SStoragecachedimage, task taskman.ITask) error {
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	userCred := task.GetUserCred().GetTokenString()
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
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
