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
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLocalImageCacheManager struct {
	SBaseImageCacheManager
	limit      int
	isTemplate bool
}

func NewLocalImageCacheManager(manager *SStorageManager, cachePath string, limit int, isTemplete bool, storagecacheId string) *SLocalImageCacheManager {
	imageCacheManager := new(SLocalImageCacheManager)
	imageCacheManager.storagemanager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.cachePath = cachePath
	imageCacheManager.limit = limit
	imageCacheManager.isTemplate = isTemplete
	imageCacheManager.cachedImages = make(map[string]IImageCache, 0)
	imageCacheManager.mutex = new(sync.Mutex)
	if !fileutils2.Exists(cachePath) {
		procutils.NewCommand("mkdir", "-p", cachePath).Run()
	}
	imageCacheManager.loadCache()
	return imageCacheManager
}

func (c *SLocalImageCacheManager) loadCache() {
	if len(c.cachePath) == 0 {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	files, _ := ioutil.ReadDir(c.cachePath)
	for _, f := range files {
		if regutils.MatchUUIDExact(f.Name()) {
			c.LoadImageCache(f.Name())
		}
	}
}

func (c *SLocalImageCacheManager) LoadImageCache(imageId string) {
	imageCache := NewLocalImageCache(imageId, c)
	if imageCache.Load() {
		c.cachedImages[imageId] = imageCache
	}
}

func (c *SLocalImageCacheManager) AcquireImage(ctx context.Context, imageId, zone, srcUrl, format string) IImageCache {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	img, ok := c.cachedImages[imageId]
	if !ok {
		img = NewLocalImageCache(imageId, c)
		c.cachedImages[imageId] = img
	}
	if img.Acquire(ctx, zone, srcUrl, format) {
		return img
	} else {
		return nil
	}
}

func (c *SLocalImageCacheManager) ReleaseImage(imageId string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

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

	imageId, err := body.GetString("image_id")
	if err != nil {
		return nil, err
	}
	format, _ := body.GetString("format")
	srcUrl, _ := body.GetString("src_url")

	if imgCache := c.AcquireImage(ctx, imageId, storageManager.GetZone(),
		srcUrl, format); imgCache != nil {
		defer imgCache.Release()

		res := jsonutils.NewDict()
		res.Set("image_id", jsonutils.NewString(imageId))
		res.Set("path", jsonutils.NewString(imgCache.GetPath()))

		var (
			name string
			size int64
		)
		if desc := imgCache.GetDesc(); desc != nil {
			name = desc.Name
			size = desc.Size
		}
		if size == 0 {
			if fi, err := os.Stat(imgCache.GetPath()); err != nil {
				size = fi.Size()
			}
		}
		if len(name) == 0 {
			name = imageId
		}

		res.Set("name", jsonutils.NewString(name))
		res.Set("size", jsonutils.NewInt(size))
		return res, nil
	} else {
		return nil, fmt.Errorf("Failed to fetch image %s", imageId)
	}
}

// TODO: AgentImageCacheManager
