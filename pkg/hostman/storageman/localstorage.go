package storageman

import (
	"os"
)

type SLocalStorage struct {
	*SBaseStorage
}

func NewLocalStorage(manager *SStorageManager, path string) *SLocalStorage {
	var ret = new(SLocalStorage)
	ret.SBaseStorage = NewBaseStorage(manager, path)
	ret.StartSnapshotRecycle()
	return ret
}

func (s *SLocalStorage) StorageType() string {
	return "local"
}

func (s *SLocalStorage) GetDiskById(diskId string) IDisk {
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			return s.Disks[i]
		}
	}
	return s.CreateDisk(diskId)
}

func (s *SLocalStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	var disk = NewLocalDisk(s, diskId)
	if disk.Probe() {
		s.Disks = append(s.Disks, disk)
		return disk
	}
	return nil
}

func (s *SLocalStorage) StartSnapshotRecycle() {
	//TODO
}

type SLocalDisk struct {
	*SBaseDisk
}

func NewLocalDisk(storage IStorage, id string) *SLocalDisk {
	var ret = new(SLocalDisk)
	ret.SBaseDisk = NewBaseDisk(storage, id)
	return ret
}

func (d *SLocalDisk) GetId() string {
	return d.Id
}

func (d *SLocalDisk) Probe() bool {
	if _, err := os.Stat(d.getPath()); !os.IsNotExist(err) {
		return true
	}
	// TODO alter ??
	return false
}
