package aws

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (self *SStoragecache) GetId() string {
	panic("implement me")
}

func (self *SStoragecache) GetName() string {
	panic("implement me")
}

func (self *SStoragecache) GetGlobalId() string {
	panic("implement me")
}

func (self *SStoragecache) GetStatus() string {
	panic("implement me")
}

func (self *SStoragecache) Refresh() error {
	panic("implement me")
}

func (self *SStoragecache) IsEmulated() bool {
	panic("implement me")
}

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	panic("implement me")
}

func (self *SStoragecache) GetManagerId() string {
	panic("implement me")
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	panic("implement me")
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	panic("implement me")
}

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, extId string, isForce bool) (string, error) {
	panic("implement me")
}

func (self *SRegion) CheckBucket(bucketName string) (*oss.Bucket, error) {
	return self.checkBucket(bucketName)
}

func (self *SRegion) checkBucket(bucketName string) (*oss.Bucket, error) {
	return nil, nil
}

func (self *SRegion) createIImage(snapshoutId, imageName, imageDesc string) (string, error) {
	return "", nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}