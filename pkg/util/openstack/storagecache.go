package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (cache *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (cache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerID, cache.region.GetId())
}

func (cache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerName, cache.region.GetId())
}

func (cache *SStoragecache) GetStatus() string {
	return "available"
}

func (cache *SStoragecache) Refresh() error {
	return nil
}

func (cache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", cache.region.client.providerID, cache.region.GetGlobalId())
}

func (cache *SStoragecache) IsEmulated() bool {
	return false
}

func (cache *SStoragecache) GetManagerId() string {
	return cache.region.client.providerID
}

func (cache *SStoragecache) fetchImages() error {
	return cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if cache.iimages == nil {
		if err := cache.fetchImages(); err != nil {
			return nil, err
		}
	}
	return cache.iimages, nil
}

func (cache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
