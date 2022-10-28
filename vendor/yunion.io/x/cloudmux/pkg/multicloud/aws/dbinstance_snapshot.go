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

package aws

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceSnapshot struct {
	multicloud.SDBInstanceBackupBase
	AwsTags
	region *SRegion

	AllocatedStorage                 int       `xml:"AllocatedStorage"`
	AvailabilityZone                 string    `xml:"AvailabilityZone"`
	DbiResourceId                    string    `xml:"DbiResourceId"`
	DBInstanceIdentifier             string    `xml:"DBInstanceIdentifier"`
	Engine                           string    `xml:"Engine"`
	VpcId                            string    `xml:"VpcId"`
	PercentProgress                  int       `xml:"PercentProgress"`
	IAMDatabaseAuthenticationEnabled bool      `xml:"IAMDatabaseAuthenticationEnabled"`
	DBSnapshotIdentifier             string    `xml:"DBSnapshotIdentifier"`
	OptionGroupName                  string    `xml:"OptionGroupName"`
	EngineVersion                    string    `xml:"EngineVersion"`
	MasterUsername                   string    `xml:"MasterUsername"`
	SnapshotType                     string    `xml:"SnapshotType"`
	InstanceCreateTime               time.Time `xml:"InstanceCreateTime"`
	DBSnapshotArn                    string    `xml:"DBSnapshotArn"`
	Encrypted                        bool      `xml:"Encrypted"`
	Port                             int       `xml:"Port"`
	LicenseModel                     string    `xml:"LicenseModel"`
	SnapshotCreateTime               time.Time `xml:"SnapshotCreateTime"`
	StorageType                      string    `xml:"StorageType"`
	Status                           string    `xml:"Status"`
}

type SDBInstanceSnapshots struct {
	Snapshots []SDBInstanceSnapshot `xml:"DBSnapshots>DBSnapshot"`
	Marker    string                `xml:"Marker"`
}

func (snapshot *SDBInstanceSnapshot) GetId() string {
	return snapshot.DBSnapshotIdentifier
}

func (snapshot *SDBInstanceSnapshot) GetGlobalId() string {
	return snapshot.DBSnapshotIdentifier
}

func (snapshot *SDBInstanceSnapshot) GetName() string {
	return snapshot.DBSnapshotIdentifier
}

func (snapshot *SDBInstanceSnapshot) GetEngine() string {
	return snapshot.Engine
}

func (snapshot *SDBInstanceSnapshot) GetEngineVersion() string {
	return snapshot.EngineVersion
}

func (snapshot *SDBInstanceSnapshot) GetStartTime() time.Time {
	return snapshot.SnapshotCreateTime
}

func (snapshot *SDBInstanceSnapshot) GetEndTime() time.Time {
	return snapshot.SnapshotCreateTime
}

func (snapshot *SDBInstanceSnapshot) GetBackupMode() string {
	switch snapshot.SnapshotType {
	case "manual":
		return api.BACKUP_MODE_MANUAL
	default:
		return api.BACKUP_MODE_AUTOMATED
	}
}

func (snapshot *SDBInstanceSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "available":
		return api.DBINSTANCE_BACKUP_READY
	default:
		log.Errorf("unknown dbinstance snapshot status: %s", snapshot.Status)
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (self *SDBInstanceSnapshot) Refresh() error {
	snap, err := self.region.GetRdsSnapshot(self.DBSnapshotIdentifier)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snap)
}

func (snapshot *SDBInstanceSnapshot) GetBackupSizeMb() int {
	return snapshot.AllocatedStorage * 1024
}

func (snapshot *SDBInstanceSnapshot) GetDBNames() string {
	return ""
}

func (snapshot *SDBInstanceSnapshot) GetDBInstanceId() string {
	return snapshot.DbiResourceId
}

func (self *SDBInstanceSnapshot) Delete() error {
	return self.region.DeleteRdsSnapshot(self.DBSnapshotIdentifier)
}

func (self *SRegion) DeleteRdsSnapshot(id string) error {
	params := map[string]string{
		"DBSnapshotIdentifier": id,
	}
	return self.rdsRequest("DeleteDBSnapshot", params, nil)
}

func (region *SRegion) GetDBInstanceSnapshots(instanceId, backupId string) ([]SDBInstanceSnapshot, error) {
	params := map[string]string{}
	if len(instanceId) > 0 {
		params["DBInstanceIdentifier"] = instanceId
	}
	if len(backupId) > 0 {
		params["DBSnapshotIdentifier"] = backupId
	}
	ret, marker := []SDBInstanceSnapshot{}, ""
	for {
		snapshots := SDBInstanceSnapshots{}
		params["Marker"] = marker
		err := region.rdsRequest("DescribeDBSnapshots", params, &snapshots)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeDBSnapshots")
		}
		ret = append(ret, snapshots.Snapshots...)
		if len(snapshots.Marker) == 0 {
			break
		}
		marker = snapshots.Marker
	}
	return ret, nil
}

func (self *SRegion) GetRdsSnapshot(id string) (*SDBInstanceSnapshot, error) {
	if len(id) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	backups, err := self.GetDBInstanceSnapshots("", id)
	if err != nil {
		return nil, err
	}
	if len(backups) == 1 {
		backups[0].region = self
		return &backups[0], nil
	}
	if len(backups) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	snapshots, err := region.GetDBInstanceSnapshots("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceSnapshots")
	}
	isnapshots := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}
