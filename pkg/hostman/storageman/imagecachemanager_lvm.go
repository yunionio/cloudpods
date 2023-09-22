package storageman

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const IMAGECACHE_PREFIX = "imagecache_"

type SLVMImageCacheManager struct {
	SBaseImageCacheManager

	lock lockman.ILockManager
}

func NewLVMImageCacheManager(manager IStorageManager, cachePath string, storagecacheId string) *SLVMImageCacheManager {
	imageCacheManager := new(SLVMImageCacheManager)
	imageCacheManager.lock = lockman.NewInMemoryLockManager()
	imageCacheManager.storageManager = manager
	imageCacheManager.storagecacaheId = storagecacheId
	imageCacheManager.cachePath = cachePath
	imageCacheManager.cachedImages = make(map[string]IImageCache, 0)

	imageCacheManager.loadCache(context.Background())
	return imageCacheManager
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
	imageCache := NewLocalImageCache(imageId, c)
	if imageCache.Load() == nil {
		c.cachedImages[imageId] = imageCache
	}
}

func (c *SLVMImageCacheManager) AcquireImage(
	ctx context.Context, input api.CacheImageInput,
	callback func(progress, progressMbps float64, totalSizeMb int64),
) (IImageCache, error) {
	c.lock.LockRawObject(ctx, "image-cache", input.ImageId)
	defer c.lock.ReleaseRawObject(ctx, "image-cache", input.ImageId)

	img, ok := c.cachedImages[input.ImageId]
	if !ok {
		img = NewLVMImageCache(input.ImageId, c)
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
		ret.Size = desc.Size
	}
	return jsonutils.Marshal(ret), nil
}

func (c *SLVMImageCacheManager) DeleteImageCache(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	body, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	imageId, _ := body.GetString("image_id")
	return nil, c.removeImage(ctx, imageId)
}

func (c *SLVMImageCacheManager) removeImage(ctx context.Context, imageId string) error {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)

	if img, ok := c.cachedImages[imageId]; ok {
		delete(c.cachedImages, imageId)
		return img.Remove(ctx)
	}
	return nil
}

func (c *SLVMImageCacheManager) ReleaseImage(ctx context.Context, imageId string) {
	lockman.LockRawObject(ctx, "image-cache", imageId)
	defer lockman.ReleaseRawObject(ctx, "image-cache", imageId)
	if img, ok := c.cachedImages[imageId]; ok {
		img.Release()
	}
}
