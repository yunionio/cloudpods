package storageman

import (
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

/***************************************************************************/
/****************************StorageManager*********************************/
/***************************************************************************/

const MINIMAL_FREE_SPACE = 128

type SStorageManager struct {
	storages               []IStorage
	LocalStorageImagecache IStorageCache
}

func NewStorageManager() *SStorageManager {
	var ret = new(SStorageManager)
	ret.storages = make([]IStorage, 0)
	var allFull = true
	for _, d := range options.HostOptions.LocalImagePath {
		s := NewLocalStorage(ret, d)
		// TODO Accessible GetFreeSizeMb
		if s.Accessible() {
			ret.storages = append(ret.storages, s)
			if allFull && s.GetFreeSizeMb > MINIMAL_FREE_SPACE {
				allFull = false
			}
		}
	}
	for _, d := range options.HostOptions.SharedStorages {
		s := NewSharedStorage(ret, d)
		ret.storages = append(ret.storages, s)
		allFull = False
	}
	if allFull {
		log.Fatalf("Not enough storage space!")
	}
	ret.initLocalStorageImagecache()
	ret.initAgentStorageImagecache()
	ret.initAgentStorage()
	return ret
}

func (s *SStorageManager) GetStorageDisk(storageId, diskId string) IDisk {
	if storage, ok := s.storages[storageId]; ok {
		return storage.GetDiskById(diskId)
	}
	return nil
}

func (s *SStorageManager) initLocalStorageImagecache() {
	var cacheDir = "image_cache"
	cachePath := options.HostOptions.ImageCachePath
	limit := options.HostOptions.ImageCacheLimit
	if len(cachePath) == 0 {
		cachePath = s.getLeasedUsedLocalStorage(cacheDir, limit)
	}
	if len(cachePath) == 0 {
		// TODO NewLocalImageCacheManager
		s.LocalStorageImagecache = NewLocalImageCacheManager(s, cachePath, limit, true)
	} else {
		log.Fatalf("Cannot allocate image cache storage")
	}
}
