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

package storageman

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

func cleanImages(ctx context.Context, manager IImageCacheManger, images map[string]IImageCache) (int64, error) {
	if !manager.IsLocal() {
		return 0, nil
	}
	storageCachedImages := make(map[string]computeapis.StoragecachedimageDetails)
	limit := 50
	total := -1
	for total < 0 || len(storageCachedImages) < total {
		params := baseoptions.BaseListOptions{}
		details := true
		params.Details = &details
		params.Limit = &limit
		offset := len(storageCachedImages)
		params.Offset = &offset
		result, err := compute.Storagecachedimages.ListDescendent(hostutils.GetComputeSession(ctx), manager.GetId(), jsonutils.Marshal(params))
		if err != nil {
			return 0, errors.Wrap(err, "List Storage Cached Images")
		}
		total = result.Total
		for i := range result.Data {
			ci := computeapis.StoragecachedimageDetails{}
			err := result.Data[i].Unmarshal(&ci)
			if err != nil {
				return 0, errors.Wrap(err, "Unmarshal")
			}
			storageCachedImages[ci.CachedimageId] = ci
		}
	}

	var inUseCacheImageIds = make(map[string]struct{})
	for i := range storageManager.Storages {
		storage := storageManager.Storages[i]
		if storage.GetStoragecacheId() != manager.GetId() {
			continue
		}
		// load storage disks used image cache
		disksPath, err := storage.GetDisksPath()
		if err != nil {
			log.Errorf("storage %s failed get disksPath: %s", storage.GetPath(), err)
			continue
		}

		for j := range disksPath {
			diskPath := disksPath[j]
			img, err := qemuimg.NewQemuImage(diskPath)
			if err != nil {
				log.Errorf("failed NewQemuImage of %s", diskPath)
				continue
			}
			backingChain, err := img.GetBackingChain()
			if err != nil {
				log.Errorf("disk %s failed get backing chain", diskPath)
				continue
			}
			for _, backingPath := range backingChain {
				if strings.HasPrefix(backingPath, manager.GetPath()) {
					imageId := strings.Trim(strings.TrimPrefix(backingPath, manager.GetPath()), "/")
					inUseCacheImageIds[imageId] = struct{}{}
				}
			}
		}
	}
	log.Infof("found image caches in use: %v", inUseCacheImageIds)

	deleteSizeMb := int64(0)
	for imageId, image := range images {
		if _, ok := storageCachedImages[imageId]; !ok {
			atime := image.GetDesc().AccessAt
			if !atime.IsZero() && time.Now().Sub(atime) > time.Duration(options.HostOptions.ImageCacheExpireDays*86400)*time.Second {
				continue
			}
			if _, ok := inUseCacheImageIds[imageId]; ok {
				log.Infof("cached image not found but referenced by disks backing file")
				continue
			}

			log.Infof("cached image %s not found on region, to delete size %dMB ...", imageId, image.GetDesc().SizeMb)
			// not found on region, clean directly
			if options.HostOptions.ImageCacheCleanupDryRun {
				continue
			}
			err := manager.RemoveImage(ctx, imageId)
			if err != nil {
				return deleteSizeMb, errors.Wrapf(err, "RemoveImage %s", imageId)
			}
			deleteSizeMb += image.GetDesc().SizeMb
		}
	}
	log.Infof("to delete non-exist image caches %dMB", deleteSizeMb)

	for imgId := range storageCachedImages {
		if _, ok := images[imgId]; !ok {
			log.Infof("cached image %s in database not exists locally, to delete remotely ...", imgId)
			_, err := modules.Storagecachedimages.Detach(hostutils.GetComputeSession(ctx), manager.GetId(), imgId, nil)
			if err != nil {
				log.Errorf("Fail to delete host cached image %s at %s: %s", imgId, manager.GetId(), err)
			}
			continue
		}
		img := storageCachedImages[imgId]
		if img.Reference == 0 && (img.Size == 0 || time.Now().Sub(img.UpdatedAt) > time.Duration(options.HostOptions.ImageCacheExpireDays*86400)*time.Second) {
			if img.Size == 0 {
				img.Size = images[imgId].GetDesc().SizeMb * 1024 * 1024
			}
			if _, ok := inUseCacheImageIds[imgId]; ok {
				log.Infof("cached image database reference zero but referenced by disks locally")
				continue
			}

			log.Infof("image reference zero, to delete %s(%s) size %dMB", img.Cachedimage, img.CachedimageId, img.Size/1024/1024)
			if options.HostOptions.ImageCacheCleanupDryRun {
				continue
			}
			err := manager.RemoveImage(ctx, imgId)
			if err != nil {
				return deleteSizeMb, errors.Wrapf(err, "RemoveImage %s", imgId)
			}
			deleteSizeMb += img.Size / 1024 / 1024
		}
	}

	return deleteSizeMb, nil
}
