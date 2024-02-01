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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/storageutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/zeroclean"
)

const MINIMAL_FREE_SPACE = 128

type IStorageManager interface {
	GetZoneId() string
}

type SStorageManager struct {
	host hostutils.IHost

	Storages     []IStorage
	AgentStorage IStorage

	LocalStorageImagecacheManager IImageCacheManger
	// AgentStorageImagecacheManager IImageCacheManger

	LVMStorageImagecacheManagers        map[string]IImageCacheManger
	SharedLVMStorageImagecacheManagers  map[string]IImageCacheManger
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
		if err := s.Accessible(); err == nil {
			StartSnapshotRecycle(s)
			ret.Storages = append(ret.Storages, s)
			if allFull && s.GetFreeSizeMb() > MINIMAL_FREE_SPACE {
				allFull = false
			}
		} else {
			log.Errorf("storage %s not accessible error: %v", s.Path, err)
		}
	}

	for _, d := range options.HostOptions.SharedStorages {
		s := ret.NewSharedStorageInstance(d, "")
		if s != nil {
			ret.Storages = append(ret.Storages, s)
			allFull = false
		}
	}

	for _, d := range options.HostOptions.LVMVolumeGroups {
		s := NewLVMStorage(ret, d)
		if err := s.Accessible(); err == nil {
			ret.Storages = append(ret.Storages, s)
			if allFull && s.GetFreeSizeMb() > MINIMAL_FREE_SPACE {
				allFull = false
			}
		} else {
			log.Errorf("lvm storage %s not accessible error: %v", s.Path, err)
		}
	}

	for _, conf := range options.HostOptions.PTNVMEConfigs {
		diskConf := strings.Split(conf, "/")
		if len(diskConf) != 2 {
			return nil, fmt.Errorf("bad nvme config %s", conf)
		}
		var pciAddr, size = diskConf[0], diskConf[1]
		sizeMb, err := fileutils.GetSizeMb(size, 'M', 1024)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse pci device %s size %s", pciAddr, size)
		}
		ret.Storages = append(ret.Storages, newNVMEStorage(ret, pciAddr, sizeMb))
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
	storageType := storage.StorageType()
	if utils.IsInStringArray(storageType, api.SHARED_FILE_STORAGE) {
		delete(s.SharedFileStorageImagecacheManagers, storage.GetStoragecacheId())
	} else if storageType == api.STORAGE_RBD {
		delete(s.RbdStorageImagecacheManagers, storage.GetStoragecacheId())
	} else if storageType == api.STORAGE_LVM {
		delete(s.LVMStorageImagecacheManagers, storage.GetStoragecacheId())
	} else if storageType == api.STORAGE_CLVM || storageType == api.STORAGE_SLVM {
		delete(s.SharedLVMStorageImagecacheManagers, storage.GetStoragecacheId())
	}
	for index, iS := range s.Storages {
		if iS.GetId() == storage.GetId() {
			s.Storages = append(s.Storages[:index], s.Storages[index+1:]...)
			break
		}
	}
}

func (s *SStorageManager) GetZoneId() string {
	return s.host.GetZoneId()
}

func (s *SStorageManager) GetHostId() string {
	return s.host.GetHostId()
}

/*func (s *SStorageManager) GetMediumType() string {
	return s.host.GetMediumType()
}*/

func (s *SStorageManager) GetKubeletConfig() kubelet.KubeletConfig {
	return s.host.GetKubeletConfig()
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
		disk, _ := storage.GetDiskById(diskId)
		return disk
	}
	return nil
}

func (s *SStorageManager) GetStoragesByPath(sPath string) ([]IStorage, error) {
	ret := []IStorage{}
	for i := range s.Storages {
		if s.Storages[i].GetPath() == sPath {
			ret = append(ret, s.Storages[i])
		}
	}
	if len(ret) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, sPath)
	}
	return ret, nil
}

func (s *SStorageManager) GetStorageByPath(sPath string) (IStorage, error) {
	for _, storage := range s.Storages {
		if storage.GetPath() == sPath {
			return storage, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, sPath)
}

func (s *SStorageManager) GetDiskById(diskId string) (IDisk, error) {
	for _, storage := range s.Storages {
		disk, err := storage.GetDiskById(diskId)
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
		if err == nil {
			return disk, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, diskId)
}

func (s *SStorageManager) GetDiskByPath(diskPath string) (IDisk, error) {
	pos := strings.LastIndex(diskPath, "/")
	sPath := diskPath[:pos]
	diskId := diskPath[pos+1:]
	pos = strings.LastIndex(diskId, ".")
	if pos > 0 {
		diskId = diskId[:pos]
	}
	storages, err := s.GetStoragesByPath(sPath)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStoragesByPath")
	}
	for i := range storages {
		disk, err := storages[i].GetDiskById(diskId)
		if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
		if err == nil {
			return disk, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, diskId)
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
	if sc, ok := s.LVMStorageImagecacheManagers[scId]; ok {
		return sc
	}
	if sc, ok := s.SharedLVMStorageImagecacheManagers[scId]; ok {
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
		if rbdStorageCache := s.GetStoragecacheById(storagecacheId); rbdStorageCache == nil {
			s.AddRbdStorageImagecache(imagecachePath, storage, storagecacheId)
		}
	} else if storageType == api.STORAGE_CLVM || storageType == api.STORAGE_SLVM {
		if sharedLVMStorageCache := s.GetStoragecacheById(storagecacheId); sharedLVMStorageCache == nil {
			s.AddSharedLVMStorageImagecache(storage.GetPath(), storage, storagecacheId)
		}
	}
}

func (s *SStorageManager) InitLVMStorageImageCache(storagecacheId, vg string) {
	if len(storagecacheId) == 0 {
		return
	}
	if s.LVMStorageImagecacheManagers == nil {
		s.LVMStorageImagecacheManagers = map[string]IImageCacheManger{}
	}
	if _, ok := s.LVMStorageImagecacheManagers[storagecacheId]; !ok {
		s.LVMStorageImagecacheManagers[storagecacheId] = NewLVMImageCacheManager(s, vg, storagecacheId)
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

func (s *SStorageManager) AddSharedLVMStorageImagecache(imagecachePath string, storage IStorage, storagecacheId string) {
	if s.SharedLVMStorageImagecacheManagers == nil {
		s.SharedLVMStorageImagecacheManagers = map[string]IImageCacheManger{}
	}
	if _, ok := s.SharedLVMStorageImagecacheManagers[storagecacheId]; !ok {
		imagecache := NewLVMImageCacheManager(s, imagecachePath, storagecacheId)
		s.SharedLVMStorageImagecacheManagers[storagecacheId] = imagecache
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
			if options.HostOptions.ZeroCleanDiskData {
				// try to zero clean files in subdir
				err := zeroclean.ZeroDir(subDirPath)
				if err != nil {
					log.Errorf("zeroclean disk %s fail %s", subDirPath, err)
				} else {
					log.Debugf("zeroclean disk %s success!", subDirPath)
				}
			}
			if output, err := procutils.NewCommand("rm", "-rf", subDirPath).Output(); err != nil {
				log.Errorf("clean recycle dir %s error: %s, %s", subDirPath, err, output)
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

func GatherHostStorageStats() api.SHostPingInput {
	stats := api.SHostPingInput{}
	stats.RootPartitionUsedCapacityMb = GetRootPartUsedCapacity()
	manager := GetManager()
	for i := 0; i < len(manager.Storages); i++ {
		iS := manager.Storages[i]
		stat, err := iS.SyncStorageSize()
		if err != nil {
			log.Errorf("sync storage %s size failed: %s", iS.GetStorageName(), err)
		} else {
			stat.StorageId = iS.GetId()
			stats.StorageStats = append(stats.StorageStats, stat)
		}
	}
	return stats
}

// func StartSyncStorageSizeTask(interval time.Duration) {
//	log.Infof("Start sync storage size task !!!")
//	for {
//		time.Sleep(interval)
//		manager := GetManager()
//		for i := 0; i < len(manager.Storages); i++ {
//			iS := manager.Storages[i]
//			if iS.StorageType() == api.STORAGE_LOCAL || iS.StorageType() == api.STORAGE_RBD {
//				err := iS.SyncStorageSize()
//				if err != nil {
//					log.Errorf("sync storage %s size failed: %s", iS.GetStorageName(), err)
//				}
//			}
//		}
//		err := manager.host.SyncRootPartitionUsedCapacity()
//		if err != nil {
//			log.Errorf("sync root partition used size failed: %s", err)
//		}
//	}
// }

func GetRootPartTotalCapacity() int {
	size, err := storageutils.GetTotalSizeMb("/")
	if err != nil {
		log.Errorf("failed get path %s total size: %s", "/", err)
		return -1
	}
	return size
}

func GetRootPartUsedCapacity() int {
	size, err := storageutils.GetUsedSizeMb("/")
	if err != nil {
		log.Errorf("failed get path %s used size: %s", "/", err)
		return -1
	}
	return size
}
