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
	"io/ioutil"
	"path"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const MINIMAL_FREE_SPACE = 128

type IStorageManager interface {
	GetZoneName() string
}

type SStorageManager struct {
	host hostutils.IHost

	Storages     []IStorage
	AgentStorage IStorage

	LocalStorageImagecacheManager IImageCacheManger
	// AgentStorageImagecacheManager IImageCacheManger

	RbdStorageImagecacheManagers        map[string]IImageCacheManger
	SharedFileStorageImagecacheManagers map[string]IImageCacheManger
}

func NewStorageManager(host hostutils.IHost) (*SStorageManager, error) {
	var (
		ret = &SStorageManager{
			host:     host,
			Storages: make([]IStorage, 0),
		}
		allFull = true
	)

	for i, d := range options.HostOptions.LocalImagePath {
		s := NewLocalStorage(ret, d, i)
		if s.Accessible() {
			StartSnapshotRecycle(s)
			ret.Storages = append(ret.Storages, s)
			if allFull && s.GetFreeSizeMb() > MINIMAL_FREE_SPACE {
				allFull = false
			}
		}
	}

	for _, d := range options.HostOptions.SharedStorages {
		s := ret.NewSharedStorageInstance(d, "")
		if s != nil {
			ret.Storages = append(ret.Storages, s)
			allFull = false
		}
	}

	if allFull {
		return nil, fmt.Errorf("Not enough storage space!")
	}

	if err := ret.initLocalStorageImagecache(); err != nil {
		return nil, fmt.Errorf("Init Local storage image cache failed: %s", err)
	}

	return ret, nil
}

func (s *SStorageManager) Remove(storage IStorage) {
	if utils.IsInStringArray(storage.StorageType(), api.SHARED_FILE_STORAGE) {
		delete(s.SharedFileStorageImagecacheManagers, storage.GetStoragecacheId())
	} else if storage.StorageType() == api.STORAGE_RBD {
		delete(s.RbdStorageImagecacheManagers, storage.GetStoragecacheId())
	}
	for index, iS := range s.Storages {
		if iS.GetId() == storage.GetId() {
			s.Storages = append(s.Storages[:index], s.Storages[index+1:]...)
			break
		}
	}
}

func (s *SStorageManager) GetZoneName() string {
	return s.host.GetZoneName()
}

func (s *SStorageManager) GetHostId() string {
	return s.host.GetHostId()
}

func (s *SStorageManager) GetMediumType() string {
	return s.host.GetMediumType()
}

func (s *SStorageManager) getLeasedUsedLocalStorage(cacheDir string, limit int) (string, error) {
	var (
		maxFree    int
		spath      string
		maxStorage IStorage
	)

	for _, storage := range s.Storages {

		if _, ok := storage.(*SLocalStorage); ok {
			cachePath := path.Join(storage.GetPath(), cacheDir)
			if fileutils2.Exists(cachePath) {
				spath = cachePath
				break
			}
			free := storage.GetFreeSizeMb()
			if maxFree < free {
				maxFree = free
				maxStorage = storage
			}
		}
	}

	if len(spath) == 0 {
		if maxFree >= limit*1024 {
			spath = path.Join(maxStorage.GetPath(), cacheDir)
		} else {
			return "", fmt.Errorf("No local storage has free space larger than %dGB", limit)
		}
	}
	return spath, nil
}

func (s *SStorageManager) initLocalStorageImagecache() error {
	var (
		cacheDir  = "image_cache"
		cachePath = options.HostOptions.ImageCachePath
		// limit     = options.HostOptions.ImageCacheLimit
	)

	if len(cachePath) == 0 {
		var err error
		cachePath, err = s.getLeasedUsedLocalStorage(cacheDir, 0)
		if err != nil {
			return err
		}
	}
	if len(cachePath) > 0 {
		s.LocalStorageImagecacheManager = NewLocalImageCacheManager(s, cachePath, "")
		return nil
	} else {
		return fmt.Errorf("Cannot allocate image cache storage")
	}
}

func (s *SStorageManager) GetStorage(storageId string) IStorage {
	for _, storage := range s.Storages {
		if storage.GetId() == storageId {
			return storage
		}
	}
	return nil
}

func (s *SStorageManager) GetStorageDisk(storageId, diskId string) IDisk {
	if storage := s.GetStorage(storageId); storage != nil {
		return storage.GetDiskById(diskId)
	}
	return nil
}

func (s *SStorageManager) GetStorageByPath(sPath string) IStorage {
	for _, storage := range s.Storages {
		if storage.GetPath() == sPath {
			return storage
		}
	}
	return nil
}

func (s *SStorageManager) GetDiskByPath(diskPath string) IDisk {
	pos := strings.LastIndex(diskPath, "/")
	sPath := diskPath[:pos]
	diskId := diskPath[pos+1:]
	pos = strings.LastIndex(diskId, ".")
	if pos > 0 {
		diskId = diskId[:pos]
	}
	storage := s.GetStorageByPath(sPath)
	if storage != nil {
		return storage.GetDiskById(diskId)
	}
	return nil
}

func (s *SStorageManager) GetTotalCapacity() int {
	var capa = 0
	for _, s := range s.Storages {
		capa += s.GetCapacity()
	}
	return capa
}

func (s *SStorageManager) GetStoragecacheById(scId string) IImageCacheManger {
	if s.LocalStorageImagecacheManager.GetId() == scId {
		return s.LocalStorageImagecacheManager
	}
	if sc, ok := s.SharedFileStorageImagecacheManagers[scId]; ok {
		return sc
	}
	if sc, ok := s.RbdStorageImagecacheManagers[scId]; ok {
		return sc
	}
	return nil
}

func (s *SStorageManager) NewSharedStorageInstance(mountPoint, storageType string) IStorage {
	return NewStorage(s, mountPoint, storageType)
}

func (s *SStorageManager) InitSharedStorageImageCache(storageType, storagecacheId, imagecachePath string, storage IStorage) {
	if utils.IsInStringArray(storageType, api.SHARED_FILE_STORAGE) {
		s.InitSharedFileStorageImagecache(storagecacheId, imagecachePath)
	} else if storageType == api.STORAGE_RBD {
		if rbdStorage := s.GetStoragecacheById(storagecacheId); rbdStorage == nil {
			s.AddRbdStorageImagecache(imagecachePath, storage, storagecacheId)
		}
	}
}

func (s *SStorageManager) InitSharedFileStorageImagecache(storagecacheId, path string) {
	if len(path) == 0 {
		return
	}
	if s.SharedFileStorageImagecacheManagers == nil {
		s.SharedFileStorageImagecacheManagers = map[string]IImageCacheManger{}
	}
	if _, ok := s.SharedFileStorageImagecacheManagers[storagecacheId]; !ok {
		s.SharedFileStorageImagecacheManagers[storagecacheId] = NewLocalImageCacheManager(s, path, storagecacheId)
	}
}

func (s *SStorageManager) AddRbdStorageImagecache(imagecachePath string, storage IStorage, storagecacheId string) {
	if s.RbdStorageImagecacheManagers == nil {
		s.RbdStorageImagecacheManagers = map[string]IImageCacheManger{}
	}
	if _, ok := s.RbdStorageImagecacheManagers[storagecacheId]; !ok {
		if imagecache := NewImageCacheManager(s, imagecachePath, storage, storagecacheId, api.STORAGE_RBD); imagecache != nil {
			s.RbdStorageImagecacheManagers[storagecacheId] = imagecache
			return
		}
		log.Errorf("failed init storagecache %s for storage %s", storagecacheId, storage.GetStorageName())
	}
}

var storageManager *SStorageManager

func GetManager() *SStorageManager {
	return storageManager
}

func Manager() *SStorageManager {
	return storageManager
}

func Init(host hostutils.IHost) error {
	lm := lockman.NewInMemoryLockManager()
	// lm := lockman.NewNoopLockManager()
	lockman.Init(lm)

	var err error
	storageManager, err = NewStorageManager(host)
	return err
}

func Stop() {
	// pass do nothing
}

func cleanDailyFiles(storagePath, subDir string, keepDay int) {
	recycleDir := path.Join(storagePath, subDir)
	if !fileutils2.Exists(recycleDir) {
		return
	}

	// before mark should be deleted
	markTime := timeutils.UtcNow().Add(time.Hour * 24 * -1 * time.Duration(keepDay))
	files, err := ioutil.ReadDir(recycleDir)
	if err != nil {
		log.Errorln(err)
		return
	}

	for _, file := range files {
		date, err := timeutils.ParseTimeStr(file.Name())
		if err != nil {
			log.Errorln(err)
			continue
		}
		if date.Before(markTime) {
			log.Infof("Cron Job Clean Recycle Bin: start delete %s", file.Name())
			subDirPath := path.Join(recycleDir, file.Name())
			if _, err := procutils.NewCommand("rm", "-rf", subDirPath).Run(); err != nil {
				log.Errorf("clean recycle dir %s error %s", subDirPath, err)
			}
		}
	}
}

func CleanRecycleDiskfiles(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	for _, d := range options.HostOptions.LocalImagePath {
		cleanDailyFiles(d, _RECYCLE_BIN_, options.HostOptions.RecycleDiskfileKeepDays)
		cleanDailyFiles(d, _IMGSAVE_BACKUPS_, options.HostOptions.RecycleDiskfileKeepDays)
	}
}
