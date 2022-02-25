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
package bingocloud

import (
	"context"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/errors"
)

type SDisk struct {
	storage *SStorage

	Device   string `json:"device"`
	Path     string `json:"path"`
	SizeInGB string `json:"sizeInGB"`
	VolumeId string `json:"volumeId"`
}

func (self *SRegion) GetDisks(storageId, vmId string) ([]SDisk, error) {
	params := url.Values{}
	filter := []string{}
	if len(storageId) > 0 {
		filter = append(filter, "container_uuid=="+storageId)
	}
	if len(vmId) > 0 {
		filter = append(filter, "vm_uuid=="+vmId)
		filter = append(filter, "attach_vm_id=="+vmId)
	}
	if len(filter) > 0 {
		params.Set("filter_criteria", strings.Join(filter, ","))
	}

	resp, err := self.invoke("", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		DiskFileInfo struct {
			Item []SDisk
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	disks := result.DiskFileInfo.Item

	return disks, self.listAll("virtual_disks", params, &disks)
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetDisk")
	}
	disk := &SDisk{}
	return disk, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetIStorageId() string {
	return ""
}

func (self *SDisk) GetDiskFormat() string {
	return ""
}

func (self *SDisk) GetDiskSizeMB() int {
	return 0
}

func (self *SDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetDiskType() string {
	return ""
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return ""
}

func (self *SDisk) GetCacheMode() string {
	return ""
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetBillingType() string {
	return ""
}

func (self *SDisk) GetCreatedAt() time.Time {
	return time.Now()
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Now()
}

func (self *SDisk) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) IsAutoRenew() bool {
	return false
}

func (self *SDisk) GetProjectId() string {
	return ""
}

func (self *SDisk) GetId() string {
	return ""
}

func (self *SDisk) GetName() string {
	return ""
}

func (self *SDisk) GetGlobalId() string {
	return ""
}

func (self *SDisk) GetStatus() string {
	return ""
}

func (self *SDisk) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetSysTags() map[string]string {
	return nil
}

func (self *SDisk) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
