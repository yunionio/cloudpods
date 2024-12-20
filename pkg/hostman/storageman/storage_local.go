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
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/hostutils/kubelet"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/backupstorage"
	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/onecloud/pkg/util/zeroclean"
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

func (s *SLocalStorage) StorageType() string {
	return api.STORAGE_LOCAL
}

func (s *SLocalStorage) IsLocal() bool {
	return true
}

func (s *SLocalStorage) GetSnapshotDir() string {
	return path.Join(s.Path, _SNAPSHOT_PATH_)
}

func (s *SLocalStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return path.Join(s.GetSnapshotDir(), diskId+options.HostOptions.SnapshotDirSuffix, snapshotId)
}

func (s *SLocalStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	return fileutils2.Exists(s.GetSnapshotPathByIds(diskId, snapshotId)), nil
}

func (s *SLocalStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%d", s.Manager.host.GetMasterIp(), s.StorageType(), s.Index)
}

func (s *SLocalStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	err := doRestoreDisk(ctx, s, input, disk, disk.GetPath())
	if err != nil {
		return errors.Wrap(err, "doRestoreDisk")
	}
	/*info := input.DiskInfo
	backupDir := s.GetBackupDir()
	if !fileutils2.Exists(backupDir) {
		output, err := procutils.NewCommand("mkdir", "-p", backupDir).Output()
		if err != nil {
			return errors.Wrapf(err, "mkdir %s failed: %s", backupDir, output)
		}
	}
	backupPath := path.Join(s.GetBackupDir(), info.Backup.BackupId)
	if !fileutils2.Exists(backupPath) {
		_, err := s.storageBackupRecovery(ctx, &SStorageBackup{
			BackupId:                input.DiskInfo.Backup.BackupId,
			BackupStorageId:         input.DiskInfo.Backup.BackupStorageId,
			BackupStorageAccessInfo: input.DiskInfo.Backup.BackupStorageAccessInfo.Copy(),
		})
		if err != nil {
			return errors.Wrap(err, "unable to storageBackupRecovery")
		}
	}*/
	/*img, err := qemuimg.NewQemuImage(backupPath)
	if err != nil {
		log.Errorf("unable to new qemu image for %s: %s", backupPath, err.Error())
		return errors.Wrapf(err, "unable to new qemu image for %s", backupPath)
	}
	if info.Encryption {
		img.SetPassword(info.EncryptInfo.Key)
	}
	_, err = img.Clone(disk.GetLvPath(), qemuimg.QCOW2, false)*/
	/*img, err := qemuimg.NewQemuImage(disk.GetPath())
	if err != nil {
		log.Errorf("NewQemuImage fail %s %s", disk.GetPath(), err)
		return errors.Wrapf(err, "unable to new qemu image for %s", disk.GetPath())
	}
	var (
		encKey string
		encFmt qemuimg.TEncryptFormat
		encAlg seclib2.TSymEncAlg
	)
	if info.Encryption {
		encKey = info.EncryptInfo.Key
		encFmt = qemuimg.EncryptFormatLuks
		encAlg = info.EncryptInfo.Alg
	}
	err = img.CreateQcow2(0, false, backupPath, encKey, encFmt, encAlg)
	if err != nil {
		log.Errorf("CreateQcow2 fail %s", err)
		return errors.Wrapf(err, "CreateQcow2 %s fail", backupPath)
	}*/
	return nil
}

/*func (s *SLocalStorage) StorageBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sbParams := params.(*SStorageBackup)
	backupStorage, err := backupstorage.GetBackupStorage(sbParams.BackupStorageId, sbParams.BackupStorageAccessInfo)
	if err != nil {
		return nil, err
	}
	backupPath := path.Join(s.GetBackupDir(), sbParams.BackupId)
	err = backupStorage.SaveBackupFrom(ctx, backupPath, sbParams.BackupId)
	if err != nil {
		return nil, err
	}
	// remove local backup
	output, err := procutils.NewCommand("rm", backupPath).Output()
	if err != nil {
		log.Errorf("rm %s failed %s", backupPath, output)
		return nil, errors.Wrapf(err, "rm %s failed %s", backupPath, output)
	}
	return nil, nil
}*/

func (s *SLocalStorage) storageBackupRecovery(ctx context.Context, sbParams *SStorageBackup) (jsonutils.JSONObject, error) {
	backupStorage, err := backupstorage.GetBackupStorage(sbParams.BackupStorageId, sbParams.BackupStorageAccessInfo)
	if err != nil {
		return nil, err
	}
	backupPath := path.Join(s.GetBackupDir(), sbParams.BackupId)
	return nil, backupStorage.RestoreBackupTo(ctx, backupPath, sbParams.BackupId)
}

func (s *SLocalStorage) StorageBackupRecovery(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	sbParams := params.(*SStorageBackup)
	return s.storageBackupRecovery(ctx, sbParams)
}

func (s *SLocalStorage) GetAvailSizeMb() int {
	sizeMb := s.SBaseStorage.GetAvailSizeMb()
	kubeletConf := s.Manager.GetKubeletConfig()
	if kubeletConf == nil {
		return sizeMb
	}

	// available size should aware of kubelet hard eviction threshold
	hardThresholds := kubeletConf.GetEvictionConfig().GetHard()
	nodeFs := hardThresholds.GetNodeFsAvailable()
	imageFs := hardThresholds.GetImageFsAvailable()
	storageDev, err := kubelet.GetDirectoryMountDevice(s.GetPath())
	if err != nil {
		log.Errorf("Get directory %s mount device: %v", s.GetPath(), err)
		return sizeMb
	}

	usablePercent := 1.0
	if kubeletConf.HasDedicatedImageFs() {
		if storageDev == kubeletConf.GetImageFsDevice() {
			usablePercent = 1 - float64(imageFs.Value.Percentage)
			log.Infof("Storage %s and kubelet imageFs %s share same device %s", s.GetPath(), kubeletConf.GetImageFs(), storageDev)
		}
	} else {
		// nodeFs and imageFs use same device
		if storageDev == kubeletConf.GetNodeFsDevice() {
			maxPercent := math.Max(float64(nodeFs.Value.Percentage), float64(imageFs.Value.Percentage))
			usablePercent = 1 - maxPercent
			log.Infof("Storage %s and kubelet nodeFs share same device %s", s.GetPath(), storageDev)
		}
	}

	sizeMb = int(float64(sizeMb) * usablePercent)
	log.Infof("Storage %s sizeMb %d, usablePercent %f", s.GetPath(), sizeMb, usablePercent)

	return sizeMb
}

func (s *SLocalStorage) GetMediumType() (string, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("lsblk -d -o name,rota $(df -P %s | awk 'NR==2{print $1}') |  awk 'NR==2{print $2}'", s.GetPath())).Output()
	if err != nil {
		return api.DISK_TYPE_ROTATE, errors.Wrapf(err, "failed get medium type %s", out)
	}
	if strings.TrimSpace(string(out)) == "0" {
		return api.DISK_TYPE_SSD, nil
	} else {
		return api.DISK_TYPE_ROTATE, nil
	}
}

func (s *SLocalStorage) getHardwareInfo() (*api.StorageHardwareInfo, error) {
	cmd := fmt.Sprintf("df -P %s | awk 'NR==2 {print $1}'", s.GetPath())
	partName, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "execute command: %s", cmd)
	}
	partPath := strings.TrimSuffix(string(partName), "\n")
	sysPath := "/sys/class/block"

	getPartPathCmd := fmt.Sprintf("readlink -f %s", filepath.Join(sysPath, filepath.Base(partPath)))
	partRealPath, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", getPartPathCmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "get partition sys path: %s", getPartPathCmd)
	}
	partRealPathStr := strings.TrimSuffix(string(partRealPath), "\n")
	blockPath := filepath.Dir(partRealPathStr)
	devicePath := filepath.Join(blockPath, "device")
	modelPath := filepath.Join(devicePath, "model")

	errs := []error{}
	ret := &api.StorageHardwareInfo{}
	model, err := ioutil.ReadFile(modelPath)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "read model file: %s", modelPath))
	} else {
		modelStr := string(model)
		ret.Model = &modelStr
	}

	vendorPath := filepath.Join(devicePath, "vendor")
	vendor, err := ioutil.ReadFile(vendorPath)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "read vendor file: %s", vendorPath))
	} else {
		vendorStr := string(vendor)
		ret.Vendor = &vendorStr
	}

	return ret, errors.NewAggregate(errs)
}

func (s *SLocalStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	name := s.GetName(s.GetComposedName)
	content.Set("name", jsonutils.NewString(name))
	content.Set("capacity", jsonutils.NewInt(int64(s.GetAvailSizeMb())))
	content.Set("actual_capacity_used", jsonutils.NewInt(int64(s.GetUsedSizeMb())))
	content.Set("storage_type", jsonutils.NewString(s.StorageType()))
	content.Set("zone", jsonutils.NewString(s.GetZoneId()))
	if len(s.Manager.LocalStorageImagecacheManager.GetId()) > 0 {
		content.Set("storagecache_id",
			jsonutils.NewString(s.Manager.LocalStorageImagecacheManager.GetId()))
	}

	hardwareInfo, err := s.getHardwareInfo()
	if err != nil {
		log.Warningf("get hardware info: storage: %s, %v", name, err)
	}
	if hardwareInfo != nil {
		content.Set("hardware_info", jsonutils.Marshal(hardwareInfo))
	}

	var (
		res jsonutils.JSONObject
	)

	log.Infof("Sync storage info %s/%s", s.StorageId, name)

	if len(s.StorageId) > 0 {
		res, err = modules.Storages.Put(
			hostutils.GetComputeSession(context.Background()),
			s.StorageId, content)
		if err != nil {
			log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
			return nil, errors.Wrapf(err, "Storages.Put %s", s.StorageId)
		}
	} else {
		res, err = modules.Storages.GetByName(hostutils.GetComputeSession(context.Background()), name, nil)
		if err == nil {
			return res, nil
		}

		var mediumType string
		mediumType, err = s.GetMediumType()
		if err != nil {
			log.Errorf("failed get medium type %s %s", s.GetPath(), err)
		} else {
			content.Set("medium_type", jsonutils.NewString(mediumType))
		}

		res, err = modules.Storages.Create(
			hostutils.GetComputeSession(context.Background()), content)
		if err != nil {
			log.Errorf("SyncStorageInfo Failed: %s: %s", content, err)
			return nil, errors.Wrapf(err, "Storages.Create %s", content)
		}
	}

	return res, nil
}

func (s *SLocalStorage) GetDiskById(diskId string) (IDisk, error) {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	for i := 0; i < len(s.Disks); i++ {
		if s.Disks[i].GetId() == diskId {
			return s.Disks[i], s.Disks[i].Probe()
		}
	}
	var disk = NewLocalDisk(s, diskId)
	err := disk.Probe()
	if err == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "probe: %s", err)
}

func (s *SLocalStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewLocalDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SLocalStorage) Accessible() error {
	var c = make(chan error)
	go func() {
		if !fileutils2.Exists(s.Path) {
			if err := procutils.NewCommand("mkdir", "-p", s.Path).Run(); err != nil {
				c <- err
				return
			}
		}
		if !fileutils2.IsDir(s.Path) {
			c <- fmt.Errorf("path %s isn't directory", s.Path)
			return
		}
		if err := s.BindMountStoragePath(s.Path); err != nil {
			c <- err
			return
		}
		if !fileutils2.Writable(s.Path) {
			c <- fmt.Errorf("dir %s not writable", s.Path)
			return
		}
		c <- nil
	}()
	var err error
	select {
	case err = <-c:
		break
	case <-time.After(time.Second * 10):
		err = ErrStorageTimeout
	}
	return err

}

func (s *SLocalStorage) Detach() error {
	return nil
}

func (s *SLocalStorage) deleteBackendFile(diskpath string, skipRecycle bool) error {
	backendPath := diskpath + ".backend"
	if !fileutils2.Exists(backendPath) {
		return nil
	}
	disk, err := qemuimg.NewQemuImage(diskpath)
	if err != nil {
		return errors.Wrapf(err, "qemuimg.NewQemuImage(%s)", diskpath)
	}
	if disk.BackFilePath != backendPath {
		return nil
	}

	destDir := s.getRecyclePath()
	if options.HostOptions.RecycleDiskfile && (!skipRecycle || options.HostOptions.AlwaysRecycleDiskfile) {
		if err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
			log.Errorf("Fail to mkdir %s for recycle: %s", destDir, err)
			return err
		}
		backendDestFile := fmt.Sprintf("%s.%d", path.Base(backendPath), time.Now().Unix())
		log.Infof("Move deleted disk file %s to recycle %s", backendPath, destDir)
		return procutils.NewCommand("mv", "-f", backendPath, path.Join(destDir, backendDestFile)).Run()
	} else {
		log.Infof("Delete disk file %s immediately", backendPath)
		if options.HostOptions.ZeroCleanDiskData {
			// try to zero clean files in subdir
			err := zeroclean.ZeroDir(backendPath)
			if err != nil {
				log.Errorf("zeroclean disk %s fail %s", backendPath, err)
			} else {
				log.Debugf("zeroclean disk %s success!", backendPath)
			}
		}
		return procutils.NewCommand("rm", "-rf", backendPath).Run()
	}
}

func (s *SLocalStorage) DeleteDiskfile(diskpath string, skipRecycle bool) error {
	log.Infof("Start Delete %s", diskpath)

	if err := s.deleteBackendFile(diskpath, skipRecycle); err != nil {
		return err
	}

	if options.HostOptions.RecycleDiskfile && (!skipRecycle || options.HostOptions.AlwaysRecycleDiskfile) {
		var (
			destDir  = s.getRecyclePath()
			destFile = fmt.Sprintf("%s.%d", path.Base(diskpath), time.Now().Unix())
		)
		if err := procutils.NewCommand("mkdir", "-p", destDir).Run(); err != nil {
			log.Errorf("Fail to mkdir %s for recycle: %s", destDir, err)
			return err
		}

		log.Infof("Move deleted disk file %s to recycle %s", diskpath, destDir)
		return procutils.NewCommand("mv", "-f", diskpath, path.Join(destDir, destFile)).Run()
	} else {
		log.Infof("Delete disk file %s immediately", diskpath)
		if options.HostOptions.ZeroCleanDiskData {
			// try to zero clean files in subdir
			err := zeroclean.ZeroDir(diskpath)
			if err != nil {
				log.Errorf("zeroclean disk %s fail %s", diskpath, err)
			} else {
				log.Debugf("zeroclean disk %s success!", diskpath)
			}
		}
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
	info, ok := params.(SStorageSaveToGlanceInfo)
	if !ok {
		return nil, hostutils.ParamsError
	}
	data := info.DiskInfo

	var (
		imageId, _   = data.GetString("image_id")
		imagePath, _ = data.GetString("image_path")
		compress     = jsonutils.QueryBoolean(data, "compress", true)
		format, _    = data.GetString("format")
		encKeyId, _  = data.GetString("encrypt_key_id")
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

	if err := s.saveToGlance(ctx, imageId, imagePath, compress, format, encKey, encFormat, encAlg); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		s.onSaveToGlanceFailed(ctx, imageId, err.Error())
		return nil, errors.Wrap(err, "saveToGlance")
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
			imagecacheManager.GetId(), imageId, "active", dstPath)
		if err != nil {
			log.Errorf("Fail to remote cache image: %s", err)
		}
	}
	return nil, nil
}

func (s *SLocalStorage) saveToGlance(ctx context.Context, imageId, imagePath string,
	compress bool, format string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
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
		return err
	}

	if compress {
		origin, err := qemuimg.NewQemuImage(imagePath)
		if err != nil {
			log.Errorln(err)
			return err
		}
		if len(encryptKey) > 0 {
			origin.SetPassword(encryptKey)
		}
		if len(format) == 0 {
			format = options.HostOptions.DefaultImageSaveFormat
		}
		if format == "qcow2" {
			if err := origin.Convert2Qcow2(true, encryptKey, encFormat, encAlg); err != nil {
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

	_, err = image.Images.Upload(hostutils.GetImageSession(ctx),
		params, f, size)
	return err
}

func (s *SLocalStorage) onSaveToGlanceFailed(ctx context.Context, imageId string, reason string) {
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

func (s *SLocalStorage) CreateSnapshotFormUrl(
	ctx context.Context, snapshotUrl, diskId, snapshotPath string,
) error {
	remoteFile := remotefile.NewRemoteFile(ctx, snapshotUrl, snapshotPath,
		false, "", -1, nil, "", "")
	err := remoteFile.Fetch(nil)
	return errors.Wrapf(err, "fetch snapshot from %s", snapshotUrl)
}

func (s *SLocalStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SStorageDeleteSnapshots)
	if !ok {
		return nil, hostutils.ParamsError
	}
	snapshotDir := path.Join(s.GetSnapshotDir(), input.DiskId+options.HostOptions.SnapshotDirSuffix)
	output, err := procutils.NewCommand("rm", "-rf", snapshotDir).Output()
	if err != nil {
		return nil, fmt.Errorf("Delete snapshot dir failed: %s", output)
	}
	return nil, nil
}

func (s *SLocalStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SStorageDeleteSnapshot)
	if !ok {
		return nil, hostutils.ParamsError
	}

	log.Errorf("input %s", jsonutils.Marshal(input))
	snapshotDir := path.Join(s.GetSnapshotDir(), input.DiskId+options.HostOptions.SnapshotDirSuffix)
	diskPath := path.Join(s.GetPath(), input.DiskId)
	err := DeleteLocalSnapshot(snapshotDir, input.SnapshotId, diskPath, input.ConvertSnapshot, input.BlockStream)
	if err != nil {
		return nil, err
	}
	res := jsonutils.NewDict()
	res.Set("deleted", jsonutils.JSONTrue)
	return res, nil
}

func DeleteLocalSnapshot(snapshotDir, snapshotId, diskPath, convertSnapshot string, blockStream bool) error {
	//snapshotDir := d.GetSnapshotDir()
	snapshotPath := path.Join(snapshotDir, snapshotId)
	if blockStream {
		//diskPath := d.getPath()
		output := diskPath + ".tmp"
		if fileutils2.Exists(output) {
			procutils.NewCommand("rm", "-f", output).Run()
		}
		img, err := qemuimg.NewQemuImage(diskPath)
		if err != nil {
			return errors.Wrap(err, "NewQemuImage")
		}
		if err = img.Convert2Qcow2To(output, false, "", "", ""); err != nil {
			log.Errorf("convert image %s to %s: %s", img.Path, output, err)
			procutils.NewCommand("rm", "-f", output).Run()
			return err
		}
		if err = procutils.NewCommand("rm", "-f", diskPath).Run(); err != nil {
			log.Errorf("rm convert disk file %s: %s", diskPath, err)
			return err
		}
		if err = procutils.NewCommand("mv", "-f", output, diskPath).Run(); err != nil {
			log.Errorf("mv disk file %s to %s: %s", output, diskPath, err)
			return err
		}
	} else if len(convertSnapshot) > 0 {
		if !fileutils2.Exists(snapshotDir) {
			err := procutils.NewCommand("mkdir", "-p", snapshotDir).Run()
			if err != nil {
				log.Errorln(err)
				return err
			}
		}
		convertSnapshotPath := path.Join(snapshotDir, convertSnapshot)
		output := convertSnapshotPath + ".tmp"
		if fileutils2.Exists(output) {
			procutils.NewCommand("rm", "-f", output).Run()
		}
		img, err := qemuimg.NewQemuImage(convertSnapshotPath)
		if err != nil {
			return errors.Wrap(err, "NewQemuImage")
		}
		if err = img.Convert2Qcow2To(output, false, "", "", ""); err != nil {
			log.Errorf("convert image %s to %s: %s", img.Path, output, err)
			procutils.NewCommand("rm", "-f", output).Run()
			return err
		}
		if err = procutils.NewCommand("rm", "-f", convertSnapshotPath).Run(); err != nil {
			log.Errorf("rm convert snapshot file %s: %s", convertSnapshotPath, err)
			return err
		}
		if err = procutils.NewCommand("mv", "-f", output, convertSnapshotPath).Run(); err != nil {
			log.Errorf("mv snapshot file %s to %s: %s", output, convertSnapshotPath, err)
			return err
		}
	}
	if fileutils2.Exists(snapshotPath) {
		out, err := procutils.NewCommand("rm", "-f", snapshotPath).Output()
		if err != nil {
			log.Errorf("rm snapshot file: %s %s", out, err)
			return errors.Wrap(err, "rm snapshot file")
		}
	}
	return nil
}

func (s *SLocalStorage) DestinationPrepareMigrate(
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
	// prepare disk snapshot dir
	if len(snapshots) > 0 && !fileutils2.Exists(disk.GetSnapshotDir()) {
		output, err := procutils.NewCommand("mkdir", "-p", disk.GetSnapshotDir()).Output()
		if err != nil {
			return errors.Wrapf(err, "mkdir %s failed: %s", disk.GetSnapshotDir(), output)
		}
	}

	// create snapshots form remote url
	var (
		diskStorageId = diskinfo.StorageId
		baseImagePath string
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
		if i == 0 && len(templateId) > 0 && sysDiskHasTemplate {
			templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
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
		snapshotPath := path.Join(disk.GetSnapshotDir(), snapId)
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
	if rebaseDisks && len(templateId) > 0 && len(baseImagePath) == 0 && sysDiskHasTemplate {
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

func doRebaseDisk(diskPath, newBasePath string, encInfo *apis.SEncryptInfo) error {
	img, err := qemuimg.NewQemuImage(diskPath)
	if err != nil {
		return errors.Wrap(err, "failed open disk as qemu image")
	}
	if encInfo != nil {
		img.SetPassword(encInfo.Key)
	}
	if err = img.Rebase(newBasePath, true); err != nil {
		return errors.Wrap(err, "failed rebase disk backing file")
	}
	log.Infof("rebase disk %s backing file to %s ", diskPath, newBasePath)
	return nil
}

func (s *SLocalStorage) DiskMigrate(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SDiskMigrate)

	disk := s.CreateDisk(input.DiskId)
	snapshots := input.SnapsChain
	diskOutChainSnaps := input.OutChainSnaps
	// prepare disk snapshot dir
	if len(snapshots) > 0 && !fileutils2.Exists(disk.GetSnapshotDir()) {
		output, err := procutils.NewCommand("mkdir", "-p", disk.GetSnapshotDir()).Output()
		if err != nil {
			return nil, errors.Wrapf(err, "mkdir %s failed: %s", disk.GetSnapshotDir(), output)
		}
	}

	baseImagePath := ""
	templateId := input.TemplateId
	for i, snapshotId := range snapshots {
		snapId, _ := snapshotId.GetString()
		snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
			input.SnapshotsUri, input.SrcStorageId, input.DiskId, snapId)
		snapshotPath := path.Join(disk.GetSnapshotDir(), snapId)
		log.Infof("Disk %s snapshot %s url: %s", input.DiskId, snapId, snapshotUrl)
		if err := s.CreateSnapshotFormUrl(ctx, snapshotUrl, input.DiskId, snapshotPath); err != nil {
			return nil, errors.Wrap(err, "create from snapshot url failed")
		}
		if i == 0 && len(templateId) > 0 && input.SysDiskHasTemplate {
			templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
			if err := doRebaseDisk(snapshotPath, templatePath, nil); err != nil {
				return nil, err
			}
		} else if len(baseImagePath) > 0 {
			if err := doRebaseDisk(snapshotPath, baseImagePath, nil); err != nil {
				return nil, err
			}
		}
		baseImagePath = snapshotPath
	}

	for _, snapshotId := range diskOutChainSnaps {
		snapId, _ := snapshotId.GetString()
		snapshotUrl := fmt.Sprintf("%s/%s/%s/%s",
			input.SnapshotsUri, input.SrcStorageId, input.DiskId, snapId)
		snapshotPath := path.Join(disk.GetSnapshotDir(), snapId)
		log.Infof("Disk %s snapshot %s url: %s", input.DiskId, snapId, snapshotUrl)
		if err := s.CreateSnapshotFormUrl(ctx, snapshotUrl, input.DiskId, snapshotPath); err != nil {
			return nil, errors.Wrap(err, "create from snapshot url failed")
		}
	}

	// download disk form remote url
	diskUrl := fmt.Sprintf("%s/%s/%s", input.DiskUri, input.SrcStorageId, input.DiskId)
	err := disk.CreateFromUrl(ctx, diskUrl, 0, func(progress, progressMbps float64, totalSizeMb int64) {
		log.Debugf("[%.2f / %d] disk %s create %.2f with speed %.2fMbps",
			progress*float64(totalSizeMb)/100, totalSizeMb, disk.GetId(), progress, progressMbps)
	})
	if err != nil {
		return nil, errors.Wrap(err, "CreateFromUrl")
	}
	if len(templateId) > 0 && len(baseImagePath) == 0 {
		templatePath := path.Join(storageManager.LocalStorageImagecacheManager.GetPath(), templateId)
		if err := doRebaseDisk(disk.GetPath(), templatePath, nil); err != nil {
			return nil, err
		}
	} else if len(baseImagePath) > 0 {
		if err := doRebaseDisk(disk.GetPath(), baseImagePath, nil); err != nil {
			return nil, err
		}
	}

	res := jsonutils.NewDict()
	res.Set("disk_path", jsonutils.NewString(disk.GetPath()))
	return nil, nil
}

func (s *SLocalStorage) CreateDiskFromSnapshot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
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
	return nil, httperrors.NewUnsupportOperationError("Unsupport protocol %s for Local storage", info.Protocol)
}

func (s *SLocalStorage) CreateDiskFromExistingPath(
	ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo,
) error {
	err := os.Link(input.DiskInfo.ExistingPath, disk.GetPath())
	if err != nil {
		return errors.Wrap(err, "os.link")
	}
	return nil
}

func (s *SLocalStorage) GetCloneTargetDiskPath(ctx context.Context, targetDiskId string) string {
	return path.Join(s.GetPath(), targetDiskId)
}

func (s *SLocalStorage) CloneDiskFromStorage(
	ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool,
) (*hostapi.ServerCloneDiskFromStorageResponse, error) {
	srcDiskPath := srcDisk.GetPath()
	srcImg, err := qemuimg.NewQemuImage(srcDiskPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Get source image %q info", srcDiskPath)
	}

	// start create target disk. if full copy is false, just create
	// empty target disk with same size and format
	accessPath := s.GetCloneTargetDiskPath(ctx, targetDiskId)
	if fullCopy {
		_, err = srcImg.Clone(s.GetCloneTargetDiskPath(ctx, targetDiskId), qemuimgfmt.QCOW2, false)
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

func (s *SLocalStorage) CleanRecycleDiskfiles(ctx context.Context) {
	cleanDailyFiles(s.Path, _RECYCLE_BIN_, options.HostOptions.RecycleDiskfileKeepDays)
	cleanDailyFiles(s.Path, _IMGSAVE_BACKUPS_, options.HostOptions.RecycleDiskfileKeepDays)
}
