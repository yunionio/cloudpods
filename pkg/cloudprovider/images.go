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

type TImageType string

const (
	IMAGE_STATUS_ACTIVE  = "active"
	IMAGE_STATUS_QUEUED  = "queued"
	IMAGE_STATUS_SAVING  = "saving"
	IMAGE_STATUS_KILLED  = "killed"
	IMAGE_STATUS_DELETED = "deleted"

	ImageTypeSystem     = TImageType("system")
	ImageTypeCustomized = TImageType("customized")
	ImageTypeShared     = TImageType("shared")
	ImageTypeMarket     = TImageType("market")
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
	PublicScope string
	ExternalId  string

	// SubImages record the subImages of the guest image.
	// For normal image, it's nil.
	SubImages []SSubImage
}

type SSubImage struct {
	Index     int
	MinDiskMB int
	MinRamMb  int
	SizeBytes int64
}

type SaveImageOptions struct {
	Name  string
	Notes string
}

func CloudImage2Image(image ICloudImage) SImage {
	return SImage{
		CreatedAt:  image.GetCreatedAt(),
		Deleted:    false,
		DiskFormat: image.GetImageFormat(),
		Id:         image.GetId(),
		IsPublic:   image.GetImageType() != ImageTypeCustomized,
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
		SizeBytes: image.GetSizeByte(),
		Status:    image.GetImageStatus(),
		SubImages: image.GetSubImages(),
	}
}

type SImageCreateOption struct {
	ImageId        string
	ExternalId     string
	ImageName      string
	OsType         string
	OsArch         string
	OsDistribution string
	OsVersion      string
	OsFullVersion  string
}
