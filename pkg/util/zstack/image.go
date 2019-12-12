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

package zstack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/imagetools"
	"yunion.io/x/onecloud/pkg/util/multipart"
)

type SBackupStorageRef struct {
	BackupStorageUUID string    `json:"backupStorageUuid"`
	CreateDate        time.Time `json:"createDate"`
	ImageUUID         string    `json:"ImageUuid"`
	InstallPath       string    `json:"installPath"`
	LastOpDate        time.Time `json:"lastOpDate"`
	Status            string    `json:"status"`
}

type SImage struct {
	storageCache *SStoragecache

	BackupStorageRefs []SBackupStorageRef `json:"backupStorageRefs"`
	ActualSize        int                 `json:"actualSize"`
	CreateDate        time.Time           `json:"createDate"`
	Description       string              `json:"description"`
	Format            string              `json:"format"`
	LastOpDate        time.Time           `json:"lastOpDate"`
	MD5Sum            string              `json:"md5sum"`
	MediaType         string              `json:"mediaType"`
	Name              string              `json:"name"`
	Platform          string              `json:"platform"`
	Size              int                 `json:"size"`
	State             string              `json:"state"`
	Status            string              `json:"Ready"`
	System            bool                `json:"system"`
	Type              string              `json:"type"`
	URL               string              `json:"url"`
	UUID              string              `json:"uuid"`
}

func (image *SImage) GetMinRamSizeMb() int {
	return 0
}

func (image *SImage) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	return data
}

func (image *SImage) GetId() string {
	return image.UUID
}

func (image *SImage) GetName() string {
	return image.Name
}

func (image *SImage) IsEmulated() bool {
	return false
}

func (image *SImage) Delete(ctx context.Context) error {
	return image.storageCache.region.DeleteImage(image.UUID)
}

func (image *SImage) GetGlobalId() string {
	return image.UUID
}

func (region *SRegion) DeleteImage(imageId string) error {
	err := region.client.delete("images", imageId, "")
	if err != nil {
		return err
	}
	params := map[string]interface{}{
		"expungeImage": jsonutils.NewDict(),
	}
	_, err = region.client.put("images", imageId, jsonutils.Marshal(params))
	return err
}

func (image *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return image.storageCache
}

func (image *SImage) GetStatus() string {
	if image.State != "Enabled" {
		return api.CACHED_IMAGE_STATUS_CACHING
	}
	switch image.Status {
	case "Ready":
		return api.CACHED_IMAGE_STATUS_READY
	case "Downloading":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "Deleted":
		return api.CACHED_IMAGE_STATUS_DELETING
	default:
		log.Errorf("Unknown image status: %s", image.Status)
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (image *SImage) GetImageStatus() string {
	switch image.Status {
	case "Ready":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	case "Deleted":
		return cloudprovider.IMAGE_STATUS_DELETED
	default:
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (image *SImage) Refresh() error {
	new, err := image.storageCache.region.GetImage(image.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(image, new)
}

func (image *SImage) GetImageType() string {
	return cloudprovider.CachedImageTypeSystem
}

func (image *SImage) GetSizeByte() int64 {
	return int64(image.Size)
}

func (image *SImage) GetOsType() string {
	return image.Platform
}

func (image *SImage) GetOsDist() string {
	osDist := imagetools.NormalizeImageInfo(image.URL, "", "", "", "").OsDistro
	if len(osDist) > 0 {
		return osDist
	}
	return image.Platform
}

func (image *SImage) GetOsVersion() string {
	return imagetools.NormalizeImageInfo(image.Name, "", "", "", "").OsVersion
}

func (image *SImage) GetOsArch() string {
	return ""
}

func (image *SImage) GetMinOsDiskSizeGb() int {
	return image.Size / 1024 / 1024 / 1024
}

func (image *SImage) GetImageFormat() string {
	return image.Format
}

func (image *SImage) GetCreateTime() time.Time {
	return image.CreateDate
}

func (region *SRegion) GetImage(imageId string) (*SImage, error) {
	image := &SImage{}
	err := region.client.getResource("images", imageId, image)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetImage")
	}
	return image, nil
}

func (region *SRegion) GetImages(zoneId string, imageId string) ([]SImage, error) {
	images := []SImage{}
	params := url.Values{}
	params.Add("q", "system=false")
	if len(zoneId) > 0 {
		params.Add("q", "backupStorage.zone.uuid="+zoneId)
	}
	if len(imageId) > 0 {
		params.Add("q", "uuid="+imageId)
	}
	if SkipEsxi {
		params.Add("q", "type!=vmware")
	}
	return images, region.client.listAll("images", params, &images)
}

func (region *SRegion) GetBackupStorageUUID(zondId string) ([]string, error) {
	imageServers, err := region.GetImageServers(zondId, "")
	if err != nil {
		return nil, err
	}
	if len(imageServers) == 0 {
		return nil, fmt.Errorf("failed to found any image servers")
	}
	servers := ImageServers(imageServers)
	sort.Sort(servers)
	return []string{servers[0].UUID}, nil
}

func (region *SRegion) CreateImage(zoneId string, imageName, format, osType, desc string, reader io.Reader, size int64) (*SImage, error) {
	backupStorageUUIDs, err := region.GetBackupStorageUUID(zoneId)
	if err != nil {
		return nil, err
	}
	platform := ""
	switch osType {
	case "linux":
		platform = "Linux"
	case "windows":
		platform = "Windows"
	default:
		platform = "Other"
	}
	parmas := map[string]interface{}{
		"params": map[string]interface{}{
			"name":               imageName,
			"url":                fmt.Sprintf("upload://%s", imageName),
			"description":        desc,
			"mediaType":          "RootVolumeTemplate",
			"system":             false,
			"format":             format,
			"platform":           platform,
			"backupStorageUuids": backupStorageUUIDs,
			"systemTags":         []string{"qemuga", "bootMode::Legacy"},
		},
	}

	if reader == nil {
		return nil, fmt.Errorf("invalid reader")
	}

	if size == 0 {
		return nil, fmt.Errorf("invalid image size")
	}

	body := multipart.NewReader(reader, "", imageName)

	image := &SImage{}
	err = region.client.create("images", jsonutils.Marshal(parmas), image)
	if err != nil {
		return nil, err
	}

	if len(image.BackupStorageRefs) < 0 {
		return nil, fmt.Errorf("no InstallPath reture")
	}
	header := http.Header{}
	header.Add("X-IMAGE-UUID", image.UUID)
	header.Add("X-IMAGE-SIZE", fmt.Sprintf("%d", size))
	header.Add("Content-Type", body.FormDataContentType())
	resp, err := httputils.Request(httputils.GetTimeoutClient(0), context.Background(), "POST", image.BackupStorageRefs[0].InstallPath, header, body, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return image, nil
}
