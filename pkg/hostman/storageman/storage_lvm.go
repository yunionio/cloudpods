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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/procutils"
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

func (s *SLVMStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%s", s.Manager.host.GetMasterIp(), s.StorageType(), s.Path)
}

func (s *SLVMStorage) GetMediumType() (string, error) {
	return api.DISK_TYPE_ROTATE, nil
}

func (s *SLVMStorage) GetFreeSizeMb() int {
	vgProps, err := lvmutils.GetVgProps(s.GetPath())
	if err != nil {
		log.Errorf("failed get vg_free: %s", err)
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

		res, err = modules.Storages.Create(hostutils.GetComputeSession(context.Background()), content)
		if err == nil {
			log.Errorf("storage created %s", res)
			storageCacheId, _ := res.GetString("storagecache_id")
			storageManager.InitLVMStorageImageCache(storageCacheId, s.GetPath())
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
	return ""
}

func (s *SLVMStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
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
		err          error
	)
	if err = s.saveToGlance(ctx, imageId, imagePath, compress); err != nil {
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

func (s *SLVMStorage) saveToGlance(ctx context.Context, imageId, imagePath string, compress bool) error {
	log.Infof("saveToGlance %s", imagePath)
	diskInfo := &deployapi.DiskInfo{
		Path: imagePath,
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

func (s *SLVMStorage) CreateDiskFromSnapshot(context.Context, IDisk, *SDiskCreateByDiskinfo) error {
	return errors.Errorf("unsupported operation")
}

func (s *SLVMStorage) CreateDiskFromExistingPath(context.Context, IDisk, *SDiskCreateByDiskinfo) error {
	return errors.Errorf("unsupported operation")
}

func (s *SLVMStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return errors.Errorf("unsupported operation")
}

func (s *SLVMStorage) GetFuseTmpPath() string {
	return ""
}

func (s *SLVMStorage) GetFuseMountPath() string {
	return ""
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
