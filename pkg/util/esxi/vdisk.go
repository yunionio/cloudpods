package esxi

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SVirtualDisk struct {
	SVirtualDevice
}

func NewVirtualDisk(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualDisk {
	return SVirtualDisk{
		NewVirtualDevice(vm, dev, index),
	}
}

func (disk *SVirtualDisk) getVirtualDisk() *types.VirtualDisk {
	return disk.dev.(*types.VirtualDisk)
}

func (disk *SVirtualDisk) getBackingInfo() *types.VirtualDiskFlatVer2BackingInfo {
	backing := disk.getVirtualDisk().Backing
	switch backing.(type) {
	case *types.VirtualDiskFlatVer2BackingInfo:
		return backing.(*types.VirtualDiskFlatVer2BackingInfo)
	case *types.VirtualDeviceFileBackingInfo:
	case *types.VirtualDiskFlatVer1BackingInfo:
	case *types.VirtualDiskLocalPMemBackingInfo:
	case *types.VirtualDiskRawDiskMappingVer1BackingInfo:
	case *types.VirtualDiskSeSparseBackingInfo:
	case *types.VirtualDiskSparseVer1BackingInfo:
	case *types.VirtualDiskSparseVer2BackingInfo:
	case *types.VirtualFloppyImageBackingInfo:
	case *types.VirtualNVDIMMBackingInfo:
	case *types.VirtualParallelPortFileBackingInfo:
	case *types.VirtualSerialPortFileBackingInfo:
	case *types.VirtualCdromIsoBackingInfo:
	}
	log.Fatalf("unsupported backing info %T", backing)
	return nil
}

func (disk *SVirtualDisk) GetId() string {
	backing := disk.getBackingInfo()
	return backing.Uuid
}

func (disk *SVirtualDisk) GetName() string {
	backing := disk.getBackingInfo()
	return path.Base(backing.FileName)
}

func (disk *SVirtualDisk) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", disk.vm.GetGlobalId(), disk.GetId())
}

func (disk *SVirtualDisk) GetStatus() string {
	return models.DISK_READY
}

func (disk *SVirtualDisk) Refresh() error {
	return nil
}

func (disk *SVirtualDisk) IsEmulated() bool {
	return false
}

func (disk *SVirtualDisk) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (disk *SVirtualDisk) GetDiskSizeMB() int {
	capa := disk.getVirtualDisk().CapacityInBytes
	if capa == 0 {
		capa = disk.getVirtualDisk().CapacityInKB * 1024
	}
	return int(capa / 1024 / 1024)
}

func (disk *SVirtualDisk) GetAccessPath() string {
	istore, err := disk.GetIStorage()
	if err != nil {
		log.Errorf("disk.GetIStorage fail %s", err)
		return ""
	}
	ds := istore.(*SDatastore)
	return ds.getFullPath(disk.getBackingInfo().FileName)
}

func (disk *SVirtualDisk) GetDiskFormat() string {
	return "vmdk"
}

func (disk *SVirtualDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	dsObj := disk.getBackingInfo().Datastore
	dc, err := disk.vm.GetDatacenter()
	if err != nil {
		log.Errorf("fail to find datacenter %s", err)
		return nil, err
	}
	istorage, err := dc.GetIStorageByMoId(moRefId(*dsObj))
	if err != nil {
		return nil, err
	}
	return istorage, nil
}

func (disk *SVirtualDisk) GetIsAutoDelete() bool {
	return true
}

func (disk *SVirtualDisk) GetTemplateId() string {
	backing := disk.getBackingInfo()
	if backing.Parent != nil {
		return path.Base(backing.Parent.FileName)
	}
	return ""
}

func (disk *SVirtualDisk) GetDiskType() string {
	backing := disk.getBackingInfo()
	if backing.Parent != nil {
		return models.DISK_TYPE_SYS
	}
	return models.DISK_TYPE_DATA
}

func (disk *SVirtualDisk) GetFsFormat() string {
	return ""
}

func (disk *SVirtualDisk) getDiskMode() string {
	backing := disk.getBackingInfo()
	return backing.DiskMode
}

func (disk *SVirtualDisk) GetIsNonPersistent() bool {
	return disk.getDiskMode() == "persistent"
}

func (disk *SVirtualDisk) GetDriver() string {
	controller := disk.vm.getVdev(disk.getControllerKey())
	name := controller.GetDriver()
	name = strings.Replace(name, "controller", "", -1)
	mapping := map[string]string{
		"ahci":        "sata",
		"parascsi":    "pvscsi",
		"buslogic":    "scsi",
		"lsilogic":    "scsi",
		"lsilogicsas": "scsi",
	}
	return mapping[name]
}

func (disk *SVirtualDisk) GetCacheMode() string {
	backing := disk.getBackingInfo()
	if backing.WriteThrough != nil && *backing.WriteThrough {
		return "writethrough"
	} else {
		return "none"
	}
}

func (disk *SVirtualDisk) GetMountpoint() string {
	return ""
}

func (disk *SVirtualDisk) Delete(ctx context.Context) error {
	istorage, err := disk.GetIStorage()
	if err != nil {
		log.Errorf("disk.GetIStorage() fail %s", err)
		return err
	}
	ds := istorage.(*SDatastore)
	return ds.DeleteVmdk(ctx, disk.getBackingInfo().FileName)
}

func (disk *SVirtualDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (disk *SVirtualDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (disk *SVirtualDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (disk *SVirtualDisk) Resize(ctx context.Context, newSizeMb int64) error {
	ndisk := disk.getVirtualDisk()
	ndisk.CapacityInKB = newSizeMb * 1024

	devSpec := types.VirtualDeviceConfigSpec{}
	devSpec.Device = ndisk
	devSpec.Operation = types.VirtualDeviceConfigSpecOperationEdit

	spec := types.VirtualMachineConfigSpec{}
	spec.DeviceChange = []types.BaseVirtualDeviceConfigSpec{&devSpec}

	vm := disk.vm.getVmObj()

	task, err := vm.Reconfigure(ctx, spec)

	if err != nil {
		log.Errorf("vm.Reconfigure fail %s", err)
		return err
	}

	err = task.Wait(ctx)
	if err != nil {
		log.Errorf("task.Wait fail %s", err)
		return err
	}

	return err
}

func (disk *SVirtualDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (disk *SVirtualDisk) GetBillingType() string {
	return ""
}

func (disk *SVirtualDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SVirtualDisk) Rebuild(ctx context.Context) error {
	return disk.vm.rebuildDisk(ctx, disk)
}
