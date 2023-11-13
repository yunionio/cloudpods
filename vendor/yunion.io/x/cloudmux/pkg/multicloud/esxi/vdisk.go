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

package esxi

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var driverMap = map[string]string{
	"ahci":        "sata",
	"parascsi":    "pvscsi",
	"buslogic":    "scsi",
	"lsilogic":    "scsi",
	"lsilogicsas": "scsi",
}

type SVirtualDisk struct {
	multicloud.SDisk
	multicloud.STagBase

	SVirtualDevice
	IsRoot bool
}

func NewVirtualDisk(vm *SVirtualMachine, dev types.BaseVirtualDevice, index int) SVirtualDisk {
	isRoot := dev.GetVirtualDevice().DeviceInfo.GetDescription().Label == rootDiskMark
	return SVirtualDisk{
		SDisk:          multicloud.SDisk{},
		SVirtualDevice: NewVirtualDevice(vm, dev, index),
		IsRoot:         isRoot,
	}
}

func (disk *SVirtualDisk) getVirtualDisk() *types.VirtualDisk {
	return disk.dev.(*types.VirtualDisk)
}

type IDiskBackingInfo interface {
	GetParent() IDiskBackingInfo
	GetUuid() string
	GetDiskMode() string
	GetPreallocation() string
	GetWriteThrough() bool
	GetFileName() string
	GetDatastore() *types.ManagedObjectReference
}

type sVirtualDiskFlatVer2BackingInfo struct {
	info *types.VirtualDiskFlatVer2BackingInfo
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetParent() IDiskBackingInfo {
	if s.info.Parent != nil {
		return &sVirtualDiskFlatVer2BackingInfo{
			info: s.info.Parent,
		}
	}
	return nil
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetUuid() string {
	return s.info.Uuid
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetPreallocation() string {
	if s.info.ThinProvisioned != nil {
		if *s.info.ThinProvisioned {
			return api.DISK_PREALLOCATION_METADATA
		}
		if s.info.EagerlyScrub == nil || !*s.info.EagerlyScrub {
			return api.DISK_PREALLOCATION_FALLOC
		}
		return api.DISK_PREALLOCATION_FULL
	}
	return api.DISK_PREALLOCATION_METADATA
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetDiskMode() string {
	return s.info.DiskMode
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetWriteThrough() bool {
	if s.info.WriteThrough != nil && *s.info.WriteThrough == true {
		return true
	} else {
		return false
	}
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetFileName() string {
	return s.info.FileName
}

func (s *sVirtualDiskFlatVer2BackingInfo) GetDatastore() *types.ManagedObjectReference {
	return s.info.Datastore
}

type sVirtualDiskSparseVer2BackingInfo struct {
	info *types.VirtualDiskSparseVer2BackingInfo
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetParent() IDiskBackingInfo {
	if s.info.Parent != nil {
		return &sVirtualDiskSparseVer2BackingInfo{
			info: s.info.Parent,
		}
	}
	return nil
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetUuid() string {
	return s.info.Uuid
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetPreallocation() string {
	return api.DISK_PREALLOCATION_METADATA
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetDiskMode() string {
	return s.info.DiskMode
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetWriteThrough() bool {
	if s.info.WriteThrough != nil && *s.info.WriteThrough == true {
		return true
	} else {
		return false
	}
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetFileName() string {
	return s.info.FileName
}

func (s *sVirtualDiskSparseVer2BackingInfo) GetDatastore() *types.ManagedObjectReference {
	return s.info.Datastore
}

type sVirtualDiskRawDiskMappingVer1BackingInfo struct {
	info *types.VirtualDiskRawDiskMappingVer1BackingInfo
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetParent() IDiskBackingInfo {
	if s.info.Parent != nil {
		return &sVirtualDiskRawDiskMappingVer1BackingInfo{
			info: s.info.Parent,
		}
	}
	return nil
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetUuid() string {
	return s.info.Uuid
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetDiskMode() string {
	return s.info.DiskMode
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetPreallocation() string {
	return api.DISK_PREALLOCATION_METADATA
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetWriteThrough() bool {
	return false
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetFileName() string {
	return s.info.FileName
}

func (s *sVirtualDiskRawDiskMappingVer1BackingInfo) GetDatastore() *types.ManagedObjectReference {
	return s.info.Datastore
}

type sVirtualDiskSparseVer1BackingInfo struct {
	info *types.VirtualDiskSparseVer1BackingInfo
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetParent() IDiskBackingInfo {
	if s.info.Parent != nil {
		return &sVirtualDiskSparseVer1BackingInfo{
			info: s.info.Parent,
		}
	}
	return nil
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetUuid() string {
	return s.info.Datastore.String() + s.info.FileName
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetPreallocation() string {
	return api.DISK_PREALLOCATION_METADATA
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetDiskMode() string {
	return s.info.DiskMode
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetWriteThrough() bool {
	if s.info.WriteThrough != nil && *s.info.WriteThrough == true {
		return true
	} else {
		return false
	}
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetFileName() string {
	return s.info.FileName
}

func (s *sVirtualDiskSparseVer1BackingInfo) GetDatastore() *types.ManagedObjectReference {
	return s.info.Datastore
}

type sVirtualDiskFlatVer1BackingInfo struct {
	info *types.VirtualDiskFlatVer1BackingInfo
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetParent() IDiskBackingInfo {
	if s.info.Parent != nil {
		return &sVirtualDiskFlatVer1BackingInfo{
			info: s.info.Parent,
		}
	}
	return nil
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetUuid() string {
	return s.info.Datastore.String() + s.info.FileName
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetDiskMode() string {
	return s.info.DiskMode
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetPreallocation() string {
	return api.DISK_PREALLOCATION_METADATA
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetWriteThrough() bool {
	if s.info.WriteThrough != nil && *s.info.WriteThrough == true {
		return true
	} else {
		return false
	}
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetFileName() string {
	return s.info.FileName
}

func (s *sVirtualDiskFlatVer1BackingInfo) GetDatastore() *types.ManagedObjectReference {
	return s.info.Datastore
}

func (disk *SVirtualDisk) getBackingInfo() IDiskBackingInfo {
	backing := disk.getVirtualDisk().Backing
	switch backing.(type) {
	case *types.VirtualDiskFlatVer2BackingInfo:
		return &sVirtualDiskFlatVer2BackingInfo{
			info: backing.(*types.VirtualDiskFlatVer2BackingInfo),
		}
	case *types.VirtualDeviceFileBackingInfo:
	case *types.VirtualDiskFlatVer1BackingInfo:
		return &sVirtualDiskFlatVer1BackingInfo{
			info: backing.(*types.VirtualDiskFlatVer1BackingInfo),
		}
	case *types.VirtualDiskLocalPMemBackingInfo:
	case *types.VirtualDiskRawDiskMappingVer1BackingInfo:
		return &sVirtualDiskRawDiskMappingVer1BackingInfo{
			info: backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo),
		}
	case *types.VirtualDiskSeSparseBackingInfo:
	case *types.VirtualDiskSparseVer1BackingInfo:
		return &sVirtualDiskSparseVer1BackingInfo{
			info: backing.(*types.VirtualDiskSparseVer1BackingInfo),
		}
	case *types.VirtualDiskSparseVer2BackingInfo:
		return &sVirtualDiskSparseVer2BackingInfo{
			info: backing.(*types.VirtualDiskSparseVer2BackingInfo),
		}
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
	return backing.GetUuid()
}

func (disk *SVirtualDisk) MatchId(id string) bool {
	vmid := disk.vm.GetGlobalId()
	if !strings.HasPrefix(id, vmid) {
		return false
	}
	backingUuid := id[len(vmid)+1:]
	backing := disk.getBackingInfo()
	for backing != nil {
		if backing.GetUuid() == backingUuid {
			return true
		}
		backing = backing.GetParent()
	}
	return false
}

func (disk *SVirtualDisk) GetName() string {
	backing := disk.getBackingInfo()
	return path.Base(backing.GetFileName())
}

func (disk *SVirtualDisk) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", disk.vm.GetGlobalId(), disk.GetId())
}

func (disk *SVirtualDisk) GetStatus() string {
	return api.DISK_READY
}

func (disk *SVirtualDisk) Refresh() error {
	return nil
}

func (disk *SVirtualDisk) IsEmulated() bool {
	return false
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
	return ds.GetFullPath(disk.getBackingInfo().GetFileName())
}

func (disk *SVirtualDisk) GetDiskFormat() string {
	return "vmdk"
}

func (disk *SVirtualDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	dsObj := disk.getBackingInfo().GetDatastore()
	dc, err := disk.vm.GetDatacenter()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDatacenter")
	}
	istorage, err := dc.GetIStorageByMoId(moRefId(*dsObj))
	if err != nil {
		return nil, err
	}
	return istorage, nil
}

func (disk *SVirtualDisk) GetIStorageId() string {
	storage, err := disk.GetIStorage()
	if err != nil {
		return ""
	}
	return storage.GetGlobalId()
}

func (disk *SVirtualDisk) GetIsAutoDelete() bool {
	return true
}

func (disk *SVirtualDisk) GetTemplateId() string {
	backing := disk.getBackingInfo()
	if backing.GetParent() != nil {
		return path.Base(backing.GetParent().GetFileName())
	}
	return ""
}

func (disk *SVirtualDisk) GetDiskType() string {
	if disk.IsRoot {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (disk *SVirtualDisk) GetFsFormat() string {
	return ""
}

func (disk *SVirtualDisk) getDiskMode() string {
	backing := disk.getBackingInfo()
	return backing.GetDiskMode()
}

func (disk *SVirtualDisk) GetIsNonPersistent() bool {
	return disk.getDiskMode() == "independent_nonpersistent"
}

func (disk *SVirtualDisk) GetDriver() string {
	controller := disk.vm.getVdev(disk.getControllerKey())
	name := controller.GetDriver()
	name = strings.Replace(name, "controller", "", -1)
	if driver, ok := driverMap[name]; ok {
		return driver
	}
	return name
}

func (disk *SVirtualDisk) GetCacheMode() string {
	backing := disk.getBackingInfo()
	if backing.GetWriteThrough() {
		return "writethrough"
	} else {
		return "none"
	}
}

func (disk *SVirtualDisk) GetMountpoint() string {
	return ""
}

func (disk *SVirtualDisk) GetPreallocation() string {

	backing := disk.getBackingInfo()
	return backing.GetPreallocation()
}

func (disk *SVirtualDisk) Delete(ctx context.Context) error {
	istorage, err := disk.GetIStorage()
	if err != nil {
		return errors.Wrapf(err, "GetIStorage")
	}
	ds := istorage.(*SDatastore)
	return ds.Delete2(ctx, disk.getBackingInfo().GetFileName(), false, false)
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
		return errors.Wrapf(err, "Reconfigure")
	}

	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrapf(err, "task.Wait")
	}

	return err
}

/*
func (disk *SVirtualDisk) ResizePartition(ctx context.Context, accessInfo vcenter.SVCenterAccessInfo) error {
	diskPath := disk.GetFilename()
	vmref := disk.vm.GetMoid()
	diskInfo := deployapi.DiskInfo{
		Path: diskPath,
	}
	vddkInfo := deployapi.VDDKConInfo{
		Host:   accessInfo.Host,
		Port:   int32(accessInfo.Port),
		User:   accessInfo.Account,
		Passwd: accessInfo.Password,
		Vmref:  vmref,
	}
	_, err := deployclient.GetDeployClient().ResizeFs(ctx, &deployapi.ResizeFsParams{
		DiskInfo:   &diskInfo,
		Hypervisor: compute.HYPERVISOR_ESXI,
		VddkInfo:   &vddkInfo,
	})
	if err != nil {
		return errors.Wrap(err, "unable to ResizeFs")
	}
	return nil
}
*/

func (disk *SVirtualDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (disk *SVirtualDisk) GetBillingType() string {
	return ""
}

// GetCreatedAt return create time by getting the Data of file stored at disk.GetAccessPath
func (disk *SVirtualDisk) GetCreatedAt() time.Time {
	path, name := disk.GetAccessPath(), disk.GetFilename()
	storage, err := disk.GetIStorage()
	if err != nil {
		return time.Time{}
	}
	ds := storage.(*SDatastore)
	files, err := ds.ListDir(context.Background(), filepath.Dir(path))
	if err != nil {
		return time.Time{}
	}
	for _, file := range files {
		if file.Name == name {
			return file.Date
		}
	}
	return time.Time{}
}

func (disk *SVirtualDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SVirtualDisk) Rebuild(ctx context.Context) error {
	return disk.vm.rebuildDisk(ctx, disk, "")
}

func (disk *SVirtualDisk) GetProjectId() string {
	return disk.vm.GetProjectId()
}

func (disk *SVirtualDisk) GetFilename() string {
	return disk.getBackingInfo().GetFileName()
}
