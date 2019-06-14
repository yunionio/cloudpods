package storageman

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func init() {
	registerStorageFactory(&SGPFSStorageFactory{})
}

type SGPFSStorageFactory struct {
}

func (factory *SGPFSStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewNFSStorage(manager, mountPoint)
}

func (factory *SGPFSStorageFactory) StorageType() string {
	return api.STORAGE_GPFS
}

type SGPFSStorage struct {
	SNasStorage
}

func (s *SGPFSStorage) newDisk(diskId string) IDisk {
	return NewGPFSDisk(s, diskId)
}

func (s *SGPFSStorage) StorageType() string {
	return api.STORAGE_GPFS
}

func NewGPFSStorage(manager *SStorageManager, path string) *SGPFSStorage {
	ret := &SGPFSStorage{}
	ret.SNasStorage = *NewNasStorage(manager, path, ret)
	return ret
}
