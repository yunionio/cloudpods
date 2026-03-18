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
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	EcloudTags
	region *SRegion
	SCreateTime

	BackupType  string `json:"backupType,omitempty"`
	CreateBy    string `json:"createBy,omitempty"`
	Description string `json:"description,omitempty"`
	EcType      string `json:"ecType,omitempty"`
	Id          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Size        int    `json:"size"`
	VolumeId    string `json:"volumeId,omitempty"`
	VolumeType  string `json:"volumeType,omitempty"`
	Status      string `json:"status,omitempty"`
	IsSystem    bool   `json:"isSystem,omitempty"`
	SystemDisk  string `json:"systemDisk,omitempty"`
	// EBS 快照列表返回 createTime
	CreateTime string `json:"createTime,omitempty"`
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

func (s *SSnapshot) GetCreatedAt() time.Time {
	if s.CreateTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", s.CreateTime); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, s.CreateTime); err == nil {
			return t
		}
	}
	return s.SCreateTime.GetCreatedAt()
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
	if s.region == nil {
		return cloudprovider.ErrNotImplemented
	}
	return s.region.DeleteEbsSnapshot(s.Id)
}

func (s *SSnapshot) GetProjectId() string {
	return ""
}

func (s *SRegion) GetSnapshots(snapshotId string, parentId string, isSystem bool) ([]SSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}
