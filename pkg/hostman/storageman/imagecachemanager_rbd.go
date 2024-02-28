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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/cephutils"
)

type SRbdImageCacheManager struct {
	SBaseImageCacheManager
	Pool, Prefix string
	storage      IStorage
}

func NewRbdImageCacheManager(manager IStorageManager, cachePath string, storage IStorage, storagecacheId string) *SRbdImageCacheManager {
	imageCacheManager := new(SRbdImageCacheManager)

	imageCacheManager.storageManager = manager
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
	imageCacheManager.loadCache(context.Background())
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

func (c *SRbdImageCacheManager) getCephClient() (*cephutils.CephClient, error) {
	storage := c.storage.(*SRbdStorage)
	return storage.getCephClient(c.Pool)
}

func (c *SRbdImageCacheManager) loadCache(ctx context.Context) {
	lockman.LockRawObject(ctx, "RBD", "image-cache")
	defer lockman.ReleaseRawObject(ctx, "RBD", "image-cache")

	cli, err := c.getCephClient()
	if err != nil {
		log.Errorf("getCephClient %s fail %s", c.storage.GetStorageName(), err)
		return
	}
	defer cli.Close()

	images, err := cli.ListImages()
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
	if imageCache.Load() == nil {
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

	input := api.CacheImageInput{}
	body.Unmarshal(&input)

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
		ret.Size = desc.Size
	}
	return jsonutils.Marshal(ret), nil
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
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages[imageId]; ok {
		delete(c.cachedImages, imageId)
		return img.Remove(ctx)
	}
	return nil
}

func (c *SRbdImageCacheManager) AcquireImage(ctx context.Context, input api.CacheImageInput, callback func(float64, float64, int64)) (IImageCache, error) {
	lockman.LockRawObject(ctx, "image-cache", input.ImageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", input.ImageId)

	img, ok := c.cachedImages[input.ImageId]
	if !ok {
		img = NewRbdImageCache(input.ImageId, c)
		c.cachedImages[input.ImageId] = img
	}

	return img, img.Acquire(ctx, input, callback)
}

func (c *SRbdImageCacheManager) ReleaseImage(ctx context.Context, imageId string) {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)
	if img, ok := c.cachedImages[imageId]; ok {
		img.Release()
	}
}
