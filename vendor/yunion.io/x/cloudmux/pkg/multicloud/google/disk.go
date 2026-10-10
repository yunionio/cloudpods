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

package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/encode"

	billing "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage
	SResourceBase
	multicloud.SDisk
	GoogleTags

	ProvisionedIops        int
	CreationTimestamp      time.Time
	SizeGB                 int
	Zone                   string
	Status                 string
	Type                   string
	SourceImage            string
	LastAttachTimestamp    time.Time
	LastDetachTimestamp    time.Time
	LabelFingerprint       string
	PhysicalBlockSizeBytes string
	ResourcePolicies       []string
	Users                  []string
	Kind                   string
	autoDelete             bool
	boot                   bool
	index                  int
}

func (region *SRegion) GetDisks(zone string, storageType string, maxResults int, pageToken string) ([]SDisk, error) {
	disks := []SDisk{}
	if len(zone) == 0 {
		return nil, fmt.Errorf("zone params can not be empty")
	}
	params := map[string]string{}
	if len(storageType) > 0 {
		params["filter"] = fmt.Sprintf(`type="%s/%s/projects/%s/zones/%s/diskTypes/%s"`, GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, region.GetProjectId(), zone, storageType)
	}
	return disks, region.List(fmt.Sprintf("zones/%s/disks", zone), params, maxResults, pageToken, &disks)
}

func (region *SRegion) GetDisk(id string) (*SDisk, error) {
	disk := &SDisk{}
	return disk, region.Get("disks", id, disk)
}

func (disk *SDisk) GetStatus() string {
	switch disk.Status {
	case "READY":
		return api.DISK_READY
	case "CREATING":
		return api.DISK_ALLOCATING
	case "RESTORING":
		return api.DISK_RESET
	case "FAILED":
		return api.DISK_ALLOC_FAILED
	case "DELETING":
		return api.DISK_DEALLOC
	default:
		return api.DISK_UNKNOWN
	}
}

func (disk *SDisk) IsEmulated() bool {
	return false
}

func (disk *SDisk) Refresh() error {
	_disk, err := disk.storage.zone.region.GetDisk(disk.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(disk, _disk)
}

func (disk *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return disk.storage, nil
}

func (disk *SDisk) GetIStorageId() string {
	return disk.storage.GetGlobalId()
}

func (disk *SDisk) GetDiskFormat() string {
	return "raw"
}

func (disk *SDisk) GetDiskSizeMB() int {
	return disk.SizeGB * 1024
}

func (disk *SDisk) GetIsAutoDelete() bool {
	if len(disk.Users) == 0 {
		return false
	}
	return disk.autoDelete
}

func (disk *SDisk) GetTemplateId() string {
	return disk.SourceImage
}

func (disk *SDisk) GetDiskType() string {
	if disk.boot && len(disk.Users) > 0 {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
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

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (disk *SDisk) GetIops() int {
	return disk.ProvisionedIops
}

func (disk *SDisk) Delete(ctx context.Context) error {
	return disk.storage.zone.region.Delete(disk.SelfLink)
}

func (disk *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := disk.storage.zone.region.CreateSnapshot(disk.SelfLink, name, desc)
	if err != nil {
		return nil, err
	}
	snapshot.region = disk.storage.zone.region
	return snapshot, nil
}

func (disk *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := disk.storage.zone.region.GetSnapshots(disk.SelfLink, 0, "")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = disk.storage.zone.region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (disk *SDisk) GetISnapshot(id string) (cloudprovider.ICloudSnapshot, error) {
	return disk.storage.zone.region.GetSnapshot(id)
}

func (disk *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return disk.storage.zone.region.ResizeDisk(disk.SelfLink, int(newSizeMB>>10))
}

func (disk *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (disk *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SDisk) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (disk *SDisk) GetCreatedAt() time.Time {
	return disk.CreationTimestamp
}

func (disk *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (disk *SDisk) GetProjectId() string {
	return disk.storage.zone.region.GetProjectId()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func diskTypeName(storageType string) string {
	if idx := strings.LastIndex(storageType, "/"); idx >= 0 {
		return storageType[idx+1:]
	}
	return storageType
}

func inRange(val, minVal, maxVal int) bool {
	return val >= minVal && val <= maxVal
}

// hyperdiskBalancedIopsRange 计算 balanced / balanced-ha 的 IOPS 区间
func hyperdiskBalancedIopsRange(sizeGb, maxIopsCap int) (minIops, maxIops int, ok bool) {
	switch {
	case sizeGb == 4:
		return 2000, 2000, true
	case sizeGb == 5:
		return 2500, 2500, true
	case sizeGb >= 6:
		return 3000, minInt(500*sizeGb, maxIopsCap), true
	default:
		return 0, 0, false
	}
}

func hyperdiskBalancedDefaultIops(sizeGb, maxIopsCap int) int {
	if sizeGb <= 6 {
		return 500 * sizeGb
	}
	// 6 GiB ~ 26.666667 TiB ≈ 27307 GiB
	if sizeGb <= 27307 {
		return minInt(6*sizeGb+3000, maxIopsCap)
	}
	return maxIopsCap
}

func hyperdiskBalancedThroughputRange(iops int) (minTp, maxTp int) {
	return maxInt(140, iops/256), minInt(2400, iops/4)
}

func setHyperdiskBalancedPerformance(params map[string]interface{}, sizeGb, iops, throughput, maxIopsCap int) {
	minIops, maxIops, ok := hyperdiskBalancedIopsRange(sizeGb, maxIopsCap)
	if !ok {
		return
	}
	effectiveIops := iops
	if iops > 0 && inRange(iops, minIops, maxIops) {
		params["provisionedIops"] = iops
	} else if throughput > 0 {
		effectiveIops = hyperdiskBalancedDefaultIops(sizeGb, maxIopsCap)
	} else {
		return
	}
	if throughput > 0 {
		minTp, maxTp := hyperdiskBalancedThroughputRange(effectiveIops)
		if inRange(throughput, minTp, maxTp) {
			params["provisionedThroughput"] = throughput
		}
	}
}

// setDiskProvisionedPerformance 按 GCP 盘型区间+容量公式写入 IOPS/吞吐，超范围或不支持则丢弃该参数
func setDiskProvisionedPerformance(params map[string]interface{}, storageType string, sizeGb, iops, throughput int) {
	if iops <= 0 && throughput <= 0 {
		return
	}
	diskType := diskTypeName(storageType)
	switch diskType {
	case api.STORAGE_GOOGLE_PD_EXTREME:
		if iops > 0 && inRange(iops, 2500, 120000) {
			params["provisionedIops"] = iops
		}
	case api.STORAGE_GOOGLE_HYPERDISK_EXTREME:
		if sizeGb < 64 || iops <= 0 {
			return
		}
		minIops := 2 * sizeGb
		maxIops := 350000
		if sizeGb <= 291 {
			maxIops = 1200 * sizeGb
		}
		if inRange(iops, minIops, maxIops) {
			params["provisionedIops"] = iops
		}
	case api.STORAGE_GOOGLE_HYPERDISK_BALANCED:
		setHyperdiskBalancedPerformance(params, sizeGb, iops, throughput, 160000)
	case "hyperdisk-balanced-high-availability":
		setHyperdiskBalancedPerformance(params, sizeGb, iops, throughput, 100000)
	case api.STORAGE_GOOGLE_HYPERDISK_THROUGHPUT:
		if throughput <= 0 || sizeGb < 2048 || sizeGb > 32768 {
			return
		}
		sizeTiB := float64(sizeGb) / 1024.0
		minTp := maxInt(20, int(5*sizeTiB))
		maxTp := minInt(int(90*sizeTiB), 2400)
		if inRange(throughput, minTp, maxTp) {
			params["provisionedThroughput"] = throughput
		}
	case api.STORAGE_GOOGLE_HYPERDISK_ML:
		if throughput <= 0 {
			return
		}
		minTp := maxInt(400, int(0.12*float64(sizeGb)))
		maxTp := minInt(2097152, 1600*sizeGb)
		if inRange(throughput, minTp, maxTp) {
			params["provisionedThroughput"] = throughput
		}
	}
}

func (region *SRegion) CreateDisk(zone string, storageType string, opts *cloudprovider.DiskCreateConfig) (*SDisk, error) {
	if !strings.HasPrefix(storageType, GOOGLE_COMPUTE_DOMAIN) {
		storageType = fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", region.GetProjectId(), zone, storageType)
	}
	labels := map[string]string{}
	for k, v := range opts.Tags {
		labels[encode.EncodeGoogleLabel(k)] = encode.EncodeGoogleLabel(v)
	}
	body := map[string]interface{}{
		"name":        normalizeString(opts.Name),
		"description": opts.Desc,
		// https://www.googleapis.com/compute/v1/projects/my-project-15390453537169/zones/us-west2-c/diskTypes/pd-standard
		// projects/my-project-15390453537169/zones/us-west2-c/diskTypes/pd-standard
		"type":   storageType,
		"labels": labels,
	}
	body["sizeGb"] = opts.SizeGb
	if len(opts.ImageId) > 0 {
		body["sourceImage"] = opts.ImageId
	}
	setDiskProvisionedPerformance(body, storageType, opts.SizeGb, opts.Iops, opts.Throughput)
	disk := &SDisk{}
	resource := fmt.Sprintf("zones/%s/disks", zone)
	err := region.Insert(resource, jsonutils.Marshal(body), disk)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (region *SRegion) ResizeDisk(id string, sizeGb int) error {
	body := map[string]int{
		"sizeGb": sizeGb,
	}
	return region.Do(id, "resize", nil, jsonutils.Marshal(body))
}

func (self *SRegion) CreateSnapshot(diskId string, name string, desc string) (*SSnapshot, error) {
	body := map[string]string{
		"name":        normalizeString(name),
		"description": desc,
	}
	err := self.Do(diskId, "createSnapshot", nil, jsonutils.Marshal(body))
	if err != nil {
		return nil, err
	}
	snapshot := &SSnapshot{region: self}
	return snapshot, self.GetBySelfId(fmt.Sprintf("projects/%s/global/snapshots/%s", self.GetProjectId(), name), snapshot)
}
