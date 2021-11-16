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

package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SStoragecache struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	region *SRegion
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Name, self.region.GetId())
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.cpcfg.Id, self.region.GetGlobalId())
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

func (self *SStoragecache) GetICustomizedCloudImages() ([]cloudprovider.ICloudImage, error) {
	images, err := self.region.GetImages("", ImageOwnerSelf, nil, "", "hvm", nil, "", true)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImages")
	}
	ret := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		ret = append(ret, &images[i])
	}
	return ret, nil
}

func (self *SStoragecache) GetICloudImages() ([]cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	if len(extId) == 0 {
		return nil, fmt.Errorf("GetIImageById image id should not be empty")
	}

	part, err := self.region.GetImage(extId)
	if err != nil {
		log.Errorf("GetImage %s %s", extId, err)
		return nil, errors.Wrap(err, "GetImage")
	}
	part.storageCache = self
	return part, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	imageId, err := self.region.createIImage(snapshotId, imageName, imageDesc)
	if err != nil {
		log.Errorf("createIImage %s %s %s: %s", snapshotId, imageName, imageDesc, err)
		return nil, errors.Wrap(err, "createIImage")
	}
	image, err := self.region.GetImage(imageId)
	if err != nil {
		log.Errorf("GetImage %s: %s", imageId, err)
		return nil, errors.Wrap(err, "GetImage")
	}
	image.storageCache = self
	iimage := make([]cloudprovider.ICloudImage, 1)
	iimage[0] = image
	//todo : implement me
	if err := cloudprovider.WaitStatus(iimage[0], "avaliable", 15*time.Second, 3600*time.Second); err != nil {
		return nil, errors.Wrap(err, "WaitStatus.iimage")
	}
	return iimage[0], nil
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return self.uploadImage(ctx, userCred, image, callback)

}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, opts *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	err := self.region.initVmimport()
	if err != nil {
		return "", errors.Wrap(err, "initVmimport")
	}

	bucketName := GetBucketName(self.region.GetId(), opts.ImageId)

	exist, err := self.region.IBucketExist(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "IBucketExist")
	}

	if !exist {
		err = self.region.CreateIBucket(bucketName, "", "")
		if err != nil {
			return "", errors.Wrap(err, "CreateIBucket")
		}
	}

	defer self.region.DeleteIBucket(bucketName)

	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	meta, reader, sizeBytes, err := modules.Images.Download(s, opts.ImageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", errors.Wrap(err, "Images.Download")
	}
	log.Debugf("Images meta data %s", meta)

	diskFormat, _ := meta.GetString("disk_format")

	bucket, err := self.region.GetIBucketByName(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	body := multicloud.NewProgress(sizeBytes, 80, reader, callback)
	err = cloudprovider.UploadObject(ctx, bucket, opts.ImageId, 0, body, sizeBytes, "", "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "cloudprovider.UploadObject")
	}

	defer bucket.DeleteObject(ctx, opts.ImageId)

	imageBaseName := opts.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", opts.ImageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	// check image name, avoid name conflict
	for {
		_, err = self.region.GetImageByName(imageName, ImageOwnerSelf)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}

		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx += 1
		log.Debugf("uploadImage Match remote name %s", imageName)
	}

	image, err := self.region.ImportImage(imageName, opts.OsArch, opts.OsType, opts.OsDistribution, diskFormat, bucketName, opts.ImageId)

	if err != nil {
		return "", errors.Wrapf(err, "ImportImage")
	}

	image.storageCache = self
	err = cloudprovider.WaitStatus(image, api.CACHED_IMAGE_STATUS_ACTIVE, time.Second*10, time.Minute*5)
	if err != nil {
		return "", errors.Wrap(err, "SStoragecache.Wait")
	}

	if callback != nil {
		callback(100)
	}
	return image.ImageId, nil
}

func (self *SRegion) CheckBucket(bucketName string) error {
	return self.checkBucket(bucketName)
}

func (self *SRegion) checkBucket(bucketName string) error {
	exists, err := self.IsBucketExist(bucketName)
	if err != nil {
		log.Errorf("IsBucketExist %s: %s", bucketName, err)
		return errors.Wrap(err, "IsBucketExist")
	}

	if !exists {
		return fmt.Errorf("bucket %s not found", bucketName)
	} else {
		return nil
	}

}

func (self *SRegion) IsBucketExist(bucketName string) (bool, error) {
	s3Client, err := self.GetS3Client()
	if err != nil {
		return false, errors.Wrap(err, "IsBucketExist.GetS3Client")
	}

	params := &s3.ListBucketsInput{}
	ret, err := s3Client.ListBuckets(params)
	if err != nil {
		return false, errors.Wrap(err, "ListBuckets")
	}

	for _, bucket := range ret.Buckets {
		if bucket.Name != nil && *bucket.Name == bucketName {
			return true, nil
		}
	}

	return false, nil
}

func (self *SRegion) GetBucketRegionId(bucketName string) (string, error) {
	s3Client, err := self.GetS3Client()
	if err != nil {
		return "", err
	}

	params := &s3.GetBucketLocationInput{Bucket: &bucketName}
	ret, err := s3Client.GetBucketLocation(params)
	if err != nil {
		return "", err
	}

	return StrVal(ret.LocationConstraint), nil
}

func (self *SRegion) GetARNPartition() string {
	// https://docs.amazonaws.cn/general/latest/gr/aws-arns-and-namespaces.html?id=docs_gateway
	// https://github.com/aws/chalice/issues/777
	// https://github.com/aws/chalice/issues/792
	/*
		I assume this is because the ARN format is slightly different for China.
		In general, ARNs follow the pattern arn:partition:service:region:account-id:resource,
		where partition is aws for most of the world and aws-cn for China.
		It looks like the more common "arn:aws" is currently hardcoded in quite a few places.
	*/
	if strings.HasPrefix(self.RegionName, "cn-") {
		return "aws-cn"
	} else {
		return "aws"
	}
}

func (self *SRegion) initVmimportRole() error {
	// search role vmimport
	roleName := "vmimport"
	_, err := self.client.GetRole(roleName)
	if err == nil || (err != nil && errors.Cause(err) != cloudprovider.ErrNotFound) {
		return err
	}
	// create it
	roleDoc := `{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Effect": "Allow",
         "Principal": { "Service": "vmie.amazonaws.com" },
         "Action": "sts:AssumeRole",
         "Condition": {
            "StringEquals":{
               "sts:Externalid": "vmimport"
            }
         }
      }
   ]
}`
	params := map[string]string{
		"RoleName":                 roleName,
		"Description":              "vmimport role for image import",
		"AssumeRolePolicyDocument": roleDoc,
	}
	return self.client.iamRequest("CreateRole", params, nil)
}

func (self *SRegion) initVmimportRolePolicy() error {
	roleName := "vmimport"
	policyName := "vmimport"
	_, err := self.client.GetRolePolicy(roleName, policyName)
	if err == nil || (err != nil && errors.Cause(err) != cloudprovider.ErrNotFound) {
		return err
	}
	partition := self.GetARNPartition()
	rolePolicy := `{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Effect":"Allow",
         "Action":[
            "s3:GetBucketLocation",
            "s3:GetObject",
            "s3:ListBucket" 
         ],
         "Resource":[
            "arn:%[1]s:s3:::%[2]s",
            "arn:%[1]s:s3:::%[2]s/*"
         ]
      },
      {
         "Effect":"Allow",
         "Action":[
            "ec2:ModifySnapshotAttribute",
            "ec2:CopySnapshot",
            "ec2:RegisterImage",
            "ec2:Describe*"
         ],
         "Resource":"*"
      }
   ]
}`
	params := map[string]string{
		"PolicyName":     policyName,
		"RoleName":       roleName,
		"PolicyDocument": fmt.Sprintf(rolePolicy, partition, "imgcache-*"),
	}
	return self.client.iamRequest("PutRolePolicy", params, nil)
}

func (self *SRegion) initVmimport() error {
	if err := self.initVmimportRole(); err != nil {
		return err
	}

	if err := self.initVmimportRolePolicy(); err != nil {
		return err
	}

	return nil
}

func (self *SRegion) createIImage(snapshotId, imageName, imageDesc string) (string, error) {
	params := map[string]string{
		"Description":                         imageDesc,
		"Name":                                imageName,
		"BlockDeviceMapping.1.DeviceName":     "/dev/sda1",
		"BlockDeviceMapping.1.Ebs.SnapshotId": snapshotId,
		"BlockDeviceMapping.1.Ebs.DeleteOnTermination": "true",
	}
	ret := struct {
		ImageId string `xml:"imageId"`
	}{}
	return ret.ImageId, self.ec2Request("CreateImage", params, &ret)
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}
