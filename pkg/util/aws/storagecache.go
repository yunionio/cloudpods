package aws

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if imageId, err := self.region.createIImage(snapshotId, imageName, imageDesc); err != nil {
		return nil, err
	} else if image, err := self.region.GetImage(imageId); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		iimage := make([]cloudprovider.ICloudImage, 1)
		iimage[0] = image
		//todo : implement me
		if err := cloudprovider.WaitStatus(iimage[0], "avaliable", 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return iimage[0], nil
	}
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
}

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, extId string, isForce bool) (string, error) {
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

	return self.uploadImage(userCred, imageId, osArch, osType, osDist, isForce)

}

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages(ImageStatusType(""), ImageOwnerSelf, nil, "", len(images), 50)
		if err != nil {
			return err
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
	}
	self.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		self.iimages[i] = &images[i]
	}
	return nil
}

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
	// todo: implement me
	bucketName := "imgcache-onecloud"
	err := self.region.initVmimport()
	if err != nil {
		return "", err
	}

	// first upload image to oss
	s := auth.GetAdminSession(options.Options.Region, "")

	meta, reader, err := modules.Images.Download(s, imageId)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

	s3Client, err := self.region.getS3Client()
	if err != nil {
		return "", nil
	}
	// 内存？
	f, err := ioutil.ReadAll(reader)
	params := &s3.PutObjectInput{}
	params.SetBucket(bucketName)
	params.SetKey(imageId)
	params.SetBody(bytes.NewReader(f))
	_, err = s3Client.PutObject(params)
	if err != nil {
		return "", nil
	}

	imageBaseName := imageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	// check image name, avoid name conflict
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
	}

	task, err := self.region.ImportImage(imageName, osArch, osType, osDist, bucketName, imageId)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s", imageId, bucketName, err)
		return "", err
	}

	// todo:// 等待镜像导入完成
	err = self.region.ec2Client.WaitUntilImageExists(&ec2.DescribeImagesInput{ImageIds: []*string{&task.ImageId}})
	return task.ImageId, err

}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	// aws 导出镜像限制比较多。https://docs.aws.amazon.com/zh_cn/vm-import/latest/userguide/vmexport.html
	bucketName := "imgcache-onecloud"
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

	s3Client, err := self.region.getS3Client()
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

	s := auth.GetAdminSession(options.Options.Region, "")
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
	s3Client, err := self.getS3Client()
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
            "arn:aws:s3:::%[1]s",
            "arn:aws:s3:::%[1]s/*"
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
		params := &iam.CreatePolicyInput{}
		params.SetDescription("vmimport policy for image import")
		params.SetPolicyDocument(fmt.Sprintf(rolePolicy, "imgcache-onecloud"))
		params.SetPolicyName(policyName)
		_, err = iamClient.CreatePolicy(params)
		return err
	}
}

func (self *SRegion) initVmimportBucket() error {
	bucketName := "imgcache-onecloud"
	// todo: "imgcache-onecloud" 使用常量
	exists, err := self.IsBucketExist("imgcache-onecloud")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	s3Client, err := self.getS3Client()
	if err != nil {
		return err
	}

	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{Bucket: &bucketName})
	return err
}

func (self *SRegion) initVmimport() error {
	if err := self.initVmimportRole(); err != nil {
		return err
	}

	if err := self.initVmimportRolePolicy(); err != nil {
		return err
	}

	if err := self.initVmimportBucket(); err != nil {
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
