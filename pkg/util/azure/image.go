package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "Creating"
	ImageStatusAvailable    ImageStatusType = "Succeeded"
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
	data := jsonutils.NewDict()
	osType := string(self.Properties.StorageProfile.OsDisk.OsType)
	if len(osType) > 0 {
		data.Add(jsonutils.NewString(osType), "os_name")
	}
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
	return strings.ToLower(self.ID)
}

func (self *SImage) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "created":
		return models.IMAGE_STATUS_QUEUED
	case "Succeeded":
		return models.IMAGE_STATUS_ACTIVE
	default:
		log.Errorf("Unknow image status: %s", self.Properties.ProvisioningState)
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
	if len(imageId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	_, resourceGroup, imageName := pareResourceGroupWithName(imageId, IMAGE_RESOURCE)
	if result, err := imageClient.Get(context.Background(), resourceGroup, imageName, ""); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if jsonutils.Update(&image, result); err != nil {
		return nil, err
	}
	return &image, nil
}

func (self *SRegion) GetImageByName(imageId string) (*SImage, error) {
	return self.GetImage(imageId)
}

func (self *SRegion) CreateImageByBlob(imageName, osType, blobURI string, diskSizeGB int32) (*SImage, error) {
	if diskSizeGB < 1 || diskSizeGB > 4095 {
		diskSizeGB = 30
	}
	imageClient := compute.NewImagesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	imageClient.Authorizer = self.client.authorizer
	storageProfile := compute.ImageStorageProfile{
		OsDisk: &compute.ImageOSDisk{
			OsType:     compute.OperatingSystemTypes(osType),
			OsState:    compute.Generalized,
			BlobURI:    &blobURI,
			DiskSizeGB: &diskSizeGB,
		},
	}
	params := compute.Image{
		Name:     &imageName,
		Location: &self.Name,
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &storageProfile,
		},
	}
	_, resourceGroup, imageName := pareResourceGroupWithName(imageName, IMAGE_RESOURCE)
	self.CreateResourceGroup(resourceGroup)
	if result, err := imageClient.CreateOrUpdate(context.Background(), resourceGroup, imageName, params); err != nil {
		log.Errorf("Create image from blob error: %v", err)
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), imageClient.Client); err != nil {
		log.Errorf("WaitForCreateImageCompletion error: %v", err)
		return nil, err
	} else if image, err := self.GetImageByName(imageName); err != nil {
		return nil, err
	} else {
		return image, nil
	}
}

func (self *SRegion) CreateImage(snapshotId, imageName, osType, imageDesc string) (*SImage, error) {
	globalId, resourceGroup, imageName := pareResourceGroupWithName(imageName, IMAGE_RESOURCE)
	imageClient := compute.NewImagesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	imageClient.Authorizer = self.client.authorizer
	if utils.IsInStringArray(osType, []string{string(compute.Linux), string(compute.Windows)}) {
		osType = string(compute.Linux)
	}
	storageProfile := compute.ImageStorageProfile{
		OsDisk: &compute.ImageOSDisk{
			OsType:  compute.OperatingSystemTypes(osType),
			OsState: compute.Generalized,
			Snapshot: &compute.SubResource{
				ID: &snapshotId,
			},
		},
	}
	params := compute.Image{
		Name:     &imageName,
		Location: &self.Name,
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &storageProfile,
		},
	}
	self.CreateResourceGroup(resourceGroup)
	if resutl, err := imageClient.CreateOrUpdate(context.Background(), resourceGroup, imageName, params); err != nil {
		return nil, err
	} else if err := resutl.WaitForCompletion(context.Background(), imageClient.Client); err != nil {
		return nil, err
	}
	return self.GetImage(globalId)
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

func (self *SRegion) DeleteImage(imageId string) error {
	imageClient := compute.NewImagesClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	imageClient.Authorizer = self.client.authorizer
	_, resourceGroup, imageName := pareResourceGroupWithName(imageId, IMAGE_RESOURCE)
	if result, err := imageClient.Delete(context.Background(), resourceGroup, imageName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), imageClient.Client); err != nil {
		return err
	}
	return nil
}

func (self *SImage) GetBlobUri() string {
	return self.Properties.StorageProfile.OsDisk.BlobURI
}

func (self *SImage) Delete() error {
	return self.storageCache.region.DeleteImage(self.ID)
}

func (self *SImage) GetOsType() string {
	return string(self.Properties.StorageProfile.OsDisk.OsType)
}
