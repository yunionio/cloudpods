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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDisk struct {
	multicloud.SDisk
	multicloud.ProxmoxTags

	region *SRegion

	Storage    string
	Node       string
	DiskDriver string
	DriverIdx  int
	VmId       int
	CacheMode  string

	Format  string `json:"format"`
	Size    int64  `json:"size"`
	VolId   string `json:"volid"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`
	Content string `json:"content"`
}

func (self *SDisk) GetName() string {
	return self.Name
}

func (self *SDisk) GetId() string {
	return self.VolId
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetCacheMode() string {
	return self.CacheMode
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return self.DiskDriver
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
	return self.region.ResizeDisk(self.VolId, int(sizeMb/1024))
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	DataStoreId := fmt.Sprintf("storage/%s/%s", self.Node, self.Storage)
	return self.region.GetStorage(DataStoreId)
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SRegion) GetDisks(storageId string) ([]SDisk, error) {
	vols := []SDisk{}
	disks := []SDisk{}

	splited := strings.Split(storageId, "/")
	nodeName := ""
	storageName := ""

	if len(splited) == 3 {
		nodeName, storageName = splited[1], splited[2]
	}

	res := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageName)
	err := self.get(res, url.Values{}, &vols)
	if err != nil {
		return nil, err
	}
	for i := range vols {
		_, diskName := ParseSubConf(vols[i].VolId, ":")
		if err != nil {
			continue
		}
		vols[i].Storage = storageName
		vols[i].Node = nodeName
		vols[i].Name = diskName.(string)

		disks = append(disks, vols[i])
	}

	return disks, nil
}

func (self *SRegion) GetDisk(Id string) (*SDisk, error) {

	vols := []SDisk{}
	nodeName := ""
	storageName, diskName := ParseSubConf(Id, ":")
	resources, err := self.GetClusterStoragesResources()
	if err != nil {
		return nil, err
	}

	if res, ok := resources[storageName]; !ok {
		return nil, errors.Errorf("self.GetDisk")
	} else {
		nodeName = res.Node
	}

	res := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageName)
	err = self.get(res, url.Values{}, &vols)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetDisk")
	}

	for _, vol := range vols {
		if vol.VolId == Id {
			ret := &SDisk{
				region:  self,
				Storage: storageName,
				Node:    nodeName,
				Format:  vol.Format,
				Size:    vol.Size,
				VolId:   vol.VolId,
				Name:    diskName.(string),
				Parent:  vol.Parent,
				VmId:    vol.VmId,
				Content: vol.Content,
			}
			return ret, nil
		}
	}

	return nil, errors.Errorf("self.GetDisk failed to get disk by %s", Id)
}

func (self *SRegion) ResizeDisk(id string, sizeGb int) error {
	disk, err := self.GetDisk(id)
	if err != nil {
		return errors.Wrapf(err, "GetDisk(%s)", id)
	}
	// not support unmount disk
	if disk.VmId < 1 {
		return nil
	}
	body := map[string]interface{}{
		"disk": fmt.Sprintf("%s%d", disk.DiskDriver, disk.DriverIdx),
		"size": sizeGb,
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%d/resize", disk.Node, disk.VmId)
	return self.put(res, nil, jsonutils.Marshal(body), nil)
}
