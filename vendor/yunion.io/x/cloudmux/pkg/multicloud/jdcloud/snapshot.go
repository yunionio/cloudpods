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

package jdcloud

import (
	"fmt"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/disk/models"

	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	JdcloudTags
	region *SRegion

	models.Snapshot
	disk *SDisk
}

func (s *SSnapshot) GetId() string {
	return s.SnapshotId
}

func (s *SSnapshot) GetName() string {
	return s.Name
}

func (s *SSnapshot) GetStatus() string {
	switch s.Status {
	case "creating":
		return api.SNAPSHOT_CREATING
	case "available", "in-use":
		return api.SNAPSHOT_READY
	case "deleting":
		return api.SNAPSHOT_DELETING
	case "error_create":
		return api.SNAPSHOT_FAILED
	case "error_delete":
		return api.SNAPSHOT_DELETE_FAILED
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (s *SSnapshot) GetSizeMb() int32 {
	return int32(s.SnapshotSizeGB) * 1024
}

func (s *SSnapshot) GetDiskId() string {
	return s.DiskId
}

func (s *SSnapshot) getDisk() (*SDisk, error) {
	if s.disk != nil {
		return s.disk, nil
	}
	disk, err := s.region.GetDiskById(s.DiskId)
	if err != nil {
		return disk, err
	}
	s.disk = disk
	return s.disk, nil
}

func (s *SSnapshot) GetDiskType() string {
	disk, err := s.getDisk()
	if err != nil {
		log.Errorf("unable to get disk %s", s.DiskId)
		return ""
	}
	return disk.GetDiskType()
}

func (s *SSnapshot) Refresh() error {
	return nil
}

func (s *SSnapshot) GetGlobalId() string {
	return s.GetId()
}

func (s *SSnapshot) IsEmulated() bool {
	return false
}

func (s *SSnapshot) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (s *SSnapshot) GetProjectId() string {
	return ""
}

func (r *SRegion) GetSnapshots(diskId string, pageNumber, pageSize int) ([]SSnapshot, int, error) {
	filters := []commodels.Filter{}
	if diskId != "" {
		filters = append(filters, commodels.Filter{
			Name:   "diskId",
			Values: []string{diskId},
		})
	}
	req := apis.NewDescribeSnapshotsRequestWithAllParams(r.ID, &pageNumber, &pageSize, nil, filters)
	client := client.NewDiskClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeSnapshots(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error.Code >= 400 {
		return nil, 0, fmt.Errorf(resp.Error.Message)
	}
	snapshots := make([]SSnapshot, len(resp.Result.Snapshots))
	for i := range resp.Result.Snapshots {
		snapshots = append(snapshots, SSnapshot{
			region:   r,
			Snapshot: resp.Result.Snapshots[i],
		})
	}
	return snapshots, resp.Result.TotalCount, nil
}

func (r *SRegion) GetSnapshotById(id string) (*SSnapshot, error) {
	req := apis.NewDescribeSnapshotRequest(r.ID, id)
	client := client.NewDiskClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeSnapshot(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		return nil, fmt.Errorf(resp.Error.Message)
	}
	snapshot := SSnapshot{
		region:   r,
		Snapshot: resp.Result.Snapshot,
	}
	return &snapshot, nil
}
