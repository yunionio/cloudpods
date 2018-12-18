package storageman

import (
	"fmt"
	"path"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
)

type IStorage interface {
	StorageType() string
	GetPath() string

	// Find owner disks first, if not found, call create disk
	GetDiskById(diskId string) IDisk
	CreateDisk(diskId string) IDisk
}

type IDisk interface {
	GetId() string
	Probe() bool
	DeployGuestFs(guestDesc *jsonutils.JSONDict,
		deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error)
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

type SBaseDisk struct {
	Id      string
	Storage IStorage
}

func (d *SBaseDisk) getPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SBaseDisk) DeployGuestFs(
	guestDesc *jsonutils.JSONDict,
	deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	// TODO
	var kvmDisk = NewKVMGuestDisk(d.getPath())
	if kvmDisk.Connect() {
		defer kvmDisk.Disconnect()
		log.Infof("Kvm Disk Connect Success !!")
		if root := kvmDisk.Mount(); root != nil {
			defer kvmDisk.Umount(root)
			return root.DeployGuestFs(root, guestDesc, deployInfo)
		}
	}
	return nil, fmt.Errorf("Kvm disk connect or mount error")
}

func NewBaseDisk(storage IStorage, id string) *SBaseDisk {
	var ret = new(SBaseDisk)
	ret.Storage = storage
	ret.Id = id
	return ret
}

type SStorageManager struct {
	storages map[string]IStorage
}

func NewStorageManager() *SStorageManager {
	var ret = new(SStorageManager)
	// TODO
	ret.storages = make(map[string]IStorage, 0)

	return ret
}

func (m *SStorageManager) GetStorageDisk(storageId, diskId string) IDisk {
	if storage, ok := m.storages[storageId]; ok {
		return storage.GetDiskById(diskId)
	}
	return nil
}
