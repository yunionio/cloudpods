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
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

var (
	_FUSE_MOUNT_PATH_ = "fusemnt"
	_FUSE_TMP_PATH_   = "fusetmp"
)

type SLocalStorage struct {
	SBaseStorage

	Index int
}

func NewLocalStorage(manager *SStorageManager, path string, index int) *SLocalStorage {
	var ret = new(SLocalStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.Index = index
	return ret
}

func (s *SLocalStorage) GetFuseTmpPath() string {
	return path.Join(s.Path, _FUSE_TMP_PATH_)
}

func (s *SLocalStorage) GetFuseMountPath() string {
	return path.Join(s.Path, _FUSE_MOUNT_PATH_)
}

func (s *SLocalStorage) StorageType() string {
	return api.STORAGE_LOCAL
}

func (s *SLocalStorage) GetSnapshotDir() string {
	return path.Join(s.Path, _SNAPSHOT_PATH_)
}

func (s *SLocalStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return path.Join(s.GetSnapshotDir(), diskId+options.HostOptions.SnapshotDirSuffix, snapshotId)
}

func (s *SLocalStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%d", s.Manager.host.GetMasterIp(), s.StorageType(), s.Index)
}

func (s *SLocalStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	content.Set("name", jsonutils.NewString(s.GetName(s.GetComposedName)))
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("medium_type", jsonutils.NewString(s.GetMediumType()))
	content.Set("zone", jsonutils.NewString(s.GetZoneName()))
	if len(s.Manager.LocalStorageImagecacheManager.GetId()) > 0 {
		content.Set("storagecache_id",
			jsonutils.NewString(s.Manager.LocalStorageImagecacheManager.GetId()))
	}
	var (
		err error
		res jsonutils.JSONObject
	)

	log.Infof("Sync storage info %s", s.StorageId)

	if len(s.StorageId) > 0 {
		res, err = modules.Storages.Put(
			hostutils.GetComputeSession(context.Background()),
			s.StorageId, content)
	} else {
		res, err = modules.Storages.Create(
			hostutils.GetComputeSession(context.Background()), content)
	}
	if err != nil {
		log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
	}
	return res, err
}

func (s *SLocalStorage) GetDiskById(diskId string) IDisk {
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
	var disk = NewLocalDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk
	} else {
		return nil
	}
}

func (s *SLocalStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewLocalDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SLocalStorage) Accessible() bool {
	if !fileutils2.Exists(s.Path) {
		if err := procutils.NewCommand("mkdir", "-p", s.Path).Run(); err != nil {
			log.Errorln(err)
		}
	}
	if fileutils2.IsDir(s.Path) && fileutils2.Writable(s.Path) {
		return true
	} else {
		return false
	}
}

func (s *SLocalStorage) DeleteDiskfile(diskpath string) error {
	log.Infof("Start Delete %s", diskpath)
	if options.HostOptions.RecycleDiskfile {
		var (
			destDir  = s.getRecyclePath()
			destFile = fmt.Sprintf("%s.%d", path.Base(diskpath), time.Now().Unix())
		)
		if err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
			return err
		}
		return procutils.NewCommand("mv", "-f", diskpath, path.Join(destDir, destFile)).Run()
	} else {
		return procutils.NewCommand("rm", "-rf", diskpath).Run()
	}
}

func (s *SLocalStorage) getRecyclePath() string {
	return s.getSubdirPath(_RECYCLE_BIN_)
}

func (s *SLocalStorage) getSubdirPath(subdir string) string {
	spath := path.Join(s.Path, subdir)
	today := timeutils.CompactTime(time.Now())
	return path.Join(spath, today)
}

func (s *SLocalStorage) GetImgsaveBackupPath() string {
	return s.getSubdirPath(_IMGSAVE_BACKUPS_)
}

func (s *SLocalStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var (
		imageId, _   = data.GetString("image_id")
		imagePath, _ = data.GetString("image_path")
		compress     = jsonutils.QueryBoolean(data, "compress", true)
		format, _    = data.GetString("format")
	)

	if err := s.saveToGlance(ctx, imageId, imagePath, compress, format); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		s.onSaveToGlanceFailed(ctx, imageId)
	}

	imagecacheManager := s.Manager.LocalStorageImagecacheManager
	if len(imagecacheManager.GetId()) > 0 {
		return nil, procutils.NewCommand("rm", "-f", imagePath).Run()
	} else {
		dstPath := path.Join(imagecacheManager.GetPath(), imageId)
		if err := procutils.NewCommand("mv", imagePath, dstPath).Run(); err != nil {
			log.Errorf("Fail to move saved image to cache: %s", err)
		}
		imagecacheManager.LoadImageCache(imageId)
		_, err := hostutils.RemoteStoragecacheCacheImage(ctx,
			imagecacheManager.GetId(), imageId, "ready", dstPath)
		if err != nil {
			log.Errorf("Fail to remote cache image: %s", err)
		}
	}
	return nil, nil
}

func (s *SLocalStorage) saveToGlance(ctx context.Context, imageId, imagePath string,
	compress bool, format string) error {
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskPath: imagePath, Compress: compress})
	if err != nil {
		return err
	}

	if compress {
		origin, err := qemuimg.NewQemuImage(imagePath)
		if err != nil {
			log.Errorln(err)
			return err
		}
		if len(format) == 0 {
			format = options.HostOptions.DefaultImageSaveFormat
		}
		if format == "qcow2" {
			if err := origin.Convert2Qcow2(true); err != nil {
				log.Errorln(err)
				return err
			}
		} else {
			if err := origin.Convert2Vmdk(true); err != nil {
				log.Errorln(err)
				return err
			}
		}
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer f.Close()
	finfo, err := f.Stat()
	if err != nil {
		return err
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

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx, s.GetZoneName()),
		params, f, size)
	return err
}

func (s *SLocalStorage) onSaveToGlanceFailed(ctx context.Context, imageId string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	_, err := modules.Images.Update(hostutils.GetImageSession(ctx, s.GetZoneName()),
		imageId, params)
	if err != nil {
		log.Errorln(err)
	}
}

func (s *SLocalStorage) CreateSnapshotFormUrl(
	ctx context.Context, snapshotUrl, diskId, snapshotPath string,
) error {
	remoteFile := remotefile.NewRemoteFile(ctx, snapshotUrl, snapshotPath,
		false, "", -1, nil, "", "")
	if remoteFile.Fetch() {
		return nil
	} else {
		return fmt.Errorf("Fail to fetch snapshot from %s", snapshotUrl)
	}
}

func (s *SLocalStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	snapshotDir := path.Join(s.GetSnapshotDir(), diskId+options.HostOptions.SnapshotDirSuffix)
	output, err := procutils.NewCommand("rm", "-rf", snapshotDir).Output()
	if err != nil {
		return nil, fmt.Errorf("Delete snapshot dir failed: %s", output)
	}
	return nil, nil
}

func (s *SLocalStorage) DestinationPrepareMigrate(
	ctx context.Context, liveMigrate bool, disksUri string, snapshotsUri string,
	desc, disksBackingFile, srcSnapshots jsonutils.JSONObject, rebaseDisks bool) error {
	disks, _ := desc.GetArray("disks")
	for i, diskinfo := range disks {
		var (
			diskId, _    = diskinfo.GetString("disk_id")
			snapshots, _ = srcSnapshots.GetArray(diskId)
			disk         = s.CreateDisk(diskId)
		)

		if disk == nil {
			return fmt.Errorf(
				"Storage %s create disk %s failed", s.GetId(), diskId)
		}

		templateId, _ := diskinfo.GetString("template_id")
		// prepare disk snapshot dir
		if len(snapshots) > 0 && !fileutils2.Exists(disk.GetSnapshotDir()) {
			_, err := procutils.NewCommand("mkdir", "-p", disk.GetSnapshotDir()).Output()
			if err != nil {
				return err
			}
		}

		// create snapshots form remote url
		var (
			diskStorageId, _ = diskinfo.GetString("storage_id")
			baseImagePath    string
		)
		for i, snapshotId := range snapshots {
			snapId, _ := snapshotId.GetString()
			snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
				snapshotsUri, diskStorageId, diskId, snapId)
			snapshotPath := path.Join(disk.GetSnapshotDir(), snapId)
			log.Infof("Disk %s snapshot %s url: %s", diskId, snapId, snapshotUrl)
			if err := s.CreateSnapshotFormUrl(ctx, snapshotUrl, diskId, snapshotPath); err != nil {
				return errors.Wrap(err, "create from snapshot url failed")
			}
			baseImagePath = snapshotPath
			if i == 0 && len(templateId) > 0 {
				templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
				if err := doRebaseDisk(snapshotPath, templatePath); err != nil {
					return err
				}
			} else if rebaseDisks {
				if err := doRebaseDisk(snapshotPath, baseImagePath); err != nil {
					return err
				}
			}
		}

		if liveMigrate {
			// create local disk
			backingFile, _ := disksBackingFile.GetString(diskId)
			size, _ := diskinfo.Int("size")
			_, err := disk.CreateRaw(ctx, int(size), "qcow2", "", false, "", backingFile)
			if err != nil {
				log.Errorln(err)
				return err
			}
		} else {
			// download disk form remote url
			diskUrl := fmt.Sprintf("%s/%s/%s", disksUri, diskStorageId, diskId)
			if err := disk.CreateFromUrl(ctx, diskUrl, 0); err != nil {
				log.Errorln(err)
				return err
			}
		}
		if rebaseDisks && len(templateId) > 0 && len(baseImagePath) == 0 {
			templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
			if err := doRebaseDisk(disk.GetPath(), templatePath); err != nil {
				return err
			}
		} else if rebaseDisks && len(baseImagePath) > 0 {
			if err := doRebaseDisk(disk.GetPath(), baseImagePath); err != nil {
				return err
			}
		}
		diskDesc, _ := disks[i].(*jsonutils.JSONDict)
		diskDesc.Set("path", jsonutils.NewString(disk.GetPath()))
	}
	return nil
}

func doRebaseDisk(diskPath, newBasePath string) error {
	img, err := qemuimg.NewQemuImage(diskPath)
	if err != nil {
		return errors.Wrap(err, "failed open disk as qemu image")
	}
	if err = img.Rebase(newBasePath, true); err != nil {
		return errors.Wrap(err, "failed rebase disk backing file")
	}
	log.Infof("rebase disk %s backing file to %s ", diskPath, newBasePath)
	return nil
}

func (s *SLocalStorage) CreateDiskFromSnapshot(
	ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo,
) error {
	var (
		snapshotUrl, _      = createParams.DiskInfo.GetString("snapshot_url")
		transferProtocol, _ = createParams.DiskInfo.GetString("protocol")
		diskSize, _         = createParams.DiskInfo.Int("size")
	)
	if transferProtocol == "fuse" {
		if err := disk.CreateFromImageFuse(ctx, snapshotUrl, diskSize); err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("Unsupport protocol %s for Local storage", transferProtocol)
	}
}
