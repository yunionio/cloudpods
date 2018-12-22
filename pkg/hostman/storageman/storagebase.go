package storageman

import (
	"sync"

	"yunion.io/x/jsonutils"
)

type IStorage interface {
	StorageType() string
	GetPath() string
	GetFreeSizeMb() int

	// Find owner disks first, if not found, call create disk
	GetDiskById(diskId string) IDisk
	CreateDisk(diskId string) IDisk
}

type SBaseStorage struct {
	Manager        *SStorageManager
	StorageId      string
	Path           string
	StorageName    string
	StorageConf    *jsonutils.JSONDict
	StoragecacheId string

	Disks    []IDisk
	DiskLock *sync.Mutex
}

func (s *SBaseStorage) GetPath() string {
	return s.Path
}

func NewBaseStorage(manager *SStorageManager, path string) *SBaseStorage {
	var ret = new(SBaseStorage)
	ret.Disks = make([]IDisk, 0)
	ret.DiskLock = new(sync.Mutex)
	ret.Manager = manager
	ret.Path = path
	return ret
}
