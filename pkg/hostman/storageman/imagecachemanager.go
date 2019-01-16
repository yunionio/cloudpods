package storageman

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/pkg/util/regutils"
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
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		exec.Command("mkdir", "-p", cachePath).Run()
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
		c.cachedImages[imageId] = NewLocalImageCache(imageId, c)
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
type SAgentImageCacheManager struct {
	storagemanager *SStorageManager
}

func NewAgentImageCacheManager(storagemanager *SStorageManager) *SAgentImageCacheManager {
	return &SAgentImageCacheManager{storagemanager}
}

type SRbdImageCacheManager struct {
	SBaseImageCacheManager
	pool, prefix string
	storage      IStorage
}

func NewRbdImageCacheManager() *SRbdImageCacheManager {
	// TODO
	return nil
}
