package qcloud

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type Placement struct {
	ProjectId int
	Zone      string
}

type SDisk struct {
	storage *SStorage

	Attached             bool
	AutoRenewFlagError   bool
	CreateTime           time.Time
	DeadlineError        bool
	DeadlineTime         time.Time
	DifferDaysOfDeadline int
	DiskChargeType       string
	DiskId               string
	DiskName             string
	DiskSize             int
	DiskState            string
	DiskType             string
	DiskUsage            string
	Encrypt              bool
	InstanceId           string
	IsReturnable         bool
	Placement            Placement
	Portable             bool
	RenewFlag            string
	ReturnFailCode       int
	RollbackPercent      int
	Rollbacking          bool
	SnapshotAbility      bool
	DeleteWithInstance   bool
}

type SDiskSet []SDisk

func (v SDiskSet) Len() int {
	return len(v)
}

func (v SDiskSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SDiskSet) Less(i, j int) bool {
	if v[i].DiskUsage == "SYSTEM_DISK" || v[j].DiskUsage == "DATA_DISK" {
		return true
	}
	return false
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	// data := jsonutils.NewDict()

	// // The pricingInfo key structure is 'RegionId::DiskCategory::DiskType
	// priceKey := fmt.Sprintf("%s::%s::%s", self.RegionId, self.Category, self.Type)
	// data.Add(jsonutils.NewString(priceKey), "price_key")

	// data.Add(jsonutils.NewString(models.HYPERVISOR_ALIYUN), "hypervisor")

	// return data
	return nil
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, category string, diskIds []string, offset int, limit int) ([]SDisk, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	filter := 0

	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "zone"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = zoneId
		filter++
	}

	if len(instanceId) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "instance-id"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = instanceId
		filter++
	}

	if len(category) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filter)] = "disk-type"
		params[fmt.Sprintf("Filters.%d.Values.0", filter)] = category
		filter++
	}
	if diskIds != nil && len(diskIds) > 0 {
		for index, diskId := range diskIds {
			params[fmt.Sprintf("DiskIds.%d", index)] = diskId
		}
	}

	body, err := self.cbsRequest("DescribeDisks", params)
	if err != nil {
		log.Errorf("GetDisks fail %s", err)
		return nil, 0, err
	}

	disks := make([]SDisk, 0)
	err = body.Unmarshal(&disks, "DiskSet")
	if err != nil {
		log.Errorf("Unmarshal disk details fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	sort.Sort(SDiskSet(disks))
	return disks, int(total), nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, total, err := self.GetDisks("", "", "", []string{diskId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &disks[0], nil
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SRegion) DeleteDisk(diskId string) error {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["DiskIds.0"] = diskId

	_, err := self.cbsRequest("TerminateDisks", params)
	return err
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.DiskId)
}

func (self *SRegion) ResizeDisk(ctx context.Context, diskId string, sizeGb int64) error {
	params := make(map[string]string)
	params["DiskId"] = diskId
	params["DiskSize"] = fmt.Sprintf("%d", sizeGb)
	startTime := time.Now()
	for {
		_, err := self.cbsRequest("ResizeDisk", params)
		if err != nil {
			if strings.Index(err.Error(), "Code=InvalidDisk.Busy") > 0 {
				log.Infof("The disk is busy, try later ...")
				time.Sleep(10 * time.Second)
				if time.Now().Sub(startTime) > time.Minute*20 {
					return cloudprovider.ErrTimeout
				}
				continue
			}
		}
		return err
	}
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return self.storage.zone.region.ResizeDisk(ctx, self.DiskId, sizeMb/1024)
}

func (self *SDisk) GetName() string {
	if len(self.DiskName) > 0 {
		return self.DiskName
	}
	return self.DiskId
}

func (self *SDisk) GetGlobalId() string {
	return self.DiskId
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetStatus() string {
	switch self.DiskState {
	case "ATTACHING", "DETACHING", "EXPANDING", "ROLLBACKING":
		return models.DISK_ALLOCATING
	default:
		return models.DISK_READY
	}
}

func (self *SDisk) Refresh() error {
	new, err := self.storage.zone.region.GetDisk(self.DiskId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshotId, err := self.storage.zone.region.CreateSnapshot(self.DiskId, name, desc)
	if err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	}
	snapshots, total, err := self.storage.zone.region.GetSnapshots("", "", "", []string{snapshotId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total == 1 {
		snapshot := &snapshots[0]
		err := cloudprovider.WaitStatus(snapshot, string(SnapshotStatusAccomplished), 15*time.Second, 3600*time.Second)
		if err != nil {
			return nil, err
		}
		return snapshot, nil
	}
	return nil, nil
}

func (self *SDisk) GetDiskType() string {
	switch self.DiskUsage {
	case "SYSTEM_DISK":
		return models.DISK_TYPE_SYS
	case "DATA_DISK":
		return models.DISK_TYPE_DATA
	default:
		return models.DISK_TYPE_DATA
	}
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

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetBillingType() string {
	switch self.DiskChargeType {
	case "PREPAID":
		return models.BILLING_TYPE_PREPAID
	case "POSTPAID_BY_HOUR":
		return models.BILLING_TYPE_POSTPAID
	default:
		return models.BILLING_TYPE_PREPAID
	}
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.DiskSize * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.DeleteWithInstance
}

func (self *SDisk) GetExpiredAt() time.Time {
	return self.DeadlineTime
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshots, total, err := self.storage.zone.region.GetSnapshots("", "", "", []string{snapshotId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total == 1 {
		return &snapshots[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	for {
		parts, total, err := self.storage.zone.region.GetSnapshots("", self.DiskId, "", []string{}, 0, 20)
		if err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		}
		snapshots = append(snapshots, parts...)
		if len(snapshots) >= total {
			break
		}
	}
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self.storage.zone.region
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) GetTemplateId() string {
	//return self.ImageId
	return ""
}

func (self *SRegion) ResetDisk(diskId, snapshotId string) error {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["DiskId"] = diskId
	params["SnapshotId"] = snapshotId
	_, err := self.cbsRequest("ApplySnapshot", params)
	if err != nil {
		log.Errorf("ResetDisk %s to snapshot %s fail %s", diskId, snapshotId, err)
		return err
	}
	return nil
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) error {
	return self.storage.zone.region.ResetDisk(self.DiskId, snapshotId)
}

func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, desc string) (string, error) {
	params := make(map[string]string)
	params["Region"] = self.Region
	params["DiskType"] = category
	params["DiskChargeType"] = "POSTPAID_BY_HOUR"
	params["DiskName"] = name
	params["Placement.Zone"] = zoneId
	//params["Encrypted"] = "false"
	params["DiskSize"] = fmt.Sprintf("%d", sizeGb)
	params["ClientToken"] = utils.GenRequestId(20)

	body, err := self.cbsRequest("CreateDisks", params)
	if err != nil {
		return "", err
	}
	diskIDSet := []string{}
	err = body.Unmarshal(&diskIDSet, "DiskIdSet")
	if err != nil {
		return "", err
	}
	if len(diskIDSet) < 1 {
		return "", fmt.Errorf("Create Disk error")
	}
	return diskIDSet[0], nil
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}
