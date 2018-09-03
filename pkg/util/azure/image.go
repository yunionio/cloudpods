package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "Creating"
	ImageStatusAvailable    ImageStatusType = "Available"
	ImageStatusUnAvailable  ImageStatusType = "UnAvailable"
	ImageStatusCreateFailed ImageStatusType = "CreateFailed"
)

type OperatingSystemStateTypes string

type ImageOSDisk struct {
	OsType             OperatingSystemTypes
	OsState            OperatingSystemStateTypes
	Snapshot           SubResource
	ManagedDisk        SubResource
	BlobURI            string
	Caching            string
	DiskSizeGB         int32
	StorageAccountType StorageAccountTypes
}

type ImageDataDisk struct {
	Lun                int32
	Snapshot           SubResource
	ManagedDisk        SubResource
	BlobURI            string
	Caching            string
	DiskSizeGB         int32
	StorageAccountType StorageAccountTypes
}

type ImageStorageProfile struct {
	OsDisk        ImageOSDisk
	DataDisks     []ImageDataDisk
	ZoneResilient bool
}

type ImageProperties struct {
	SourceVirtualMachine SubResource
	StorageProfile       ImageStorageProfile
	ProvisioningState    ImageStatusType
}

type SImage struct {
	storageCache *SStoragecache

	Properties ImageProperties
	ID         string
	Name       string
	Location   string
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SImage) GetId() string {
	return self.ID
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", DefaultResourceGroup["image"], self.Name)
}

func (self *SImage) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "created":
		return models.IMAGE_STATUS_QUEUED
	// case ImageStatusAvailable:
	// 	return models.IMAGE_STATUS_ACTIVE
	// case ImageStatusUnAvailable:
	// 	return models.IMAGE_STATUS_DELETED
	// case ImageStatusCreateFailed:
	// 	return models.IMAGE_STATUS_KILLED
	default:
		return models.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	new, err := self.storageCache.region.GetImage(self.Name)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	if image, err := self.GetImage(imageId); err != nil {
		return "", err
	} else {
		return image.Properties.ProvisioningState, nil
	}
}

func (self *SRegion) GetImage(imageId string) (*SImage, error) {
	image := SImage{}
	imageClient := compute.NewImagesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	imageClient.Authorizer = self.client.authorizer
	if result, err := imageClient.Get(context.Background(), DefaultResourceGroup["image"], imageId, ""); err != nil {
		return nil, err
	} else if jsonutils.Update(&image, result); err != nil {
		return nil, err
	}
	return &image, nil
}

func (self *SRegion) GetImages() ([]SImage, error) {
	images := make([]SImage, 0)
	imageClient := compute.NewImagesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	imageClient.Authorizer = self.client.authorizer
	if result, err := imageClient.List(context.Background()); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&images, result.Values()); err != nil {
		return nil, err
	}
	return images, nil
}
