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

package zstack

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	multicloud.SDisk
	ZStackTags

	localStorage *SLocalStorage
	storage      *SStorage
	region       *SRegion

	ZStackBasic
	PrimaryStorageUUID string  `json:"primaryStorageUuid"`
	VMInstanceUUID     string  `json:"vmInstanceUuid"`
	DiskOfferingUUID   string  `json:"diskOfferingUuid"`
	RootImageUUID      string  `json:"rootImageUuid"`
	InstallPath        string  `json:"installPath"`
	Type               string  `json:"type"`
	Format             string  `json:"format"`
	Size               int     `json:"size"`
	ActualSize         int     `json:"actualSize"`
	DeviceID           float32 `json:"deviceId"`
	State              string  `json:"state"`
	Status             string  `json:"status"`

	ZStackTime
}

func (region *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disk := &SDisk{region: region}
	err := region.client.getResource("volumes", diskId, disk)
	if err != nil {
		return nil, err
	}
	if disk.Status == "NotInstantiated" || disk.Status == "Deleted" {
		return nil, cloudprovider.ErrNotFound
	}
	return disk, nil
}

func (region *SRegion) GetDiskWithStorage(diskId string) (*SDisk, error) {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		log.Errorf("Get Disk %s error: %v", diskId, err)
		return nil, err
	}
	storage, err := region.GetStorage(disk.PrimaryStorageUUID)
	if err != nil {
		log.Errorf("Get primary storage %s error: %v", disk.PrimaryStorageUUID, err)
		return nil, err
	}
	switch storage.Type {
	case StorageTypeLocal:
		if len(disk.VMInstanceUUID) > 0 {
			instance, err := region.GetInstance(disk.VMInstanceUUID)
			if err != nil {
				return nil, err
			}
			hostId := instance.LastHostUUID
			if len(hostId) == 0 {
				hostId = instance.HostUUID
			}
			disk.localStorage = &SLocalStorage{region: region, primaryStorageID: storage.UUID, HostUUID: hostId}
			return disk, nil
		}
		tags, err := region.GetResourceSysTags("", "VolumeVO", disk.UUID, "")
		if err != nil {
			log.Errorf("get disk tag error: %v", err)
			return nil, err
		}
		for i := 0; i < len(tags); i++ {
			if strings.HasPrefix(tags[i].Tag, "localStorage::hostUuid::") {
				hostInfo := strings.Split(tags[i].Tag, "localStorage::hostUuid::")
				if len(hostInfo) == 2 {
					localStorage, err := region.GetLocalStorage(storage.UUID, hostInfo[1])
					if err != nil {
						return nil, err
					}
					disk.localStorage = localStorage
					return disk, nil
				}
				return nil, fmt.Errorf("invalid host info %s from disk %s", tags[i].Tag, disk.Name)
			}
		}
		return nil, cloudprovider.ErrNotFound
	default:
		disk.storage = storage
		return disk, nil
	}
}

func (region *SRegion) GetDisks(storageId string, diskIds []string, diskType string) ([]SDisk, error) {
	disks := []SDisk{}
	params := url.Values{}
	params.Add("q", "status!=Deleted")
	params.Add("q", "status!=NotInstantiated")
	if len(storageId) > 0 {
		params.Add("q", "primaryStorageUuid="+storageId)
	}
	if len(diskIds) > 0 {
		params.Add("q", "uuid?="+strings.Join(diskIds, ","))
	}
	if len(diskType) > 0 {
		params.Add("q", "type="+diskType)
	}
	return disks, region.client.listAll("volumes", params, &disks)
}

func (disk *SDisk) GetSysTags() map[string]string {
	data := map[string]string{}
	data["hypervisor"] = api.HYPERVISOR_ZSTACK
	return data
}

func (disk *SDisk) GetId() string {
	return disk.UUID
}

func (disk *SDisk) Delete(ctx context.Context) error {
	if disk.Status == "Deleted" {
		return disk.region.ExpungeDisk(disk.UUID)
	}
	return disk.region.DeleteDisk(disk.UUID)
}

func (disk *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return disk.region.ResizeDisk(disk.UUID, sizeMb)
}

func (disk *SDisk) GetName() string {
	return disk.Name
}

func (disk *SDisk) GetGlobalId() string {
	return disk.UUID
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	if disk.localStorage != nil {
		return disk.localStorage, nil
	}
	if disk.storage != nil {
		return disk.storage, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (disk *SDisk) GetIStorageId() string {
	storage, err := disk.region.GetStorage(disk.PrimaryStorageUUID)
	if err != nil {
		return disk.PrimaryStorageUUID
	} else if storage.Type == StorageTypeLocal && len(disk.VMInstanceUUID) > 0 {
		instnace, err := disk.region.GetInstance(disk.VMInstanceUUID)
		if err != nil {
			log.Warningf("failed to get instance %s for disk %s(%s) error: %v", disk.VMInstanceUUID, disk.Name, disk.UUID, err)
			return ""
		}
		return fmt.Sprintf("%s/%s", disk.PrimaryStorageUUID, instnace.LastHostUUID)
	}
	return disk.PrimaryStorageUUID
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "Ready":
		return api.DISK_READY
	case "NotInstantiated":
		//数据云盘特有的状态。在这个连接状态中，数据云盘只存在于数据库的表记录中。NotInstantiated状态的数据云盘可以挂载到任何类型虚拟机管理程序管理的云主机上；当挂载到云主机上后，数据云盘的hypervisorType域会存储云主机对应的虚拟机管理程序类型，在主存储上被实例化为虚拟机管理程序类型的实际二进制文件，同时连接状态会改为就绪（Ready）；在这之后，这些数据云盘就只能被重新挂载到相同类型虚拟机管理程序管理的云主机上了。
		return api.DISK_INIT
	case "Creating":
		return api.DISK_ALLOCATING
	case "Deleted":
		return api.DISK_DEALLOC
	default:
		log.Errorf("Unknown disk %s(%s) status %s", disk.Name, disk.UUID, disk.Status)
		return api.DISK_UNKNOWN
	}
}

func (disk *SDisk) Refresh() error {
	new, err := disk.region.GetDisk(disk.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, new)
}

func (disk *SDisk) GetDiskFormat() string {
	return disk.Format
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.Size / 1024 / 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	return disk.GetDiskType() == api.DISK_TYPE_SYS || disk.localStorage != nil
}

func (disk *SDisk) GetTemplateId() string {
	return disk.RootImageUUID
}

func (disk *SDisk) GetDiskType() string {
	switch disk.Type {
	case "Root":
		return api.DISK_TYPE_SYS
	default:
		return api.DISK_TYPE_DATA
	}
}

func (disk *SDisk) GetFsFormat() string {
	return ""
}

func (disk *SDisk) GetIsNonPersistent() bool {
	return false
}

func (disk *SDisk) GetDriver() string {
	return "scsi"
}

func (disk *SDisk) GetCacheMode() string {
	return "none"
}

func (disk *SDisk) GetMountpoint() string {
	return ""
}

func (region *SRegion) CreateDisk(name string, storageId string, hostId string, poolName string, sizeGb int, desc string) (*SDisk, error) {
	offerings, err := region.GetDiskOfferings(sizeGb)
	if err != nil {
		return nil, err
	}
	diskOfferingUuid := ""
	if len(offerings) > 0 {
		diskOfferingUuid = offerings[0].UUID
	} else {
		offering, err := region.CreateDiskOffering(sizeGb)
		if err != nil {
			return nil, err
		}
		diskOfferingUuid = offering.UUID
		defer region.DeleteDiskOffering(diskOfferingUuid)
	}
	params := map[string]interface{}{
		"params": map[string]string{
			"name":               name,
			"description":        desc,
			"diskOfferingUuid":   diskOfferingUuid,
			"primaryStorageUuid": storageId,
		},
	}
	if len(hostId) > 0 {
		params["systemTags"] = []string{"localStorage::hostUuid::" + hostId}
	}
	if len(poolName) > 0 {
		params["systemTags"] = []string{"ceph::pool::" + poolName}
	}
	resp, err := region.client.post("volumes/data", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	disk := &SDisk{region: region}
	return disk, resp.Unmarshal(disk, "inventory")
}

func (region *SRegion) ExpungeDisk(diskId string) error {
	params := map[string]interface{}{
		"expungeDataVolume": jsonutils.NewDict(),
	}
	_, err := region.client.put("volumes", diskId, jsonutils.Marshal(params))
	return err
}

func (region *SRegion) DeleteDisk(diskId string) error {
	err := region.client.delete("volumes", diskId, "Enforcing")
	if err != nil {
		return err
	}
	return region.ExpungeDisk(diskId)
}

func (region *SRegion) ResizeDisk(diskId string, sizeMb int64) error {
	disk, err := region.GetDisk(diskId)
	if err != nil {
		return err
	}

	if disk.GetDiskSizeMB() == int(sizeMb) {
		return nil
	}

	params := jsonutils.Marshal(map[string]interface{}{
		fmt.Sprintf("resize%sVolume", disk.Type): map[string]int64{
			"size": sizeMb * 1024 * 1024,
		},
	})
	resource := "volumes/resize"
	if disk.Type == "Data" {
		resource = "volumes/data/resize"
	}
	_, err = region.client.put(resource, diskId, params)
	return err
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return disk.region.CreateSnapshot(name, disk.UUID, desc)
}

func (disk *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.region.GetSnapshots(snapshotId, disk.UUID)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 1 {
		if snapshots[0].UUID == snapshotId {
			return &snapshots[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(snapshots) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return nil, cloudprovider.ErrNotFound
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.region.GetSnapshots("", disk.UUID)
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	_, err := disk.region.ResetDisks(snapshotId)
	return disk.UUID, err
}

func (region *SRegion) ResetDisks(snapshotId string) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"revertVolumeFromSnapshot": jsonutils.NewDict(),
	}
	return region.client.put("volume-snapshots", snapshotId, jsonutils.Marshal(params))
}

func (disk *SDisk) GetBillingType() string {
	return ""
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreateDate
}

func (disk *SDisk) GetAccessPath() string {
	return disk.InstallPath
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return disk.region.RebuildDisk(disk.UUID)
}

func (region *SRegion) RebuildDisk(diskId string) error {
	params := map[string]interface{}{
		"recoverDataVolume": jsonutils.NewDict(),
	}
	_, err := region.client.put("volumes", diskId, jsonutils.Marshal(params))
	return err
}

func (disk *SDisk) GetProjectId() string {
	return ""
}
