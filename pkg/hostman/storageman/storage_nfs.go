package storageman

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	registerStorageFactory(&SNFSStorageFactory{})
}

type SNFSStorageFactory struct {
}

func (factory *SNFSStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewNFSStorage(manager, mountPoint)
}

func (factory *SNFSStorageFactory) StorageType() string {
	return storagetypes.STORAGE_NFS
}

type SNFSStorage struct {
	SLocalStorage
}

func NewNFSStorage(manager *SStorageManager, path string) *SNFSStorage {
	ret := &SNFSStorage{}
	ret.SLocalStorage = *NewLocalStorage(manager, path, 0)
	if !fileutils2.Exists(path) {
		procutils.NewCommand("mkdir", "-p", path).Run()
	}
	return ret
}

func (s *SNFSStorage) StorageType() string {
	return storagetypes.STORAGE_NFS
}

func (s *SNFSStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewNFSDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SNFSStorage) GetDiskById(diskId string) IDisk {
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
	var disk = NewNFSDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk
	} else {
		return nil
	}
}

func (s *SNFSStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	if len(s.StorageId) == 0 {
		return nil, fmt.Errorf("Sync nfs storage without storage id")
	}

	content := jsonutils.NewDict()
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("status", jsonutils.NewString(models.STORAGE_ONLINE))
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

func (s *SNFSStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) {
	s.SLocalStorage.SetStorageInfo(storageId, storageName, conf)
	if err := s.checkAndMount(); err != nil {
		log.Errorf("Fail to mount storage to mountpoint: %s, %s", s.Path, err)
	}
}

func (s *SNFSStorage) checkAndMount() error {
	if _, err := procutils.NewCommand("mountpoint", s.Path).Run(); err == nil {
		return nil
	}
	if s.StorageConf == nil {
		return fmt.Errorf("Storage conf is nil")
	}
	host, err := s.StorageConf.GetString("nfs_host")
	if err != nil {
		return fmt.Errorf("Storage conf missing nfs_host ")
	}
	sharedDir, err := s.StorageConf.GetString("nfs_shared_dir")
	if err != nil {
		return fmt.Errorf("Storage conf missing nfs_shared_dir ")
	}
	output, err := procutils.NewCommand(
		"mount", "-t", "nfs", fmt.Sprintf("%s:%s", host, sharedDir), s.Path).RunWithTimeout(10 * time.Second)
	if err != nil {
		return fmt.Errorf("%s", output)
	}
	return nil
}
