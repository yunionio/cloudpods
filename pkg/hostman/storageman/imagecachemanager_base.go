package storageman

import (
	"context"
	"sync"

	"yunion.io/x/jsonutils"
)

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
