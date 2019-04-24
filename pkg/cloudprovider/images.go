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

package cloudprovider

import (
	"time"
)

const (
	IMAGE_STATUS_ACTIVE  = "active"
	IMAGE_STATUS_QUEUED  = "queued"
	IMAGE_STATUS_SAVING  = "saving"
	IMAGE_STATUS_KILLED  = "killed"
	IMAGE_STATUS_DELETED = "deleted"

	CachedImageTypeSystem     = "system"
	CachedImageTypeCustomized = "customized"
	CachedImageTypeShared     = "shared"
	CachedImageTypeMarket     = "market"
)

type SImage struct {
	Checksum string
	// ContainerFormat string
	CreatedAt  time.Time
	Deleted    bool
	DiskFormat string
	Id         string
	IsPublic   bool
	MinDiskMB  int `json:"min_disk"`
	MinRamMB   int `json:"min_ram"`
	Name       string
	Owner      string
	Properties map[string]string
	Protected  bool
	SizeBytes  int64 `json:"size"`
	Status     string
	// UpdatedAt       time.Time
}

func CloudImage2Image(image ICloudImage) SImage {
	return SImage{
		CreatedAt:  image.GetCreateTime(),
		Deleted:    false,
		DiskFormat: image.GetImageFormat(),
		Id:         image.GetId(),
		IsPublic:   image.GetImageType() != CachedImageTypeCustomized,
		MinDiskMB:  image.GetMinOsDiskSizeGb() * 1024,
		MinRamMB:   image.GetMinRamSizeMb(),
		Name:       image.GetName(),
		Properties: map[string]string{
			"os_type":         image.GetOsType(),
			"os_distribution": image.GetOsDist(),
			"os_version":      image.GetOsVersion(),
			"os_arch":         image.GetOsArch(),
		},
		Protected: true,
		SizeBytes: image.GetSize(),
		Status:    image.GetImageStatus(),
	}
}
