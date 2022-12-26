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

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStoragecache struct {
	multicloud.SResourceBase
	AwsTags
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

func (self *SStoragecache) DownloadImage(imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(imageId, extId)
}

func (self *SStoragecache) UploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	return self.uploadImage(ctx, image, callback)

}

func (self *SStoragecache) uploadImage(ctx context.Context, image *cloudprovider.SImageCreateOption, callback func(progress float32)) (string, error) {
	err := self.region.initVmimport()
	if err != nil {
		return "", errors.Wrap(err, "initVmimport")
	}

	bucketName := GetBucketName(self.region.GetId(), image.ImageId)

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

	reader, sizeBytes, err := image.GetReader(image.ImageId, string(qemuimgfmt.VMDK))
	if err != nil {
		return "", errors.Wrapf(err, "GetReader")
	}

	bucket, err := self.region.GetIBucketByName(bucketName)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	body := multicloud.NewProgress(sizeBytes, 80, reader, callback)
	err = cloudprovider.UploadObject(ctx, bucket, image.ImageId, 0, body, sizeBytes, "", "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "cloudprovider.UploadObject")
	}

	defer bucket.DeleteObject(ctx, image.ImageId)

	imageBaseName := image.ImageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", image.ImageId)
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

	task, err := self.region.ImportImage(imageName, image.OsArch, image.OsType, image.OsDistribution, string(qemuimgfmt.VMDK), bucketName, image.ImageId)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s", image.ImageId, bucketName, err)
		return "", err
	}

	err = cloudprovider.Wait(2*time.Minute, 4*time.Hour, func() (bool, error) {
		status := task.GetStatus()
		if status == ImageImportStatusDeleted {
			return false, errors.Wrap(errors.ErrInvalidStatus, "SStoragecache.ImageImportStatusDeleted")
		}

		if status == ImageImportStatusCompleted {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "SStoragecache.Wait")
	}

	// add name tag
	self.region.addTags(task.ImageId, "Name", image.ImageId)
	if callback != nil {
		callback(100)
	}
	return task.ImageId, nil
}

func (self *SStoragecache) downloadImage(imageId string, extId string) (jsonutils.JSONObject, error) {
	// aws 导出镜像限制比较多。https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmexport.html
	bucketName := GetBucketName(self.region.GetId(), imageId)
	if err := self.region.checkBucket(bucketName); err != nil {
		log.Errorf("checkBucket %s: %s", bucketName, err)
		return nil, errors.Wrap(err, "checkBucket")
	}

	instanceId, err := self.region.GetInstanceIdByImageId(extId)
	if err != nil {
		log.Errorf("GetInstanceIdByImageId %s: %s", extId, err)
		return nil, errors.Wrap(err, "GetInstanceIdByImageId")
	}

	ec2Client, err := self.region.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	task, err := self.region.ExportImage(instanceId, imageId)
	if err != nil {
		log.Errorf("ExportImage %s %s: %s", instanceId, imageId, err)
		return nil, errors.Wrap(err, "ExportImage")
	}

	taskParams := &ec2.DescribeExportTasksInput{}
	taskParams.SetExportTaskIds([]*string{&task.TaskId})
	if err := ec2Client.WaitUntilExportTaskCompleted(taskParams); err != nil {
		log.Errorf("WaitUntilExportTaskCompleted %#v %s", taskParams, err)
		return nil, errors.Wrap(err, "WaitUntilExportTaskCompleted")
	}

	return nil, cloudprovider.ErrNotSupported
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
	if strings.HasPrefix(self.RegionId, "cn-") {
		return "aws-cn"
	} else {
		return "aws"
	}
}

func (self *SRegion) initVmimportRole() error {
	/*需要api access token 具备iam Full access权限*/
	iamClient, err := self.getIamClient()
	if err != nil {
		return err
	}

	// search role vmimport
	rolename := "vmimport"
	ret, _ := iamClient.GetRole(&iam.GetRoleInput{RoleName: &rolename})
	// todo: 这里得区分是not found.还是其他错误
	if ret.Role != nil && ret.Role.RoleId != nil {
		return nil
	} else {
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
		params := &iam.CreateRoleInput{}
		params.SetDescription("vmimport role for image import")
		params.SetRoleName(rolename)
		params.SetAssumeRolePolicyDocument(roleDoc)

		_, err = iamClient.CreateRole(params)
		return err
	}
}

func (self *SRegion) initVmimportRolePolicy() error {
	/*需要api access token 具备iam Full access权限*/
	iamClient, err := self.getIamClient()
	if err != nil {
		return err
	}

	partition := self.GetARNPartition()
	roleName := "vmimport"
	policyName := "vmimport"
	ret, err := iamClient.GetRolePolicy(&iam.GetRolePolicyInput{RoleName: &roleName, PolicyName: &policyName})
	// todo: 这里得区分是not found.还是其他错误.
	if ret.PolicyName != nil {
		return nil
	} else {
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
		params := &iam.PutRolePolicyInput{}
		params.SetPolicyDocument(fmt.Sprintf(rolePolicy, partition, "imgcache-*"))
		params.SetPolicyName(policyName)
		params.SetRoleName(roleName)
		_, err = iamClient.PutRolePolicy(params)
		return err
	}
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
	params := &ec2.CreateImageInput{}
	params.SetDescription(imageDesc)
	params.SetName(imageName)
	block := &ec2.BlockDeviceMapping{}
	block.SetDeviceName("/dev/sda1")
	ebs := &ec2.EbsBlockDevice{}
	ebs.SetSnapshotId(snapshotId)
	ebs.SetDeleteOnTermination(true)
	block.SetEbs(ebs)
	blockList := []*ec2.BlockDeviceMapping{block}
	params.SetBlockDeviceMappings(blockList)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return "", errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.CreateImage(params)
	if err != nil {
		return "", err
	}
	return *ret.ImageId, nil
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
