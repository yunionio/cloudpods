package storageman

import (
	"os/exec"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

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

func (s *SLocalStorage) Accessible() bool {
	if !fileutils2.Exists(s.Path) {
		if err := exec.Command("mkdir", "-p", s.Path).Run(); err != nil {
			log.Errorln(err)
		}
	}
	if fileutils2.IsDir(s.Path) && fileutils2.Writable(s.Path) {
		return true
	} else {
		return false
	}
}

func (s *SLocalStorage) GetFreeSizeMb() int {
	var stat syscall.Statfs_t
	err := syscall.Statfs(s.Path, &stat)
	if err != nil {
		log.Errorln(err)
		return -1
	}
	return int(stat.Bavail * uint64(stat.Bsize) / 1024 / 1024)
}
