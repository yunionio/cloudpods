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

package ecloud

import (
	"context"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	multicloud.EcloudTags
	region *SRegion
	SCreateTime

	BackupType  string
	CreateBy    string
	Description string
	EcType      string
	Id          string
	Name        string
	Size        int
	VolumeId    string
	VolumeType  string
	Status      string
	IsSystem    bool
	SystemDisk  string
}

func (s *SSnapshot) GetId() string {
	return s.Id
}

func (s *SSnapshot) GetName() string {
	return s.Name
}

func (s *SSnapshot) GetStatus() string {
	switch strings.ToLower(s.Status) {
	case "available", "in_use", "active":
		return api.SNAPSHOT_READY
	case "creating", "attaching", "backing_up", "saving", "queued":
		return api.SNAPSHOT_CREATING
	case "deleting":
		return api.SNAPSHOT_DELETING
	case "error_deleting":
		return api.SNAPSHOT_DELETE_FAILED
	case "error":
		return api.SNAPSHOT_FAILED
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (s *SSnapshot) GetSizeMb() int32 {
	return int32(s.Size)
}

func (s *SSnapshot) GetDiskId() string {
	if s.IsSystem {
		return s.SystemDisk
	}
	return s.VolumeId
}

func (s *SSnapshot) GetDiskType() string {
	if s.IsSystem {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (s *SSnapshot) GetGlobalId() string {
	return s.Id
}

func (s *SSnapshot) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (s *SSnapshot) GetProjectId() string {
	return ""
}

func (s *SRegion) GetSnapshots(snapshotId string, parentId string, isSystem bool) ([]SSnapshot, error) {
	var apiRequest *SApiRequest
	query := map[string]string{}
	if len(snapshotId) > 0 {
		query["backupId"] = snapshotId
	}
	if isSystem {
		if len(parentId) > 0 {
			query["serverId"] = parentId
		}
		apiRequest = NewApiRequest(s.ID, "/api/v2/vmBackup", query, nil)
	} else {
		if len(parentId) > 0 {
			query["volumeId"] = parentId
		}
		apiRequest = NewApiRequest(s.ID, "/api/v2/volume/volumebackup", query, nil)
	}
	request := NewNovaRequest(apiRequest)
	snapshots := make([]SSnapshot, 0)
	err := s.client.doList(context.Background(), request, &snapshots)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
