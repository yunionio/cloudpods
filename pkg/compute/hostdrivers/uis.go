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
	"io"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

type SUisHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SUisHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SUisHostDriver) GetHostType() string {
	return api.HOST_TYPE_UIS
}

func (self *SUisHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_UIS
}

func (self *SUisHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_UIS
}

func (self *SUisHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (self *SUisHostDriver) CheckAndSetCacheImage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	input := api.CacheImageInput{}
	task.GetParams().Unmarshal(&input)
	opts := &cloudprovider.SImageCreateOption{}
	task.GetParams().Unmarshal(&opts)

	if len(input.ImageId) == 0 {
		return fmt.Errorf("no image_id params")
	}

	if input.Format != "iso" {
		return fmt.Errorf("invalid image format %s", input.Format)
	}

	imageSize := int64(0)

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		lockman.LockRawObject(ctx, models.CachedimageManager.Keyword(), fmt.Sprintf("%s-%s", storageCache.Id, input.ImageId))
		defer lockman.ReleaseRawObject(ctx, models.CachedimageManager.Keyword(), fmt.Sprintf("%s-%s", storageCache.Id, input.ImageId))

		log.Debugf("XXX Hold lockman key %p cachedimages %s-%s", ctx, storageCache.Id, input.ImageId)

		image, err := models.CachedimageManager.GetCachedimageById(ctx, userCred, input.ImageId, false)
		if err != nil {
			return nil, errors.Wrapf(err, "CachedimageManager.FetchById(%s)", input.ImageId)
		}

		if len(image.ExternalId) > 0 {
			storages, err := image.GetStorages()
			if err != nil {
				return nil, err
			}
			find := false
			for i := range storages {
				iStorage, _ := storages[i].GetIStorage(ctx)
				if gotypes.IsNil(iStorage) {
					continue
				}
				iCache := iStorage.GetIStoragecache()
				if gotypes.IsNil(iCache) {
					continue
				}
				iImage, err := iCache.GetIImageById(image.ExternalId)
				if err == nil {
					imageSize = iImage.GetSizeByte()
					find = true
					break
				}
			}
			if !find {
				return nil, errors.Wrapf(cloudprovider.ErrNotFound, image.ExternalId)
			}
			opts.ExternalId = image.ExternalId
		} else {
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
				log.Infof("Upload image %s from storagecache %s%s status: %.2f%%", opts.ImageName, storageCache.Name, guestInfo, progress)
			}

			storages, err := host.GetStorages()
			if err != nil {
				return nil, errors.Wrapf(err, "GetStorages")
			}

			opts.ExternalId, err = func() (string, error) {
				s := auth.GetAdminSession(ctx, options.Options.Region)
				info, err := modules.Images.Get(s, input.ImageId, nil)
				if err != nil {
					return "", errors.Wrapf(err, "Images.Get(%s)", input.ImageId)
				}
				opts.Description, _ = info.GetString("description")
				opts.Checksum, _ = info.GetString("checksum")
				minDiskMb, _ := info.Int("min_disk")
				opts.MinDiskMb = int(minDiskMb)
				minRamMb, _ := info.Int("min_ram")
				opts.MinRamMb = int(minRamMb)
				opts.TmpPath = options.Options.TempPath

				opts.GetReader = func(imageId, format string) (io.Reader, int64, error) {
					_, reader, sizeByte, err := modules.Images.Download(s, imageId, format, false)
					return reader, sizeByte, err
				}

				for i := range storages {
					cache := storages[i].GetStoragecache()
					iCache, _ := cache.GetIStorageCache(ctx)
					if gotypes.IsNil(iCache) {
						continue
					}

					ret, err := iCache.UploadImage(ctx, opts, callback)
					if err != nil {
						if errors.Cause(err) == cloudprovider.ErrNotSupported {
							continue
						}
						return "", errors.Wrapf(err, "UploadImage")
					}

					region, err := host.GetRegion()
					if err != nil {
						return ret, nil
					}

					obj, err := models.CachedimageManager.FetchById(input.ImageId)
					if err != nil {
						return ret, errors.Wrapf(err, "CachedimageManager.FetchById")
					}
					cachedImage := obj.(*models.SCachedimage)
					db.Update(cachedImage, func() error {
						cachedImage.ExternalId = ret
						return nil
					})

					cache.SyncCloudImages(ctx, userCred, iCache, region, true)
					return ret, nil
				}

				return "", fmt.Errorf("no valid storagecache for upload image")
			}()
			if err != nil {
				return nil, err
			}
			log.Infof("upload image %s id: %s", opts.ImageName, image.ExternalId)
		}

		ret := jsonutils.NewDict()
		ret.Add(jsonutils.NewString(opts.ExternalId), "image_id")
		ret.Add(jsonutils.NewInt(imageSize), "size")
		return ret, nil
	})
	return nil
}
