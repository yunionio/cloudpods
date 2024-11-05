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
	"io/ioutil"
	"path"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/storageutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const (
	_RECYCLE_BIN_     = "recycle_bin"
	_IMGSAVE_BACKUPS_ = "imgsave_backups"
	_SNAPSHOT_PATH_   = "snapshots"
	_BACKUP_PATH_     = "backups"

	ErrStorageTimeout = constError("storage accessible check timeout")
	TempBindMountPath = "/opt/cloud/workspace/temp-bind"
)

type constError string

func (e constError) Error() string { return string(e) }

var DELETEING_SNAPSHOTS = sync.Map{}

type IStorageFactory interface {
	NewStorage(manager *SStorageManager, mountPoint string) IStorage
	StorageType() string
}

var (
	storagesFactories = make([]IStorageFactory, 0)
)

// for shared storages
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
	GetZoneId() string

	SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error
	SyncStorageInfo() (jsonutils.JSONObject, error)
	SyncStorageSize() (api.SHostStorageStat, error)
	StorageType() string
	GetStorageConf() *jsonutils.JSONDict
	GetStoragecacheId() string
	SetStoragecacheId(storagecacheId string)

	IsLocal() bool
	Lvmlockd() bool

	SetPath(string)
	GetPath() string

	GetSnapshotDir() string
	GetSnapshotPathByIds(diskId, snapshotId string) string
	DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	IsSnapshotExist(diskId, snapshotId string) (bool, error)

	GetBackupDir() string
	StorageBackup(ctx context.Context, params *SStorageBackup) (jsonutils.JSONObject, error)
	StorageBackupRecovery(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	GetFreeSizeMb() int
	GetCapacityMb() int

	// Find owner disks first, if not found, call create disk
	GetDiskById(diskId string) (IDisk, error)
	CreateDisk(diskId string) IDisk
	RemoveDisk(IDisk)
	GetDisksPath() ([]string, error)

	// DeleteDisk(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	// *SDiskCreateByDiskinfo
	CreateDiskByDiskinfo(context.Context, interface{}) (jsonutils.JSONObject, error)
	SaveToGlance(context.Context, interface{}) (jsonutils.JSONObject, error)
	CreateDiskFromSnapshot(context.Context, IDisk, *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error)
	CreateDiskFromExistingPath(context.Context, IDisk, *SDiskCreateByDiskinfo) error
	CreateDiskFromBackup(context.Context, IDisk, *SDiskCreateByDiskinfo) error
	DiskMigrate(context.Context, interface{}) (jsonutils.JSONObject, error)

	// GetCloneTargetDiskPath generate target disk path by target disk id
	GetCloneTargetDiskPath(ctx context.Context, targetDiskId string) string
	// CloneDiskFromStorage clone disk from other storage
	CloneDiskFromStorage(ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool) (*hostapi.ServerCloneDiskFromStorageResponse, error)

	CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error

	DeleteDiskfile(diskPath string, skipRecycle bool) error
	GetFuseTmpPath() string
	GetFuseMountPath() string
	GetImgsaveBackupPath() string

	DestinationPrepareMigrate(ctx context.Context, liveMigrate bool, disksUri string, snapshotsUri string,
		disksBackingFile, diskSnapsChain, outChainSnaps jsonutils.JSONObject, rebaseDisks bool, diskDesc *desc.SGuestDisk, serverId string, idx, totalDiskCount int, encInfo *apis.SEncryptInfo, sysDiskHasTemplate bool) error

	Accessible() error
	Detach() error

	CleanRecycleDiskfiles(ctx context.Context)
}

type SBaseStorage struct {
	Manager          *SStorageManager
	StorageId        string
	Path             string
	StorageName      string
	StorageConf      *jsonutils.JSONDict
	StoragecacheId   string
	isSetStorageInfo bool

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

func (s *SBaseStorage) Lvmlockd() bool {
	return false
}

func (s *SBaseStorage) GetFuseTmpPath() string {
	return path.Join(s.Path, _FUSE_TMP_PATH_)
}

func (s *SBaseStorage) GetFuseMountPath() string {
	return path.Join(s.Path, _FUSE_MOUNT_PATH_)
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

func (s *SBaseStorage) StorageBackup(ctx context.Context, params *SStorageBackup) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (s *SBaseStorage) StorageBackupRecovery(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
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

func (s *SBaseStorage) GetZoneId() string {
	return s.Manager.GetZoneId()
}

func (d *SBaseStorage) GetDisksPath() ([]string, error) {
	spath := d.GetPath()
	files, err := ioutil.ReadDir(spath)
	if err != nil {
		return nil, err
	}
	disksPath := make([]string, 0)
	for i := range files {
		if !files[i].IsDir() && regutils.MatchUUIDExact(files[i].Name()) {
			disksPath = append(disksPath, path.Join(spath, files[i].Name()))
		}
	}

	return disksPath, nil
}

func (s *SBaseStorage) GetCapacityMb() int {
	return s.GetAvailSizeMb()
}

func (s *SBaseStorage) GetStorageConf() *jsonutils.JSONDict {
	return s.StorageConf
}

func (s *SBaseStorage) GetAvailSizeMb() int {
	return s.GetTotalSizeMb()
}

func (s *SBaseStorage) GetUsedSizeMb() int {
	size, err := storageutils.GetUsedSizeMb(s.Path)
	if err != nil {
		log.Errorf("failed get %s used size: %s", s.Path, err)
		return -1
	}
	return size
}

/*func (s *SBaseStorage) GetMediumType() string {
	return s.Manager.GetMediumType()
}*/

func (s *SBaseStorage) GetFreeSizeMb() int {
	size, err := storageutils.GetFreeSizeMb(s.Path)
	if err != nil {
		log.Errorf("failed get %s free size: %s", s.Path, err)
		return -1
	}
	return size
}

func (s *SBaseStorage) GetTotalSizeMb() int {
	size, err := storageutils.GetTotalSizeMb(s.Path)
	if err != nil {
		log.Errorf("failed get %s total size: %s", s.Path, err)
		return -1
	}
	return size
}

func (s *SBaseStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	s.BindMountStoragePath(s.Path)
	return nil
}

func (s *SBaseStorage) SyncStorageSize() (api.SHostStorageStat, error) {
	stat := api.SHostStorageStat{
		StorageId: s.StorageId,
	}
	stat.CapacityMb = int64(s.GetCapacityMb())
	stat.ActualCapacityUsedMb = int64(s.GetUsedSizeMb())
	return stat, nil
}

func (s *SBaseStorage) BindMountStoragePath(sPath string) error {
	if strings.HasPrefix(sPath, "/opt/cloud") {
		return nil
	}
	if s.isSetStorageInfo || !options.HostOptions.EnableRemoteExecutor {
		return nil
	}

	err := s.bindMountTo(sPath)
	if err == nil {
		s.isSetStorageInfo = true
	}
	return err
}

func (s *SBaseStorage) bindMountTo(sPath string) error {
	tempPath := path.Join(TempBindMountPath, sPath)
	out, err := procutils.NewCommand("mkdir", "-p", tempPath).Output()
	if err != nil {
		return errors.Errorf("mkdir temp mount path %s failed %s", tempPath, out)
	}
	out, err = procutils.NewCommand("mkdir", "-p", sPath).Output()
	if err != nil {
		return errors.Errorf("mkdir mount path %s failed %s", sPath, out)
	}
	if procutils.NewCommand("mountpoint", tempPath).Run() != nil {
		out, err = procutils.NewRemoteCommandAsFarAsPossible("mount", "--bind", sPath, tempPath).Output()
		if err != nil {
			return errors.Errorf("bind mount to temp path failed %s", out)
		}
	}
	if procutils.NewCommand("mountpoint", sPath).Run() != nil {
		out, err = procutils.NewCommand("mount", "--bind", tempPath, sPath).Output()
		if err != nil {
			return errors.Errorf("bind mount temp path to local image path failed %s", out)
		}
	}
	log.Infof("bind mount %s -> %s", tempPath, sPath)
	return nil
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

func (s *SBaseStorage) DeleteDiskfile(diskpath string, skipRecycle bool) error {
	return fmt.Errorf("Not Implement")
}

func (s *SBaseStorage) CreateDiskByDiskinfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	createParams, ok := params.(*SDiskCreateByDiskinfo)
	if !ok {
		return nil, hostutils.ParamsError
	}

	if createParams.Disk != nil {
		if !createParams.DiskInfo.Rebuild {
			return nil, fmt.Errorf("Disk exist")
		}
		if createParams.DiskInfo.Backup != nil && createParams.DiskInfo.Backup.BackupAsTar != nil {
			log.Infof("skip OnRebuildRoot step when backup_as_tar is provided")
		} else {
			if err := createParams.Disk.OnRebuildRoot(ctx, createParams.DiskInfo); err != nil {
				return nil, err
			}
		}
	}

	disk := createParams.Storage.CreateDisk(createParams.DiskId)
	if disk == nil {
		return nil, fmt.Errorf("Fail to Create disk %s", createParams.DiskId)
	}

	log.Infof("storage %s %s(%s) start create disk", createParams.Storage.StorageType(), createParams.Storage.GetStorageName(), createParams.Storage.GetId())
	switch {
	case len(createParams.DiskInfo.SnapshotId) > 0:
		log.Infof("CreateDiskFromSnpashot %s", createParams)
		return s.CreateDiskFromSnpashot(ctx, disk, createParams)
	case len(createParams.DiskInfo.ImageId) > 0 && createParams.DiskInfo.ImageFormat != imageapi.IMAGE_DISK_FORMAT_TGZ:
		log.Infof("CreateDiskFromTemplate %s", createParams)
		return s.CreateDiskFromTemplate(ctx, disk, createParams)
	case createParams.DiskInfo.Backup != nil:
		log.Infof("CreateDiskFromBackup %s", createParams)
		return s.createDiskFromBackup(ctx, disk, createParams)
	case len(createParams.DiskInfo.ExistingPath) > 0:
		return s.createDiskFromExistingPath(ctx, disk, createParams)
	case createParams.DiskInfo.DiskSizeMb > 0:
		log.Infof("CreateRawDisk %s", createParams)
		return s.CreateRawDisk(ctx, disk, createParams)
	default:
		return nil, fmt.Errorf("CreateDiskByDiskinfo with params %s empty snapshot_id, image_id or disk_size_mb", jsonutils.Marshal(createParams.DiskInfo))
	}
}

func (s *SBaseStorage) CreateRawDisk(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var encryptInfo *apis.SEncryptInfo
	if input.DiskInfo.Encryption {
		encryptInfo = &input.DiskInfo.EncryptInfo
	}
	return disk.CreateRaw(ctx, input.DiskInfo.DiskSizeMb,
		input.DiskInfo.Format,
		input.DiskInfo.FsFormat,
		input.DiskInfo.FsFeatures,
		encryptInfo, input.DiskId, "")
}

func (s *SBaseStorage) CreateDiskFromTemplate(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var (
		format = "qcow2" // force qcow2
	)

	var encryptInfo *apis.SEncryptInfo
	if input.DiskInfo.Encryption {
		encryptInfo = &input.DiskInfo.EncryptInfo
	}

	return disk.CreateFromTemplate(ctx, input.DiskInfo.ImageId, format, int64(input.DiskInfo.DiskSizeMb), encryptInfo)
}

func (s *SBaseStorage) CreateDiskFromSnpashot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var storage = input.Storage
	if len(input.DiskInfo.SnapshotUrl) == 0 {
		return nil, httperrors.NewMissingParameterError("snapshot_url")
	}

	return storage.CreateDiskFromSnapshot(ctx, disk, input)
}

func (s *SBaseStorage) createDiskFromExistingPath(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var storage = input.Storage
	if len(input.DiskInfo.ExistingPath) == 0 {
		return nil, httperrors.NewMissingParameterError("existing_path")
	}
	if !strings.HasPrefix(input.DiskInfo.ExistingPath, storage.GetPath()) {
		return nil, errors.Errorf("disk %s not in storage %s", input.DiskInfo.ExistingPath, storage.GetPath())
	}
	err := storage.CreateDiskFromExistingPath(ctx, disk, input)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDiskFromExistingPath")
	}

	return disk.GetDiskDesc(), nil
}

func (s *SBaseStorage) createDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	var storage = input.Storage
	if input.DiskInfo.Backup == nil {
		return nil, httperrors.NewMissingParameterError("Backup")
	}

	err := storage.CreateDiskFromBackup(ctx, disk, input)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDiskFromBackup")
	}

	return disk.GetDiskDesc(), nil
}

func (s *SBaseStorage) DestinationPrepareMigrate(
	ctx context.Context, liveMigrate bool, disksUri string, snapshotsUri string,
	disksBackingFile, diskSnapsChain, outChainSnaps jsonutils.JSONObject, rebaseDisks bool, diskinfo *desc.SGuestDisk, serverId string, idx, totalDiskCount int, encInfo *apis.SEncryptInfo, sysDiskHasTemplate bool,
) error {
	return nil
}

func (s *SBaseStorage) GetCloneTargetDiskPath(ctx context.Context, targetDiskId string) string {
	return ""
}

func (s *SBaseStorage) CloneDiskFromStorage(
	ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool,
) (*hostapi.ServerCloneDiskFromStorageResponse, error) {
	return nil, httperrors.ErrNotImplemented
}

func (s *SBaseStorage) GetBackupDir() string {
	return path.Join(s.Path, _BACKUP_PATH_)
}

func (s *SBaseStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	info := input.DiskInfo
	backupPath := path.Join(s.GetBackupDir(), info.Backup.BackupId)
	img, err := qemuimg.NewQemuImage(backupPath)
	if err != nil {
		log.Errorf("unable to new qemu image for %s: %s", backupPath, err.Error())
		return errors.Wrapf(err, "unable to new qemu image for %s", backupPath)
	}
	_, err = img.Clone(disk.GetPath(), qemuimgfmt.QCOW2, false)
	return err
}

func (s *SBaseStorage) DiskMigrate(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, httperrors.ErrNotImplemented
}

func (s *SBaseStorage) onSaveToGlanceFailed(ctx context.Context, imageId string, reason string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	params.Set("reason", jsonutils.NewString(reason))
	_, err := image.Images.PerformAction(
		hostutils.GetImageSession(ctx),
		imageId, "update-status", params,
	)
	if err != nil {
		log.Errorln(err)
	}
}

func requestDeleteSnapshot(
	storage IStorage, diskId, snapshotPath, deleteSnapshot, convertSnapshotPath,
	outfile string, pendingDelete bool,
) {
	deleteSnapshotPath := path.Join(snapshotPath, deleteSnapshot)
	DELETEING_SNAPSHOTS.Store(diskId, true)
	defer DELETEING_SNAPSHOTS.Delete(diskId)
	_, err := modules.Snapshots.PerformAction(hostutils.GetComputeSession(context.Background()),
		deleteSnapshot, "deleted", nil)
	if err != nil {
		log.Errorln(err)
		return
	}
	if out, err := procutils.NewCommand("rm", "-f", convertSnapshotPath).Output(); err != nil {
		log.Errorf("%s", out)
		return
	}
	if out, err := procutils.NewCommand("mv", "-f", outfile, convertSnapshotPath).Output(); err != nil {
		log.Errorf("%s", out)
		return
	}
	if !pendingDelete {
		if err := storage.DeleteDiskfile(deleteSnapshotPath, false); err != nil {
			log.Errorln(err)
			return
		}
	}
}
