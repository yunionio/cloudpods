// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storageman

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
)

func init() {
	registerStorageFactory(&SCLVMStorageFactory{})
}

type SCLVMStorageFactory struct {
}

func (factory *SCLVMStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewCLVMStorage(manager, mountPoint)
}

func (factory *SCLVMStorageFactory) StorageType() string {
	return api.STORAGE_CLVM
}

type SCLVMStorage struct {
	*SLVMStorage
}

func NewCLVMStorage(manager *SStorageManager, vgName string) *SCLVMStorage {
	var ret = new(SCLVMStorage)
	ret.SLVMStorage = NewLVMStorage(manager, vgName)
	return ret
}

func (s *SCLVMStorage) newDisk(diskId string) IDisk {
	return NewCLVMDisk(s, diskId)
}

func (s *SCLVMStorage) StorageType() string {
	return api.STORAGE_CLVM
}

func (s *SCLVMStorage) IsLocal() bool {
	return false
}

func (s *SCLVMStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewCLVMDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SCLVMStorage) GetDiskById(diskId string) (IDisk, error) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			err := s.Disks[i].Probe()
			if err != nil {
				return nil, errors.Wrapf(err, "disk.Probe")
			}
			return s.Disks[i], nil
		}
	}

	var disk = NewCLVMDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.ErrNotFound
}

func (s *SCLVMStorage) Accessible() error {
	if err := lvmutils.VgDisplay(s.Path); err != nil {
		return err
	}
	return nil
}

func (s *SCLVMStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}

	if err := s.Accessible(); err != nil {
		return err
	}
	return nil
}
