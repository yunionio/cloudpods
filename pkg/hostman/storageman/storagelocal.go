package storageman

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/pkg/util/timeutils"
)

var (
	_FUSE_MOUNT_PATH_ = "fusemnt"
	_FUSE_TMP_PATH_   = "fusetmp"
	_SNAPSHOT_PATH_   = "snapshots"
)

type SLocalStorage struct {
	SBaseStorage
}

func NewLocalStorage(manager *SStorageManager, path string) *SLocalStorage {
	var ret = new(SLocalStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.StartSnapshotRecycle()
	return ret
}

func (s *SLocalStorage) GetFuseTmpPath() string {
	return path.Join(s.Path, _FUSE_TMP_PATH_)
}

func (s *SLocalStorage) GetFuseMountPath() string {
	return path.Join(s.Path, _FUSE_MOUNT_PATH_)
}

func (s *SLocalStorage) StorageType() string {
	return storagetypes.STORAGE_LOCAL
}

func (s *SLocalStorage) GetSnapshotDir() string {
	return path.Join(s.Path, _SNAPSHOT_PATH_)
}

func (s *SLocalStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return path.Join(s.GetSnapshotDir(),
		diskId+options.HostOptions.SnapshotDirSuffix, snapshotId)
}

func (s *SLocalStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	content.Set("name", jsonutils.NewString(s.StorageName))
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("medium_type", jsonutils.NewString(s.GetMediumType()))
	content.Set("zone", jsonutils.NewString(s.GetZone()))
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
	return NewLocalDisk(s, diskId)
}

func (s *SLocalStorage) StartSnapshotRecycle() {
	//TODO
}

func (s *SLocalStorage) Accessible() bool {
	if !fileutils2.Exists(s.Path) {
		if _, err := procutils.NewCommand("mkdir", "-p", s.Path).Run(); err != nil {
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
	if options.HostOptions.RecycleDiskfile {
		var (
			destDir  = s.getRecyclePath()
			destFile = fmt.Sprintf("%s.%d", path.Base(diskpath), time.Now().Unix())
		)
		if _, err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
			return err
		}
		_, err := procutils.NewCommand("mv", "-f", diskpath, path.Join(destDir, destFile)).Run()
		return err
	} else {
		_, err := procutils.NewCommand("rm", "-rf", diskpath).Run()
		return err
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
		_, err := procutils.NewCommand("rm", "-f", imagePath).Run()
		return nil, err
	} else {
		dstPath := path.Join(imagecacheManager.GetPath(), imageId)
		if _, err := procutils.NewCommand("mv", imagePath, dstPath).Run(); err != nil {
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
	var (
		kvmDisk = NewKVMGuestDisk(imagePath)
		osInfo  string
		relInfo *fsdriver.SReleaseInfo
	)

	if err := func() error {
		if kvmDisk.Connect() {
			defer kvmDisk.Disconnect()

			if root := kvmDisk.Mount(); root != nil {
				defer kvmDisk.Umount(root)

				osInfo = root.GetOs()
				relInfo = root.GetReleaseInfo(root.GetPartition())
				if compress {
					if err := root.PrepareFsForTemplate(root.GetPartition()); err != nil {
						log.Errorln(err)
						return err
					}
				}
			}

			if compress {
				kvmDisk.Zerofree()
			}
		}
		return nil
	}(); err != nil {
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
	if len(osInfo) > 0 {
		params.Set("os_type", jsonutils.NewString(osInfo))
	}
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

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx, s.GetZone()),
		params, f, size)
	// TODO
	// notify_template_ready
	return err
}

func (s *SLocalStorage) onSaveToGlanceFailed(ctx context.Context, imageId string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	_, err := modules.Images.Update(hostutils.GetImageSession(ctx, s.GetZone()),
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
	output, err := procutils.NewCommand("rm", "-rf", snapshotDir).Run()
	if err != nil {
		return nil, fmt.Errorf("Delete snapshot dir failed: %s", output)
	}
	return nil, nil
}
