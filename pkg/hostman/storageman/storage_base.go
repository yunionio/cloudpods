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
	"sync"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

var (
	_RECYCLE_BIN_     = "recycle_bin"
	_IMGSAVE_BACKUPS_ = "imgsave_backups"
)

type IStorageFactory interface {
	NewStorage(manager *SStorageManager, mountPoint string) IStorage
	StorageType() string
}

var (
	storagesFactories = make([]IStorageFactory, 0)
)

func registerStorageFactory(factory IStorageFactory) {
	storagesFactories = append(storagesFactories, factory)
}

func NewStorage(manager *SStorageManager, mountPoint, storageType string) IStorage {
	for i := range storagesFactories {
		if storageType == storagesFactories[i].StorageType() || strings.HasPrefix(mountPoint, storagesFactories[i].StorageType()) {
			return storagesFactories[i].NewStorage(manager, mountPoint)
		}
	}
	log.Errorf("no storage driver for %s found", storageType)
	return nil
}

type IStorage interface {
	GetId() string
	GetStorageName() string
	GetZone() string

	SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject)
	SyncStorageInfo() (jsonutils.JSONObject, error)
	StorageType() string
	GetStorageConf() *jsonutils.JSONDict
	GetStoragecacheId() string
	SetStoragecacheId(storagecacheId string)

	SetPath(string)
	GetPath() string

	GetSnapshotDir() string
	GetSnapshotPathByIds(diskId, snapshotId string) string
	DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	GetFreeSizeMb() int
	GetCapacity() int

	// Find owner disks first, if not found, call create disk
	GetDiskById(diskId string) IDisk
	CreateDisk(diskId string) IDisk
	RemoveDisk(IDisk)

	// DeleteDisk(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	// *SDiskCreateByDiskinfo
	CreateDiskByDiskinfo(context.Context, interface{}) (jsonutils.JSONObject, error)
	SaveToGlance(context.Context, interface{}) (jsonutils.JSONObject, error)

	CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error

	DeleteDiskfile(diskPath string) error
	GetFuseTmpPath() string
	GetFuseMountPath() string
	GetImgsaveBackupPath() string
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

func NewBaseStorage(manager *SStorageManager, path string) *SBaseStorage {
	var ret = new(SBaseStorage)
	ret.Disks = make([]IDisk, 0)
	ret.DiskLock = new(sync.Mutex)
	ret.Manager = manager
	ret.Path = path
	return ret
}

func (s *SBaseStorage) GetId() string {
	return s.StorageId
}

func (s *SBaseStorage) GetStorageName() string {
	return s.StorageName
}

func (s *SBaseStorage) GetStoragecacheId() string {
	return s.StoragecacheId
}

func (s *SBaseStorage) SetStoragecacheId(storagecacheId string) {
	s.StoragecacheId = storagecacheId
}

func (s *SBaseStorage) GetName(generateName func() string) string {
	if len(s.StorageName) > 0 {
		return s.StorageName
	} else {
		return generateName()
	}
}

func (s *SBaseStorage) GetPath() string {
	return s.Path
}

func (s *SBaseStorage) SetPath(p string) {
	s.Path = p
}

func (s *SBaseStorage) GetZone() string {
	return s.Manager.GetZone()
}

func (s *SBaseStorage) GetCapacity() int {
	return s.GetAvailSizeMb()
}

func (s *SBaseStorage) GetStorageConf() *jsonutils.JSONDict {
	return s.StorageConf
}

func (s *SBaseStorage) GetAvailSizeMb() int {
	return s.GetTotalSizeMb()
}

func (s *SBaseStorage) GetMediumType() string {
	return s.Manager.GetMediumType()
}

func (s *SBaseStorage) GetFreeSizeMb() int {
	var stat syscall.Statfs_t
	err := syscall.Statfs(s.Path, &stat)
	if err != nil {
		log.Errorln(err)
		return -1
	}
	return int(stat.Bavail * uint64(stat.Bsize) / 1024 / 1024)
}

func (s *SBaseStorage) GetTotalSizeMb() int {
	var stat syscall.Statfs_t
	err := syscall.Statfs(s.Path, &stat)
	if err != nil {
		log.Errorln(err)
		return -1
	}
	return int(stat.Blocks * uint64(stat.Bsize) / 1024 / 1024)
}

func (s *SBaseStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
}

func (s *SBaseStorage) RemoveDisk(d IDisk) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()

	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == d.GetId() {
			s.Disks = append(s.Disks[:i], s.Disks[i+1:]...)
			break
		}
	}
}

func (s *SBaseStorage) DeleteDiskfile(diskpath string) error {
	return fmt.Errorf("Not Implement")
}

func (s *SBaseStorage) CreateDiskByDiskinfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	createParams, ok := params.(*SDiskCreateByDiskinfo)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if createParams.Disk != nil {
		if !jsonutils.QueryBoolean(createParams.DiskInfo, "rebuild", false) {
			return nil, fmt.Errorf("Disk exist")
		}
		if _, err := createParams.Disk.Delete(ctx, params); err != nil {
			return nil, err
		}
	}

	disk := createParams.Storage.CreateDisk(createParams.DiskId)
	if disk == nil {
		return nil, fmt.Errorf("Fail to Create disk %s", createParams.DiskId)
	}

	switch {
	case createParams.DiskInfo.Contains("snapshot"):
		log.Infof("CreateDiskFromSnpashot %s", createParams)
		return s.CreateDiskFromSnpashot(ctx, disk, createParams)
	case createParams.DiskInfo.Contains("image_id"):
		log.Infof("CreateDiskFromTemplate %s", createParams)
		return s.CreateDiskFromTemplate(ctx, disk, createParams)
	case createParams.DiskInfo.Contains("size"):
		log.Infof("CreateRawDisk %s", createParams)
		return s.CreateRawDisk(ctx, disk, createParams)
	default:
		return nil, fmt.Errorf("Not fount")
	}
}

func (s *SBaseStorage) CreateRawDisk(ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	size, _ := createParams.DiskInfo.Int("size")
	diskFromat, _ := createParams.DiskInfo.GetString("format")
	fsFormat, _ := createParams.DiskInfo.GetString("fs_format")
	encryption := jsonutils.QueryBoolean(createParams.DiskInfo, "encryption", false)

	return disk.CreateRaw(ctx, int(size), diskFromat, fsFormat, encryption, createParams.DiskId, "")
}

func (s *SBaseStorage) CreateDiskFromTemplate(ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var (
		imageId, _ = createParams.DiskInfo.GetString("image_id")
		format     = "qcow2" // force qcow2
		size, _    = createParams.DiskInfo.Int("size")
	)

	return disk.CreateFromTemplate(ctx, imageId, format, size)
}

func (s *SBaseStorage) CreateDiskFromSnpashot(ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var (
		// diskPath            = path.Join(s.Path, createParams.DiskId)
		snapshotUrl, _      = createParams.DiskInfo.GetString("snapshot_url")
		transferProtocol, _ = createParams.DiskInfo.GetString("url")
	)

	if len(snapshotUrl) == 0 || len(transferProtocol) == 0 {
		return nil, fmt.Errorf("Create disk form snapshot missing params snapshot url or protocol")
	}

	if transferProtocol == "url" {
		// TODO
		// snapshotOutOfChain := jsonutils.QueryBoolean(
		// 	createParams.DiskInfo, "snapshot_out_of_chain", false)
		// // you wen ti...
		// if err := s.CreateDiskFromUrl(ctx, snapshotUrl, diskPath, !snapshotOutOfChain); err != nil {
		// 	return nil, err
		// }
	} else if transferProtocol == "fuse" {
		if err := disk.CreateFromImageFuse(ctx, snapshotUrl); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Unkown protocol %s", transferProtocol)
	}
	return disk.GetDiskDesc(), nil
}

func (s *SLocalStorage) GetImgsaveBackupPath() string {
	return s.getSubdirPath(_IMGSAVE_BACKUPS_)
}
