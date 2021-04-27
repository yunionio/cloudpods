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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type IImageCacheMangerFactory interface {
	NewImageCacheManager(manager *SStorageManager, cachePath string, storage IStorage, storagecacheId string) IImageCacheManger
	StorageType() string
}

var (
	imageCacheManagerFactories = make(map[string]IImageCacheMangerFactory)
)

func registerimageCacheManagerFactory(factory IImageCacheMangerFactory) {
	imageCacheManagerFactories[factory.StorageType()] = factory
}

func NewImageCacheManager(manager *SStorageManager, cachePath string, storage IStorage, storagecacheId string, storageType string) IImageCacheManger {
	if factory, ok := imageCacheManagerFactories[storageType]; ok {
		return factory.NewImageCacheManager(manager, cachePath, storage, storagecacheId)
	}
	log.Errorf("no image cache manager driver for %s found", storageType)
	return nil
}

type IImageCacheManger interface {
	GetId() string
	GetPath() string
	SetStoragecacheId(string)

	// for diskhandler
	PrefetchImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error)
	DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error)

	AcquireImage(ctx context.Context, imageId, zone, srcUrl, format, checksum string) IImageCache
	ReleaseImage(ctx context.Context, imageId string)
	LoadImageCache(imageId string)
}

type SBaseImageCacheManager struct {
	storageManager  IStorageManager
	storagecacaheId string
	cachePath       string
	cachedImages    map[string]IImageCache
}

func (c *SBaseImageCacheManager) GetPath() string {
	return c.cachePath
}

func (c *SBaseImageCacheManager) GetId() string {
	return c.storagecacaheId
}

func (c *SBaseImageCacheManager) SetStoragecacheId(scid string) {
	c.storagecacaheId = scid
}

func (c *SBaseImageCacheManager) GetStorageManager() IStorageManager {
	return c.storageManager
}
