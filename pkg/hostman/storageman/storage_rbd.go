package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
)

type SRBDStorage struct {
	SBaseStorage
}

func NewRBDStorage(manager *SStorageManager, path string) *SRBDStorage {
	var ret = new(SRBDStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	return ret
}

func (s *SRBDStorage) StorageType() string {
	return storagetypes.STORAGE_RBD
}

func (s *SRBDStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
}

func (s *SRBDStorage) GetSnapshotDir() string {
	return ""
}

func (s *SRBDStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SRBDStorage) GetFuseMountPath() string {
	return ""
}

func (s *SRBDStorage) GetImgsaveBackupPath() string {
	return ""
}

func (s *SRBDStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (s *SRBDStorage) GetDiskById(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			if s.Disks[i].Probe() == nil {
				return s.Disks[i]
			}
		}
	}
	var disk = NewRBDDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk
	} else {
		return nil
	}
}

func (s *SRBDStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewRBDDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SRBDStorage) Accessible() bool {
	return true
}

func (s *SRBDStorage) DeleteDiskfile(diskpath string) error {
	return nil
}

func (s *SRBDStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (s *SRBDStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return nil
}

func (s *SRBDStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}
