package multicloud

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SNoObjectStorageRegion struct{}

///////////////// S3 ///////////////////

func (cli *SNoObjectStorageRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) DeleteIBucket(name string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) IBucketExist(name string) (bool, error) {
	return false, cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotSupported
}

////////////////// END S3 fake API //////////
