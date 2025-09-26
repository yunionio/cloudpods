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

package ksyun

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
)

/*
   {
     "SnapshotId": "780618d7-05d0-4445-8905-94a5a1fe8a3d",
     "SnapshotName": "test-snap",
     "SnapshotDesc": "",
     "VolumeId": "d9329bb8-fa1d-4e13-ac36-64096fea1a3a",
     "Size": 20,
     "BillSize": "0",
     "ProjectAvailable": true,
     "ProjectId": 0,
     "CreateTime": "2025-09-25 16:53:41",
     "SnapshotStatus": "available",
     "VolumeCategory": "data",
     "VolumeName": "test-disk",
     "VolumeType": "SSD3.0",
     "Progress": "100%",
     "AvailabilityZone": "cn-beijing-6a",
     "VolumeStatus": "available",
     "SnapshotType": "CommonSnapShot",
     "AutoSnapshot": false,
     "ImageRelated": false,
     "CopyFrom": false,
     "EbsClusterType": "Public"
   }
*/

type SSnapshot struct {
	multicloud.SVirtualResourceBase
	SKsyunTags
	region *SRegion

	SnapshotId       string
	VolumeId         string
	SnapshotName     string
	SnapshotDesc     string
	Size             int
	BillSize         string
	ProjectAvailable bool
	ProjectId        int
	SnapshotStatus   string
	VolumeCategory   string
	VolumeName       string
	VolumeType       string
	Progress         string
	AvailabilityZone string
	VolumeStatus     string
	SnapshotType     string
	AutoSnapshot     bool
	ImageRelated     bool
	CopyFrom         bool
	EbsClusterType   string
	CreateTime       time.Time
}

func (snap *SSnapshot) GetId() string {
	return snap.SnapshotId
}

func (snap *SSnapshot) GetName() string {
	if len(snap.SnapshotName) > 0 {
		return snap.SnapshotName
	}
	return snap.SnapshotId
}

func (snap *SSnapshot) GetStatus() string {
	switch snap.SnapshotStatus {
	case "available":
		return api.SNAPSHOT_READY
	case "creating":
		return api.SNAPSHOT_CREATING
	case "failed":
		return api.SNAPSHOT_FAILED
	default:
		return strings.ToLower(snap.SnapshotStatus)
	}
}

func (snap *SSnapshot) GetSizeMb() int32 {
	return int32(snap.Size * 1024)
}

func (snap *SSnapshot) GetDiskId() string {
	return snap.VolumeId
}

func (snap *SSnapshot) GetDiskType() string {
	if snap.VolumeCategory == "data" {
		return api.DISK_TYPE_DATA
	} else if snap.VolumeCategory == "system" {
		return api.DISK_TYPE_SYS
	} else {
		return ""
	}
}

func (snap *SSnapshot) Refresh() error {
	ret, err := snap.region.GetSnapshot(snap.SnapshotId)
	if err != nil {
		return err
	}
	return jsonutils.Update(snap, ret)
}

func (snap *SSnapshot) GetGlobalId() string {
	return snap.SnapshotId
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := region.GetSnapshots("", "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = region
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (region *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]string{
		"VolumeId":     diskId,
		"SnapshotName": name,
		"SnapshotDesc": desc,
	}

	body, err := region.ebsRequest("CreateSnapshot", params)
	if err != nil {
		return nil, err
	}
	id, err := body.GetString("SnapshotId")
	if err != nil {
		return nil, err
	}
	return region.GetSnapshot(id)
}

func (region *SRegion) GetSnapshots(snapshotId, volumeId string) ([]SSnapshot, error) {
	params := map[string]string{
		"PageSize": "1000",
	}
	if len(snapshotId) > 0 {
		params["SnapshotId"] = snapshotId
	}
	if len(volumeId) > 0 {
		params["VolumeId"] = volumeId
	}
	PageNumber := 1
	ret := []SSnapshot{}
	for {
		params["PageNumber"] = fmt.Sprintf("%d", PageNumber)
		body, err := region.ebsRequest("DescribeSnapshots", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Snapshots []SSnapshot
			Page      struct {
				TotalCount int64
			}
		}{}
		err = body.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Snapshots...)
		if len(part.Snapshots) == 0 || len(ret) > int(part.Page.TotalCount) {
			break
		}
		PageNumber++
	}
	return ret, nil
}

func (region *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshots, err := region.GetSnapshots("", id)
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		snapshots[i].region = region
		if snapshots[i].SnapshotId == id {
			return &snapshots[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	params := map[string]string{
		"SnapshotId": snapshotId,
	}
	_, err := region.ebsRequest("DeleteSnapshot", params)
	return err
}
