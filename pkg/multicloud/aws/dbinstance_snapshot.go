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

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceSnapshot struct {
	multicloud.SDBInstanceBackupBase
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

func (snapshot *SDBInstanceSnapshot) GetBackupSizeMb() int {
	return snapshot.AllocatedStorage * 1024
}

func (snapshot *SDBInstanceSnapshot) GetDBNames() string {
	return ""
}

func (snapshot *SDBInstanceSnapshot) GetDBInstanceId() string {
	return snapshot.DbiResourceId
}

func (region *SRegion) GetDBInstanceSnapshots(instanceId string) ([]SDBInstanceSnapshot, error) {
	params := map[string]string{}
	if len(instanceId) > 0 {
		params["DbiResourceId"] = instanceId
	}

	snapshots := SDBInstanceSnapshots{}

	err := region.rdsRequest("DescribeDBSnapshots", params, &snapshots)
	if err != nil {
		return nil, err
	}
	return snapshots.Snapshots, nil
}

func (region *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	snapshots, err := region.GetDBInstanceSnapshots("")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudDBInstanceBackup{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}
