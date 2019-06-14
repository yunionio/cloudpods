package storageman

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type INasStorage interface {
	newDisk(diskId string) IDisk
}

type SNasStorage struct {
	SLocalStorage

	ins INasStorage
}

func NewNasStorage(manager *SStorageManager, path string, ins INasStorage) *SNasStorage {
	return &SNasStorage{*NewLocalStorage(manager, path, 0), ins}
}

func (s *SNasStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := s.ins.newDisk(diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SNasStorage) GetDiskById(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			if s.Disks[i].Probe() == nil {
				return s.Disks[i]
			} else {
				return nil
			}
		}
	}
	var disk = s.ins.newDisk(diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk
	} else {
		return nil
	}
}

func (s *SNasStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	if len(s.StorageId) == 0 {
		return nil, fmt.Errorf("Sync nfs storage without storage id")
	}
	content := jsonutils.NewDict()
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("zone", jsonutils.NewString(s.GetZone()))
	log.Infof("Sync storage info %s", s.StorageId)
	res, err := modules.Storages.Put(
		hostutils.GetComputeSession(context.Background()),
		s.StorageId, content)
	if err != nil {
		log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
	}
	return res, err
}
