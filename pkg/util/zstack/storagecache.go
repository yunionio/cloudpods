package zstack

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SStoragecache struct {
	region *SRegion
}

func (scache *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (scache *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", scache.region.client.providerID, scache.region.GetId())
}

func (scache *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", scache.region.client.providerName, scache.region.GetId())
}

func (scache *SStoragecache) GetStatus() string {
	return "available"
}

func (scache *SStoragecache) Refresh() error {
	return nil
}

func (scache *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", scache.region.client.providerID, scache.region.GetGlobalId())
}

func (scache *SStoragecache) IsEmulated() bool {
	return false
}

func (scache *SStoragecache) GetManagerId() string {
	return scache.region.client.providerID
}

func (scache *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	images, err := scache.region.GetImages("")
	if err != nil {
		return nil, err
	}
	iImages := []cloudprovider.ICloudImage{}
	for i := 0; i < len(images); i++ {
		images[i].storageCache = scache
		iImages = append(iImages, &images[i])
	}
	return iImages, nil
}

func (scache *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	image, err := scache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	image.storageCache = scache
	return image, nil
}

func (scache *SStoragecache) GetPath() string {
	return ""
}

func (scache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (scache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
