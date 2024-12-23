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
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
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
	lvmlockd bool

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

func (s *SSLVMStorage) Lvmlockd() bool {
	return s.lvmlockd
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
	err := disk.Probe()
	if err == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	} else {
		log.Errorf("failed probe slvm disk %s: %s", diskId, err)
	}
	return nil, errors.ErrNotFound
}

func (s *SSLVMStorage) CreateDiskFromSnapshot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	snapshotLocation := disk.GetSnapshotPath(input.DiskInfo.SnapshotId)

	return disk.CreateFromSnapshotLocation(ctx, snapshotLocation, int64(input.DiskInfo.DiskSizeMb), &input.DiskInfo.EncryptInfo)
}

func (s *SSLVMStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SStorageDeleteSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if input.BlockStream {
		diskLvPath := path.Join("/dev", s.GetPath(), input.DiskId)
		err := lvmutils.LVActive(diskLvPath, false, s.Lvmlockd())
		if err != nil {
			return nil, errors.Wrap(err, "lvactive exclusive")
		}

		err = ConvertLVMDisk(s.GetPath(), input.DiskId, input.EncryptInfo)
		if err != nil {
			return nil, err
		}

	} else if len(input.ConvertSnapshot) > 0 {
		convertSnapshotName := "snap_" + input.ConvertSnapshot
		convertSnapshotPath := path.Join("/dev", s.GetPath(), convertSnapshotName)
		err := lvmutils.LVActive(convertSnapshotPath, false, s.Lvmlockd())
		if err != nil {
			return nil, errors.Wrap(err, "lvactive exclusive")
		}

		if err := ConvertLVMDisk(s.GetPath(), convertSnapshotName, input.EncryptInfo); err != nil {
			return nil, err
		}
	}

	snapName := "snap_" + input.SnapshotId
	snapId := path.Join("/dev", s.GetPath(), snapName)
	err := lvmutils.LvRemove(snapId)
	if err != nil {
		return nil, err
	}

	res := jsonutils.NewDict()
	res.Set("deleted", jsonutils.JSONTrue)
	return res, nil
}

func (s *SSLVMStorage) Accessible() error {
	if err := lvmutils.VgActive(s.Path, true); err != nil {
		log.Warningf("vgactive got %s", err)
	}

	if err := lvmutils.VgDisplay(s.Path); err != nil {
		return err
	}
	return nil
}

func (s *SSLVMStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		if jsonutils.QueryBoolean(dconf, "enabled_lvmlockd", false) {
			log.Infof("storage %s(%s) enabled lvmlockd", s.StorageId, s.StorageName)
			s.lvmlockd = true
		}
		s.StorageConf = dconf
	}

	if err := s.Accessible(); err != nil {
		return err
	}
	return nil
}

func (s *SSLVMStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	err := s.SLVMStorage.CreateDiskFromBackup(ctx, disk, input)
	if err != nil {
		return err
	}
	err = lvmutils.LVDeactivate(disk.GetPath())
	if err != nil {
		return errors.Wrap(err, "LVDeactivate")
	}
	return nil
}
