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
	"fmt"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
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

func (r *SRbdImageCache) Load() error {
	log.Debugf("loading rbd imagecache %s", r.GetPath())
	origin, err := qemuimg.NewQemuImage(r.GetPath())
	if err != nil {
		return errors.Wrapf(err, "NewQemuImage at host %s", options.HostOptions.Hostname)
	}
	if origin.IsValid() {
		return nil
	}
	return fmt.Errorf("invalid rbd image %s at host %s", origin.String(), options.HostOptions.Hostname)
}

func (r *SRbdImageCache) Acquire(ctx context.Context, input api.CacheImageInput, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	input.ImageId = r.imageId
	localImageCache, err := storageManager.LocalStorageImagecacheManager.AcquireImage(ctx, input, func(progress, progressMbps float64, totalSizeMb int64) {
		if len(input.ServerId) > 0 {
			hostutils.UpdateServerProgress(context.Background(), input.ServerId, progress/1.2, progressMbps)
		}
	})
	if err != nil {
		return errors.Wrapf(err, "LocalStorage.AcquireImage")
	}
	r.imageName = localImageCache.GetName()
	if r.Load() != nil {
		log.Infof("convert local image %s to rbd pool %s", r.imageId, r.Manager.GetPath())
		err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(),
			"convert", "-W", "-m", "16", "-O", "raw", localImageCache.GetPath(), r.GetPath()).Run()
		if err != nil {
			return errors.Wrapf(err, "convert local image %s to rbd pool %s at host %s", r.imageId, r.Manager.GetPath(), options.HostOptions.Hostname)
		}
		if len(input.ServerId) > 0 {
			modules.Servers.Update(hostutils.GetComputeSession(context.Background()), input.ServerId, jsonutils.Marshal(map[string]float32{"progress": 100.0}))
		}
	}
	return r.Load()
}

func (r *SRbdImageCache) Release() {
	return
}

func (r *SRbdImageCache) Remove(ctx context.Context) error {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)

	cli, err := imageCacheManger.getCephClient()
	if err != nil {
		return errors.Wrap(err, "getCephClient")
	}

	img, err := cli.GetImage(r.GetName())
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetImage")
	}
	err = img.Delete()
	if err != nil {
		return errors.Wrap(err, "Delete")
	}

	go func() {
		_, err := modules.Storagecachedimages.Detach(hostutils.GetComputeSession(ctx),
			r.Manager.GetId(), r.imageId, nil)
		if err != nil && httputils.ErrorCode(err) != 404 {
			log.Errorf("Fail to delete host cached image: %s", err)
		}
	}()
	return nil
}

func (r *SRbdImageCache) GetDesc() *remotefile.SImageDesc {
	imageCacheManger := r.Manager.(*SRbdImageCacheManager)

	desc := &remotefile.SImageDesc{
		SizeMb: -1,
		Name:   r.imageName,
	}

	cli, err := imageCacheManger.getCephClient()
	if err != nil {
		log.Errorf("getCephClient fail %s", err)
		return desc
	}
	img, err := cli.GetImage(r.GetName())
	if err != nil {
		log.Errorf("getName fail %s", err)
		return desc
	}
	info, err := img.GetInfo()
	if err != nil {
		log.Errorf("GetInfo fail %s", err)
		return desc
	}
	desc.SizeMb = info.SizeByte / 1024 / 1024
	desc.AccessAt = info.AccessTimestamp

	return desc
}

func (r *SRbdImageCache) GetImageId() string {
	return r.imageId
}

func (r *SRbdImageCache) GetAccessDirectory() (string, error) {
	return "", errors.ErrNotImplemented
}
