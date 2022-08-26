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
	"regexp"
	"strconv"
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
	CacheMode  string

	Format  string `json:"format"`
	Size    int64  `json:"size"`
	VolId   string `json:"volid"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`
	VmId    int    `json:"vmid"`
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

var (
	vmIdPattern        = regexp.MustCompile("-(.*?)-disk-")
	DiskKeyNamePattern = regexp.MustCompile("(scsi|ide|virtio|sata|unused)(\\d)")
)

func (self *SRegion) GetVMByDiskId(id string) (vmId int, storage string, err error) {

	splitedFrist := strings.Split(id, ":")
	storage = splitedFrist[0]

	splitedSecond := strings.Split(splitedFrist[1], "/")
	disk := splitedSecond[len(splitedSecond)-1]
	matchArr := vmIdPattern.FindStringSubmatch(disk)

	if len(matchArr) == 0 {
		vmId = 0

		return vmId, storage, errors.Errorf("failed to get vm number by %d", id)
	} else {
		num, _ := strconv.Atoi(matchArr[len(matchArr)-1])
		vmId = num
	}
	return vmId, storage, nil
}

func (self *SRegion) GetQemuDiskConfig(node, id string, vmId int) (diskDriver, cacheMode string, driverIdx int, err error) {
	params := make(map[string]string)

	res := fmt.Sprintf("/nodes/%s/qemu/%d/config", node, vmId)
	err = self.get(res, url.Values{}, &params)

	if err != nil {
		return "", "", 0, errors.Wrapf(err, "self.GetQemuDiskConfig self.get in %s", res)
	}

	for k, v := range params {

		match, _ := regexp.MatchString("(scsi|ide|virtio|sata|unused)", k)

		if match == true {
			part1 := strings.Split(v, ",")
			if part1[0] == id {
				diskParms := make(map[string]string)
				for _, p := range part1 {
					part2 := strings.Split(p, "=")
					if len(part2) != 2 {
						continue
					}
					diskParms[part2[0]] = part2[1]
				}

				if v, ok := diskParms["cache"]; ok {
					cacheMode = v
				} else {
					cacheMode = "none"
				}

				part3 := DiskKeyNamePattern.FindStringSubmatch(k)

				driverIdx, _ = strconv.Atoi(part3[2])
				diskDriver = part3[1]

				return diskDriver, cacheMode, driverIdx, nil
			}
		}

	}

	return "", "", 0, errors.Wrapf(err, "self.GetQemuDiskConfig not fond ")
}

func (self *SRegion) GetDisks(storageId string) ([]SDisk, error) {
	vols := []SDisk{}
	disks := []SDisk{}

	splited := strings.Split(storageId, "/")
	nodeName := ""
	storageName := ""

	if len(splited) == 3 {
		nodeName = splited[1]
		storageName = splited[2]
	}

	res := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storageName)
	err := self.get(res, url.Values{}, &vols)

	resources, _ := self.GetClusterVmResources()

	if err != nil {
		return nil, err
	}

	for _, vol := range vols {
		if vol.VmId > 0 {

			if _, ok := resources[vol.VmId]; !ok {
				continue
			}

			diskDriver, cacheMode, driverIdx, _ := self.GetQemuDiskConfig(nodeName, vol.VolId, vol.VmId)

			vol.Storage = storageName
			vol.Node = nodeName
			vol.DiskDriver = diskDriver
			vol.CacheMode = cacheMode
			vol.DriverIdx = driverIdx

			disks = append(disks, vol)
		}
	}

	return disks, nil

}

func (self *SRegion) GetDisk(Id string) (*SDisk, error) {

	vols := []SDisk{}
	nodeName := ""

	vmId, storageName, err := self.GetVMByDiskId(Id)

	if err != nil {
		return nil, errors.Wrapf(err, "self.GetDisk")
	}

	resources, _ := self.GetClusterVmResources()

	if res, ok := resources[vmId]; !ok {
		return nil, errors.Wrapf(err, "self.GetDisk")
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

			diskDriver, cacheMode, driverIdx, err := self.GetQemuDiskConfig(nodeName, Id, vol.VmId)

			if err != nil {
				return nil, errors.Errorf("self.GetDisk failed to get disk by %d", Id)
			}

			ret := &SDisk{
				region:     self,
				Storage:    storageName,
				Node:       nodeName,
				Format:     vol.Format,
				Size:       vol.Size,
				VolId:      vol.VolId,
				Name:       vol.Name,
				Parent:     vol.Parent,
				VmId:       vol.VmId,
				Content:    vol.Content,
				DiskDriver: diskDriver,
				DriverIdx:  driverIdx,
				CacheMode:  cacheMode,
			}

			return ret, nil
		}
	}

	return nil, errors.Errorf("self.GetDisk failed to get disk by %d", Id)
}

func (self *SRegion) ResizeDisk(id string, sizeGb int) error {
	disk, err := self.GetDisk(id)
	if err != nil {
		return errors.Wrapf(err, "GetDisk(%s)", id)
	}
	body := map[string]interface{}{
		"disk": fmt.Sprintf("%s%d", disk.DiskDriver, disk.DriverIdx),
		"size": sizeGb,
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%d/resize", disk.Node, disk.VmId)
	return self.put(res, nil, jsonutils.Marshal(body), nil)
}
