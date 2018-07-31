package hostdrivers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/pkg/util/httputils"
)

type SKVMHostDriver struct {
}

func init() {
	driver := SKVMHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SKVMHostDriver) GetHostType() string {
	return models.HOST_TYPE_HYPERVISOR
}

func (self *SKVMHostDriver) CheckAndSetCacheImage(ctx context.Context, host *SHost, storageCache *SStoragecache, scimg *SStoragecachedimage, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	cacheImage, err := models.CachedimageManager.FetchById(imageId)
	if err != nil {
		return err
	}
	srcHostCacheImage, err := cacheImage.ChooseSourceStoragecacheInRange(models.HOST_TYPE_HYPERVISOR, []string{host.Id}, []*SZone{host.GetZone()})
	if err != nil {
		return err
	}

	content := jsonutils.NewDict()
	content.Add(jsonutils.NewString(imageId), "image_id")
	if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}
		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return err
		}
		srcUrl := fmt.Sprintf("%s/download/images/%s", srcHost.ManagerUri, imageId)
		content.Add(jsonutils.NewString(srcUrl), "src_url")
	}
	url := "/disks/image_cache"

	if isForce {
		content.Add(jsonutils.NewBool(true), "is_force")
	}
	content.Add(jsonutils.NewString(storageCache.Id), "storagecache_id")
	body := jsonutils.NewDict()
	body.Add(content, "disk")
	header := http.Header{}
	header.Set("X-Auth-Token", task.GetUserCred().GetTokenString())
	header.Set("X-Task-Id", task.GetTaskId())
	header.Set("X-Region-Version", "v2")
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	if err != nil {
		return err
	}
	return nil
}
