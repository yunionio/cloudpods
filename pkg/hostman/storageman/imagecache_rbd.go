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

// +build linux,cgo

package storageman

import (
	"context"
	"fmt"
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

type SRbdImageCache struct {
	imageId   string
	imageName string
	cond      *sync.Cond
	Manager   IImageCacheManger
}

func NewRbdImageCache(imageId string, imagecacheManager IImageCacheManger) *SRbdImageCache {
	imageCache := new(SRbdImageCache)
	imageCache.imageId = imageId
	imageCache.Manager = imagecacheManager
	imageCache.cond = sync.NewCond(new(sync.Mutex))
	return imageCache
}

func (r *SRbdImageCache) GetName() string {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)
	return fmt.Sprintf("%s%s", imageCacheManger.Prefix, r.imageId)
}

func (r *SRbdImageCache) GetPath() string {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)
	storage := imageCacheManger.storage.(*SRbdStorage)
	return fmt.Sprintf("rbd:%s/%s%s", imageCacheManger.GetPath(), r.GetName(), storage.getStorageConfString())
}

func (r *SRbdImageCache) Load() bool {
	log.Debugf("loading rbd imagecache %s", r.GetPath())
	origin, err := qemuimg.NewQemuImage(r.GetPath())
	if err != nil {
		return false
	}
	return origin.IsValid()
}

func (r *SRbdImageCache) Acquire(ctx context.Context, zone, srcUrl, format, checksum string) bool {
	localImageCache := storageManager.LocalStorageImagecacheManager.AcquireImage(ctx, r.imageId, zone, srcUrl, format, checksum)
	if localImageCache == nil {
		log.Errorf("failed to acquireimage %s ", r.imageId)
		return false
	}
	r.imageName = localImageCache.GetName()
	if !r.Load() {
		log.Infof("convert local image %s to rbd pool %s", r.imageId, r.Manager.GetPath())
		err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(),
			"convert", "-O", "raw", localImageCache.GetPath(), r.GetPath()).Run()
		if err != nil {
			log.Errorf("failed to convert image %s", err)
			return false
		}
	}
	return r.Load()
}

func (r *SRbdImageCache) Release() {
	return
}

func (r *SRbdImageCache) Remove(ctx context.Context) error {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)
	storage := imageCacheManger.storage.(*SRbdStorage)
	if err := storage.deleteImage(r.Manager.GetPath(), r.GetName()); err != nil {
		return err
	}

	go func() {
		_, err := modules.Storagecachedimages.Detach(hostutils.GetComputeSession(ctx),
			r.Manager.GetId(), r.imageId, nil)
		if err != nil {
			log.Errorf("Fail to delete host cached image: %s", err)
		}
	}()
	return nil
}

func (r *SRbdImageCache) GetDesc() *remotefile.SImageDesc {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)
	storage := imageCacheManger.storage.(*SRbdStorage)

	size := storage.getImageSizeMb(imageCacheManger.Pool, r.GetName())
	return &remotefile.SImageDesc{
		Size: int64(size),
		Name: r.imageName,
	}
}

func (r *SRbdImageCache) GetImageId() string {
	return r.imageId
}
