package storageman

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"
)

type IImageCacheManger interface {
	// LoadCache() error ???
	PrefetchImageCache(ctx context.Context, data jsonutils.JSONObject) error
	DeleteImageCache(ctx context.Context, data jsonutils.JSONObject) error

	AcquireImage(ctx context.Context, imageId, zone, srcUrl string) IImageCache
	ReleaseImage(imageId string)

	GetPath() string
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
			c.loadImageCache(f.Name())
		}
	}
}

func (c *SLocalImageCacheManager) loadImageCache(file string) {
	imageCache := NewLocalImageCache(file, c)
	if err := imageCache.Load(); err != nil {
		c.cachedImages[imageCache.GetImageId()] = imageCache
	}
}

func DeleteImageCache(ctx context.Context, data jsonutils.JSONObject) error {

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
}
