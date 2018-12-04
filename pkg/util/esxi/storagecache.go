package esxi

import (
	"context"
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	IMAGE_CACHE_DIR_NAME = "image_cache"
)

type SDatastoreImageCache struct {
	datastore *SDatastore
	host      *SHost
}

func (self *SDatastoreImageCache) GetId() string {
	if self.host != nil {
		return self.host.GetGlobalId()
	} else {
		return self.datastore.GetGlobalId()
	}
}

func (self *SDatastoreImageCache) GetName() string {
	if self.host != nil {
		return fmt.Sprintf("storage-cache-%s", self.host.GetName())
	} else {
		return fmt.Sprintf("storage-cache-%s", self.datastore.GetName())
	}
}

func (self *SDatastoreImageCache) GetGlobalId() string {
	return self.GetId()
}

func (self *SDatastoreImageCache) GetStatus() string {
	return "available"
}

func (self *SDatastoreImageCache) Refresh() error {
	return nil
}

func (self *SDatastoreImageCache) IsEmulated() bool {
	return false
}

func (self *SDatastoreImageCache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDatastoreImageCache) GetPath() string {
	return path.Join(self.datastore.GetMountPoint(), IMAGE_CACHE_DIR_NAME)
}

func (self *SDatastoreImageCache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	ctx := context.Background()

	files, err := self.datastore.ListDir(ctx, IMAGE_CACHE_DIR_NAME)
	if err != nil {
		log.Errorf("GetIImages ListDir fail %s", err)
		return nil, err
	}

	ret := make([]cloudprovider.ICloudImage, 0)

	validFilenames := make(map[string]bool)

	for i := 0; i < len(files); i += 1 {
		filename := path.Join(IMAGE_CACHE_DIR_NAME, files[i].Name)
		if err := self.datastore.CheckVmdk(ctx, filename); err != nil {
			continue
		}
		image := SImage{
			cache:    self,
			filename: filename,
		}
		ret = append(ret, &image)
		vmdkName := files[i].Name
		vmdkExtName := fmt.Sprintf("%s-flat.vmdk", vmdkName[:len(vmdkName)-5])
		validFilenames[vmdkName] = true
		validFilenames[vmdkExtName] = true
	}

	log.Debugf("storage cache contains %#v", validFilenames)
	// cleanup storage cache!!!
	for i := 0; i < len(files); i += 1 {
		if _, ok := validFilenames[files[i].Name]; !ok {
			log.Debugf("delete invalid vmdk file %s!!!", files[i].Name)
			self.datastore.Delete(ctx, path.Join(IMAGE_CACHE_DIR_NAME, files[i].Name))
		}
	}

	return ret, nil
}

func (self *SDatastoreImageCache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	images, err := self.GetIImages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i += 1 {
		if images[i].GetGlobalId() == extId {
			return images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SDatastoreImageCache) GetManagerId() string {
	return self.datastore.manager.providerId
}

func (self *SDatastoreImageCache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastoreImageCache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastoreImageCache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
