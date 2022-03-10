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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const (
	_TMP_SUFFIX_ = ".tmp"
	_INF_SUFFIX_ = ".inf"

	CHECK_TIMEOUT = 3600 * time.Second
)

type SLocalImageCache struct {
	imageId string
	Manager IImageCacheManger
	Size    int64
	Desc    *remotefile.SImageDesc

	consumerCount int
	cond          *sync.Cond
	lastCheckTime time.Time

	remoteFile *remotefile.SRemoteFile
}

func NewLocalImageCache(imageId string, imagecacheManager IImageCacheManger) *SLocalImageCache {
	imageCache := new(SLocalImageCache)
	imageCache.imageId = imageId
	imageCache.Manager = imagecacheManager
	imageCache.cond = sync.NewCond(new(sync.Mutex))
	return imageCache
}

func (l *SLocalImageCache) GetDesc() *remotefile.SImageDesc {
	return l.Desc
}

func (l *SLocalImageCache) GetImageId() string {
	return l.imageId
}

func (l *SLocalImageCache) GetName() string {
	if l.Desc != nil && len(l.Desc.Name) > 0 {
		return l.Desc.Name
	}
	return l.imageId
}

func (l *SLocalImageCache) Load() error {
	var (
		imgPath = l.GetPath()
		infPath = l.GetInfPath()
		desc    = &remotefile.SImageDesc{}
	)
	if fileutils2.Exists(imgPath) {
		if !fileutils2.Exists(infPath) {
			img, err := qemuimg.NewQemuImage(imgPath)
			if err != nil {
				return errors.Wrapf(err, "NewQemuImage(%s)", imgPath)
			}
			if !img.IsValid() {
				return fmt.Errorf("invalid local image %s", img.String())
			}
			chksum, err := fileutils2.MD5(imgPath)
			if err != nil {
				return errors.Wrapf(err, "fileutils2.MD5(%s)", imgPath)
			}
			desc = &remotefile.SImageDesc{
				Format: string(img.Format),
				Id:     l.imageId,
				Chksum: chksum,
				Path:   imgPath,
				Size:   l.GetSize(),
			}
			err = fileutils2.FilePutContents(infPath, jsonutils.Marshal(desc).PrettyString(), false)
			if err != nil {
				return errors.Wrapf(err, "fileutils2.FilePutContents(%s)", infPath)
			}
		} else {
			sdesc, err := fileutils2.FileGetContents(infPath)
			if err != nil {
				return errors.Wrapf(err, "fileutils2.FileGetContents(%s)", infPath)
			}
			err = json.Unmarshal([]byte(sdesc), desc)
			if err != nil {
				return errors.Wrapf(err, "jsonutils.Unmarshal(%s)", infPath)
			}
		}
		if len(desc.Chksum) > 0 && len(desc.Id) > 0 && desc.Id == l.imageId {
			l.Desc = desc
			return nil
		}
	}

	tmpPath := l.GetTmpPath()
	if fileutils2.Exists(tmpPath) {
		syscall.Unlink(tmpPath)
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, imgPath)
}

func (l *SLocalImageCache) needCheck() bool {
	if time.Now().Sub(l.lastCheckTime) > CHECK_TIMEOUT {
		return true
	}
	return false
}

func (l *SLocalImageCache) Release() {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	l.consumerCount -= 1
}

func (l *SLocalImageCache) Acquire(ctx context.Context, input api.CacheImageInput, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	isOk, err := l.prepare(ctx, input)
	if err != nil {
		return errors.Wrapf(err, "prepare")
	}
	if isOk {
		return nil
	}
	return l.fetch(ctx, input, callback)
}

func (l *SLocalImageCache) prepare(ctx context.Context, input api.CacheImageInput) (bool, error) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	for l.remoteFile != nil {
		l.cond.Wait()
	}

	if l.remoteFile == nil && l.Desc != nil && (l.consumerCount > 0 || !l.needCheck()) {
		l.consumerCount++
		return true, nil
	}
	url, err := auth.GetServiceURL(apis.SERVICE_TYPE_IMAGE, "", input.Zone, "")
	if err != nil {
		return false, errors.Wrapf(err, "GetServiceURL(%s)", apis.SERVICE_TYPE_IMAGE)
	}
	url += fmt.Sprintf("/images/%s", l.imageId)
	if len(input.Format) == 0 {
		input.Format = "qcow2"
	}
	url += fmt.Sprintf("?format=%s&scope=system", input.Format)

	l.remoteFile = remotefile.NewRemoteFile(ctx, url,
		l.GetPath(), false, input.Checksum, -1, nil, l.GetTmpPath(), input.SrcUrl)
	return false, nil
}

func (l *SLocalImageCache) fetch(ctx context.Context, input api.CacheImageInput, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	// Whether successful or not, fetch should reset the condition variable and wakes up other waiters
	defer func() {
		l.cond.L.Lock()
		l.remoteFile = nil
		l.cond.Broadcast()
		l.cond.L.Unlock()
	}()
	var _fetch = func() error {
		if len(l.Manager.GetId()) > 0 {
			_, err := hostutils.RemoteStoragecacheCacheImage(ctx,
				l.Manager.GetId(), l.imageId, "active", l.GetPath())
			if err != nil {
				log.Errorf("Fail to update host cached image: %s", err)
			}
		}
		l.cond.L.Lock()
		defer l.cond.L.Unlock()

		var err error
		l.Desc, err = l.remoteFile.GetInfo()
		if err != nil {
			return errors.Wrapf(err, "remoteFile.GetInfo")
		}

		l.Size = l.GetSize() / 1024 / 1024
		l.Desc.Id = l.imageId
		l.lastCheckTime = time.Now()
		l.consumerCount++

		bDesc, err := json.Marshal(l.Desc)
		if err != nil {
			return errors.Wrapf(err, "json.Marshal(%#v)", l.Desc)
		}

		err = fileutils2.FilePutContents(l.GetInfPath(), string(bDesc), false)
		if err != nil {
			return errors.Wrapf(err, "FilePutContents(%s)", string(bDesc))
		}
		return nil
	}
	if fileutils2.Exists(l.GetPath()) && l.remoteFile.VerifyIntegrity(callback) == nil {
		return _fetch()
	}
	err := l.remoteFile.Fetch(callback)
	if err != nil {
		return errors.Wrapf(err, "remoteFile.Fetch")
	}
	return _fetch()
}

func (l *SLocalImageCache) Remove(ctx context.Context) error {
	if fileutils2.Exists(l.GetPath()) {
		if err := syscall.Unlink(l.GetPath()); err != nil {
			return err
		}
	}
	if fileutils2.Exists(l.GetInfPath()) {
		if err := syscall.Unlink(l.GetInfPath()); err != nil {
			return err
		}
	}
	if fileutils2.Exists(l.GetTmpPath()) {
		if err := syscall.Unlink(l.GetTmpPath()); err != nil {
			return err
		}
	}

	go func() {
		_, err := modules.Storagecachedimages.Detach(hostutils.GetComputeSession(ctx),
			l.Manager.GetId(), l.imageId, nil)
		if err != nil {
			log.Errorf("Fail to delete host cached image: %s", err)
		}
	}()

	return nil
}

func (l *SLocalImageCache) GetPath() string {
	return path.Join(l.Manager.GetPath(), l.imageId)
}

func (l *SLocalImageCache) GetTmpPath() string {
	return l.GetPath() + _TMP_SUFFIX_
}

func (l *SLocalImageCache) GetInfPath() string {
	return l.GetPath() + _INF_SUFFIX_
}

func (l *SLocalImageCache) GetSize() int64 {
	if fi, err := os.Stat(l.GetPath()); err != nil {
		log.Errorln(err)
		return 0
	} else {
		return fi.Size()
	}
}
