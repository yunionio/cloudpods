package storageman

import (
	"context"
	"fmt"
	"path"
	"sync"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

var (
	_RECYCLE_BIN_     = "recycle_bin"
	_IMGSAVE_BACKUPS_ = "imgsave_backups"
)

type IStorage interface {
	GetId() string
	GetZone() string

	StorageType() string
	GetPath() string
	GetFreeSizeMb() int
	GetCapacity() int

	// Find owner disks first, if not found, call create disk
	GetDiskById(diskId string) IDisk
	CreateDisk(diskId string) IDisk
	RemoveDisk(IDisk)

	DeleteDisk(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	// *SDiskCreateByDiskinfo
	CreateDiskByDiskinfo(context.Context, interface{}) (jsonutils.JSONObject, error)

	DeleteDiskfile(diskPath string) error
	GetFuseTmpPath() string
	GetFuseMountPath() string
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

func (s *SBaseStorage) GetPath() string {
	return s.Path
}

func (s *SBaseStorage) GetZone() string {
	return s.Manager.GetZone()
}

func (s *SBaseStorage) GetCapacity() int {
	return s.GetAvailSizeMb()
}

func (s *SBaseStorage) GetAvailSizeMb() int {
	return s.GetTotalSizeMb()
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

func (s *SBaseStorage) RemoveDisk(d IDisk) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()

	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == d.GetId() {
			s.Disks = append(s.Disks[:i], s.Disks[i+1:]...)
		}
	}
}

func (s *SBaseStorage) DeleteDiskfile(diskpath string) error {
	return fmt.Errorf("Not Implement")
}

func (s *SBaseStorage) CreateDiskByDiskinfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	createParams, ok := params.(*SDiskCreateByDiskinfo)
	if !ok {
		return nil, fmt.Errorf("Unknown params")
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
		return s.CreateDiskFromSnpashot(ctx, disk, createParams) // TODO
	case createParams.DiskInfo.Contains("image_id"):
		return s.CreateDiskFromTemplate(ctx, disk, createParams) // TODO
	case createParams.DiskInfo.Contains("size"):
		return s.CreateRawDisk(ctx, disk, createParams) // TODO
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
		diskPath            = path.Join(s.Path, createParams.DiskId)
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
