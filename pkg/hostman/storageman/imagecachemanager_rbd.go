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

// +build linux

package storageman

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SRbdImageCacheManager struct {
	SBaseImageCacheManager
	Pool, Prefix string
	storage      IStorage
}

func NewRbdImageCacheManager(manager *SStorageManager, cachePath string, storage IStorage, storagecacheId string) *SRbdImageCacheManager {
	imageCacheManager := new(SRbdImageCacheManager)

	imageCacheManager.storagemanager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.storage = storage

	// cachePath like `rbd:pool/imagecache` or `rbd:pool`
	cachePath = strings.TrimPrefix(cachePath, "rbd:")
	poolInfo := strings.Split(cachePath, "/")
	if len(poolInfo) == 2 {
		imageCacheManager.Pool, imageCacheManager.Prefix = poolInfo[0], poolInfo[1]
	} else {
		imageCacheManager.Pool, imageCacheManager.Prefix = cachePath, "image_cache_"
	}
	imageCacheManager.cachedImages = make(map[string]IImageCache, 0)
	imageCacheManager.mutex = new(sync.Mutex)
	imageCacheManager.loadCache()
	return imageCacheManager
}

type SRbdImageCacheManagerFactory struct {
}

func (factory *SRbdImageCacheManagerFactory) NewImageCacheManager(manager *SStorageManager, cachePath string, storage IStorage, storagecacheId string) IImageCacheManger {
	return NewRbdImageCacheManager(manager, cachePath, storage, storagecacheId)
}

func (factory *SRbdImageCacheManagerFactory) StorageType() string {
	return api.STORAGE_RBD
}

func init() {
	registerimageCacheManagerFactory(&SRbdImageCacheManagerFactory{})
}

func (c *SRbdImageCacheManager) loadCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	storage := c.storage.(*SRbdStorage)

	images, err := storage.listImages(c.Pool)
	if err != nil {
		log.Errorf("get storage %s images error; %v", c.storage.GetStorageName(), err)
		return
	}
	for _, image := range images {
		if strings.HasPrefix(image, c.Prefix) {
			imageId := strings.TrimPrefix(image, c.Prefix)
			c.LoadImageCache(imageId)
		} else {
			log.Debugf("find image %s from stroage %s", image, c.storage.GetStorageName())
		}
	}
}

func (c *SRbdImageCacheManager) LoadImageCache(imageId string) {
	imageCache := NewRbdImageCache(imageId, c)
	if imageCache.Load() {
		c.cachedImages[imageId] = imageCache
	}
}

func (c *SRbdImageCacheManager) GetPath() string {
	return c.Pool
}

func (c *SRbdImageCacheManager) PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, err := body.GetString("image_id")
	if err != nil {
		return nil, err
	}
	format, _ := body.GetString("format")
	srcUrl, _ := body.GetString("src_url")
	zone, _ := body.GetString("zone")

	cache := c.AcquireImage(ctx, imageId, zone, srcUrl, format)
	if cache == nil {
		return nil, fmt.Errorf("failed to cache image %s.%s", imageId, format)
	}

	res := map[string]interface{}{
		"image_id": imageId,
		"path":     cache.GetPath(),
	}
	if desc := cache.GetDesc(); desc != nil {
		res["name"] = desc.Name
		res["size"] = desc.Size
	}
	return jsonutils.Marshal(res), nil
}

func (c *SRbdImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, _ := body.GetString("image_id")
	return nil, c.removeImage(ctx, imageId)
}

func (c *SRbdImageCacheManager) removeImage(ctx context.Context, imageId string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if img, ok := c.cachedImages[imageId]; ok {
		delete(c.cachedImages, imageId)
		return img.Remove(ctx)
	}
	return nil
}

func (c *SRbdImageCacheManager) AcquireImage(ctx context.Context, imageId, zone, srcUrl, format string) IImageCache {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	img, ok := c.cachedImages[imageId]
	if !ok {
		img = NewRbdImageCache(imageId, c)
		c.cachedImages[imageId] = img
	}
	if img.Acquire(ctx, zone, srcUrl, format) {
		return img
	}
	return nil
}

func (c *SRbdImageCacheManager) ReleaseImage(imageId string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if img, ok := c.cachedImages[imageId]; ok {
		img.Release()
	}
}
