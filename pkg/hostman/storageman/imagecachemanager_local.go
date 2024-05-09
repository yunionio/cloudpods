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
	"os"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLocalImageCacheManager struct {
	SBaseImageCacheManager
	// limit      int
	// isTemplate bool
	lock lockman.ILockManager

	storage IStorage
}

func NewLocalImageCacheManager(manager IStorageManager, cachePath string, storagecacheId string, storage IStorage) *SLocalImageCacheManager {
	imageCacheManager := new(SLocalImageCacheManager)
	imageCacheManager.lock = lockman.NewInMemoryLockManager()
	imageCacheManager.storageManager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.cachePath = cachePath
	imageCacheManager.storage = storage
	// imageCacheManager.limit = limit
	// imageCacheManager.isTemplate = isTemplete
	imageCacheManager.cachedImages = &sync.Map{} // make(map[string]IImageCache, 0)
	if !fileutils2.Exists(cachePath) {
		procutils.NewCommand("mkdir", "-p", cachePath).Run()
	}
	imageCacheManager.loadCache(context.Background())
	return imageCacheManager
}

func (c *SLocalImageCacheManager) IsLocal() bool {
	if c.storage != nil {
		return false
	}
	return true
}

func (c *SLocalImageCacheManager) loadCache(ctx context.Context) {
	if len(c.cachePath) == 0 {
		return
	}
	c.lock.LockRawObject(ctx, "LOCAL", "image-cache")
	defer c.lock.ReleaseRawObject(ctx, "LOCAL", "image-cache")
	files, _ := os.ReadDir(c.cachePath)
	for _, f := range files {
		if regutils.MatchUUIDExact(f.Name()) {
			c.LoadImageCache(f.Name())
		}
	}
}

func (c *SLocalImageCacheManager) LoadImageCache(imageId string) {
	imageCache := NewLocalImageCache(imageId, c)
	if imageCache.Load() == nil {
		c.cachedImages.Store(imageId, imageCache)
	}
}

func (c *SLocalImageCacheManager) AcquireImage(ctx context.Context, input api.CacheImageInput, callback func(progress, progressMbps float64, totalSizeMb int64)) (IImageCache, error) {
	c.lock.LockRawObject(ctx, "image-cache", input.ImageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", input.ImageId)

	imgObj, ok := c.cachedImages.Load(input.ImageId)
	if !ok {
		imgObj = NewLocalImageCache(input.ImageId, c)
		c.cachedImages.Store(input.ImageId, imgObj)
	}
	if callback == nil && len(input.ServerId) > 0 {
		callback = func(progress, progressMbps float64, totalSizeMb int64) {
			if len(input.ServerId) > 0 {
				hostutils.UpdateServerProgress(ctx, input.ServerId, progress, progressMbps)
			}
		}
	}
	img := imgObj.(IImageCache)
	return img, img.Acquire(ctx, input, callback)
}

func (c *SLocalImageCacheManager) ReleaseImage(ctx context.Context, imageId string) {
	c.lock.LockRawObject(ctx, "image-cache", imageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages.Load(imageId); ok {
		img.(IImageCache).Release()
	}
}

func (c *SLocalImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, _ := body.GetString("image_id")
	return nil, c.RemoveImage(ctx, imageId)
}

func (c *SLocalImageCacheManager) RemoveImage(ctx context.Context, imageId string) error {
	c.lock.LockRawObject(ctx, "image-cache", imageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages.Load(imageId); ok {
		c.cachedImages.Delete(imageId)
		return img.(IImageCache).Remove(ctx)
	}
	return nil
}

func (c *SLocalImageCacheManager) PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	input := api.CacheImageInput{}
	body.Unmarshal(&input)
	input.Zone = c.GetStorageManager().GetZoneId()

	if len(input.ImageId) == 0 {
		return nil, httperrors.NewMissingParameterError("image_id")
	}

	ret := struct {
		ImageId string
		Path    string
		Name    string
		Size    int64
	}{}

	imgCache, err := c.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imgCache.Release()

	ret.ImageId = input.ImageId
	ret.Path = imgCache.GetPath()

	if desc := imgCache.GetDesc(); desc != nil {
		ret.Name = desc.Name
		ret.Size = desc.SizeMb * 1024 * 1024 // ??? convert back to bytes?
	}
	if ret.Size == 0 {
		fi, err := os.Stat(imgCache.GetPath())
		if err != nil {
			log.Errorf("os.Stat(%s) error: %v", imgCache.GetPath(), err)
		} else {
			ret.Size = fi.Size()
		}
	}
	if len(ret.Name) == 0 {
		ret.Name = input.ImageId
	}

	return jsonutils.Marshal(ret), nil
}

func (c *SLocalImageCacheManager) getTotalSize(ctx context.Context) (int64, map[string]IImageCache) {
	total := int64(0)
	images := make(map[string]IImageCache)
	c.cachedImages.Range(func(imgId, imgObj any) bool {
		img := imgObj.(IImageCache)
		total += img.GetDesc().SizeMb
		images[imgId.(string)] = img
		return true
	})
	return total, images
}

func (c *SLocalImageCacheManager) CleanImageCachefiles(ctx context.Context) {
	totalSize, images := c.getTotalSize(ctx)
	storageSize := 0
	if c.storage != nil {
		// shared file storage
		storageSize = c.storage.GetCapacityMb()
	} else {
		storageSize = c.storageManager.(*SStorageManager).GetTotalLocalCapacity()
	}
	ratio := float64(totalSize) / float64(storageSize)
	log.Infof("SLocalImageCacheManager %s total size %dMB storage %dMB ratio %f expect ratio %d", c.cachePath, totalSize, storageSize, ratio, options.HostOptions.ImageCacheCleanupPercentage)
	if int(ratio*100) < options.HostOptions.ImageCacheCleanupPercentage {
		return
	}

	deletedMb, err := cleanImages(ctx, c, images)
	if err != nil {
		log.Errorf("SLocalImageCacheManager clean image %s fail %s", c.cachePath, err)
	} else {
		log.Infof("SLocalImageCacheManager %s cleanup %dMB", c.cachePath, deletedMb)
	}
}
