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

package ucloud

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (self *SStoragecache) fetchImages() error {
	images, err := self.region.GetImages("", "")
	if err != nil {
		return err
	}

	for i := range images {
		image := images[i]
		image.storageCache = self
		self.iimages = append(self.iimages, &image)
	}

	return nil
}

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), strings.ToLower(imageId))
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		err := self.fetchImages()
		if err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := self.region.GetImage(extId)
	image.storageCache = self
	return &image, err
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// https://docs.ucloud.cn/api/uhost-api/import_custom_image
func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.ExternalId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", image.ExternalId)

		img, err := self.region.GetImage(image.ExternalId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if img.GetStatus() == api.CACHED_IMAGE_STATUS_READY && !isForce {
			return image.ExternalId, nil
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return self.uploadImage(ctx, userCred, image, isForce)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, isForce bool) (string, error) {
	if len(image.OsVersion) == 0 {
		return "", fmt.Errorf("uploadImage os version is empty")
	}

	bucketName := GetBucketName(self.region.GetId(), image.ImageId)

	// create bucket
	exist, err := self.region.IBucketExist(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "self.region.IBucketExist")
	}
	if !exist {
		err = self.region.CreateBucket(bucketName, "private")
		if err != nil {
			return "", errors.Wrap(err, "CreateBucket")
		}
	}
	defer func() {
		e := self.region.DeleteBucket(bucketName)
		if e != nil {
			log.Errorf("uploadImage delete bucket %s", e.Error())
		}
	}()

	// upload to  ucloud
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	meta, reader, err := modules.Images.Download(s, image.ImageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", err
	}
	log.Debugf("Images meta data %s", meta)
	minDiskMB, _ := meta.Int("min_disk")
	minDiskGB := int64(math.Ceil(float64(minDiskMB) / 1024))

	if minDiskGB < 40 {
		minDiskGB = 40
	} else if minDiskGB > 1024 {
		minDiskGB = 1024
	}
	size, _ := meta.Int("size")
	md5, _ := meta.GetString("checksum")
	diskFormat, _ := meta.GetString("disk_format")
	// upload to ucloud
	bucket, err := self.region.GetIBucketById(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	file := SFile{
		bucket: bucket.(*SBucket),
		file:   reader,

		Size:     size,
		FileName: image.ImageId,
		Hash:     md5,
	}

	err = file.Upload()
	if err != nil {
		return "", err
	}
	defer func() {
		e := file.Delete()
		if e != nil {
			log.Errorf("uploadImage delete object %s", e.Error())
		}
	}() // remove object

	// check image name, avoid name conflict
	imageBaseName := image.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	for {
		_, err = self.region.GetImageByName(imageName)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}

		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx += 1
		log.Debugf("uploadImage Match remote name %s", imageName)
	}

	imgId, err := self.region.ImportImage(imageName, file.FetchFileUrl(), image.OsDistribution, image.OsVersion, diskFormat)

	if err != nil {
		log.Errorf("ImportImage error %s %s", file.FetchFileUrl(), err)
		return "", err
	}

	// timeout: 1hour = 3600 seconds
	err = cloudprovider.WaitCreated(60*time.Second, 3600*time.Second, func() bool {
		image, err := self.region.GetImage(imgId)
		if err == nil && image.State == "Available" {
			return true
		}

		return false
	})
	if err != nil {
		return "", errors.Wrap(err, "UploadImage")
	}
	return imgId, err
}

// https://docs.ucloud.cn/api/uhost-api/create_custom_image
func (self *SRegion) createIImage(snapshotId, imageName, imageDesc string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

/*func (self *SRegion) GetBucketDomain(name string) (string, error) {
	params := NewUcloudParams()
	params.Set("BucketName", name)

	res := make([]SBucket, 0)
	err := self.client.DoListAll("DescribeBucket", params, &res)
	if err != nil {
		return "", err
	}

	if len(res) == 1 && len(res[0].Domain.Src) >= 1 {
		return res[0].Domain.Src[0], nil
	} else {
		return "", fmt.Errorf("GetBucketDomain failed. %v", res)
	}
}*/

// https://docs.ucloud.cn/api/ufile-api/create_bucket
func (self *SRegion) CreateBucket(name, bucketType string) error {
	params := NewUcloudParams()
	params.Set("BucketName", name)
	params.Set("Type", bucketType)
	err := self.DoAction("CreateBucket", params, nil)
	if err != nil {
		return err
	}
	self.client.invalidateIBuckets()
	return nil
}

// https://docs.ucloud.cn/api/ufile-api/delete_bucket
func (self *SRegion) DeleteBucket(name string) error {
	params := NewUcloudParams()
	params.Set("BucketName", name)
	err := self.DoAction("DeleteBucket", params, nil)
	if err != nil {
		return err
	}
	self.client.invalidateIBuckets()
	return nil
}

// https://docs.ucloud.cn/api/ufile-api/put_file
func (self *SRegion) uploadObj(name string) error {
	params := NewUcloudParams()
	params.Set("BucketName", name)
	return self.DoAction("PutFile", params, nil)
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	images, err := self.GetImages("Custom", "")
	if err != nil {
		return nil, err
	}

	for i := range images {
		if images[i].GetName() == name {
			return &images[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func normalizeOsType(osType string) string {
	switch strings.ToLower(osType) {
	case "centos":
		return "CentOS"
	case "ubuntu":
		return "Ubuntu"
	case "windows":
		return "Windows"
	case "redHat":
		return "RedHat"
	case "debian":
		return "Debian"
	default:
		return "Other"
	}
}

func normalizeOsName(normalizeOsType string, fullVersion string) string {
	if utils.IsInStringArray(normalizeOsType, []string{"CentOS", "RedHat", "Debian"}) {
		if len(fullVersion) >= 3 {
			return fmt.Sprintf("%s %s 64位", normalizeOsType, fullVersion[0:3])
		} else {
			return fmt.Sprintf("%s %s.0 64位", normalizeOsType, fullVersion[0:1])
		}
	}

	if normalizeOsType == "Ubuntu" {
		if len(fullVersion) >= 5 {
			return fmt.Sprintf("Ubuntu %s 64位", fullVersion[0:5])
		} else if len(fullVersion) >= 2 {
			return fmt.Sprintf("Ubuntu %s.04 64位", fullVersion[0:2])
		} else {
			// 默认猜一个？
			return "Ubuntu 14.04 64位"
		}
	}

	if normalizeOsType == "Windows" {
		if len(fullVersion) >= 4 {
			return fmt.Sprintf("Windows %s 64位", fullVersion[0:4])
		} else {
			// 默认猜一个？
			return "Windows 2016 64位"
		}
	}

	return "Other"
}

func normalizeDiskFormat(diskFormat string) (string, error) {
	switch strings.ToLower(diskFormat) {
	case "raw":
		return "RAW", nil
	case "vhd":
		return "VHD", nil
	case "vmdk":
		return "VMDK", nil
	case "qcow2":
		return "qcow2", nil
	default:
		return "", fmt.Errorf("unsupported image format %s", diskFormat)
	}
}

// https://docs.ucloud.cn/api/uhost-api/import_custom_image
func (self *SRegion) ImportImage(name string, ufileUrl string, osType string, osVersion string, diskFormat string) (string, error) {
	format, err := normalizeDiskFormat(diskFormat)
	if err != nil {
		return "", err
	}

	nOsType := normalizeOsType(osType)

	params := NewUcloudParams()
	params.Set("ImageName", name)
	params.Set("UFileUrl", ufileUrl)
	// 操作系统平台，比如CentOS、Ubuntu、Windows、RedHat等，请参考控制台的镜像版本；若导入控制台上没有的操作系统，参数为Other
	params.Set("OsType", nOsType)
	// 操作系统详细版本，请参考控制台的镜像版本；OsType为Other时，输入参数为Other
	params.Set("OsName", normalizeOsName(nOsType, osVersion))
	params.Set("Format", format)
	params.Set("Auth", "True")
	params.Set("ImageDescription", osVersion)

	type SImageId struct {
		ImageId string
	}

	ret := SImageId{}
	log.Debugf("ImportImage with params %s", params.String())
	err = self.DoAction("ImportCustomImage", params, &ret)
	if err != nil {
		return "", err
	}

	return ret.ImageId, nil
}
