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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

func (self *SRegion) GetElasticacheSnapshots(replicaGroupId string, snapshotName string) ([]SElasticacheSnapshot, error) {
	params := map[string]string{}
	if len(replicaGroupId) > 0 {
		params["ReplicationGroupId"] = replicaGroupId
	}
	if len(snapshotName) > 0 {
		params["SnapshotName"] = snapshotName
	}
	ret := []SElasticacheSnapshot{}
	for {
		result := struct {
			Marker    string                 `xml:"Marker"`
			Snapshots []SElasticacheSnapshot `xml:"Snapshots>Snapshot"`
		}{}
		err := self.redisRequest("DescribeSnapshots", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSnapshots")
		}
		ret = append(ret, result.Snapshots...)
		if len(result.Marker) == 0 || len(result.Snapshots) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

type NodeGroupConfiguration struct {
	NodeGroupId              string   `xml:"NodeGroupId"`
	PrimaryAvailabilityZone  string   `xml:"PrimaryAvailabilityZone"`
	PrimaryOutpostArn        string   `xml:"PrimaryOutpostArn"`
	ReplicaAvailabilityZones []string `xml:"ReplicaAvailabilityZones>AvailabilityZone"`
	ReplicaCount             int64    `xml:"ReplicaCount"`
	ReplicaOutpostArns       []string `xml:"ReplicaOutpostArns>OutpostArn"`
	Slots                    string   `xml:"Slots"`
}

type NodeSnapshot struct {
	CacheClusterId         string                 `xml:"CacheClusterId"`
	CacheNodeCreateTime    time.Time              `xml:"CacheNodeCreateTime"`
	CacheNodeId            string                 `xml:"CacheNodeId"`
	CacheSize              string                 `xml:"CacheSize"`
	NodeGroupConfiguration NodeGroupConfiguration `xml:"NodeGroupConfiguration"`
	NodeGroupId            string                 `xml:"NodeGroupId"`
	SnapshotCreateTime     time.Time              `xml:"SnapshotCreateTime"`
}

type SElasticacheSnapshot struct {
	multicloud.SElasticcacheBackupBase
	multicloud.AwsTags
	cache *SElasticache

	ARN                         string         `xml:"ARN"`
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
	ReplicationGroupId          string         `xml:"ReplicationGroupId"`
	SnapshotName                string         `xml:"SnapshotName"`
	SnapshotRetentionLimit      int64          `xml:"SnapshotRetentionLimit"`
	SnapshotSource              string         `xml:"SnapshotSource"`
	SnapshotStatus              string         `xml:"SnapshotStatus"`
	SnapshotWindow              string         `xml:"SnapshotWindow"`
	TopicArn                    string         `xml:"TopicArn"`
	VpcId                       string         `xml:"VpcId"`
}

func (self *SElasticacheSnapshot) GetId() string {
	return self.SnapshotName
}

func (self *SElasticacheSnapshot) GetName() string {
	return self.SnapshotName
}

func (self *SElasticacheSnapshot) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheSnapshot) GetStatus() string {
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
		return api.ELASTIC_CACHE_BACKUP_STATUS_UNKNOWN
	}
}

func (self *SElasticacheSnapshot) Refresh() error {
	snapshots, err := self.cache.region.GetElasticacheSnapshots(self.cache.ReplicationGroupId, self.GetName())
	if err != nil {
		return errors.Wrapf(err, `self.region.DescribeSnapshots("", %s)`, self.GetName())
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == self.GetGlobalId() {
			return jsonutils.Update(self, snapshots[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.GetGlobalId())
}

func (self *SElasticacheSnapshot) GetBackupSizeMb() int {
	total := 0
	for _, node := range self.NodeSnapshots {
		splited := strings.Split(node.CacheSize, " ")
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

func (self *SElasticacheSnapshot) GetBackupType() string {
	return api.ELASTIC_CACHE_BACKUP_TYPE_FULL
}

func (self *SElasticacheSnapshot) GetBackupMode() string {
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

func (self *SElasticacheSnapshot) GetDownloadURL() string {
	return ""
}

func (self *SElasticacheSnapshot) GetStartTime() time.Time {
	for _, node := range self.NodeSnapshots {
		return node.SnapshotCreateTime
	}
	return time.Time{}
}

func (self *SElasticacheSnapshot) GetEndTime() time.Time {
	return time.Time{}
}

func (self *SElasticacheSnapshot) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticacheSnapshot) RestoreInstance(instanceId string) error {
	return cloudprovider.ErrNotSupported
}
