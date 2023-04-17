package storageman

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLVMStorage struct {
	SBaseStorage

	Index int
}

func NewLVMStorage(manager *SStorageManager, vgName string, index int) *SLVMStorage {
	var ret = new(SLVMStorage)
	ret.SBaseStorage = *NewBaseStorage(manager, vgName)
	ret.Index = index
	return ret
}

func (s *SLVMStorage) StorageType() string {
	return api.STORAGE_LVM
}

func (s *SLVMStorage) GetComposedName() string {
	return fmt.Sprintf("host_%s_%s_storage_%d", s.Manager.host.GetMasterIp(), s.StorageType(), s.Index)
}

func (s *SLVMStorage) GetMediumType() (string, error) {
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

func (s *SLVMStorage) GetFreeSizeMb() int {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "vgs", "--reportformat", "json", "-o", "vg_free", "--units=B", s.Path,
	).Output()
	if err != nil {
		log.Errorf("failed get vg free size %s: %s", err, out)
		return -1
	}
	res, err := jsonutils.Parse(out)
	if err != nil {
		log.Errorf("failed parse vg free json: %s", err)
		return -1
	}
	vg, _ := res.GetArray("report")
	if len(vg) != 1 {
		log.Errorf("failed get vg report")
		return -1
	}
	vgs, err := vg[0].GetArray("vg")
	if err != nil {
		log.Errorf("failed get vgs")
		return -1
	}
	vgFree, err := vgs[0].GetString("vg_free")
	if err != nil {
		log.Errorf("failed get vg_free: %s", err)
		return -1
	}
	size, err := strconv.ParseInt(strings.TrimSuffix(vgFree, "B"), 10, 64)
	if err != nil {
		log.Errorf("failed parse vg_free %s", err)
		return -1
	}
	return int(size / 1024 / 1024)
}

func (s *SLVMStorage) GetAvailSizeMb() (int64, error) {
	size, err := s.getAvailSize()
	if err != nil {
		return size, err
	}

	log.Infof("LVM Storage %s sizeMb %d", s.GetPath(), size)
	return size, nil
}

func (s *SLVMStorage) getAvailSize() (int64, error) {
	// lvm vgs --reportformat json -o vg_size --units=B <VGNAME>
	/*
			{
		      "report": [
		          {
		              "vg": [
		                  {"vg_size":"1600319913984B"}
		              ]
		          }
		      ]
			}
	*/
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "vgs", "--reportformat", "json", "-o", "vg_size", "--units=B", s.Path,
	).Output()
	if err != nil {
		return -1, errors.Wrap(err, "exec lvm command")
	}
	res, err := jsonutils.Parse(out)
	if err != nil {
		return -1, errors.Wrap(err, "json parse")
	}
	vg, _ := res.GetArray("report")
	if len(vg) != 1 {
		return -1, errors.Errorf("get ")
	}
	vgs, err := vg[0].GetArray("vg")
	if err != nil {
		return -1, errors.Wrap(err, "get vg")
	}
	vgSizeB, err := vgs[0].GetString("vg_size")
	if err != nil {
		return -1, errors.Wrap(err, "get vg_size")
	}
	size, err := strconv.ParseInt(strings.TrimSuffix(vgSizeB, "B"), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "parse vg_size")
	}
	return size / 1024 / 1024, nil
}

func (s *SLVMStorage) GetUsedSizeMb() (int64, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "vgs", "--reportformat", "json", "-o", "vg_size,vg_free", "--units=B", s.Path,
	).Output()
	if err != nil {
		return -1, errors.Wrap(err, "exec lvm command")
	}
	res, err := jsonutils.Parse(out)
	if err != nil {
		return -1, errors.Wrap(err, "json parse")
	}
	vg, _ := res.GetArray("report")
	if len(vg) != 1 {
		return -1, errors.Errorf("get vgs")
	}
	vgs, err := vg[0].GetArray("vg")
	if err != nil {
		return -1, errors.Wrap(err, "get vg")
	}
	vgSizeB, err := vgs[0].GetString("vg_size")
	if err != nil {
		return -1, errors.Wrap(err, "get vg_size")
	}
	size, err := strconv.ParseInt(strings.TrimSuffix(vgSizeB, "B"), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "parse vg_size")
	}
	vgFreeB, err := vgs[0].GetString("vg_free")
	if err != nil {
		return -1, errors.Wrap(err, "get vg_free")
	}
	freeSize, err := strconv.ParseInt(strings.TrimSuffix(vgFreeB, "B"), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "parse vg_free")
	}
	return (size - freeSize) / 1024 / 1024, nil
}

func (s *SLVMStorage) SyncStorageSize() (api.SHostStorageStat, error) {
	stat := api.SHostStorageStat{
		StorageId: s.StorageId,
	}
	size, err := s.GetAvailSizeMb()
	if err != nil {
		return stat, err
	}
	stat.CapacityMb = size
	stat.ActualCapacityUsedMb = 0
	return stat, nil
}

func (s *SLVMStorage) SyncStorageInfo() (jsonutils.JSONObject, error) {
	content := jsonutils.NewDict()
	name := s.GetName(s.GetComposedName)
	content.Set("name", jsonutils.NewString(name))
	sizeMb, err := s.GetAvailSizeMb()
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

		res, err = modules.Storages.Create(
			hostutils.GetComputeSession(context.Background()), content)
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

func (s *SLVMStorage) SaveToGlance(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
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
	return ""
}

func (s *SLVMStorage) Accessible() error {
	return nil
}

func (s *SLVMStorage) Detach() error {
	return nil
}
