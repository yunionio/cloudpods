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

package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SDisk struct {
	multicloud.SDisk
	ProxmoxTags

	storage *SStorage

	Format  string `json:"format"`
	Size    int64  `json:"size"`
	Vmid    string
	VolId   string `json:"volid"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`
	Content string `json:"content"`
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	info := strings.Split(self.VolId, ":")
	if len(info) == 2 {
		return info[1]
	}
	return self.VolId
}

func (self *SDisk) GetId() string {
	return self.VolId
}

func (self *SDisk) GetGlobalId() string {
	if self.storage.Shared == 1 {
		return fmt.Sprintf("%s|%s", self.storage.Storage, self.VolId)
	}
	return fmt.Sprintf("%s|%s|%s", self.storage.Node, self.storage.Storage, self.VolId)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) Refresh() error {
	disks, err := self.storage.zone.region.GetDisks(self.storage.Node, self.storage.Storage)
	if err != nil {
		return err
	}
	for i := range disks {
		disks[i].storage = self.storage
		if disks[i].GetGlobalId() == self.GetGlobalId() {
			return jsonutils.Update(self, disks[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.VolId)
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "virto"
}

func (self *SDisk) GetDiskType() string {
	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetDiskFormat() string {
	return strings.ToLower(self.Format)
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.Size / 1024 / 1024)
}

func (self *SDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	vm, err := self.storage.zone.region.GetInstance(self.Vmid)
	if err != nil {
		return errors.Wrapf(err, "GetInstance")
	}
	for _storageName, disks := range vm.QemuDisks {
		if _storageName != self.storage.Storage {
			continue
		}
		for _, disk := range disks {
			if disk.DiskId != self.VolId {
				continue
			}
			return self.storage.zone.region.ResizeDisk(vm.Node, self.Vmid, disk.Driver, int(sizeMb-int64(self.GetDiskSizeMB()))/1024)
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.VolId)
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SRegion) GetDisks(node, storageName string) ([]SDisk, error) {
	vols := []SDisk{}
	params := url.Values{}
	params.Set("content", "images")
	res := fmt.Sprintf("/nodes/%s/storage/%s/content", node, storageName)
	err := self.get(res, params, &vols)
	if err != nil {
		return nil, err
	}
	return vols, nil
}

func (self *SRegion) ResizeDisk(node string, vmId string, driver string, sizeGb int) error {
	body := map[string]interface{}{
		"disk": driver,
		"size": fmt.Sprintf("+%dG", sizeGb),
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%s/resize", node, vmId)
	return self.put(res, nil, jsonutils.Marshal(body))
}
