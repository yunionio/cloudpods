package azure

import (
	"strings"

	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	OsType             string       `json:"osType,omitempty"`
	OsState            string       `json:"osState,omitempty"`
	Snapshot           *SubResource `json:"snapshot,omitempty"`
	ManagedDisk        *SubResource
	BlobURI            string `json:"blobUri,omitempty"`
	Caching            string `json:"caching,omitempty"`
	DiskSizeGB         int32  `json:"diskSizeGB,omitempty"`
	StorageAccountType string `json:"storageAccountType,omitempty"`
}

type ImageDataDisk struct {
	Lun                int32
	Snapshot           SubResource
	ManagedDisk        SubResource
	BlobURI            string
	Caching            string
	DiskSizeGB         int32
	StorageAccountType string
}

type DataDisks []ImageDataDisk

type ImageStorageProfile struct {
	OsDisk        ImageOSDisk `json:"osDisk,omitempty"`
	DataDisks     *DataDisks
	ZoneResilient *bool
}

type ImageProperties struct {
	SourceVirtualMachine *SubResource
	StorageProfile       ImageStorageProfile `json:"storageProfile,omitempty"`
	ProvisioningState    ImageStatusType
}

type SImage struct {
	storageCache *SStoragecache

	Properties ImageProperties `json:"properties,omitempty"`
	ID         string
	Name       string
	Type       string ``
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
	return &image, self.client.Get(imageId, []string{}, &image)
}

func (self *SRegion) GetImageByName(name string) (*SImage, error) {
	images := []SImage{}
	err := self.client.ListAll("Microsoft.Compute/images", &images)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i++ {
		if images[i].Name == name {
			return &images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetImageById(idstr string) (*SImage, error) {
	images := []SImage{}
	err := self.client.ListAll("Microsoft.Compute/images", &images)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i++ {
		if images[i].ID == idstr {
			return &images[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) CreateImageByBlob(imageName, osType, blobURI string, diskSizeGB int32) (*SImage, error) {
	if diskSizeGB < 1 || diskSizeGB > 4095 {
		diskSizeGB = 30
	}
	image := SImage{
		Name:     imageName,
		Location: self.Name,
		Properties: ImageProperties{
			StorageProfile: ImageStorageProfile{
				OsDisk: ImageOSDisk{
					OsType:     osType,
					OsState:    "Generalized",
					BlobURI:    blobURI,
					DiskSizeGB: diskSizeGB,
				},
			},
		},
		Type: "Microsoft.Compute/images",
	}
	return &image, self.client.Create(jsonutils.Marshal(image), &image)
}

func (self *SRegion) CreateImage(snapshotId, imageName, osType, imageDesc string) (*SImage, error) {
	image := SImage{
		Name:     imageName,
		Location: self.Name,
		Properties: ImageProperties{
			StorageProfile: ImageStorageProfile{
				OsDisk: ImageOSDisk{
					OsType:  osType,
					OsState: "Generalized",
					Snapshot: &SubResource{
						ID: snapshotId,
					},
				},
			},
		},
		Type: "Microsoft.Compute/images",
	}
	return &image, self.client.Create(jsonutils.Marshal(image), &image)
}

func (self *SRegion) GetImages() ([]SImage, error) {
	result := []SImage{}
	images := []SImage{}
	err := self.client.ListAll("Microsoft.Compute/images", &images)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i++ {
		if images[i].Location == self.Name {
			result = append(result, images[i])
		}
	}
	return result, nil
}

func (self *SRegion) DeleteImage(imageId string) error {
	return self.client.Delete(imageId)
}

func (self *SImage) GetBlobUri() string {

	return self.Properties.StorageProfile.OsDisk.BlobURI
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.ID)
}

func (self *SImage) GetOsType() string {
	return string(self.Properties.StorageProfile.OsDisk.OsType)
}
