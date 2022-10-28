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

	"github.com/aws/aws-sdk-go/service/elasticache"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (region *SRegion) DescribeSnapshots(replicaGroupId string, snapshotName string) ([]*elasticache.Snapshot, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}

	input := elasticache.DescribeSnapshotsInput{}
	if len(replicaGroupId) > 0 {
		input.ReplicationGroupId = &replicaGroupId
	}
	if len(snapshotName) > 0 {
		input.SnapshotName = &snapshotName
	}
	marker := ""
	maxrecords := (int64)(50)
	input.MaxRecords = &maxrecords

	snapshot := []*elasticache.Snapshot{}
	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeSnapshots(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ecClient.DescribeCacheClusters")
		}
		snapshot = append(snapshot, out.Snapshots...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return snapshot, nil
}

type SElasticacheSnapshop struct {
	multicloud.SElasticcacheBackupBase
	AwsTags
	region *SRegion

	snapshot *elasticache.Snapshot
}

func (self *SElasticacheSnapshop) GetId() string {
	return *self.snapshot.SnapshotName
}

func (self *SElasticacheSnapshop) GetName() string {
	return *self.snapshot.SnapshotName
}

func (self *SElasticacheSnapshop) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheSnapshop) GetStatus() string {
	if self.snapshot == nil || self.snapshot.SnapshotStatus == nil {
		return api.ELASTIC_CACHE_BACKUP_STATUS_UNKNOWN
	}
	// creating | available | restoring | copying | deleting
	switch *self.snapshot.SnapshotStatus {
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
		return ""
	}
}

func (self *SElasticacheSnapshop) Refresh() error {
	snapshots, err := self.region.DescribeSnapshots("", self.GetName())
	if err != nil {
		return errors.Wrapf(err, `self.region.DescribeSnapshots("", %s)`, self.GetName())
	}

	if len(snapshots) == 0 {
		return cloudprovider.ErrNotFound
	}
	if len(snapshots) > 1 {
		return cloudprovider.ErrDuplicateId
	}

	self.snapshot = snapshots[0]
	return nil
}

func (self *SElasticacheSnapshop) GetBackupSizeMb() int {
	total := 0
	if self.snapshot != nil && len(self.snapshot.NodeSnapshots) > 0 {
		for i := range self.snapshot.NodeSnapshots {
			if self.snapshot.NodeSnapshots[i] != nil && self.snapshot.NodeSnapshots[0].CacheSize != nil {
				sizeStr := *self.snapshot.NodeSnapshots[0].CacheSize
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
		}
	}

	return total
}

func (self *SElasticacheSnapshop) GetBackupType() string {
	return api.ELASTIC_CACHE_BACKUP_TYPE_FULL
}

func (self *SElasticacheSnapshop) GetBackupMode() string {
	if self.snapshot != nil && self.snapshot.SnapshotSource != nil {
		source := *self.snapshot.SnapshotSource
		// automated manual
		switch source {
		case "automated":
			return api.ELASTIC_CACHE_BACKUP_MODE_AUTOMATED
		case "manual":
			return api.ELASTIC_CACHE_BACKUP_MODE_MANUAL
		default:
			return source
		}
	}
	return ""
}

func (self *SElasticacheSnapshop) GetDownloadURL() string {
	return ""
}

func (self *SElasticacheSnapshop) GetStartTime() time.Time {
	if self.snapshot == nil {
		return time.Time{}
	}
	for _, nodeSnapshot := range self.snapshot.NodeSnapshots {
		if nodeSnapshot != nil && nodeSnapshot.SnapshotCreateTime != nil {
			return *nodeSnapshot.SnapshotCreateTime
		}
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
