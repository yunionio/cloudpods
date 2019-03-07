package storageman

import (
	"context"
	"sync"

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

	AcquireImage(ctx context.Context, imageId, zone, srcUrl, format string) IImageCache
	ReleaseImage(imageId string)
	LoadImageCache(imageId string)
}

type SBaseImageCacheManager struct {
	storagemanager  *SStorageManager
	storagecacaheId string
	cachePath       string
	cachedImages    map[string]IImageCache
	mutex           *sync.Mutex
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
