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
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func init() {
	registerStorageFactory(&SGPFSStorageFactory{})
}

type SGPFSStorageFactory struct {
}

func (factory *SGPFSStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewGPFSStorage(manager, mountPoint)
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
