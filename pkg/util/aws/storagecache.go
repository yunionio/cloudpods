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
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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
	if len(extId) == 0 {
		return nil, fmt.Errorf("GetIImageById image id should not be empty")
	}

	part, err := self.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	part.storageCache = self
	return part, nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	imageId, err := self.region.createIImage(snapshotId, imageName, imageDesc)
	if err != nil {
		return nil, err
	}
	image, err := self.region.GetImage(imageId)
	if err != nil {
		return nil, err
	}
	image.storageCache = self
	iimage := make([]cloudprovider.ICloudImage, 1)
	iimage[0] = image
	//todo : implement me
	if err := cloudprovider.WaitStatus(iimage[0], "avaliable", 15*time.Second, 3600*time.Second); err != nil {
		return nil, err
	}
	return iimage[0], nil
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", extId)

		status, err := self.region.GetImageStatus(extId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return self.uploadImage(ctx, userCred, imageId, osArch, osType, osDist, isForce)

}

func (self *SStoragecache) fetchImages() error {
	images, err := self.region.GetImages("", ImageOwnerSelfSystem, nil, "", "hvm", nil, "", true)
	if err != nil {
		return err
	}
	self.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		self.iimages[i] = &images[i]
	}
	return nil
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
	bucketName := GetBucketName(self.region.GetId(), imageId)
	err := self.region.initVmimport(bucketName)
	if err != nil {
		return "", err
	}

	// checking remote
	s3client, err := self.region.GetS3Client()
	if err != nil {
		return "", err
	}

	defer s3client.DeleteBucket(&s3.DeleteBucketInput{Bucket: &bucketName}) // remove bucket

	var diskFormat string
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	_, err = s3client.GetObject(&s3.GetObjectInput{Bucket: &bucketName, Key: &imageId})
	if err != nil {
		// first upload image to oss
		meta, reader, err := modules.Images.Download(s, imageId, string(qemuimg.VMDK), false)
		if err != nil {
			return "", err
		}
		log.Debugf("Images meta data %s", meta)

		diskFormat, err = meta.GetString("disk_format")
		if err != nil {
			return "", err
		}

		// uploader to aws s3
		input := &s3manager.UploadInput{
			Bucket: &bucketName,
			Key:    &imageId,
			Body:   reader,
		}

		s3Session, err := self.region.getAwsSession()
		if err != nil {
			return "", err
		}

		uploader := s3manager.NewUploader(s3Session)
		_, err = uploader.Upload(input)
		if err != nil {
			return "", err
		}
		defer s3client.DeleteObject(&s3.DeleteObjectInput{Bucket: &bucketName, Key: &imageId}) // remove object
	} else {
		meta, _, err := modules.Images.Download(s, imageId, string(qemuimg.VMDK), false)
		if err != nil {
			return "", err
		}

		diskFormat, err = meta.GetString("disk_format")
		if err != nil {
			return "", err
		}
	}

	imageBaseName := imageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	// check image name, avoid name conflict
	for {
		_, err = self.region.GetImageByName(imageName, ImageOwnerSelf)
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

	task, err := self.region.ImportImage(imageName, osArch, osType, osDist, diskFormat, bucketName, imageId)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s", imageId, bucketName, err)
		return "", err
	}

	// todo:// 等待镜像导入完成
	for i := 1; i < 120; i++ {
		time.Sleep(2 * time.Minute)
		ret, err := self.region.ec2Client.DescribeImportImageTasks(&ec2.DescribeImportImageTasksInput{ImportTaskIds: []*string{&task.TaskId}})
		if err != nil {
			return "", err
		}

		err = FillZero(ret)
		if err != nil {
			return "", err
		}

		log.Debugf("DescribeImportImage Task %s", ret.String())
		for _, item := range ret.ImportImageTasks {
			if *item.Status == "completed" {
				// add name tag
				self.region.addTags(*item.ImageId, "Name", imageId)
				return *item.ImageId, nil
			}
		}
	}

	return task.ImageId, fmt.Errorf("uploadImage uncompleted: %s", task)

}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	// aws 导出镜像限制比较多。https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmexport.html
	bucketName := GetBucketName(self.region.GetId(), imageId)
	if err := self.region.checkBucket(bucketName); err != nil {
		return nil, err
	}

	instanceId, err := self.region.GetInstanceIdByImageId(extId)
	if err != nil {
		return nil, err
	}

	task, err := self.region.ExportImage(instanceId, imageId)
	if err != nil {
		return nil, err
	}

	taskParams := &ec2.DescribeExportTasksInput{}
	taskParams.SetExportTaskIds([]*string{&task.TaskId})
	if err := self.region.ec2Client.WaitUntilExportTaskCompleted(taskParams); err != nil {
		return nil, err
	}

	s3Client, err := self.region.GetS3Client()
	if err != nil {
		return nil, err
	}

	i := &s3.GetObjectInput{}
	i.SetBucket(bucketName)
	i.SetKey(fmt.Sprintf("%s.%s", task.TaskId, "ova"))
	ret, err := s3Client.GetObject(i)
	if err != nil {
		return nil, err
	}

	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	params := jsonutils.Marshal(map[string]string{"image_id": imageId, "disk-format": "raw"})
	if result, err := modules.Images.Upload(s, params, ret.Body, IntVal(ret.ContentLength)); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func (self *SRegion) CheckBucket(bucketName string) error {
	return self.checkBucket(bucketName)
}

func (self *SRegion) checkBucket(bucketName string) error {
	exists, err := self.IsBucketExist(bucketName)
	if err != nil {
		return err
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
		return false, err
	}

	params := &s3.ListBucketsInput{}
	ret, err := s3Client.ListBuckets(params)
	if err != nil {
		return false, err
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

func (self *SRegion) initVmimportBucket(bucketName string) error {
	exists, err := self.IsBucketExist(bucketName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	s3Client, err := self.GetS3Client()
	if err != nil {
		return err
	}

	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{Bucket: &bucketName})
	return err
}

func (self *SRegion) initVmimport(bucketName string) error {
	if err := self.initVmimportRole(); err != nil {
		return err
	}

	if err := self.initVmimportRolePolicy(); err != nil {
		return err
	}

	if err := self.initVmimportBucket(bucketName); err != nil {
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

	ret, err := self.ec2Client.CreateImage(params)
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
