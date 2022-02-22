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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/errors"
)

type SDisk struct {
	Device   string `json:"device"`
	Path     string `json:"path"`
	SizeInGB string `json:"sizeInGB"`
	VolumeId string `json:"volumeId"`
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
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
	return result.DiskFileInfo.Item, nil
}

// func (self *SRegion) GetDisks(instanceId string, zoneId string, category string, diskIds []string, offset int, limit int) ([]SDisk, int, error) {
// 	if limit > 50 || limit <= 0 {
// 		limit = 50
// 	}
// 	params := make(map[string]string)
// 	params["RegionId"] = self.RegionId
// 	params["PageSize"] = fmt.Sprintf("%d", limit)
// 	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

// 	if len(instanceId) > 0 {
// 		params["InstanceId"] = instanceId
// 	}
// 	if len(zoneId) > 0 {
// 		params["ZoneId"] = zoneId
// 	}
// 	if len(category) > 0 {
// 		params["Category"] = category
// 	}
// 	if diskIds != nil && len(diskIds) > 0 {
// 		params["DiskIds"] = jsonutils.Marshal(diskIds).String()
// 	}

// 	body, err := self.invoke("DescribeInstanceDiskFile", params)
// 	if err != nil {
// 		log.Errorf("GetDisks fail %s", err)
// 		return nil, 0, err
// 	}

// 	disks := make([]SDisk, 0)
// 	err = body.Unmarshal(&disks, "Disks", "Disk")
// 	if err != nil {
// 		log.Errorf("Unmarshal disk details fail %s", err)
// 		return nil, 0, err
// 	}
// 	total, _ := body.Int("TotalCount")
// 	return disks, int(total), nil
// }

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetDisk")
	}

	return nil, cloudprovider.ErrNotImplemented
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
