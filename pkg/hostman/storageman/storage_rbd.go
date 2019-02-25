package storageman

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"
)

const (
	RBD_FEATURE = 2
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
	for _, key := range []string{"mon_host", "key", "rados_osd_op_timeout", "rados_mon_op_timeout", "client_mount_timeout", "rbd_default_format"} {
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
	_stats, err := s.withCluster(func(conn *rados.Conn) (interface{}, error) {
		return conn.GetClusterStats()
	})
	if err != nil {
		return 0, err
	}
	stats := _stats.(rados.ClusterStat)
	return stats.Kb / 1024, nil
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
	return nil, nil
}

func (s *SRbdStorage) CreateSnapshotFormUrl(ctx context.Context, snapshotUrl, diskId, snapshotPath string) error {
	return nil
}

func (s *SRbdStorage) DeleteSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskId, ok := params.(string)
	if !ok {
		return nil, hostutils.ParamsError
	}
	pool, _ := s.GetStorageConf().GetString("pool")
	return nil, s.deleteSnapshot(pool, diskId, "")
}
