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
	"os"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SLVMStorage struct {
	SBaseStorage
}

func NewLVMStorage(manager *SStorageManager, vgName string) *SLVMStorage {
	var ret = new(SLVMStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, vgName)
	return ret
}

func (s *SLVMStorage) StorageType() string {
	return api.STORAGE_LVM
}

func (s *SLVMStorage) IsLocal() bool {
	return true
}

func (s *SLVMStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%s", s.Manager.host.GetMasterIp(), s.StorageType(), s.Path)
}

func (s *SLVMStorage) GetMediumType() (string, error) {
	return api.DISK_TYPE_ROTATE, nil
}

func (s *SLVMStorage) GetFreeSizeMb() int {
	vgProps, err := lvmutils.GetVgProps(s.GetPath())
	if err != nil {
		log.Errorf("failed get vg_free %s", err)
		return -1
	}

	log.Infof("LVM Storage %s sizeMb %d", s.GetPath(), vgProps.VgSize/1024/1024)
	return int(vgProps.VgFree / 1024 / 1024)
}

func (s *SLVMStorage) GetAvailSizeMb() int {
	avaSize, _ := s.getAvailSizeMb()
	return int(avaSize)
}

func (s *SLVMStorage) getAvailSizeMb() (int64, error) {
	vgProps, err := lvmutils.GetVgProps(s.GetPath())
	if err != nil {
		return -1, err
	}

	log.Debugf("LVM Storage %s sizeMb %d", s.GetPath(), vgProps.VgSize/1024/1024)
	return vgProps.VgSize / 1024 / 1024, nil
}

func (s *SLVMStorage) GetUsedSizeMb() (int64, error) {
	vgProps, err := lvmutils.GetVgProps(s.GetPath())
	if err != nil {
		return -1, err
	}
	return (vgProps.VgSize - vgProps.VgFree) / 1024 / 1024, nil
}

func (s *SLVMStorage) SyncStorageSize() (api.SHostStorageStat, error) {
	stat := api.SHostStorageStat{
		StorageId: s.StorageId,
	}
	sizeMb, err := s.getAvailSizeMb()
	if err != nil {
		return stat, err
	}
	stat.CapacityMb = sizeMb
	stat.ActualCapacityUsedMb = 0
	return stat, nil
}

func (s *SLVMStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	name := s.GetName(s.GetComposedName)
	content.Set("name", jsonutils.NewString(name))
	sizeMb, err := s.getAvailSizeMb()
	if err != nil {
		return nil, errors.Wrap(err, "GetAvailSizeMb")
	}
	usedSizeMb, err := s.GetUsedSizeMb()
	if err != nil {
		return nil, errors.Wrap(err, "GetUsedSizeMb")
	}

	content.Set("capacity", jsonutils.NewInt(sizeMb))

	content.Set("actual_capacity_used", jsonutils.NewInt(usedSizeMb))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("zone", jsonutils.NewString(s.GetZoneId()))

	var (
		res jsonutils.JSONObject
	)

	log.Infof("Sync storage info %s/%s", s.StorageId, name)
	if len(s.StorageId) > 0 {
		res, err = modules.Storages.Put(
			hostutils.GetComputeSession(context.Background()),
			s.StorageId, content)
	} else {
		mediumType, err := s.GetMediumType()
		if err != nil {
			log.Errorf("failed get medium type %s %s", s.GetPath(), err)
		} else {
			content.Set("medium_type", jsonutils.NewString(mediumType))
		}
		// reserved for imagecache
		reserved := sizeMb / 10
		if reserved > 1024*1024 {
			reserved = 1024 * 1024
		}
		content.Set("reserved", jsonutils.NewInt(reserved))

		res, err = modules.Storages.Create(hostutils.GetComputeSession(context.Background()), content)
		if err == nil {
			log.Errorf("storage created %s", res)
			storageCacheId, _ := res.GetString("storagecache_id")
			storageManager.InitLVMStorageImageCache(storageCacheId, s.GetPath(), s)
			s.SetStoragecacheId(storageCacheId)
		}
	}
	if err != nil {
		log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
	}
	return res, err
}

func (s *SLVMStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	return nil
}

func (s *SLVMStorage) GetSnapshotDir() string {
	return ""
}

func (s *SLVMStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	disk, err := s.GetDiskById(diskId)
	if err != nil {
		log.Errorf("lvm failed get disk by id %s: %s", diskId, err)
		return ""
	}
	return disk.GetSnapshotPath(snapshotId)
}

func (s *SLVMStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SStorageDeleteSnapshots)
	for i := range input.SnapshotIds {
		lvPath := path.Join("/dev", s.GetPath(), "snap_"+input.SnapshotIds[i])
		if err := lvmutils.LvRemove(lvPath); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (s *SLVMStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SStorageDeleteSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}
	if input.BlockStream {
		if err := ConvertLVMDisk(s.GetPath(), input.DiskId, input.EncryptInfo); err != nil {
			return nil, err
		}
	} else if len(input.ConvertSnapshot) > 0 {
		convertSnapshotName := "snap_" + input.ConvertSnapshot
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

func (s *SLVMStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	return false, errors.Errorf("unsupported operation")
}

func (s *SLVMStorage) GetDiskById(diskId string) (IDisk, error) {
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
	var disk = NewLVMDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.ErrNotFound
}

func (s *SLVMStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewLVMDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SLVMStorage) SaveToGlance(ctx context.Context, input interface{}) (jsonutils.JSONObject, error) {
	info, ok := input.(SStorageSaveToGlanceInfo)
	if !ok {
		return nil, hostutils.ParamsError
	}
	data := info.DiskInfo

	var (
		imageId, _   = data.GetString("image_id")
		imagePath, _ = data.GetString("image_path")
		compress     = jsonutils.QueryBoolean(data, "compress", true)
		encKeyId, _  = data.GetString("encrypt_key_id")
		err          error
	)

	var (
		encKey    string
		encFormat qemuimg.TEncryptFormat
		encAlg    seclib2.TSymEncAlg
	)

	if len(encKeyId) > 0 {
		session := auth.GetSession(ctx, info.UserCred, consts.GetRegion())
		key, err := identity_modules.Credentials.GetEncryptKey(session, encKeyId)
		if err != nil {
			return nil, errors.Wrap(err, "GetEncryptKey")
		}
		encKey = key.Key
		encFormat = qemuimg.EncryptFormatLuks
		encAlg = key.Alg
	}

	if err = s.saveToGlance(ctx, imageId, imagePath, compress, encKey, encFormat, encAlg); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		s.onSaveToGlanceFailed(ctx, imageId, err.Error())
	}

	imagecacheManager := s.Manager.LocalStorageImagecacheManager
	if err != nil || len(imagecacheManager.GetId()) > 0 {
		return nil, procutils.NewCommand("rm", "-f", imagePath).Run()
	} else {
		dstPath := path.Join(imagecacheManager.GetPath(), imageId)
		if err := procutils.NewCommand("mv", imagePath, dstPath).Run(); err != nil {
			log.Errorf("Fail to move saved image to cache: %s", err)
		}
		imagecacheManager.LoadImageCache(imageId)
		_, err := hostutils.RemoteStoragecacheCacheImage(ctx,
			imagecacheManager.GetId(), imageId, "active", dstPath)
		if err != nil {
			log.Errorf("Fail to remote cache image: %s", err)
		}
	}
	return nil, nil
}

func (s *SLVMStorage) saveToGlance(
	ctx context.Context, imageId, imagePath string, compress bool,
	encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg,
) error {
	log.Infof("saveToGlance %s", imagePath)
	diskInfo := &deployapi.DiskInfo{
		Path: imagePath,
	}
	if len(encryptKey) > 0 {
		diskInfo.EncryptPassword = encryptKey
		diskInfo.EncryptFormat = string(encFormat)
		diskInfo.EncryptAlg = string(encAlg)
	}
	ret, err := deployclient.GetDeployClient().SaveToGlance(ctx,
		&deployapi.SaveToGlanceParams{DiskInfo: diskInfo, Compress: compress})
	if err != nil {
		log.Errorf("GetDeployClient.SaveToGlance fail %s", err)
		return errors.Wrap(err, "deployclient.GetDeployClient().SaveToGlance")
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return errors.Wrap(err, "open image")
	}
	defer f.Close()
	finfo, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "f.Stat image")
	}
	size := finfo.Size()
	var params = jsonutils.NewDict()
	if len(ret.OsInfo) > 0 {
		params.Set("os_type", jsonutils.NewString(ret.OsInfo))
	}
	relInfo := ret.ReleaseInfo
	if relInfo != nil {
		params.Set("os_distribution", jsonutils.NewString(relInfo.Distro))
		if len(relInfo.Version) > 0 {
			params.Set("os_version", jsonutils.NewString(relInfo.Version))
		}
		if len(relInfo.Arch) > 0 {
			params.Set("os_arch", jsonutils.NewString(relInfo.Arch))
		}
		if len(relInfo.Version) > 0 {
			params.Set("os_language", jsonutils.NewString(relInfo.Language))
		}
	}
	params.Set("image_id", jsonutils.NewString(imageId))

	_, err = image.Images.Upload(hostutils.GetImageSession(ctx),
		params, f, size)
	if err != nil {
		return errors.Wrap(err, "image upload")
	}
	return nil
}

func (s *SLVMStorage) DestinationPrepareMigrate(
	ctx context.Context, liveMigrate bool, disksUri string, snapshotsUri string,
	disksBackingFile, diskSnapsChain, outChainSnaps jsonutils.JSONObject,
	rebaseDisks bool,
	diskinfo *desc.SGuestDisk,
	serverId string, idx, totalDiskCount int,
	encInfo *apis.SEncryptInfo, sysDiskHasTemplate bool,
) error {
	var (
		diskId               = diskinfo.DiskId
		snapshots, _         = diskSnapsChain.GetArray(diskId)
		disk                 = s.CreateDisk(diskId)
		diskOutChainSnaps, _ = outChainSnaps.GetArray(diskId)
	)

	if disk == nil {
		return fmt.Errorf(
			"Storage %s create disk %s failed", s.GetId(), diskId)
	}

	templateId := diskinfo.TemplateId
	// create snapshots form remote url
	var (
		diskStorageId = diskinfo.StorageId
		baseImagePath string
	)
	for i, snapshotId := range snapshots {
		snapId, _ := snapshotId.GetString()
		snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
			snapshotsUri, diskStorageId, diskId, snapId)
		snapshotPath := path.Join("/dev", s.GetPath(), "snap_"+snapId)
		log.Infof("Disk %s snapshot %s url: %s", diskId, snapId, snapshotUrl)
		if err := s.CreateSnapshotFormUrl(ctx, snapshotUrl, diskId, snapshotPath); err != nil {
			return errors.Wrap(err, "create from snapshot url failed")
		}
		if i == 0 && len(templateId) > 0 && sysDiskHasTemplate {
			templatePath := path.Join("/dev", s.GetPath(), "imagecache_"+templateId)
			// check if template is encrypted
			img, err := qemuimg.NewQemuImage(templatePath)
			if err != nil {
				return errors.Wrap(err, "template image probe fail")
			}
			if img.Encrypted {
				templatePath = qemuimg.GetQemuFilepath(templatePath, "sec0", qemuimg.EncryptFormatLuks)
			}
			if err := doRebaseDisk(snapshotPath, templatePath, encInfo); err != nil {
				return err
			}
		} else if rebaseDisks && len(baseImagePath) > 0 {
			if encInfo != nil {
				baseImagePath = qemuimg.GetQemuFilepath(baseImagePath, "sec0", qemuimg.EncryptFormatLuks)
			}
			if err := doRebaseDisk(snapshotPath, baseImagePath, encInfo); err != nil {
				return err
			}
		}
		baseImagePath = snapshotPath
	}

	for _, snapshotId := range diskOutChainSnaps {
		snapId, _ := snapshotId.GetString()
		snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
			snapshotsUri, diskStorageId, diskId, snapId)
		snapshotPath := disk.GetSnapshotPath(snapId)
		log.Infof("Disk %s snapshot %s url: %s", diskId, snapId, snapshotUrl)
		if err := s.CreateSnapshotFormUrl(ctx, snapshotUrl, diskId, snapshotPath); err != nil {
			return errors.Wrap(err, "create from snapshot url failed")
		}
	}

	if liveMigrate {
		// create local disk
		backingFile, _ := disksBackingFile.GetString(diskId)
		_, err := disk.CreateRaw(ctx, int(diskinfo.Size), "qcow2", "", nil, encInfo, "", backingFile)
		if err != nil {
			log.Errorln(err)
			return err
		}
	} else {
		// download disk form remote url
		diskUrl := fmt.Sprintf("%s/%s/%s", disksUri, diskStorageId, diskId)
		err := disk.CreateFromUrl(ctx, diskUrl, 0, func(progress, progressMbps float64, totalSizeMb int64) {
			log.Debugf("[%.2f / %d] disk %s create %.2f with speed %.2fMbps", progress*float64(totalSizeMb)/100, totalSizeMb, disk.GetId(), progress, progressMbps)
			newProgress := float64(idx-1)/float64(totalDiskCount)*100.0 + 1/float64(totalDiskCount)*progress
			if len(serverId) > 0 {
				log.Debugf("server %s migrate %.2f with speed %.2fMbps", serverId, newProgress, progressMbps)
				hostutils.UpdateServerProgress(context.Background(), serverId, newProgress, progressMbps)
			}
		})
		if err != nil {
			return errors.Wrap(err, "CreateFromUrl")
		}
	}
	if rebaseDisks && len(templateId) > 0 && len(baseImagePath) == 0 {
		templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
		// check if template is encrypted
		img, err := qemuimg.NewQemuImage(templatePath)
		if err != nil {
			return errors.Wrap(err, "template image probe fail")
		}
		if img.Encrypted {
			templatePath = qemuimg.GetQemuFilepath(templatePath, "sec0", qemuimg.EncryptFormatLuks)
		}
		if err := doRebaseDisk(disk.GetPath(), templatePath, encInfo); err != nil {
			return err
		}
	} else if rebaseDisks && len(baseImagePath) > 0 {
		if encInfo != nil {
			baseImagePath = qemuimg.GetQemuFilepath(baseImagePath, "sec0", qemuimg.EncryptFormatLuks)
		}
		if err := doRebaseDisk(disk.GetPath(), baseImagePath, encInfo); err != nil {
			return err
		}
	}
	diskinfo.Path = disk.GetPath()
	return nil
}

func (s *SLVMStorage) CreateDiskFromSnapshot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	info := input.DiskInfo
	if info.Protocol == "fuse" {
		var encryptInfo *apis.SEncryptInfo
		if info.Encryption {
			encryptInfo = &info.EncryptInfo
		}
		err := disk.CreateFromImageFuse(ctx, info.SnapshotUrl, int64(info.DiskSizeMb), encryptInfo)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateFromImageFuse")
		}
		return disk.GetDiskDesc(), nil
	}
	return nil, httperrors.NewUnsupportOperationError("Unsupport protocol %s for lvm storage", info.Protocol)
}

func (s *SLVMStorage) CreateDiskFromExistingPath(context.Context, IDisk, *SDiskCreateByDiskinfo) error {
	return errors.Errorf("unsupported operation")
}

func (s *SLVMStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	lvSizeMb := lvmutils.GetQcow2LvSize(int64(input.DiskInfo.DiskSizeMb))
	if err := lvmutils.LvCreate(s.GetPath(), disk.GetId(), lvSizeMb*1024*1024); err != nil {
		return errors.Wrap(err, "CreateRaw")
	}

	err := doRestoreDisk(ctx, s, input, disk, disk.GetPath())
	if err != nil {
		return errors.Wrap(err, "doRestoreDisk")
	}
	return nil
}

func (s *SLVMStorage) GetFuseTmpPath() string {
	localPath := options.HostOptions.ImageCachePath
	if len(options.HostOptions.LocalImagePath) > 0 {
		localPath = options.HostOptions.LocalImagePath[0]
	}

	return path.Join(localPath, _FUSE_TMP_PATH_)
}

func (s *SLVMStorage) GetFuseMountPath() string {
	localPath := options.HostOptions.ImageCachePath
	if len(options.HostOptions.LocalImagePath) > 0 {
		localPath = options.HostOptions.LocalImagePath[0]
	}

	return path.Join(localPath, _FUSE_MOUNT_PATH_)
}

func (s *SLVMStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	remoteFile := remotefile.NewRemoteFile(ctx, snapshotUrl, snapshotPath,
		false, "", -1, nil, "", "")
	err := remoteFile.Fetch(nil)
	return errors.Wrapf(err, "fetch snapshot from %s", snapshotUrl)
}

func (s *SLVMStorage) GetImgsaveBackupPath() string {
	return path.Join(options.HostOptions.ImageCachePath, "image_save")
}

func (s *SLVMStorage) Accessible() error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("pvscan", "--cache").Output()
	if err != nil {
		return errors.Wrapf(err, "pvscan --cache failed %s", out)
	}
	if err := lvmutils.VgDisplay(s.Path); err != nil {
		return err
	}
	return nil
}

func (s *SLVMStorage) Detach() error {
	return nil
}

func (s *SLVMStorage) StorageBackup(ctx context.Context, params *SStorageBackup) (jsonutils.JSONObject, error) {
	backupStorage, err := backupstorage.GetBackupStorage(params.BackupStorageId, params.BackupStorageAccessInfo)
	if err != nil {
		return nil, err
	}
	backupPath := params.BackupLocalPath
	err = backupStorage.SaveBackupFrom(ctx, backupPath, params.BackupId)
	if err != nil {
		return nil, err
	}
	// remove local backup
	if err = lvmutils.LvRemove(backupPath); err != nil {
		return nil, errors.Wrap(err, "On backuped lvremove")
	}
	return nil, nil
}

func (s *SLVMStorage) CloneDiskFromStorage(
	ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool,
) (*hostapi.ServerCloneDiskFromStorageResponse, error) {
	srcDiskPath := srcDisk.GetPath()
	srcImg, err := qemuimg.NewQemuImage(srcDiskPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Get source image %q info", srcDiskPath)
	}

	// create target disk lv
	lvSize := lvmutils.GetQcow2LvSize(srcImg.SizeBytes/1024/1024) * 1024 * 1024
	if err = lvmutils.LvCreate(s.GetPath(), targetDiskId, lvSize); err != nil {
		return nil, errors.Wrap(err, "lvcreate")
	}

	// start create target disk. if full copy is false, just create
	// empty target disk with same size and format
	accessPath := path.Join("/dev", s.GetPath(), targetDiskId)
	if fullCopy {
		_, err = srcImg.Clone(accessPath, qemuimgfmt.QCOW2, false)
	} else {
		newImg, err := qemuimg.NewQemuImage(accessPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed new qemu image")
		}

		err = newImg.CreateQcow2(srcImg.GetSizeMB(), false, "", "", "", "")
	}
	if err != nil {
		return nil, errors.Wrap(err, "Clone source disk to target local storage")
	}
	return &hostapi.ServerCloneDiskFromStorageResponse{
		TargetAccessPath: accessPath,
		TargetFormat:     qemuimgfmt.QCOW2.String(),
	}, nil
}

func (s *SLVMStorage) CleanRecycleDiskfiles(ctx context.Context) {
	log.Infof("SLVMStorage CleanRecycleDiskfiles do nothing!")
}

func (d *SLVMStorage) GetDisksPath() ([]string, error) {
	spath := d.GetPath()
	lvNames, err := lvmutils.GetLvNames(spath)
	if err != nil {
		return nil, err
	}

	disksPath := make([]string, 0)
	for _, f := range lvNames {
		if regutils.MatchUUIDExact(f) {
			disksPath = append(disksPath, path.Join("/dev", spath, f))
		}
	}

	return disksPath, nil
}

func ConvertLVMDisk(vgName, lvName string, encryptInfo apis.SEncryptInfo) error {
	diskPath := path.Join("/dev", vgName, lvName)
	qemuImg, err := qemuimg.NewQemuImage(diskPath)
	if err != nil {
		log.Errorln(err)
		return err
	}
	lvSize, err := lvmutils.GetLvSize(diskPath)
	if err != nil {
		return err
	}

	tmpVolume := lvName + "-convert.tmp"
	tmpVolumePath := path.Join("/dev", vgName, tmpVolume)
	// create /dev/vg/disk-convert.tmp
	if err := lvmutils.LvCreate(vgName, tmpVolume, lvSize); err != nil {
		return errors.Wrap(err, "delete snapshot LvCreate")
	}
	srcInfo := qemuimg.SImageInfo{
		Path:    diskPath,
		Format:  qemuImg.Format,
		IoLevel: qemuimg.IONiceNone,

		Password:      encryptInfo.Key,
		EncryptAlg:    encryptInfo.Alg,
		EncryptFormat: qemuimg.EncryptFormatLuks,
	}
	destInfo := qemuimg.SImageInfo{
		Path:    tmpVolumePath,
		Format:  qemuimgfmt.QCOW2,
		IoLevel: qemuimg.IONiceNone,

		Password:      encryptInfo.Key,
		EncryptAlg:    encryptInfo.Alg,
		EncryptFormat: qemuimg.EncryptFormatLuks,
	}
	// convert /dev/vg/disk to /dev/vg/disk-convert.tmp
	if err = qemuimg.Convert(srcInfo, destInfo, false, nil); err != nil {
		lvmutils.LvRemove(tmpVolumePath)
		return errors.Wrap(err, "failed convert tmp disk")
	}
	tmpVolume2 := lvName + "-convert.tmp2"
	tmpVolume2Path := path.Join("/dev", vgName, tmpVolume2)
	// rename /dev/vg/disk to /dev/vg/disk-convert.tmp2
	err = lvmutils.LvRename(vgName, diskPath, tmpVolume2)
	if err != nil {
		return errors.Wrap(err, "failed rename disk to tmp")
	}
	// rename /dev/vg/disk-convert.tmp to /dev/vg/disk
	err = lvmutils.LvRename(vgName, tmpVolume, diskPath)
	if err != nil {
		return errors.Wrap(err, "failed rename tmp to disk")
	}
	// delete /dev/vg/disk-convert.tmp2
	err = lvmutils.LvRemove(tmpVolume2Path)
	if err != nil {
		return errors.Wrap(err, "failed remove tmp disk")
	}
	return nil
}
