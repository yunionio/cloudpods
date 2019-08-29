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

// +build linux

package storageman

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

const (
	RBD_FEATURE = 3
	RBD_ORDER   = 22 //为rbd对应到rados中每个对象的大小，默认为4MB
)

var (
	ErrNoSuchImage    = errors.New("no such image")
	ErrNoSuchSnapshot = errors.New("no such snapshot")
)

type SRbdStorage struct {
	SBaseStorage
}

func NewRBDStorage(manager *SStorageManager, path string) *SRbdStorage {
	var ret = new(SRbdStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, path)
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

func (s *SRbdStorage) GetSnapshotPathByIds(diskId, snapshotId string) string {
	return ""
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

//Tip Configuration values containing :, @, or = can be escaped with a leading \ character.
func (s *SRbdStorage) getStorageConfString() string {
	conf := ""
	for _, key := range []string{"mon_host", "key"} {
		if value, _ := s.StorageConf.GetString(key); len(value) > 0 {
			if key == "mon_host" {
				value = strings.Replace(value, ",", `\;`, -1)
			}
			for _, keyworkd := range []string{":", "@", "="} {
				if strings.Index(value, keyworkd) != -1 {
					value = strings.Replace(value, keyworkd, fmt.Sprintf(`\%s`, keyworkd), -1)
				}
			}
			conf += fmt.Sprintf(":%s=%s", key, value)
		}
	}
	for key, _timeout := range map[string]int64{
		"rados_mon_op_timeout": api.RBD_DEFAULT_MON_TIMEOUT,
		"rados_osd_op_timeout": api.RBD_DEFAULT_OSD_TIMEOUT,
		"client_mount_timeout": api.RBD_DEFAULT_MOUNT_TIMEOUT,
	} {
		if timeout, _ := s.StorageConf.Int(key); timeout > 0 {
			conf += fmt.Sprintf(":%s=%d", key, timeout)
		} else {
			conf += fmt.Sprintf(":%s=%d", key, _timeout)
		}
	}
	return conf
}

func (s *SRbdStorage) getImageSizeMb(pool string, name string) uint64 {
	size, err := s.withImage(pool, name, func(image *rbd.Image) (interface{}, error) {
		size, err := image.GetSize()
		if err != nil {
			return nil, err
		}
		return size / 1024 / 1024, nil
	})
	if err != nil {
		log.Errorf("get image error: %v", err)
		return 0
	}
	return size.(uint64)
}

func (s *SRbdStorage) resizeImage(pool string, name string, sizeMb uint64) error {
	_, err := s.withImage(pool, name, func(image *rbd.Image) (interface{}, error) {
		return nil, image.Resize(sizeMb * 1024 * 1024)
	})
	return err
}

func (s *SRbdStorage) deleteImage(pool string, name string) error {
	_, err := s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
		names, err := rbd.GetImageNames(ioctx)
		if err != nil {
			return nil, err
		}
		if !utils.IsInStringArray(name, names) {
			return nil, nil
		}

		image := rbd.GetImage(ioctx, name)
		err = image.Open()
		if err != nil {
			return nil, errors.Wrap(err, "image.Open()")
		}

		//需要先删除image底下的snap
		snapInfos, err := image.GetSnapshotNames()
		if err != nil {
			return nil, errors.Wrap(err, "image.GetSnapshotNames()")
		}
		for _, snapInfo := range snapInfos {
			image.Close()

			err = image.Open(snapInfo.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "image.Open(%s)", snapInfo.Name)
			}

			pools, images, err := image.ListChildren()
			if err != nil {
				return nil, errors.Wrap(err, "image.ListChildren")
			}

			for i, _pool := range pools {
				//需要解除snap底下的image关系
				_, err = s.withIOContext(_pool, func(ioctx *rados.IOContext) (interface{}, error) {
					_image := rbd.GetImage(ioctx, images[i])
					err = _image.Open()
					if err != nil {
						return nil, errors.Wrap(err, "_image.Open()")
					}
					defer _image.Close()
					log.Debugf("start flatten %s/%s@%s => %s/%s", pool, name, snapInfo.Name, _pool, images[i])
					err := _image.Flatten()
					if err != nil {
						return nil, errors.Wrap(err, "_image.Flatten")
					}
					return nil, nil
				})
				if err != nil {
					return nil, errors.Wrapf(err, "flatten child %s/%s", _pool, images[i])
				}
			}

			snapshot := image.GetSnapshot(snapInfo.Name)
			protect, err := snapshot.IsProtected()
			if err != nil {
				return nil, errors.Wrapf(err, "snapshot.IsProtected() %s", snapInfo.Name)
			}

			if protect {
				for i := 0; i < 3; i++ {
					err = snapshot.Unprotect()
					if err == nil {
						break
					}
					//Resource busy
					if strings.Contains(err.Error(), "16") {
						log.Warningf("snapshot is busy, try unprotect after %d seconds", (i+1)*5)
						time.Sleep(time.Second * time.Duration(i+1) * 5)
						continue
					}
					return nil, errors.Wrapf(err, "snapshot.Unprotect() %s", snapInfo.Name)
				}
			}

			err = snapshot.Remove()
			if err != nil {
				return nil, errors.Wrapf(err, "snapshot.Remove() %s", snapInfo.Name)
			}
		}

		image.Close()

		err = image.Remove()
		if err != nil {
			return nil, errors.Wrapf(err, "image.Remove() %s/%s", pool, name)
		}
		return nil, nil
	})
	return err
}

// 比较费时
func (s *SRbdStorage) copyImage(srcPool string, srcImage string, destPool string, destImage string) error {
	_, err := s.withImage(srcPool, srcImage, func(src *rbd.Image) (interface{}, error) {
		imageSize, err := src.GetSize()
		if err != nil {
			return nil, err
		}
		if err := s.createImage(destPool, destImage, imageSize/1024/1024); err != nil {
			log.Errorf("create image dest pool: %s dest image: %s image size: %dMb error: %v", destPool, destImage, imageSize/1024/1024, err)
			return nil, err
		}
		_, err = s.withImage(destPool, destImage, func(dest *rbd.Image) (interface{}, error) {
			return nil, src.Copy(*dest)
		})
		return nil, err
	})
	return err
}

// 速度快
func (s *SRbdStorage) cloneImage(srcPool string, srcImage string, destPool string, destImage string) error {
	_, err := s.withImage(srcPool, srcImage, func(src *rbd.Image) (interface{}, error) {
		snapInfos, err := src.GetSnapshotNames()
		if err != nil {
			return nil, errors.Wrap(err, "image.GetSnapshotNames()")
		}
		var snapshot *rbd.Snapshot = nil
		for _, snap := range snapInfos {
			if snap.Name == destImage {
				snapshot = src.GetSnapshot(destImage)
				break
			}
		}
		if snapshot == nil {
			snapshot, err = src.CreateSnapshot(destImage)
			if err != nil {
				return nil, errors.Wrap(err, "src.CreateSnapshot")
			}
		}

		isProtect, err := snapshot.IsProtected()
		if err != nil {
			return nil, err
		}
		if !isProtect {
			if err := snapshot.Protect(); err != nil {
				return nil, errors.Wrap(err, "snapshot.Protect")
			}
		}
		defer snapshot.Unprotect()

		return s.withIOContext(destPool, func(ioctx *rados.IOContext) (interface{}, error) {
			_, err := src.Clone(destImage, ioctx, destImage, RBD_FEATURE, RBD_ORDER)
			if err != nil {
				return nil, errors.Wrapf(err, "src.Clone")
			}
			return nil, nil
		})
	})
	return err
}

func (s *SRbdStorage) cloneFromSnapshot(srcImage, srcPool, srcSnapshot, newImage, pool string) error {
	_, err := s.withImage(srcPool, srcImage, func(src *rbd.Image) (interface{}, error) {
		snapshot := src.GetSnapshot(srcSnapshot)
		isProtect, err := snapshot.IsProtected()
		if err != nil {
			return nil, errors.Wrap(err, "snapshot is protected")
		}
		if !isProtect {
			if err := snapshot.Protect(); err != nil {
				return nil, errors.Wrap(err, "snapshot protect")
			}
			defer snapshot.Unprotect()
		}
		return s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
			_, err := src.Clone(srcSnapshot, ioctx, newImage, RBD_FEATURE, RBD_ORDER)
			if err != nil {
				return nil, errors.Wrap(err, "clone from snapshot")
			}
			return nil, nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "clone from snapshot")
	}
	return nil
}

func (s *SRbdStorage) withImage(pool string, name string, doFunc func(*rbd.Image) (interface{}, error)) (interface{}, error) {
	return s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
		image := rbd.GetImage(ioctx, name)
		if err := image.Open(); err != nil {
			log.Errorf("open image %s name error: %v", name, err)
			return nil, err
		}
		defer image.Close()
		return doFunc(image)
	})
}

func (s *SRbdStorage) withIOContext(pool string, doFunc func(*rados.IOContext) (interface{}, error)) (interface{}, error) {
	return s.withCluster(func(conn *rados.Conn) (interface{}, error) {
		ioctx, err := conn.OpenIOContext(pool)
		if err != nil {
			return nil, errors.Wrapf(err, "conn.OpenIOContext(%s)", pool)
		}
		return doFunc(ioctx)
	})
}

func (s *SRbdStorage) listImages(pool string) ([]string, error) {
	images, err := s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
		return rbd.GetImageNames(ioctx)
	})
	if err != nil {
		return nil, err
	}
	return images.([]string), nil
}

func (s *SRbdStorage) withCluster(doFunc func(*rados.Conn) (interface{}, error)) (interface{}, error) {
	conn, _ := rados.NewConn()
	for _, key := range []string{"mon_host", "key"} {
		if value, _ := s.StorageConf.GetString(key); len(value) > 0 {
			if err := conn.SetConfigOption(key, value); err != nil {
				return nil, err
			}
		}
	}
	for key, timeout := range map[string]int64{
		"rados_osd_op_timeout": api.RBD_DEFAULT_OSD_TIMEOUT,
		"rados_mon_op_timeout": api.RBD_DEFAULT_MON_TIMEOUT,
		"client_mount_timeout": api.RBD_DEFAULT_MOUNT_TIMEOUT,
	} {

		_timeout, _ := s.StorageConf.Int(key)
		if _timeout > 0 {
			timeout = _timeout
		}
		if err := conn.SetConfigOption(key, fmt.Sprintf("%d", timeout)); err != nil {
			return nil, err
		}
	}
	if err := conn.Connect(); err != nil {
		return nil, errors.Wrapf(err, "conn.Connect() %s", s.StorageName)
	}
	defer conn.Shutdown()
	return doFunc(conn)
}

func (s *SRbdStorage) createImage(pool string, name string, sizeMb uint64) error {
	_, err := s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
		image, err := rbd.Create(ioctx, name, sizeMb*1024*1024, RBD_ORDER, RBD_FEATURE)
		if err != nil {
			return nil, err
		}
		defer image.Close()
		return nil, nil
	})
	return err
}

func (s *SRbdStorage) renameImage(pool string, src string, dest string) error {
	_, err := s.withImage(pool, src, func(image *rbd.Image) (interface{}, error) {
		return nil, image.Rename(dest)
	})
	return err
}

func (s *SRbdStorage) resetDisk(pool string, diskId string, snapshotId string) error {
	_, err := s.withImage(pool, diskId, func(image *rbd.Image) (interface{}, error) {
		snap := image.GetSnapshot(snapshotId)
		return nil, snap.Rollback()
	})
	return err
}

func (s *SRbdStorage) createSnapshot(pool string, diskId string, snapshotId string) error {
	_, err := s.withImage(pool, diskId, func(image *rbd.Image) (interface{}, error) {
		return image.CreateSnapshot(snapshotId)
	})
	return err
}

func (s *SRbdStorage) deleteSnapshot(pool string, diskId string, snapshotId string) error {
	_, err := s.withImage(pool, diskId, func(image *rbd.Image) (interface{}, error) {
		snapshot := image.GetSnapshot(snapshotId)
		isProtect, err := snapshot.IsProtected()
		if err != nil {
			return nil, errors.Wrap(err, "snapshot is protected")
		}
		if isProtect {
			image.Close()
			if err := image.Open(snapshotId); err != nil {
				return nil, errors.Wrap(err, "image open snapshot")
			}
			pools, childImgs, err := image.ListChildren()
			if err != nil {
				return nil, errors.Wrap(err, "image list children")
			}
			for i, iPool := range pools {
				_, err = s.withIOContext(iPool, func(ioctx *rados.IOContext) (interface{}, error) {
					_image := rbd.GetImage(ioctx, childImgs[i])
					err = _image.Open()
					if err != nil {
						return nil, errors.Wrap(err, "open child image")
					}
					defer _image.Close()

					log.Infof("start flatten %s/%s@%s => %s/%s", pool, diskId, snapshotId, iPool, childImgs[i])
					err := _image.Flatten()
					if err != nil {
						return nil, errors.Wrap(err, "child image flatten")
					}
					return nil, nil
				})
				if err != nil {
					return nil, errors.Wrapf(err, "flatten child %s/%s", iPool, childImgs[i])
				}
			}
			for i := 0; i < 3; i++ {
				err = snapshot.Unprotect()
				if err == nil {
					break
				}
				//Resource busy
				if strings.Contains(err.Error(), "16") {
					log.Warningf("snapshot is busy, try unprotect after %d seconds", (i+1)*5)
					time.Sleep(time.Second * time.Duration(i+1) * 5)
					continue
				}
				return nil, errors.Wrapf(err, "snapshot.Unprotect() %s", snapshotId)
			}
		}
		err = snapshot.Remove()
		if err != nil {
			return nil, errors.Wrap(err, "snapshot remove")
		}
		return nil, nil
	})
	return err
}

type SMonCommand struct {
	Prefix string
	Pool   string
	Format string
}

func (s *SRbdStorage) getCapacity() (uint64, error) {
	_sizeKb, err := s.withCluster(func(conn *rados.Conn) (interface{}, error) {
		stats, err := conn.GetClusterStats()
		if err != nil {
			return nil, err
		}
		clusterSizeKb := stats.Kb
		pool, _ := s.StorageConf.GetString("pool")
		cmd := SMonCommand{Prefix: "osd pool get-quota", Pool: pool, Format: "json"}
		bufer, _, err := conn.MonCommand([]byte(jsonutils.Marshal(cmd).String()))
		if err != nil {
			return nil, err
		}
		result, err := jsonutils.Parse(bufer)
		if err != nil {
			log.Errorf("parse %s json err: %v", string(bufer), err)
			return nil, err
		}
		maxBytes, _ := result.Int("quota_max_bytes")
		if maxBytes == 0 || uint64(maxBytes) > clusterSizeKb*1024 {
			return clusterSizeKb, nil
		}
		return uint64(maxBytes) / 1024, nil
	})
	if err != nil {
		return 0, errors.Wrap(err, "getCapacity")
	}
	sizeKb := _sizeKb.(uint64)
	return sizeKb / 1024, nil
}

func (s *SRbdStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := map[string]interface{}{}
	if len(s.StorageId) > 0 {
		capacity, err := s.getCapacity()
		if err != nil {
			return modules.Storages.PerformAction(hostutils.GetComputeSession(context.Background()), s.StorageId, "offline", nil)
		}
		content = map[string]interface{}{
			"name":     s.StorageName,
			"capacity": capacity,
			"status":   api.STORAGE_ONLINE,
			"zone":     s.GetZone(),
		}
		return modules.Storages.Put(hostutils.GetComputeSession(context.Background()), s.StorageId, jsonutils.Marshal(content))
	}
	return modules.Storages.Get(hostutils.GetComputeSession(context.Background()), s.StorageName, jsonutils.Marshal(content))
}

func (s *SRbdStorage) GetDiskById(diskId string) IDisk {
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

func (s *SRbdStorage) CreateDisk(diskId string) IDisk {
	s.DiskLock.Lock()
	defer s.DiskLock.Unlock()
	disk := NewRBDDisk(s, diskId)
	s.Disks = append(s.Disks, disk)
	return disk
}

func (s *SRbdStorage) Accessible() bool {
	_, err := s.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return conn.ListPools()
	})
	return err == nil
}

func (s *SRbdStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

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
		s.onSaveToGlanceFailed(ctx, imageId)
	}

	rbdImageCache.LoadImageCache(imageId)
	_, err := hostutils.RemoteStoragecacheCacheImage(ctx, rbdImageCache.GetId(), imageId, "ready", imagePath)
	if err != nil {
		log.Errorf("Fail to remote cache image: %v", err)
	}
	return nil, nil
}

func (s *SRbdStorage) onSaveToGlanceFailed(ctx context.Context, imageId string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	_, err := modules.Images.Update(hostutils.GetImageSession(ctx, s.GetZone()),
		imageId, params)
	if err != nil {
		log.Errorln(err)
	}
}

func (s *SRbdStorage) saveToGlance(ctx context.Context, imageId, imagePath string, compress bool, format string) error {
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskPath: imagePath, Compress: compress})
	if err != nil {
		return err
	}

	tmpImageFile := fmt.Sprintf("/tmp/%s.img", imageId)
	if len(format) == 0 {
		format = options.HostOptions.DefaultImageSaveFormat
	}

	_, err = procutils.NewCommand(qemutils.GetQemuImg(), "convert", "-f", "raw", "-O", format, imagePath, tmpImageFile).Run()
	if err != nil {
		return err
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

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx, s.GetZone()),
		params, f, size)
	return err
}

func (s *SRbdStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return fmt.Errorf("Not support")
}

func (s *SRbdStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not support delete snapshots")
}

func (s *SRbdStorage) CreateDiskFromSnapshot(
	ctx context.Context, disk IDisk, createParams *SDiskCreateByDiskinfo,
) error {
	var (
		snapshotUrl, _ = createParams.DiskInfo.GetString("snapshot_url")
		srcDiskId, _   = createParams.DiskInfo.GetString("src_disk_id")
		srcPool, _     = createParams.DiskInfo.GetString("src_pool")
	)
	return disk.CreateFromRbdSnapshot(ctx, snapshotUrl, srcDiskId, srcPool)
}
