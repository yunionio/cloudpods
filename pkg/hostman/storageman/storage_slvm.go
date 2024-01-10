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
	registerStorageFactory(&SSLVMStorageFactory{})
}

type SSLVMStorageFactory struct {
}

func (factory *SSLVMStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewSLVMStorage(manager, mountPoint)
}

func (factory *SSLVMStorageFactory) StorageType() string {
	return api.STORAGE_SLVM
}

type SSLVMStorage struct {
	*SLVMStorage
}

func NewSLVMStorage(manager *SStorageManager, vgName string) *SSLVMStorage {
	var ret = new(SSLVMStorage)
	ret.SLVMStorage = NewLVMStorage(manager, vgName)
	return ret
}

func (s *SSLVMStorage) newDisk(diskId string) IDisk {
	return NewSLVMDisk(s, diskId)
}

func (s *SSLVMStorage) StorageType() string {
	return api.STORAGE_SLVM
}

func (s *SSLVMStorage) IsLocal() bool {
	return false
}

func (s *SSLVMStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewSLVMDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SSLVMStorage) GetDiskById(diskId string) (IDisk, error) {
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

	var disk = NewSLVMDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.ErrNotFound
}

func (s *SSLVMStorage) Accessible() error {
	if err := lvmutils.VgDisplay(s.Path); err != nil {
		return err
	}
	return nil
}

func (s *SSLVMStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
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
