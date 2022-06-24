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

package incloudsphere

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type RelatedVms struct {
	Id            string `json:"id"`
	Text          string `json:"text"`
	IconCls       string `json:"iconCls"`
	State         string `json:"state"`
	Children      string `json:"children"`
	Object        string `json:"object"`
	Name          string `json:"name"`
	ViewId        string `json:"viewId"`
	TargetType    string `json:"targetType"`
	InnerName     string `json:"innerName"`
	ServiceType   string `json:"serviceType"`
	StorageType   string `json:"storageType"`
	HciType       string `json:"hciType"`
	VMType        string `json:"vmType"`
	HostId        string `json:"hostId"`
	ParentId      string `json:"parentId"`
	DatastoreRole string `json:"datastoreRole"`
	Vlan          string `json:"vlan"`
	DhcpEnabled   bool   `json:"dhcpEnabled"`
	ConnectMode   string `json:"connectMode"`
	DataStoreType string `json:"dataStoreType"`
	Hypervisor    string `json:"hypervisor"`
}

type SDisk struct {
	multicloud.SDisk
	multicloud.InCloudSphereTags

	region *SRegion

	Id                 string       `json:"id"`
	UUID               string       `json:"uuid"`
	Size               float64      `json:"size"`
	RealSize           float64      `json:"realSize"`
	Name               string       `json:"name"`
	FileName           string       `json:"fileName"`
	Offset             int          `json:"offset"`
	Shared             bool         `json:"shared"`
	DeleteModel        string       `json:"deleteModel"`
	VolumePolicy       string       `json:"volumePolicy"`
	Format             string       `json:"format"`
	BlockDeviceId      string       `json:"blockDeviceId"`
	DiskType           string       `json:"diskType"`
	DataStoreId        string       `json:"dataStoreId"`
	DataStoreName      string       `json:"dataStoreName"`
	DataStoreSize      float64      `json:"dataStoreSize"`
	FreeStorage        float64      `json:"freeStorage"`
	DataStoreType      string       `json:"dataStoreType"`
	DataStoreReplicate int          `json:"dataStoreReplicate"`
	VMName             string       `json:"vmName"`
	VMStatus           string       `json:"vmStatus"`
	Type               string       `json:"type"`
	Description        string       `json:"description"`
	Bootable           bool         `json:"bootable"`
	VolumeStatus       string       `json:"volumeStatus"`
	MountedHostIds     string       `json:"mountedHostIds"`
	Md5                string       `json:"md5"`
	DataSize           int          `json:"dataSize"`
	OpenStackId        string       `json:"openStackId"`
	VvSourceDto        string       `json:"vvSourceDto"`
	FormatDisk         bool         `json:"formatDisk"`
	ToBeConverted      bool         `json:"toBeConverted"`
	RelatedVms         []RelatedVms `json:"relatedVms"`
	XactiveDataStoreId string       `json:"xactiveDataStoreId"`
	ClusterSize        int          `json:"clusterSize"`
	ScsiId             string       `json:"scsiId"`
	SecondaryUUId      string       `json:"secondaryUuid"`
	SecondaryVolumes   string       `json:"secondaryVolumes"`
}

func (self *SDisk) GetName() string {
	return self.Name
}

func (self *SDisk) GetId() string {
	return self.Id
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
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
	return "scsi"
}

func (self *SDisk) GetDiskType() string {
	if strings.HasSuffix(self.Name, "1") || strings.HasSuffix(self.Name, "0") {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetDiskFormat() string {
	return strings.ToLower(self.Format)
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.Size * 1024)
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
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return self.FileName
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.region.GetStorage(self.DataStoreId)
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetDisks(storageId string) ([]SDisk, error) {
	ret := []SDisk{}
	res := fmt.Sprintf("/storages/%s/volumes", storageId)
	return ret, self.list(res, url.Values{}, &ret)
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	ret := &SDisk{region: self}
	res := fmt.Sprintf("/volumes/%s", id)
	return ret, self.get(res, nil, ret)
}
