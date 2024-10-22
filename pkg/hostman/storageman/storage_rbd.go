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
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/cephutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

const (
	RBD_FEATURE = 3
	RBD_ORDER   = 22 //为rbd对应到rados中每个对象的大小，默认为4MB
)

type sStorageConf struct {
	MonHost string
	Key     string
	Pool    string
	// https://docs.ceph.com/en/latest/rados/configuration/msgr2/
	// The messenger v2 protocol, or msgr2, is the second major
	// revision on Ceph’s on-wire protocol.
	EnableMessengerV2  bool
	RadosMonOpTimeout  int64
	RadosOsdOpTimeout  int64
	ClientMountTimeout int64
}

type SRbdStorage struct {
	SBaseStorage
	sStorageConf
}

func NewRBDStorage(manager *SStorageManager, path string) *SRbdStorage {
	var ret = new(SRbdStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
	ret.sStorageConf = sStorageConf{}
	return ret
}

type SRbdStorageFactory struct {
}

func (factory *SRbdStorageFactory) NewStorage(manager *SStorageManager, mountPoint string) IStorage {
	return NewRBDStorage(manager, mountPoint)
}

func (factory *SRbdStorageFactory) StorageType() string {
	return api.STORAGE_RBD
}

func init() {
	registerStorageFactory(&SRbdStorageFactory{})
}

func (s *SRbdStorage) StorageType() string {
	return api.STORAGE_RBD
}

func (s *SRbdStorage) IsLocal() bool {
	return false
}

func (s *SRbdStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
}

func (s *SRbdStorage) getCephClient(pool string) (*cephutils.CephClient, error) {
	if pool == "" {
		pool = s.Pool
	}
	return cephutils.NewClient(s.MonHost, s.Key, pool, s.EnableMessengerV2)
}

func (s *SRbdStorage) getClient() (*cephutils.CephClient, error) {
	return s.getCephClient("")
}

func (s *SRbdStorage) IsSnapshotExist(diskId, snapshotId string) (bool, error) {
	client, err := s.getClient()
	if err != nil {
		return false, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(diskId)
	if err != nil {
		return false, errors.Wrapf(err, "GetImage")
	}
	return img.IsSnapshotExist(snapshotId)
}

func (s *SRbdStorage) GetSnapshotDir() string {
	return ""
}

func (s *SRbdStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SRbdStorage) GetFuseMountPath() string {
	return ""
}

func (s *SRbdStorage) GetImgsaveBackupPath() string {
	return ""
}

// Tip Configuration values containing :, @, or = can be escaped with a leading \ character.
func (s *SRbdStorage) getStorageConfString() string {
	conf := []string{}
	conf = append(conf, "mon_host="+strings.ReplaceAll(s.MonHost, ",", `\;`))
	key := s.Key
	if len(key) > 0 {
		for _, k := range []string{":", "@", "="} {
			key = strings.ReplaceAll(key, k, fmt.Sprintf(`\%s`, k))
		}
		conf = append(conf, "key="+key)
	}
	for k, timeout := range map[string]int64{
		"rados_mon_op_timeout": s.RadosMonOpTimeout,
		"rados_osd_op_timeout": s.RadosOsdOpTimeout,
		"client_mount_timeout": s.ClientMountTimeout,
	} {
		conf = append(conf, fmt.Sprintf("%s=%d", k, timeout))
	}
	return ":" + strings.Join(conf, ":")
}

func (s *SRbdStorage) listImages() ([]string, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	return client.ListImages()
}

func (s *SRbdStorage) IsImageExist(name string) (bool, error) {
	images, err := s.listImages()
	if err != nil {
		return false, errors.Wrapf(err, "listImages")
	}
	if utils.IsInStringArray(name, images) {
		return true, nil
	}
	return false, nil
}

func (s *SRbdStorage) getImageSizeMb(name string) (uint64, error) {
	client, err := s.getClient()
	if err != nil {
		return 0, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(name)
	if err != nil {
		return 0, errors.Wrapf(err, "GetImage")
	}
	info, err := img.GetInfo()
	if err != nil {
		return 0, errors.Wrapf(err, "GetInfo")
	}
	return uint64(info.SizeByte) / 1024 / 1024, nil
}

func (s *SRbdStorage) resizeImage(name string, sizeMb uint64) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(name)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	info, err := img.GetInfo()
	if err != nil {
		return errors.Wrapf(err, "img.GetInfo")
	}
	if uint64(info.SizeByte/1024/1024) >= sizeMb {
		return nil
	}
	return img.Resize(int64(sizeMb))
}

func (s *SRbdStorage) deleteImage(name string, skipRecycle bool) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(name)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "GetImage")
	}
	return img.Delete()
}

func (s *SRbdStorage) cloneImage(ctx context.Context, srcPool string, srcImage string, destPool string, destImage string) error {
	cli, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer cli.Close()
	srcCli := cli.Child(srcPool)

	img, err := srcCli.GetImage(srcImage)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	_, err = img.Clone(ctx, destPool, destImage)
	if err != nil {
		return errors.Wrap(err, "Clone")
	}
	return nil
}

func (s *SRbdStorage) cloneFromSnapshot(srcImage, srcPool, srcSnapshot, newImage, pool string) error {
	cli, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer cli.Close()

	srcCli := cli.Child(srcPool)

	img, err := srcCli.GetImage(srcImage)
	if err != nil {
		return errors.Wrapf(err, "GetImage(%s/%s)", srcPool, srcImage)
	}
	snap, err := img.GetSnapshot(srcSnapshot)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot(%s)", srcSnapshot)
	}
	_, err = snap.Clone(pool, newImage, true)
	if err != nil {
		return errors.Wrap(err, "Clone")
	}
	return nil
}

func (s *SRbdStorage) createImage(name string, sizeMb uint64) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	_, err = client.CreateImage(name, int64(sizeMb))
	return err
}

func (s *SRbdStorage) renameImage(pool string, src string, dest string) error {
	cli, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer cli.Close()

	srcCli := cli.Child(pool)
	img, err := srcCli.GetImage(src)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	return img.Rename(dest)
}

func (s *SRbdStorage) resetDisk(diskId string, snapshotId string) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(snapshotId)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot")
	}
	return snap.Rollback()
}

func (s *SRbdStorage) createBackup(ctx context.Context, diskId string, diskBackup *SDiskBackup) (int, error) {
	client, err := s.getClient()
	if err != nil {
		return 0, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()

	img, err := client.GetImage(diskId)
	if err != nil {
		return 0, errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(diskBackup.SnapshotId)
	if err != nil {
		return 0, errors.Wrapf(err, "unable to GetSnapshot %s of Image %s", diskBackup.SnapshotId, diskId)
	}
	backupName := fmt.Sprintf("backup_%s", diskBackup.BackupId)
	// no need to flattern, as image will be delete right after backup
	backupImg, err := snap.Clone(s.sStorageConf.Pool, backupName, false)
	if err != nil {
		return 0, errors.Wrapf(err, "unable to Clone snap %s", fmt.Sprintf("%s@%s", diskId, diskBackup.SnapshotId))
	}
	defer backupImg.Delete()

	srcPath := fmt.Sprintf("rbd:%s%s", backupImg.GetName(), s.getStorageConfString())

	size, err := doBackupDisk(ctx, srcPath, diskBackup)
	if err != nil {
		return 0, errors.Wrap(err, "doBackupDisk")
	}

	return size, nil
}

func (s *SRbdStorage) createSnapshot(diskId string, snapshotId string) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	_, err = img.CreateSnapshot(snapshotId)
	return err
}

func (s *SRbdStorage) deleteSnapshot(diskId string, snapshotId string) error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	img, err := client.GetImage(diskId)
	if err != nil {
		return errors.Wrapf(err, "GetImage")
	}
	snap, err := img.GetSnapshot(snapshotId)
	if err != nil {
		return errors.Wrapf(err, "GetSnapshot")
	}
	return snap.Delete()
}

func (s *SRbdStorage) SyncStorageSize() (api.SHostStorageStat, error) {
	stat := api.SHostStorageStat{
		StorageId: s.StorageId,
	}

	client, err := s.getClient()
	if err != nil {
		return stat, errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	capacity, err := client.GetCapacity()
	if err != nil {
		return stat, errors.Wrapf(err, "GetCapacity")
	}
	stat.CapacityMb = capacity.CapacitySizeKb / 1024
	stat.ActualCapacityUsedMb = capacity.UsedCapacitySizeKb / 1024
	return stat, nil
}

func (s *SRbdStorage) GetCapacityMb() int {
	capa, err := s.getRbdCapacity()
	if err != nil {
		return -1
	}
	return int(capa.CapacitySizeKb) / 1024
}

func (s *SRbdStorage) getRbdCapacity() (*cephutils.SCapacity, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, errors.Wrap(err, "getClient")
	}
	defer client.Close()
	capacity, err := client.GetCapacity()
	if err != nil {
		return nil, errors.Wrap(err, "GetCapacity")
	}
	return capacity, nil
}

func (s *SRbdStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := map[string]interface{}{}
	if len(s.StorageId) > 0 {
		capacity, err := s.getRbdCapacity()
		if err != nil {
			reason := jsonutils.Marshal(map[string]string{"reason": errors.Wrapf(err, "GetCapacity").Error()})
			return modules.Storages.PerformAction(hostutils.GetComputeSession(context.Background()), s.StorageId, api.STORAGE_OFFLINE, reason)
		}

		content = map[string]interface{}{
			"name":                 s.StorageName,
			"capacity":             capacity.CapacitySizeKb / 1024,
			"actual_capacity_used": capacity.UsedCapacitySizeKb / 1024,
			"status":               api.STORAGE_ONLINE,
			"zone":                 s.GetZoneId(),
		}
		return modules.Storages.Put(hostutils.GetComputeSession(context.Background()), s.StorageId, jsonutils.Marshal(content))
	}
	return modules.Storages.Get(hostutils.GetComputeSession(context.Background()), s.StorageName, jsonutils.Marshal(content))
}

func (s *SRbdStorage) GetDiskById(diskId string) (IDisk, error) {
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
	var disk = NewRBDDisk(s, diskId)
	if disk.Probe() == nil {
		s.Disks = append(s.Disks, disk)
		return disk, nil
	}
	return nil, errors.ErrNotFound
}

func (s *SRbdStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewRBDDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SRbdStorage) Accessible() error {
	client, err := s.getClient()
	if err != nil {
		return errors.Wrapf(err, "GetClient")
	}
	defer client.Close()
	_, err = client.GetCapacity()
	return err
}

func (s *SRbdStorage) Detach() error {
	return nil
}

func (s *SRbdStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	info, ok := params.(SStorageSaveToGlanceInfo)
	if !ok {
		return nil, hostutils.ParamsError
	}
	data := info.DiskInfo

	rbdImageCache := storageManager.GetStoragecacheById(s.GetStoragecacheId())
	if rbdImageCache == nil {
		return nil, fmt.Errorf("failed to find storage image cache for storage %s", s.GetStorageName())
	}

	imagePath, _ := data.GetString("image_path")
	compress := jsonutils.QueryBoolean(data, "compress", true)
	format, _ := data.GetString("format")
	imageId, _ := data.GetString("image_id")
	imageName := "image_cache_" + imageId
	if err := s.renameImage(rbdImageCache.GetPath(), imagePath, imageName); err != nil {
		return nil, err
	}

	imagePath = fmt.Sprintf("rbd:%s/%s%s", rbdImageCache.GetPath(), imageName, s.getStorageConfString())

	if err := s.saveToGlance(ctx, imageId, imagePath, compress, format); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		s.onSaveToGlanceFailed(ctx, imageId, err.Error())
	}

	rbdImageCache.LoadImageCache(imageId)
	_, err := hostutils.RemoteStoragecacheCacheImage(ctx, rbdImageCache.GetId(), imageId, "active", imagePath)
	if err != nil {
		log.Errorf("Fail to remote cache image: %v", err)
	}
	return nil, nil
}

func (s *SRbdStorage) saveToGlance(ctx context.Context, imageId, imagePath string, compress bool, format string) error {
	diskInfo := &deployapi.DiskInfo{
		Path: imagePath,
	}
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskInfo: diskInfo, Compress: compress})
	if err != nil {
		return err
	}

	tmpPath := "/opt/cloud/tmp"
	err = os.MkdirAll(tmpPath, 0755)
	if err != nil {
		log.Errorf("failed mkdir %s", tmpPath)
		return errors.Wrap(err, "os.MkdirAll")
	}

	tmpFileDir, err := os.MkdirTemp(tmpPath, "ceph_save_images")
	if err != nil {
		log.Errorf("fail to obtain tempFile for ceph save glance image: %s", err)
		return errors.Wrap(err, "os.MkdirTemp")
	}
	defer func() {
		log.Debugf("clean up temp dir for glance image save %s", tmpFileDir)
		output, err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-fr", tmpFileDir).Output()
		if err != nil {
			log.Errorf("rm %s fail %s %s", tmpFileDir, output, err)
		}
	}()

	tmpImageFile := filepath.Join(tmpFileDir, imageId)

	if len(format) == 0 {
		format = options.HostOptions.DefaultImageSaveFormat
	}

	img, err := qemuimg.NewQemuImage(imagePath)
	if err != nil {
		return errors.Wrapf(err, "NewQemuImage %s", imagePath)
	}
	_, err = img.Clone(tmpImageFile, qemuimgfmt.String2ImageFormat(format), compress)
	if err != nil {
		return errors.Wrapf(err, "Clone %s", tmpImageFile)
	}

	f, err := os.Open(tmpImageFile)
	if err != nil {
		return err
	}
	defer os.Remove(tmpImageFile)
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

	_, err = image.Images.Upload(hostutils.GetImageSession(ctx), params, f, size)
	return err
}

func (s *SRbdStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return fmt.Errorf("Not support")
}

func (s *SRbdStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not support delete snapshots")
}

func (s *SRbdStorage) DeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not support delete snapshot")
}

func (s *SRbdStorage) CreateDiskFromSnapshot(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) (jsonutils.JSONObject, error) {
	info := input.DiskInfo
	err := disk.CreateFromRbdSnapshot(ctx, info.SnapshotUrl, info.SrcDiskId, info.SrcPool)
	if err != nil {
		return nil, err
	}
	return disk.GetDiskDesc(), nil
}

func (s *SRbdStorage) GetBackupDir() string {
	return ""
}

func (s *SRbdStorage) CreateDiskFromExistingPath(context.Context, IDisk, *SDiskCreateByDiskinfo) error {
	return fmt.Errorf("Not support")
}

func (s *SRbdStorage) CreateDiskFromBackup(ctx context.Context, disk IDisk, input *SDiskCreateByDiskinfo) error {
	pool, _ := s.StorageConf.GetString("pool")
	destPath := fmt.Sprintf("rbd:%s/%s%s", pool, disk.GetId(), s.getStorageConfString())
	err := doRestoreDisk(ctx, s, input, disk, destPath)
	if err != nil {
		return errors.Wrap(err, "doRestore")
	}
	/*backup := input.DiskInfo.Backup
	backupStorage, err := backupstorage.GetBackupStorage(backup.BackupStorageId, backup.BackupStorageAccessInfo)
	if err != nil {
		return errors.Wrap(err, "unable to GetBackupStorage")
	}
	err = backupStorage.ConvertTo(destPath, qemuimgfmt.RAW, backup.BackupId)
	if err != nil {
		return errors.Wrapf(err, "unable to Convert to with destPath %s and format %s", destPath, qemuimgfmt.RAW.String())
	}*/
	return nil
}

func (s *SRbdStorage) getDiskPath(diskId string) string {
	storageConf := s.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return fmt.Sprintf("rbd:%s/%s", pool, diskId)
}

func (s *SRbdStorage) GetDiskPath(diskId string) string {
	return fmt.Sprintf("%s%s", s.getDiskPath(diskId), s.getStorageConfString())
}

func (s *SRbdStorage) GetCloneTargetDiskPath(ctx context.Context, targetDiskId string) string {
	return s.GetDiskPath(targetDiskId)
}

func (s *SRbdStorage) CloneDiskFromStorage(
	ctx context.Context, srcStorage IStorage, srcDisk IDisk, targetDiskId string, fullCopy bool,
) (*hostapi.ServerCloneDiskFromStorageResponse, error) {
	srcDiskPath := srcDisk.GetPath()
	srcImg, err := qemuimg.NewQemuImage(srcDiskPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Get source image %q info", srcDiskPath)
	}
	accessPath := s.GetCloneTargetDiskPath(ctx, targetDiskId)
	if fullCopy {
		_, err = srcImg.Clone(accessPath, qemuimgfmt.RAW, false)
		if err != nil {
			return nil, errors.Wrap(err, "Clone source disk to target rbd storage")
		}
	} else {
		err = s.createImage(targetDiskId, uint64(srcImg.GetSizeMB()))
		if err != nil {
			return nil, errors.Wrap(err, "Create rbd image")
		}
	}

	return &hostapi.ServerCloneDiskFromStorageResponse{
		TargetAccessPath: accessPath,
		TargetFormat:     qemuimgfmt.RAW.String(),
	}, nil
}

func (s *SRbdStorage) SetStorageInfo(storageId, storageName string, conf jsonutils.JSONObject) error {
	s.StorageId = storageId
	s.StorageName = storageName
	if gotypes.IsNil(conf) {
		return fmt.Errorf("empty storage conf for storage %s(%s)", storageName, storageId)
	}
	if dconf, ok := conf.(*jsonutils.JSONDict); ok {
		s.StorageConf = dconf
	}
	conf.Unmarshal(&s.sStorageConf)
	if s.RadosMonOpTimeout == 0 {
		s.RadosMonOpTimeout = api.RBD_DEFAULT_MON_TIMEOUT
	}
	if s.RadosOsdOpTimeout == 0 {
		s.RadosOsdOpTimeout = api.RBD_DEFAULT_OSD_TIMEOUT
	}
	if s.ClientMountTimeout == 0 {
		s.ClientMountTimeout = api.RBD_DEFAULT_MOUNT_TIMEOUT
	}
	return nil
}

func (s *SRbdStorage) CleanRecycleDiskfiles(ctx context.Context) {
	log.Infof("SRbdStorage CleanRecycleDiskfiles do nothing!")
}
