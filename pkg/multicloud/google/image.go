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

package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/imagetools"
)

type GuestOsFeature struct {
	Type string
}

type SDeprecated struct {
	State       string
	Replacement string
	Deprecated  bool
}

type SImage struct {
	storagecache *SStoragecache
	SResourceBase

	Id                string
	CreationTimestamp time.Time
	Description       string
	SourceType        string
	RawDisk           map[string]string
	Deprecated        SDeprecated
	Status            string
	ArchiveSizeBytes  int64
	DiskSizeGb        int
	Licenses          []string
	Family            string
	LabelFingerprint  string
	GuestOsFeatures   []GuestOsFeature
	LicenseCodes      []string
	StorageLocations  []string
	Kind              string
}

func (region *SRegion) SetProjectId(id string) {
	region.client.projectId = id
}

func (region *SRegion) GetAllAvailableImages() ([]SImage, error) {
	images := []SImage{}
	projectId := region.client.projectId
	for _, project := range []string{
		"centos-cloud",
		"ubuntu-os-cloud",
		"windows-cloud",
		"windows-sql-cloud",
		"suse-cloud",
		"suse-sap-cloud",
		"rhel-cloud",
		"rhel-sap-cloud",
		"cos-cloud",
		"coreos-cloud",
		"debian-cloud",
		projectId,
	} {
		_images, err := region.GetImages(project, 0, "")
		if err != nil {
			return nil, err
		}
		for _, image := range _images {
			if image.Deprecated.State == "" {
				images = append(images, image)
			}
		}
	}
	return images, nil
}

func (region *SRegion) GetImages(project string, maxResults int, pageToken string) ([]SImage, error) {
	images := []SImage{}
	resource := "global/images"
	params := map[string]string{}
	if len(project) > 0 {
		region.SetProjectId(project)
	}
	return images, region.List(resource, params, maxResults, pageToken, &images)
}

func (region *SRegion) GetImage(id string) (*SImage, error) {
	image := &SImage{}
	return image, region.Get(id, image)
}

func (image *SImage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (image *SImage) GetMinRamSizeMb() int {
	return 0
}

func (image *SImage) GetStatus() string {
	switch image.Status {
	case "READY":
		return api.CACHED_IMAGE_STATUS_READY
	case "FAILED":
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	case "PENDING":
		return api.CACHED_IMAGE_STATUS_SAVING
	default:
		log.Errorf("Unknown image status: %s", image.Status)
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (image *SImage) GetImageStatus() string {
	switch image.Status {
	case "READY":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "FAILED":
		return cloudprovider.IMAGE_STATUS_KILLED
	case "PENDING":
		return cloudprovider.IMAGE_STATUS_QUEUED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (image *SImage) Refresh() error {
	_image, err := image.storagecache.region.GetImage(image.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(image, _image)
}

func (image *SImage) GetImageType() string {
	if strings.Index(image.SelfLink, image.storagecache.region.GetProjectId()) >= 0 {
		return cloudprovider.CachedImageTypeCustomized
	}
	return cloudprovider.CachedImageTypeSystem
}

func (image *SImage) GetSizeByte() int64 {
	return image.ArchiveSizeBytes
}

func (image *SImage) GetOsType() string {
	return imagetools.NormalizeImageInfo(image.Name, "", "", "", "").OsType
}

func (image *SImage) GetOsDist() string {
	return imagetools.NormalizeImageInfo(image.Name, "", "", "", "").OsDistro
}

func (image *SImage) GetOsVersion() string {
	return imagetools.NormalizeImageInfo(image.Name, "", "", "", "").OsVersion
}

func (image *SImage) GetOsArch() string {
	return imagetools.NormalizeImageInfo(image.Name, "", "", "", "").OsArch
}

func (image *SImage) GetMinOsDiskSizeGb() int {
	return image.DiskSizeGb
}

func (image *SImage) GetCreatedAt() time.Time {
	return image.CreationTimestamp
}

func (image *SImage) GetImageFormat() string {
	return "raw"
}

func (image *SImage) IsEmulated() bool {
	return false
}

func (image *SImage) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (image *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return image.storagecache
}

func (region *SRegion) fetchImages() ([]SImage, error) {
	if len(region.client.images) > 0 {
		return region.client.images, nil
	}
	images, err := region.GetAllAvailableImages()
	if err != nil {
		return nil, err
	}
	region.client.images = images
	return images, nil
}

func (region *SRegion) CreateImage(name string, desc string, bucketName string, sourceFile string) (*SImage, error) {
	body := map[string]interface{}{
		"timeout": "7200s",
		"steps": []struct {
			Args []string
			Name string
		}{
			{
				Args: []string{
					fmt.Sprintf("-source_file=gs://%s/%s", bucketName, sourceFile),
					"-data_disk",
					"-timeout=7056s",
					"-image_name=" + name,
					"-no_guest_environment",
					"-client_id=onecloud",
					"-description=" + desc,
				},
				Name: "gcr.io/compute-image-tools/gce_vm_image_import:release",
			},
		},
		"tags": []string{"gce-daisy", "gce-daisy-image-import"},
	}
	err := region.CloudbuildInsert(jsonutils.Marshal(body))
	if err != nil {
		return nil, err
	}
	return region.GetImage(fmt.Sprintf("projects/%s/global/images/%s", region.GetProjectId(), name))
}
