package openstack

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
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
	images, err := cache.region.GetImages("", ACTIVE, "")
	if err != nil {
		return err
	}
	cache.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i++ {
		images[i].storageCache = cache
		cache.iimages[i] = &images[i]
	}
	return nil
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
	image, err := cache.region.GetImage(extId)
	if err != nil {
		return nil, err
	}
	image.storageCache = cache
	return image, nil
}

func (cache *SStoragecache) GetPath() string {
	return ""
}

func (cache *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", extId)

		statsu, err := cache.region.GetImageStatus(extId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if statsu == ACTIVE && !isForce {
			return extId, nil
		}
	}
	log.Debugf("UploadImage: no external ID")
	return cache.uploadImage(ctx, userCred, imageId, osArch, osType, osDist, osVersion, isForce)
}

func (cache *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, isForce bool) (string, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, err := modules.Images.Download(s, imageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", err
	}
	log.Infof("meta data %s", meta)

	imageBaseName := imageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId)
	}
	imageName := imageBaseName
	nameIdx := 1

	for {
		_, err = cache.region.GetImageByName(imageName)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}
		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx++
	}

	image, err := cache.region.CreateImage(imageName)
	if err != nil {
		return "", err
	}

	image.storageCache = cache

	_, err = cache.region.client.StreamRequest(cache.region.Name, "image", "PUT", fmt.Sprintf("/v2/images/%s/file", image.ID), "", reader)
	if err != nil {
		return "", err
	}
	return image.ID, cloudprovider.WaitStatus(image, models.CACHED_IMAGE_STATUS_READY, 15*time.Second, 3600*time.Second)
}

func (cache *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (cache *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}
