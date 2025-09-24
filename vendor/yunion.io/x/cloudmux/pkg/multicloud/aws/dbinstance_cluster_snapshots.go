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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SDBInstanceClusterSnapshot struct {
	multicloud.SDBInstanceBackupBase
	AwsTags

	region *SRegion

	AllocatedStorage                 int       `xml:"AllocatedStorage"`
	AvailabilityZones                []string  `xml:"AvailabilityZones>AvailabilityZone"`
	EngineMode                       string    `xml:"EngineMode"`
	Engine                           string    `xml:"Engine"`
	PercentProgress                  int       `xml:"PercentProgress"`
	VpcId                            string    `xml:"VpcId"`
	IAMDatabaseAuthenticationEnabled bool      `xml:"IAMDatabaseAuthenticationEnabled"`
	DBClusterSnapshotIdentifier      string    `xml:"DBClusterSnapshotIdentifier"`
	ClusterCreateTime                time.Time `xml:"ClusterCreateTime"`
	EngineVersion                    string    `xml:"EngineVersion"`
	MasterUsername                   string    `xml:"MasterUsername"`
	SnapshotType                     string    `xml:"SnapshotType"`
	StorageEncrypted                 bool      `xml:"StorageEncrypted"`
	TagList                          []struct {
		Value string `xml:"Value"`
		Key   string `xml:"Key"`
	} `xml:"TagList>Tag"`
	Port                 int       `xml:"Port"`
	SnapshotCreateTime   time.Time `xml:"SnapshotCreateTime"`
	LicenseModel         string    `xml:"LicenseModel"`
	KmsKeyId             string    `xml:"KmsKeyId"`
	DBClusterIdentifier  string    `xml:"DBClusterIdentifier"`
	DBClusterSnapshotArn string    `xml:"DBClusterSnapshotArn"`
	DbClusterResourceId  string    `xml:"DbClusterResourceId"`
	Status               string    `xml:"Status"`
}

func (snapshot *SDBInstanceClusterSnapshot) GetId() string {
	return snapshot.DBClusterSnapshotArn
}

func (snapshot *SDBInstanceClusterSnapshot) GetGlobalId() string {
	return snapshot.DBClusterSnapshotArn
}

func (snapshot *SDBInstanceClusterSnapshot) GetName() string {
	return snapshot.DBClusterSnapshotIdentifier
}

func (snapshot *SDBInstanceClusterSnapshot) GetEngine() string {
	return snapshot.Engine
}

func (snapshot *SDBInstanceClusterSnapshot) GetEngineVersion() string {
	return snapshot.EngineVersion
}

func (snapshot *SDBInstanceClusterSnapshot) GetStartTime() time.Time {
	return snapshot.SnapshotCreateTime
}

func (snapshot *SDBInstanceClusterSnapshot) GetEndTime() time.Time {
	return snapshot.SnapshotCreateTime
}

func (snapshot *SDBInstanceClusterSnapshot) GetBackupMode() string {
	switch snapshot.SnapshotType {
	case "manual":
		return api.BACKUP_MODE_MANUAL
	default:
		return api.BACKUP_MODE_AUTOMATED
	}
}

func (snapshot *SDBInstanceClusterSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "available":
		return api.DBINSTANCE_BACKUP_READY
	default:
		log.Errorf("unknown dbinstance snapshot status: %s", snapshot.Status)
		return api.DBINSTANCE_BACKUP_UNKNOWN
	}
}

func (self *SDBInstanceClusterSnapshot) Refresh() error {
	snap, err := self.region.GetDBClusterSnapshot(self.DBClusterSnapshotIdentifier)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snap)
}

func (snapshot *SDBInstanceClusterSnapshot) GetBackupSizeMb() int {
	return snapshot.AllocatedStorage * 1024
}

func (snapshot *SDBInstanceClusterSnapshot) GetDBNames() string {
	return ""
}

func (snapshot *SDBInstanceClusterSnapshot) GetDBInstanceId() string {
	return snapshot.DBClusterIdentifier
}

func (self *SDBInstanceClusterSnapshot) Delete() error {
	return self.region.DeleteRdsSnapshot(self.DBClusterSnapshotIdentifier)
}

func (self *SRegion) GetDBClusterSnapshot(id string) (*SDBInstanceClusterSnapshot, error) {
	snapshots, err := self.DescribeDBClusterSnapshots("", id)
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == id {
			return &snapshots[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (region *SRegion) DescribeDBClusterSnapshots(clusterId, snapshotId string) ([]SDBInstanceClusterSnapshot, error) {
	params := map[string]string{}
	if len(clusterId) > 0 {
		params["DBClusterIdentifier"] = clusterId
	}
	if len(snapshotId) > 0 {
		params["DBClusterSnapshotIdentifier"] = snapshotId
	}
	ret := []SDBInstanceClusterSnapshot{}
	for {
		part := struct {
			DBClusterSnapshots struct {
				DBClusterSnapshot []SDBInstanceClusterSnapshot `xml:"DBClusterSnapshot"`
			}
			Marker string `xml:"Marker"`
		}{}
		err := region.rdsRequest("DescribeDBClusterSnapshots", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.DBClusterSnapshots.DBClusterSnapshot...)
		if len(part.DBClusterSnapshots.DBClusterSnapshot) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}
