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

package cloudpods

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SDisk struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	CloudpodsTags
	region *SRegion

	api.DiskDetails
	guestDisk *api.GuestDiskDetails
}

func (self *SDisk) GetId() string {
	return self.Id
}

func (self *SDisk) GetGlobalId() string {
	return self.Id
}

func (self *SDisk) GetName() string {
	return self.Name
}

func (self *SDisk) GetStatus() string {
	return self.Status
}

func (self *SDisk) GetIops() int {
	return 0
}

func (self *SDisk) Refresh() error {
	disk, err := self.region.GetDisk(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) GetDiskFormat() string {
	return self.DiskFormat
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.DiskSize
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.AutoDelete
}

func (self *SDisk) GetTemplateId() string {
	return self.TemplateId
}

func (self *SDisk) GetDiskType() string {
	return self.DiskType
}

func (self *SDisk) GetFsFormat() string {
	return self.FsFormat
}

func (self *SDisk) GetIsNonPersistent() bool {
	return self.Nonpersistent
}

func (self *SDisk) fetchGuestDisk() error {
	if self.guestDisk != nil {
		return nil
	}
	if len(self.Guests) > 0 {
		gds, err := self.region.GetGuestDisks("", self.Id)
		if err != nil {
			return err
		}
		for i := range gds {
			self.guestDisk = &gds[i]
			break
		}
	}
	return nil
}

func (self *SDisk) GetDriver() string {
	self.fetchGuestDisk()
	if self.guestDisk != nil {
		return self.guestDisk.Driver
	}
	return ""
}

func (self *SDisk) GetCacheMode() string {
	self.fetchGuestDisk()
	if self.guestDisk != nil {
		return self.guestDisk.CacheMode
	}
	return ""
}

func (self *SDisk) GetPreallocation() string {
	return ""
}

func (self *SDisk) GetMountpoint() string {
	self.fetchGuestDisk()
	if self.guestDisk != nil {
		return self.guestDisk.Mountpoint
	}
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return self.AccessPath
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.region.GetSnapshots(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = self.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	ret := []string{}
	for _, policy := range self.Snapshotpolicies {
		ret = append(ret, policy.Id)
	}
	return ret, nil
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	input := api.DiskResizeInput{}
	input.Size = fmt.Sprintf("%dM", sizeMb)
	_, err := self.region.perform(&modules.Disks, self.Id, "resize", input)
	return err
}

func (self *SDisk) Reset(ctx context.Context, snapId string) (string, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(snapId), "snapshot")
	_, err := self.region.perform(&modules.Disks, self.Id, "disk-reset", params)
	return self.Id, err
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SDisk) GetExpiredAt() time.Time {
	return self.ExpiredAt
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	input := api.SnapshotCreateInput{}
	input.Name = name
	input.Description = desc
	input.DiskId = self.Id
	input.ProjectId = self.TenantId
	snapshot := &SSnapshot{region: self.region}
	return snapshot, self.region.create(&modules.Snapshots, input, snapshot)
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	storage, err := self.region.GetStorage(self.StorageId)
	if err != nil {
		return nil, err
	}
	return storage, nil
}

func (self *SDisk) GetIStorageId() string {
	return self.StorageId
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.region.cli.delete(&modules.Disks, self.Id)
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.region.GetDisks(self.Id, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].region = self.region
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (self *SStorage) CreateIDisk(opts *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	input := api.DiskCreateInput{
		DiskConfig: &api.DiskConfig{},
	}
	input.Name = opts.Name
	input.Description = opts.Desc
	input.SizeMb = opts.SizeGb * 1024
	input.Storage = self.Id
	input.ProjectId = opts.ProjectId
	disk := &SDisk{region: self.region}
	return disk, self.region.create(&modules.Disks, input, disk)
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	disk := &SDisk{region: self}
	return disk, self.cli.get(&modules.Disks, id, nil, disk)
}

func (self *SRegion) GetDisks(storageId, serverId string) ([]SDisk, error) {
	params := map[string]interface{}{}
	if len(storageId) > 0 {
		params["storage_id"] = storageId
	}
	if len(serverId) > 0 {
		params["server_id"] = serverId
	}
	disks := []SDisk{}
	return disks, self.list(&modules.Disks, params, &disks)
}

func (self *SRegion) GetGuestDisks(guestId string, diskId string) ([]api.GuestDiskDetails, error) {
	params := map[string]interface{}{
		"scope": "system",
	}
	if len(guestId) > 0 {
		params["server_id"] = guestId
	}
	if len(diskId) > 0 {
		params["disk_id"] = diskId
	}
	ret := []api.GuestDiskDetails{}
	resp, err := modules.Serverdisks.List(self.cli.s, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	return ret, jsonutils.Update(&ret, resp.Data)
}
