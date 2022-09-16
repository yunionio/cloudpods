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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLocalImageCacheManager struct {
	SBaseImageCacheManager
	// limit      int
	// isTemplate bool
	lock lockman.ILockManager
}

func NewLocalImageCacheManager(manager IStorageManager, cachePath string, storagecacheId string) *SLocalImageCacheManager {
	imageCacheManager := new(SLocalImageCacheManager)
	imageCacheManager.lock = lockman.NewInMemoryLockManager()
	imageCacheManager.storageManager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.cachePath = cachePath
	// imageCacheManager.limit = limit
	// imageCacheManager.isTemplate = isTemplete
	imageCacheManager.cachedImages = make(map[string]IImageCache, 0)
	if !fileutils2.Exists(cachePath) {
		procutils.NewCommand("mkdir", "-p", cachePath).Run()
	}
	imageCacheManager.loadCache(context.Background())
	return imageCacheManager
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
		c.cachedImages[imageId] = imageCache
	}
}

func (c *SLocalImageCacheManager) AcquireImage(ctx context.Context, input api.CacheImageInput, callback func(progress, progressMbps float64, totalSizeMb int64)) (IImageCache, error) {
	c.lock.LockRawObject(ctx, "image-cache", input.ImageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", input.ImageId)

	img, ok := c.cachedImages[input.ImageId]
	if !ok {
		img = NewLocalImageCache(input.ImageId, c)
		c.cachedImages[input.ImageId] = img
	}
	if callback == nil && len(input.ServerId) > 0 {
		callback = func(progress, progressMbps float64, totalSizeMb int64) {
			if len(input.ServerId) > 0 {
				hostutils.UpdateServerProgress(context.Background(), input.ServerId, progress, progressMbps)
			}
		}
	}
	return img, img.Acquire(ctx, input, callback)
}

func (c *SLocalImageCacheManager) ReleaseImage(ctx context.Context, imageId string) {
	c.lock.LockRawObject(ctx, "image-cache", imageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages[imageId]; ok {
		img.Release()
	}
}

func (c *SLocalImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, _ := body.GetString("image_id")
	return nil, c.removeImage(ctx, imageId)
}

func (c *SLocalImageCacheManager) removeImage(ctx context.Context, imageId string) error {
	c.lock.LockRawObject(ctx, "image-cache", imageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages[imageId]; ok {
		delete(c.cachedImages, imageId)
		return img.Remove(ctx)
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

	var (
		name string
		size int64
	)
	if desc := imgCache.GetDesc(); desc != nil {
		ret.Name = desc.Name
		ret.Size = desc.Size
	}
	if size == 0 {
		fi, err := os.Stat(imgCache.GetPath())
		if err != nil {
			log.Errorf("os.Stat(%s) error: %v", imgCache.GetPath(), err)
		} else {
			ret.Size = fi.Size()
		}
	}
	if len(name) == 0 {
		ret.Name = input.ImageId
	}

	return jsonutils.Marshal(ret), nil
}
