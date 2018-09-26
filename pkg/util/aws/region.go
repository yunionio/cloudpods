package aws

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SRegion struct {
	client *SAwsClient

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	ID             string
	SubscriptionID string
	Name           string
	DisplayName    string
	Latitude       float32
	Longitude      float32
}
