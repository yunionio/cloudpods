// +build linux

package storageman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
)

const (
	RBD_FEATURE     = 3
	RBD_ORDER       = 22  //为rbd对应到rados中每个对象的大小，默认为4MB
	DEFAULT_TIMEOUT = 240 //4 minutes
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
	return storagetypes.STORAGE_RBD
}

func init() {
	registerStorageFactory(&SRbdStorageFactory{})
}

func (s *SRbdStorage) StorageType() string {
	return storagetypes.STORAGE_RBD
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
	for _, key := range []string{"rados_osd_op_timeout", "rados_mon_op_timeout", "client_mount_timeout"} {
		var timeout int64
		if timeout, _ = s.StorageConf.Int(key); timeout == 0 {
			timeout = DEFAULT_TIMEOUT
		}
		conf += fmt.Sprintf(":%s=%d", key, timeout)
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
		if err := image.Remove(); err != nil {
			log.Errorf("remove image %s from pool %s error: %v", name, pool, err)
			return nil, err
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
		snapshot, err := src.CreateSnapshot(destImage)
		if err != nil {
			log.Errorf("create snapshot error: %v", err)
			return nil, err
		}
		defer snapshot.Remove()
		isProtect, err := snapshot.IsProtected()
		if err != nil {
			return nil, err
		}
		if !isProtect {
			if err := snapshot.Protect(); err != nil {
				log.Errorf("snapshot protect error: %v", err)
				return nil, err
			}
		}
		defer snapshot.Unprotect()

		return s.withIOContext(destPool, func(ioctx *rados.IOContext) (interface{}, error) {
			dest, err := src.Clone(destImage, ioctx, destImage, RBD_FEATURE, RBD_ORDER)
			if err != nil {
				return nil, err
			}
			defer dest.Close()
			return nil, dest.Flatten()
		})
	})
	return err
}

func (s *SRbdStorage) withImage(pool string, name string, doFunc func(*rbd.Image) (interface{}, error)) (interface{}, error) {
	return s.withIOContext(pool, func(ioctx *rados.IOContext) (interface{}, error) {
		names, err := rbd.GetImageNames(ioctx)
		if err != nil {
			return nil, err
		}
		if !utils.IsInStringArray(name, names) {
			return nil, ErrNoSuchImage
		}

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
			log.Errorf("get ioctx for pool %s error: %v", pool, err)
			return nil, err
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
	if err := conn.Connect(); err != nil {
		log.Errorf("connect rbd cluster %s error: %v", s.StorageName, err)
		return nil, err
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

func (s *SRbdStorage) createSnapshot(pool string, diskId string, snapshotId string) error {
	_, err := s.withImage(pool, diskId, func(image *rbd.Image) (interface{}, error) {
		return image.CreateSnapshot(snapshotId)
	})
	return err
}

func (s *SRbdStorage) deleteSnapshot(pool string, diskId string, snapshotId string) error {
	_, err := s.withImage(pool, diskId, func(image *rbd.Image) (interface{}, error) {
		snapshots, err := image.GetSnapshotNames()
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots {
			if len(snapshotId) == 0 || snapshot.Name == snapshotId {
				if err := image.GetSnapshot(snapshot.Name).Remove(); err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
}

func (s *SRbdStorage) getCapacity() (uint64, error) {
	_sizeKb, err := s.withCluster(func(conn *rados.Conn) (interface{}, error) {
		stats, err := conn.GetClusterStats()
		if err != nil {
			return nil, err
		}
		clusterSizeKb := stats.Kb
		pool, _ := s.StorageConf.GetString("pool")
		bufer, _, err := conn.MonCommand([]byte(fmt.Sprintf(`{"prefix":"osd pool get-quota", "pool":"%s"}`, pool)))
		if err != nil {
			return nil, err
		}
		for _, v := range strings.Split(string(bufer), "\n") {
			v = strings.ToLower(v)
			if strings.Index(v, "max bytes") != -1 {
				if strings.Index(v, "n/a") == -1 {
					if info := strings.Split(v, ":"); len(info) == 2 {
						_size := strings.Trim(info[1], " ")
						for k, v := range map[string]uint64{"kb": 1, "mb": 1024, "gb": 1024 * 1024, "tb": 1024 * 1024 * 1024, "pb": 1014 * 1024 * 1024 * 1024} {
							if strings.Index(_size, k) != -1 {
								sizeStr := strings.TrimSuffix(_size, k)
								size, err := strconv.Atoi(sizeStr)
								if err != nil {
									return clusterSizeKb, nil
								}
								if uint64(size)*v > clusterSizeKb {
									return clusterSizeKb, nil
								}
								return uint64(size) * v, nil
							}
						}
					}
				}
			}
		}
		return clusterSizeKb, nil
	})
	if err != nil {
		log.Errorf("get capacity error: %v", err)
		return 0, err
	}
	sizeKb := _sizeKb.(uint64)
	return sizeKb / 1024, nil
}

func (s *SRbdStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := map[string]interface{}{}
	if len(s.StorageId) > 0 {
		capacity, err := s.getCapacity()
		if err != nil {
			return nil, err
		}
		content = map[string]interface{}{
			"name":     s.StorageName,
			"capacity": capacity,
			"status":   models.STORAGE_ONLINE,
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
	var (
		kvmDisk = NewKVMGuestDisk(imagePath)
		osInfo  string
		relInfo *fsdriver.SReleaseInfo
	)

	if err := func() error {
		if kvmDisk.Connect() {
			defer kvmDisk.Disconnect()

			if root := kvmDisk.MountKvmRootfs(); root != nil {
				defer kvmDisk.UmountKvmRootfs(root)

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

	tmpImageFile := fmt.Sprintf("/tmp/%s.img", imageId)
	if len(format) == 0 {
		format = options.HostOptions.DefaultImageSaveFormat
	}

	_, err := procutils.NewCommand(qemutils.GetQemuImg(), "convert", "-f", "raw", "-O", format, imagePath, tmpImageFile).Run()
	if err != nil {
		return err
	}

	f, err := os.Open(tmpImageFile)
	if err != nil {
		return err
	}

	defer os.Remove(tmpImageFile)

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
	f.Close()
	// TODO
	// notify_template_ready
	return err
}

func (s *SRbdStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return fmt.Errorf("Not support")
}

func (s *SRbdStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	pool, _ := s.GetStorageConf().GetString("pool")
	return nil, s.deleteSnapshot(pool, diskId, "")
}
