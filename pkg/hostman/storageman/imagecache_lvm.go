package storageman

import (
	"context"
	"fmt"
	"path"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

type SLVMImageCache struct {
	imageId string
	cond    *sync.Cond
	Manager IImageCacheManger
}

func NewLVMImageCache(imageId string, imagecacheManager IImageCacheManger) *SLVMImageCache {
	imageCache := new(SLVMImageCache)
	imageCache.imageId = imageId
	imageCache.Manager = imagecacheManager
	imageCache.cond = sync.NewCond(new(sync.Mutex))
	return imageCache
}

func (c *SLVMImageCache) GetPath() string {
	return path.Join("/dev", c.Manager.GetPath(), IMAGECACHE_PREFIX+c.imageId)
}

func (c *SLVMImageCache) GetName() string {
	return IMAGECACHE_PREFIX + c.imageId
}

func (c *SLVMImageCache) GetDesc() *remotefile.SImageDesc {
	var sizeMb int64
	img, err := qemuimg.NewQemuImage(c.GetPath())
	if err != nil {
		log.Errorf("failed NewQemuImage for imagecache %s: %s", c.GetPath(), err)
	} else {
		sizeMb = img.SizeBytes / 1024 / 1024
	}

	return &remotefile.SImageDesc{
		Size: sizeMb,
		Name: c.GetName(),
	}
}

func (c *SLVMImageCache) Load() error {
	log.Debugf("loading lvm imagecache %s", c.GetPath())
	origin, err := qemuimg.NewQemuImage(c.GetPath())
	if err != nil {
		return errors.Wrap(err, "NewQemuImage")
	}
	if origin.IsValid() {
		return nil
	}
	return fmt.Errorf("invalid lvm image %s", origin.String())
}

func (c *SLVMImageCache) Acquire(
	ctx context.Context, input api.CacheImageInput,
	callback func(progress, progressMbps float64, totalSizeMb int64),
) error {
	input.ImageId = c.imageId
	localImageCache, err := storageManager.LocalStorageImagecacheManager.AcquireImage(ctx, input, func(progress, progressMbps float64, totalSizeMb int64) {
		if len(input.ServerId) > 0 {
			hostutils.UpdateServerProgress(context.Background(), input.ServerId, progress/1.2, progressMbps)
		}
	})
	if err != nil {
		return errors.Wrapf(err, "LocalStorage.AcquireImage")
	}
	if c.Load() != nil {
		localImg, err := qemuimg.NewQemuImage(localImageCache.GetPath())
		if err != nil {
			return errors.Wrapf(err, "NewQemuImage for local image path %s", localImageCache.GetPath())
		}
		err = lvmutils.LvCreate(c.Manager.GetPath(), c.GetName(), localImg.SizeBytes)
		if err != nil {
			return errors.Wrap(err, "lvm image cache acquire")
		}

		log.Infof("convert local image %s to lvm %s", c.imageId, c.GetPath())
		err = procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(),
			"convert", "-W", "-m", "16", "-O", "raw", localImageCache.GetPath(), c.GetPath()).Run()
		if err != nil {
			return errors.Wrapf(err, "convert local image %s to lvm %s", c.imageId, c.GetPath())
		}
		if len(input.ServerId) > 0 {
			modules.Servers.Update(hostutils.GetComputeSession(context.Background()), input.ServerId, jsonutils.Marshal(map[string]float32{"progress": 100.0}))
		}
	}
	return c.Load()
}

func (c *SLVMImageCache) GetImageId() string {
	return c.imageId
}

func (r *SLVMImageCache) Release() {
	return
}

func (c *SLVMImageCache) Remove(ctx context.Context) error {
	if err := lvmutils.LvRemove(c.GetPath()); err != nil {
		return errors.Wrap(err, "lvmImageCache Remove")
	}

	go func() {
		_, err := modules.Storagecachedimages.Detach(hostutils.GetComputeSession(ctx),
			c.Manager.GetId(), c.imageId, nil)
		if err != nil {
			log.Errorf("Fail to delete host cached image: %s", err)
		}
	}()
	return nil
}
