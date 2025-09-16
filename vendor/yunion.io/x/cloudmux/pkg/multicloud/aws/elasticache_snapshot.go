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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type NodeSnapshot struct {
	CacheClusterId      string    `xml:"CacheClusterId"`
	CacheNodeCreateTime time.Time `xml:"CacheNodeCreateTime"`
	CacheNodeId         string    `xml:"CacheNodeId"`
	CacheSize           string    `xml:"CacheSize"`
	NodeGroupId         string    `xml:"NodeGroupId"`
	SnapshotCreateTime  time.Time `xml:"SnapshotCreateTime"`
}

type SElasticacheSnapshop struct {
	multicloud.SElasticcacheBackupBase
	AwsTags
	region *SRegion

	ARN                         string         `xml:"Arn"`
	AutoMinorVersionUpgrade     bool           `xml:"AutoMinorVersionUpgrade"`
	AutomaticFailover           string         `xml:"AutomaticFailover"`
	CacheClusterCreateTime      time.Time      `xml:"CacheClusterCreateTime"`
	CacheClusterId              string         `xml:"CacheClusterId"`
	CacheNodeType               string         `xml:"CacheNodeType"`
	CacheParameterGroupName     string         `xml:"CacheParameterGroupName"`
	CacheSubnetGroupName        string         `xml:"CacheSubnetGroupName"`
	Engine                      string         `xml:"Engine"`
	EngineVersion               string         `xml:"EngineVersion"`
	KmsKeyId                    string         `xml:"KmsKeyId"`
	NodeSnapshots               []NodeSnapshot `xml:"NodeSnapshots>NodeSnapshot"`
	NumCacheNodes               int64          `xml:"NumCacheNodes"`
	NumNodeGroups               int64          `xml:"NumNodeGroups"`
	Port                        int64          `xml:"Port"`
	PreferredAvailabilityZone   string         `xml:"PreferredAvailabilityZone"`
	PreferredMaintenanceWindow  string         `xml:"PreferredMaintenanceWindow"`
	PreferredOutpostArn         string         `xml:"PreferredOutpostArn"`
	ReplicationGroupDescription string         `xml:"ReplicationGroupDescription"`
	ReplicationGroupId          string         `xml:"ReplicaGroupId"`
	SnapshotName                string         `xml:"SnapshotName"`
	SnapshotRetentionLimit      int64          `xml:"SnapshotRetentionLimit"`
	SnapshotSource              string         `xml:"SnapshotSource"`
	SnapshotStatus              string         `xml:"SnapshotStatus"`
	SnapshotWindow              string         `xml:"SnapshotWindow"`
	TopicArn                    string         `xml:"TopicArn"`
	VpcId                       string         `xml:"VpcId"`
}

func (self *SElasticacheSnapshop) GetId() string {
	return self.SnapshotName
}

func (self *SElasticacheSnapshop) GetName() string {
	return self.SnapshotName
}

func (self *SElasticacheSnapshop) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheSnapshop) GetStatus() string {
	// creating | available | restoring | copying | deleting
	switch self.SnapshotStatus {
	case "creating":
		return api.ELASTIC_CACHE_BACKUP_STATUS_CREATING
	case "available":
		return api.ELASTIC_CACHE_BACKUP_STATUS_SUCCESS
	case "restoring":
		return api.ELASTIC_CACHE_BACKUP_STATUS_RESTORING
	case "copying":
		return api.ELASTIC_CACHE_BACKUP_STATUS_COPYING
	case "deleting":
		return api.ELASTIC_CACHE_BACKUP_STATUS_DELETING
	default:
		return self.SnapshotStatus
	}
}

func (self *SElasticacheSnapshop) Refresh() error {
	snapshots, err := self.region.GetCacheSnapshots(self.ReplicationGroupId, self.GetName())
	if err != nil {
		return errors.Wrapf(err, `self.region.DescribeSnapshots("", %s)`, self.GetName())
	}
	for i := range snapshots {
		if snapshots[i].SnapshotName == self.SnapshotName {
			return jsonutils.Update(self, snapshots[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.GetName())
}

func (self *SElasticacheSnapshop) GetBackupSizeMb() int {
	total := 0
	for i := range self.NodeSnapshots {
		sizeStr := self.NodeSnapshots[i].CacheSize
		splited := strings.Split(sizeStr, " ")
		if len(splited) != 2 {
			return 0
		}
		size, err := strconv.Atoi(splited[0])
		if err != nil {
			return 0
		}
		switch splited[1] {
		case "MB":
			total += size
		case "GB":
			total += size * 1024
		case "TB":
			total += size * 1024 * 1024
		}
	}
	return total
}

func (self *SElasticacheSnapshop) GetBackupType() string {
	return api.ELASTIC_CACHE_BACKUP_TYPE_FULL
}

func (self *SElasticacheSnapshop) GetBackupMode() string {
	// automated manual
	switch self.SnapshotSource {
	case "automated":
		return api.ELASTIC_CACHE_BACKUP_MODE_AUTOMATED
	case "manual":
		return api.ELASTIC_CACHE_BACKUP_MODE_MANUAL
	default:
		return self.SnapshotSource
	}
	return ""
}

func (self *SElasticacheSnapshop) GetDownloadURL() string {
	return ""
}

func (self *SElasticacheSnapshop) GetStartTime() time.Time {
	for _, nodeSnapshot := range self.NodeSnapshots {
		return nodeSnapshot.SnapshotCreateTime
	}
	return time.Time{}
}

func (self *SElasticacheSnapshop) GetEndTime() time.Time {
	return time.Time{}
}

func (self *SElasticacheSnapshop) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticacheSnapshop) RestoreInstance(instanceId string) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) GetCacheSnapshots(replicaGroupId string, snapshotName string) ([]SElasticacheSnapshop, error) {
	params := map[string]string{}
	if len(replicaGroupId) > 0 {
		params["ReplicationGroupId"] = replicaGroupId
	}
	if len(snapshotName) > 0 {
		params["SnapshotName"] = snapshotName
	}
	ret := []SElasticacheSnapshop{}
	for {
		part := struct {
			Snapshots []SElasticacheSnapshop `xml:"Snapshots>Snapshot"`
			Marker    string
		}{}
		err := region.ecRequest("DescribeSnapshots", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Snapshots...)
		if len(part.Snapshots) == 0 || len(part.Marker) == 0 {
			return nil, err
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}
