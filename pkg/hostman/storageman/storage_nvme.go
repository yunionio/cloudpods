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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SNVMEStorage struct {
	SBaseStorage

	sizeMB int
}

func (s *SNVMEStorage) GetAvailSizeMb() int {
	return s.sizeMB
}

func (s *SNVMEStorage) GetCapacityMb() int {
	return s.GetAvailSizeMb()
}

func (s *SNVMEStorage) SyncStorageSize() (api.SHostStorageStat, error) {
	stat := api.SHostStorageStat{
		StorageId: s.StorageId,
	}

	stat.CapacityMb = int64(s.GetAvailSizeMb())
	stat.ActualCapacityUsedMb = 0
	return stat, nil
}

func (s *SNVMEStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	name := s.GetName(s.GetComposedName)
	content.Set("name", jsonutils.NewString(name))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("status", jsonutils.NewString(api.STORAGE_ONLINE))
	content.Set("zone", jsonutils.NewString(s.GetZoneId()))
	if s.GetAvailSizeMb() > 0 {
		content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	}

	var (
		err error
		res jsonutils.JSONObject
	)

	log.Infof("Sync storage info %s/%s", s.StorageId, name)

	if len(s.StorageId) > 0 {
		res, err = modules.Storages.Put(
			hostutils.GetComputeSession(context.Background()),
			s.StorageId, content)
	} else {
		content.Set("medium_type", jsonutils.NewString(api.DISK_TYPE_SSD))
		res, err = modules.Storages.Create(
			hostutils.GetComputeSession(context.Background()), content)
	}
	if err != nil {
		log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
	}
	return res, err
}

func (s *SNVMEStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	return nil
}

func (s *SNVMEStorage) GetSnapshotDir() string {
	return ""
}

func (s *SNVMEStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
}

func (s *SNVMEStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	return false, errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) GetDiskById(diskId string) (IDisk, error) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			err := s.Disks[i].Probe()
			if err != nil {
				return nil, errors.Wrapf(err, "disk.Prob")
			}
			return s.Disks[i], nil
		}
	}
	var disk = NewNVMEDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.ErrNotFound
}

func (s *SNVMEStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewNVMEDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SNVMEStorage) SaveToGlance(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) CreateDiskFromSnapshot(context.Context, IDisk, *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) CreateDiskFromExistingPath(context.Context, IDisk, *SDiskCreateByDiskinfo) error {
	return errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return errors.Errorf("unsupported operation")
}

func (s *SNVMEStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SNVMEStorage) GetFuseMountPath() string {
	return ""
}

func (s *SNVMEStorage) GetImgsaveBackupPath() string {
	return ""
}

func (s *SNVMEStorage) Accessible() error {
	return nil
}

func (s *SNVMEStorage) Detach() error {
	return nil
}

func newNVMEStorage(manager *SStorageManager, path string, sizeMB int) *SNVMEStorage {
	var ret = new(SNVMEStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.sizeMB = sizeMB
	return ret
}

func (s *SNVMEStorage) StorageType() string {
	return api.STORAGE_NVME_PT
}

func (s *SNVMEStorage) IsLocal() bool {
	return true
}

func (s *SNVMEStorage) GetComposedName() string {
	p := strings.ReplaceAll(s.Path, ".", "_")
	p = strings.ReplaceAll(s.Path, ":", "_")
	return fmt.Sprintf("host_%s_%s_storage_%s", s.Manager.host.GetMasterIp(), s.StorageType(), p)
}

func (s *SNVMEStorage) CleanRecycleDiskfiles(ctx context.Context) {
	log.Infof("SNVMEStorage CleanRecycleDiskfiles do nothing!")
}
