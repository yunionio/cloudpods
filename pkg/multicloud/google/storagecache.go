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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SStoragecache struct {
	multicloud.SResourceBase
	multicloud.GoogleTags
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (cache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.cpcfg.Id, cache.region.GetGlobalId())
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
	return cache.GetId()
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

func (cache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return cache.uploadImage(ctx, userCred, image, callback)
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

func (cache *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)

	meta, reader, sizeBytes, err := modules.Images.Download(s, image.ImageId, string(qemuimg.QCOW2), false)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

	info := struct {
		Id          string
		Name        string
		Size        int64
		Description string
	}{}
	meta.Unmarshal(&info)

	bucketName := fmt.Sprintf("imagecache-%s", info.Id)
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
	for _, s := range strings.ToLower(info.Name) {
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

	_image, err := cache.region.CreateImage(imageName, info.Description, bucketName, info.Name)
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

func (cache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
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
