package aws

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
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

func (self *SRegion) CheckBucket(bucketName string) (*oss.Bucket, error) {
	return self.checkBucket(bucketName)
}

func (self *SRegion) checkBucket(bucketName string) (*oss.Bucket, error) {
	return nil, nil
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

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
	// todo: implement me
	return "", nil
}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
   //  todo: implement me
   return nil, nil
}