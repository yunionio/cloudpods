package storageman

import "yunion.io/x/log"

type SLocalStorage struct {
	SBaseStorage
}

func NewLocalStorage(manager *SStorageManager, path string) *SLocalStorage {
	var ret = new(SLocalStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.StartSnapshotRecycle()
	return ret
}

func (s *SLocalStorage) StorageType() string {
	return storagetypes.STORAGE_LOCAL
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
	if err := disk.Probe(); err != nil {
		s.Disks = append(s.Disks, disk)
		return disk
	} else {
		log.Errorln(err)
		return nil
	}
}

func (s *SLocalStorage) StartSnapshotRecycle() {
	//TODO
}
