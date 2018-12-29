package storageman

import "context"

type IImageCache interface {
	GetPath() string
	Load() error
	Acquire(context.Context, string, string) error
	Release()
	Remove() error
	GetImageId() string
}

type SImageCacheDesc struct {
	name   string
	format string
	id     string
	chksum string
	path   string
	size   int64
}

type SLocalImageCache struct {
	imageId string
	Manager IImageCacheManger
	Size    int64
	Desc    *SImageCacheDesc

	consumerCount int
}

func NewLocalImageCache(imageId string, imagecacheManager IImageCacheManger) *SLocalImageCache {
	imageCache := new(SLocalImageCache)
	imageCache.imageId = imageId
	imageCache.Manager = imagecacheManager
	return imageCache
}

func (l *SLocalImageCache) Load() error {
	// TODO
	return nil
}

type SRbdImageCache struct {
	imageId string
	Manager IImageCacheManger
}
