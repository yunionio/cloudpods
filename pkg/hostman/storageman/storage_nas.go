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
	"context"
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type INasStorage interface {
	newDisk(diskId string) IDisk
	StorageType() string
}

type SNasStorage struct {
	SLocalStorage

	ins INasStorage
}

func NewNasStorage(manager *SStorageManager, path string, ins INasStorage) *SNasStorage {
	ret := &SNasStorage{*NewLocalStorage(manager, path, 0), ins}
	return ret
}

func (s *SNasStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%d", s.Manager.host.GetMasterIp(), s.ins.StorageType(), s.Index)
}

func (s *SNasStorage) GetSnapshotDir() string {
	return path.Join(s.Path, s.GetComposedName(), _SNAPSHOT_PATH_)
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
	content.Set("storage_type", jsonutils.NewString(s.ins.StorageType()))
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

func (s *SNasStorage) CreateDiskFromSnapshot(
	ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo,
) error {
	var (
		snapshotUrl, _ = createParams.DiskInfo.GetString("snapshot_url")
		diskSize, _    = createParams.DiskInfo.Int("size")
	)
	return disk.CreateFromSnapshotLocation(ctx, snapshotUrl, diskSize)
}
