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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

func (l *SLocalImageCache) Load() bool {
	var (
		imgPath = l.GetPath()
		infPath = l.GetInfPath()
		desc    = &remotefile.SImageDesc{}
	)
	if fileutils2.Exists(imgPath) {
		if !fileutils2.Exists(infPath) {
			img, err := qemuimg.NewQemuImage(imgPath)
			if err != nil {
				log.Errorln(err)
				return false
			}
			if !img.IsValid() {
				return false
			}
			chksum, err := fileutils2.MD5(imgPath)
			if err != nil {
				log.Errorln(err)
				return false
			}
			desc = &remotefile.SImageDesc{
				Format: string(img.Format),
				Id:     l.imageId,
				Chksum: chksum,
				Path:   imgPath,
				Size:   l.GetSize(),
			}
			bdesc, err := json.Marshal(desc)
			if err != nil {
				log.Errorln(err)
				return false
			}
			err = fileutils2.FilePutContents(infPath, string(bdesc), false)
			if err != nil {
				log.Errorf("File put content error %s", err)
				return false
			}
		} else {
			sdesc, err := fileutils2.FileGetContents(infPath)
			if err != nil {
				log.Errorf("File get contents error %s", err)
				return false
			}
			err = json.Unmarshal([]byte(sdesc), desc)
			if err != nil {
				log.Errorf("Unmarshal desc %s error %s", sdesc, err)
				return false
			}
		}
		if len(desc.Chksum) > 0 && len(desc.Id) > 0 && desc.Id == l.imageId {
			l.Desc = desc
			return true
		}
	}

	tmpPath := l.GetTmpPath()
	if fileutils2.Exists(tmpPath) {
		syscall.Unlink(tmpPath)
	}
	return false
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

func (l *SLocalImageCache) Acquire(ctx context.Context, zone, srcUrl, format, preChksum string) bool {
	ret, exit := l.prepare(ctx, zone, srcUrl, format, preChksum)
	if exit {
		return ret
	}
	return l.fetch(ctx, zone, srcUrl, format)
}

func (l *SLocalImageCache) prepare(ctx context.Context, zone, srcUrl, format, preChksum string) (bool, bool) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	for l.remoteFile != nil {
		l.cond.Wait()
	}

	if l.remoteFile == nil && l.Desc != nil && (l.consumerCount > 0 || !l.needCheck()) {
		l.consumerCount++
		return true, true
	}
	url, err := auth.GetServiceURL(apis.SERVICE_TYPE_IMAGE, "", zone, "")
	if err != nil {
		log.Errorf("Failed to acquire image %s", err)
		return false, true
	}
	url += fmt.Sprintf("/images/%s", l.imageId)
	if len(format) == 0 {
		format = "qcow2"
	}
	url += fmt.Sprintf("?format=%s&scope=system", format)

	l.remoteFile = remotefile.NewRemoteFile(ctx, url,
		l.GetPath(), false, preChksum, -1, nil, l.GetTmpPath(), srcUrl)
	return false, false
}

func (l *SLocalImageCache) fetch(ctx context.Context, zone, srcUrl, format string) bool {
	if (fileutils2.Exists(l.GetPath()) && l.remoteFile.VerifyIntegrity()) ||
		l.remoteFile.Fetch() {
		if len(l.Manager.GetId()) > 0 {
			_, err := hostutils.RemoteStoragecacheCacheImage(ctx,
				l.Manager.GetId(), l.imageId, "ready", l.GetPath())
			if err != nil {
				log.Errorf("Fail to update host cached image: %s", err)
			}
		}
		l.cond.L.Lock()
		defer l.cond.L.Unlock()

		l.Desc = l.remoteFile.GetInfo()
		if l.Desc == nil {
			l.remoteFile = nil
			return false
		}
		l.Size = l.GetSize() / 1024 / 1024
		l.Desc.Id = l.imageId
		l.remoteFile = nil
		l.lastCheckTime = time.Now()
		l.consumerCount++
		l.cond.Broadcast()

		bDesc, err := json.Marshal(l.Desc)
		if err != nil {
			log.Errorf("Marshal image desc error %s", err)
			return false
		}

		err = fileutils2.FilePutContents(l.GetInfPath(), string(bDesc), false)
		if err != nil {
			log.Errorf("File put content error %s", err)
			return false
		}
		return true
	} else {
		l.cond.L.Lock()
		defer l.cond.L.Unlock()
		l.Desc = nil
		l.remoteFile = nil
		l.cond.Broadcast()
		return false
	}
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
