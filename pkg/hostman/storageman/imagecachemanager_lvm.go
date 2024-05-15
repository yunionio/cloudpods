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
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const IMAGECACHE_PREFIX = "imagecache_"

type SLVMImageCacheManager struct {
	SBaseImageCacheManager

	storage IStorage

	lvmlockd bool
	lock     lockman.ILockManager
}

func NewLVMImageCacheManager(manager IStorageManager, cachePath, storagecacheId string, storage IStorage, lvmlockd bool) *SLVMImageCacheManager {
	imageCacheManager := new(SLVMImageCacheManager)
	imageCacheManager.lock = lockman.NewInMemoryLockManager()
	imageCacheManager.storageManager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.cachePath = cachePath
	imageCacheManager.cachedImages = &sync.Map{} // make(map[string]IImageCache, 0)
	imageCacheManager.storage = storage
	imageCacheManager.lvmlockd = lvmlockd

	imageCacheManager.loadCache(context.Background())
	return imageCacheManager
}

func (c *SLVMImageCacheManager) IsLocal() bool {
	return c.storage.IsLocal()
}

func (c *SLVMImageCacheManager) Lvmlockd() bool {
	return c.lvmlockd
}

func (c *SLVMImageCacheManager) loadLvNames() []string {
	lvNames, err := lvmutils.GetLvNames(c.cachePath)
	if err != nil {
		log.Errorf("failed get lvm %s lvs %s", c.cachePath, err)
		return nil
	}
	return lvNames
}

func (c *SLVMImageCacheManager) loadCache(ctx context.Context) {
	if len(c.cachePath) == 0 { // cachePath is lvm vg
		return
	}
	c.lock.LockRawObject(ctx, "LOCAL", "image-cache")
	defer c.lock.ReleaseRawObject(ctx, "LOCAL", "image-cache")

	lvNames := c.loadLvNames()
	for _, f := range lvNames {
		if strings.HasPrefix(f, IMAGECACHE_PREFIX) && regutils.MatchUUIDExact(f[len(IMAGECACHE_PREFIX):]) {
			c.LoadImageCache(f[len(IMAGECACHE_PREFIX):])
		}
	}
}

func (c *SLVMImageCacheManager) LoadImageCache(imageId string) {
	imageCache := NewLVMImageCache(imageId, c)
	if err := imageCache.Load(); err == nil {
		c.cachedImages.Store(imageId, imageCache)
	} else {
		log.Errorf("failed load cache %s %s", c.GetPath(), err)
	}
}

func (c *SLVMImageCacheManager) AcquireImage(
	ctx context.Context, input api.CacheImageInput,
	callback func(progress, progressMbps float64, totalSizeMb int64),
) (IImageCache, error) {
	c.lock.LockRawObject(ctx, "image-cache", input.ImageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", input.ImageId)

	imgObj, ok := c.cachedImages.Load(input.ImageId)
	if !ok {
		imgObj = NewLVMImageCache(input.ImageId, c)
		c.cachedImages.Store(input.ImageId, imgObj)
	}
	if callback == nil && len(input.ServerId) > 0 {
		callback = func(progress, progressMbps float64, totalSizeMb int64) {
			if len(input.ServerId) > 0 {
				hostutils.UpdateServerProgress(context.Background(), input.ServerId, progress, progressMbps)
			}
		}
	}
	img := imgObj.(IImageCache)
	return img, img.Acquire(ctx, input, callback)
}

func (c *SLVMImageCacheManager) PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	input := api.CacheImageInput{}
	if err := body.Unmarshal(&input); err != nil {
		return nil, err
	}

	if len(input.ImageId) == 0 {
		return nil, httperrors.NewMissingParameterError("image_id")
	}

	cache, err := c.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	ret := struct {
		ImageId string
		Path    string
		Name    string
		Size    int64
	}{
		ImageId: input.ImageId,
		Path:    cache.GetPath(),
	}

	if desc := cache.GetDesc(); desc != nil {
		ret.Name = desc.Name
		ret.Size = desc.SizeMb
	}
	return jsonutils.Marshal(ret), nil
}

func (c *SLVMImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, _ := body.GetString("image_id")
	if jsonutils.QueryBoolean(body, "deactivate_image", false) {
		return nil, c.DeactiveImageCacahe(ctx, imageId)
	} else {
		return nil, c.RemoveImage(ctx, imageId)
	}
}

func (c *SLVMImageCacheManager) DeactiveImageCacahe(ctx context.Context, imageId string) error {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages.Load(imageId); ok {
		c.cachedImages.Delete(imageId)
		return lvmutils.LVDeactivate(img.(IImageCache).GetPath())
	}
	return nil
}

func (c *SLVMImageCacheManager) RemoveImage(ctx context.Context, imageId string) error {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages.Load(imageId); ok {
		c.cachedImages.Delete(imageId)
		return img.(IImageCache).Remove(ctx)
	}
	return nil
}

func (c *SLVMImageCacheManager) ReleaseImage(ctx context.Context, imageId string) {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)
	if img, ok := c.cachedImages.Load(imageId); ok {
		img.(IImageCache).Release()
	}
}

func (c *SLVMImageCacheManager) getTotalSize(ctx context.Context) (int64, map[string]IImageCache) {
	total := int64(0)
	images := make(map[string]IImageCache, 0)
	c.cachedImages.Range(func(imgId, imgObj any) bool {
		img := imgObj.(IImageCache)
		total += img.GetDesc().SizeMb
		images[imgId.(string)] = img
		return true
	})
	return total, images
}

func (c *SLVMImageCacheManager) CleanImageCachefiles(ctx context.Context) {
	totalSize, images := c.getTotalSize(ctx)
	ratio := float64(totalSize) / float64(c.storage.GetCapacityMb())
	log.Infof("SLVMImageCacheManager %s total size %dMB storage capacity %dMB ratio %f expect ratio %d", c.cachePath, totalSize, c.storage.GetCapacityMb(), ratio, options.HostOptions.ImageCacheCleanupPercentage)
	if int(ratio*100) < options.HostOptions.ImageCacheCleanupPercentage {
		return
	}

	deletedMb, err := cleanImages(ctx, c, images)
	if err != nil {
		log.Errorf("SLVMImageCacheManager clean image %s fail %s", c.cachePath, err)
	} else {
		log.Infof("SLVMImageCacheManager %s cleanup %dMB", c.cachePath, deletedMb)
	}
}
