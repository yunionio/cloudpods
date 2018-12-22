package storageman

import (
	"fmt"
	"os"
	"path"
	"strings"

	"yunion.io/x/onecloud/pkg/hostman/options"
)

const MINIMAL_FREE_SPACE = 128

type SStorageManager struct {
	storages                      []IStorage
	AgentStorage                  IStorage
	LocalStorageImagecacheManager IImageCacheManger
	AgentStorageImagecacheManager IImageCacheManger
	NfsStorageImagecacheMangers   []IImageCacheManger
}

func NewStorageManager() (*SStorageManager, error) {
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
		return nil, fmt.Errorf("Not enough storage space!")
	}
	if err := ret.initLocalStorageImagecache(); err != nil {
		return nil, fmt.Errorf("Init Local storage image cache failed: %s", err)
	}
	ret.initAgentStorageImagecache()
	if err := ret.initAgentStorage(); err != nil {
		return nil, fmt.Errorf("Init agent storage failed: %s", err)
	}
	return ret, nil
}

func (s *SStorageManager) getLeasedUsedLocalStorage(cacheDir string, limit int) (string, error) {
	var maxFree int
	var spath string
	var maxStorage IStorage
	for _, storage := range s.storages {
		if _, ok := storage.(*SLocalStorage); ok {
			cachePath := path.Join(storage.GetPath(), cacheDir)
			if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
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
	var cacheDir = "image_cache"
	cachePath := options.HostOptions.ImageCachePath
	limit := options.HostOptions.ImageCacheLimit
	if len(cachePath) == 0 {
		var err error
		cachePath, err = s.getLeasedUsedLocalStorage(cacheDir, limit)
		if err != nil {
			return err
		}
	}
	if len(cachePath) == 0 {
		s.LocalStorageImagecacheManager = NewLocalImageCacheManager(s, cachePath, limit, true, "")
		return nil
	} else {
		fmt.Errorf("Cannot allocate image cache storage")
	}
}

func (s *SStorageManager) initAgentStorageImagecache() {
	s.AgentStorageImagecacheManager = NewAgentImageCacheManager(s)
}

func (s *SStorageManager) initAgentStorage() error {
	var cacheDir = "agent_tmp"
	var spath = options.HostOptions.AgentTempPath
	var limit = options.HostOptions.AgentTempLimit
	if len(spath) == 0 {
		var err error
		spath, err = s.getLeasedUsedLocalStorage(cacheDir, limit)
		if err != nil {
			return err
		}
	}
	if len(spath) != nil {
		// TODO: NewAgentStorage
		s.AgentStorage = NewAgentStorage(s, spath)
	} else {
		return fmt.Errorf("Cannot allocate agent storage")
	}
}

func (s *SStorageManager) AddNfsStorage(storagecacheId, cachePath string) {
	if len(cachePath) == 0 {
		return
	}
	if s.NfsStorageImagecacheMangers == nil {
		s.NfsStorageImagecacheMangers = make(map[string]IImageCacheManger, 0)
	}
	s.NfsStorageImagecacheMangers[storagecacheId] = NewLocalImageCacheManager(s, cachePath,
		options.HostOptions.ImageCacheLimit, true, storagecacheId)
}

func (s *SStorageManager) GetStorageDisk(storageId, diskId string) IDisk {
	if storage, ok := s.storages[storageId]; ok {
		return storage.GetDiskById(diskId)
	}
	return nil
}

func (s *SStorageManager) GetStorageByPath(sPath string) IStorage {
	for _, storage := range s.storages {
		if storage.GetPath() == sPath {
			return storage
		}
	}
	return nil
}

func (s *SStorageManager) GetDiskByPath(diskPath string) IDisk {
	pos := strings.LastIndex(diskPath, "/")
	sPath := path[:pos]
	diskId := path[pos+1:]
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
