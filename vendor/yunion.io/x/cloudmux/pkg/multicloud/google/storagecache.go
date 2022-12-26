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
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	GoogleTags
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (cache *SStoragecache) GetId() string {
	return cache.region.client.cpcfg.Id
}

func (cache *SStoragecache) GetName() string {
	return cache.region.client.cpcfg.Name
}

func (cache *SStoragecache) GetStatus() string {
	return "available"
}

func (cache *SStoragecache) Refresh() error {
	return nil
}

func (cache *SStoragecache) GetGlobalId() string {
	return cache.region.client.cpcfg.Id
}

func (cache *SStoragecache) IsEmulated() bool {
	return true
}

func (cache *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := cache.region.GetImages(cache.region.client.projectId, 1000, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := range images {
		images[i].storagecache = cache
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (cache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := cache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return cache.uploadImage(ctx, image, callback)
}

func (region *SRegion) checkAndCreateBucket(bucketName string) (*SBucket, error) {
	bucket, err := region.GetBucket(bucketName)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			bucket, err = region.CreateBucket(bucketName, "", cloudprovider.ACLPrivate)
			if err != nil {
				return nil, errors.Wrapf(err, "region.CreateBucket(%s)", bucketName)
			}
		} else {
			return nil, errors.Wrapf(err, "region.StorageGet(%s)", bucketName)
		}
	}
	return bucket, nil
}

func (cache *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	reader, sizeBytes, err := image.GetReader(image.ImageId, string(qemuimgfmt.QCOW2))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	bucketName := fmt.Sprintf("imagecache-%s", image.ImageId)
	bucket, err := cache.region.checkAndCreateBucket(bucketName)
	if err != nil {
		return "", errors.Wrapf(err, "checkAndCreateBucket(%s)", bucketName)
	}

	defer cache.region.DeleteBucket(bucket.Name)

	log.Debugf("To upload image to bucket %s ...", bucketName)
	body := multicloud.NewProgress(sizeBytes, 80, reader, callback)
	err = cloudprovider.UploadObject(context.Background(), bucket, image.ImageId, 0, body, sizeBytes, "", "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "UploadObjectWithProgress")
	}

	images, err := cache.region.GetImages(cache.region.GetProjectId(), 0, "")
	if err != nil {
		return "", errors.Wrap(err, "region.GetImages")
	}
	imageNames := []string{}
	for _, image := range images {
		imageNames = append(imageNames, image.Name)
	}

	imageName := "img-"
	for _, s := range strings.ToLower(image.ImageName) {
		if unicode.IsDigit(s) || unicode.IsLetter(s) || s == '-' {
			imageName = fmt.Sprintf("%s%c", imageName, s)
		} else {
			imageName = fmt.Sprintf("%s-", imageName)
		}
	}

	baseName := imageName
	for i := 0; i < 30; i++ {
		if !utils.IsInStringArray(imageName, imageNames) {
			break
		}
		imageName = fmt.Sprintf("%s-%d", baseName, i)
	}

	_image, err := cache.region.CreateImage(imageName, image.Description, bucketName, image.ImageName)
	if err != nil {
		return "", errors.Wrap(err, "region.CreateImage")
	}

	if callback != nil {
		callback(100)
	}

	return _image.GetGlobalId(), nil
}

func (cache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	cache := &SStoragecache{region: region}
	return []cloudprovider.ICloudStoragecache{cache}, nil
}

func (region *SRegion) getStoragecache() *SStoragecache {
	return &SStoragecache{region: region}
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	cache := region.getStoragecache()
	if id == cache.GetGlobalId() {
		return cache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
