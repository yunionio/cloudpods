// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "Creating"
	ImageStatusAvailable    ImageStatusType = "Succeeded"
	ImageStatusUnAvailable  ImageStatusType = "UnAvailable"
	ImageStatusCreateFailed ImageStatusType = "CreateFailed"
)

type OperatingSystemStateTypes string

type SubResource struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type ImageOSDisk struct {
	OsType             string       `json:"osType,omitempty"`
	OsState            string       `json:"osState,omitempty"`
	Snapshot           *SubResource `json:"snapshot,omitempty"`
	ManagedDisk        *SubResource
	BlobURI            string `json:"blobUri,omitempty"`
	Caching            string `json:"caching,omitempty"`
	DiskSizeGB         int32  `json:"diskSizeGB,omitzero"`
	StorageAccountType string `json:"storageAccountType,omitempty"`
	OperatingSystem    string `json:"operatingSystem,omitempty"`
}

type ImageDataDisk struct {
	Lun                int32
	Snapshot           SubResource
	ManagedDisk        SubResource
	BlobURI            string
	Caching            string
	DiskSizeGB         int32 `json:"diskSizeGB,omitzero"`
	StorageAccountType string
}

type ImageStorageProfile struct {
	OsDisk        ImageOSDisk     `json:"osDisk,omitempty"`
	DataDisks     []ImageDataDisk `json:"dataDisks,omitempty"`
	ZoneResilient bool            `json:"zoneResilient,omitfalse"`
}

type SAutomaticOSUpgradeProperties struct {
	AutomaticOSUpgradeSupported bool
}

type ImageProperties struct {
	SourceVirtualMachine *SubResource
	StorageProfile       ImageStorageProfile `json:"storageProfile,omitempty"`
	ProvisioningState    ImageStatusType
	HyperVGeneration     string `json:"hyperVGeneration,omitempty"`
}

type SImage struct {
	multicloud.SImageBase
	AzureTags
	storageCache *SStoragecache

	Properties ImageProperties `json:"properties,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string
	Type       string
	Location   string

	Publisher string
	Offer     string
	Sku       string
	Version   string

	ImageType cloudprovider.TImageType
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) GetSysTags() map[string]string {
	data := map[string]string{}
	osType := string(self.Properties.StorageProfile.OsDisk.OsType)
	if len(osType) > 0 {
		data["os_name"] = osType
	}
	return data
}

func (self *SImage) GetId() string {
	return self.ID
}

func (self *SImage) GetName() string {
	return self.Name
}

func (self *SImage) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SImage) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "created":
		return api.CACHED_IMAGE_STATUS_CACHING
	case "Succeeded":
		return api.CACHED_IMAGE_STATUS_ACTIVE
	default:
		log.Errorf("Unknow image status: %s", self.Properties.ProvisioningState)
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
}

func (self *SImage) GetImageStatus() string {
	switch self.Properties.ProvisioningState {
	case "created":
		return cloudprovider.IMAGE_STATUS_QUEUED
	case "Succeeded":
		return cloudprovider.IMAGE_STATUS_ACTIVE
	default:
		log.Errorf("Unknow image status: %s", self.Properties.ProvisioningState)
		return cloudprovider.IMAGE_STATUS_KILLED
	}
}

func (self *SImage) Refresh() error {
	image, err := self.storageCache.region.GetImageById(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, image)
}

func (self *SImage) GetImageType() cloudprovider.TImageType {
	return cloudprovider.TImageType(self.ImageType)
}

func (self *SImage) GetSizeByte() int64 {
	return int64(self.Properties.StorageProfile.OsDisk.DiskSizeGB) * 1024 * 1024 * 1024
}

func (i *SImage) GetFullOsName() string {
	return ""
}

func (self *SImage) GetOsType() cloudprovider.TOsType {
	osType := self.Properties.StorageProfile.OsDisk.OsType
	if len(osType) == 0 {
		osType = publisherGetOsType(self.Publisher)
	}
	return cloudprovider.TOsType(osType)
}

func (self *SImage) GetOsArch() string {
	if self.GetImageType() == cloudprovider.ImageTypeCustomized {
		return apis.OS_ARCH_X86_64
	}
	return publisherGetOsArch(self.Publisher, self.Offer, self.Sku, self.Version)
}

func (self *SImage) GetOsDist() string {
	if self.GetImageType() == cloudprovider.ImageTypeCustomized {
		return ""
	}
	return publisherGetOsDist(self.Publisher, self.Offer, self.Sku, self.Version)
}

func (self *SImage) GetOsVersion() string {
	return publisherGetOsVersion(self.Publisher, self.Offer, self.Sku, self.Version)
}

func (self *SImage) GetOsLang() string {
	return ""
}

func (i *SImage) GetBios() cloudprovider.TBiosType {
	if i.Properties.HyperVGeneration == "V2" {
		return cloudprovider.UEFI
	} else {
		return cloudprovider.BIOS
	}
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	if self.Properties.StorageProfile.OsDisk.DiskSizeGB > 0 {
		return int(self.Properties.StorageProfile.OsDisk.DiskSizeGB)
	}
	return 30
}

func (self *SImage) GetImageFormat() string {
	return "vhd"
}

func (self *SImage) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.storageCache
}

func (self *SRegion) GetImageStatus(imageId string) (ImageStatusType, error) {
	if image, err := self.GetImageById(imageId); err != nil {
		return "", err
	} else {
		return image.Properties.ProvisioningState, nil
	}
}

func isPrivateImageID(imageId string) bool {
	return strings.HasPrefix(strings.ToLower(imageId), "/subscriptions/")
}

func (self *SRegion) GetImageById(imageId string) (SImage, error) {
	if isPrivateImageID(imageId) {
		return self.getPrivateImage(imageId)
	} else {
		return self.getOfferedImage(imageId)
	}
}

func (self *SRegion) getPrivateImage(imageId string) (SImage, error) {
	image := SImage{}
	err := self.get(imageId, url.Values{}, &image)
	if err != nil {
		return image, err
	}
	return image, nil
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
	return &image, self.create("", jsonutils.Marshal(image), &image)
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
	return &image, self.create("", jsonutils.Marshal(image), &image)
}

func (self *SRegion) getOfferedImages(publishersFilter []string, offersFilter []string, skusFilter []string, verFilter []string, imageType cloudprovider.TImageType, latestVer bool) ([]SImage, error) {
	images := make([]SImage, 0)
	idMap, err := self.GetOfferedImageIDs(publishersFilter, offersFilter, skusFilter, verFilter, latestVer)
	if err != nil {
		return nil, err
	}
	for id, _image := range idMap {
		image, err := self.getOfferedImage(id)
		if err == nil {
			image.ImageType = imageType
			image.Properties.StorageProfile.OsDisk.DiskSizeGB = int32(_image.Properties.OsDiskImage.SizeInGb)
			image.Properties.StorageProfile.OsDisk.OsType = _image.Properties.OsDiskImage.OperatingSystem
			image.Properties.HyperVGeneration = _image.Properties.HyperVGeneration
			images = append(images, image)
		}
	}
	return images, nil
}

func (self *SRegion) GetOfferedImageIDs(publishersFilter []string, offersFilter []string, skusFilter []string, verFilter []string, latestVer bool) (map[string]SAzureImageResource, error) {
	idMap := map[string]SAzureImageResource{}
	publishers, err := self.GetImagePublishers(toLowerStringArray(publishersFilter))
	if err != nil {
		return nil, err
	}
	for _, publisher := range publishers {
		offers, err := self.getImageOffers(publisher, toLowerStringArray(offersFilter))
		if err != nil {
			log.Errorf("failed to found offers for publisher %s error: %v", publisher, err)
			if errors.Cause(err) != cloudprovider.ErrNotFound {
				return nil, errors.Wrap(err, "getImageOffers")
			}
			continue
		}
		for _, offer := range offers {
			skus, err := self.getImageSkus(publisher, offer, toLowerStringArray(skusFilter))
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotFound {
					return nil, errors.Wrap(err, "getImageSkus")
				}
				log.Errorf("failed to found skus for publisher %s offer %s error: %v", publisher, offer, err)
				continue
			}
			for _, sku := range skus {
				verFilter = toLowerStringArray(verFilter)
				vers, err := self.getImageVersions(publisher, offer, sku, verFilter, latestVer)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotFound {
						return nil, errors.Wrap(err, "getImageVersions")
					}
					log.Errorf("failed to found publisher %s offer %s sku %s version error: %v", publisher, offer, sku, err)
					continue
				}
				for _, ver := range vers {
					idStr := strings.Join([]string{publisher, offer, sku, ver}, "/")
					image, err := self.getImageDetail(publisher, offer, sku, ver)
					if err != nil {
						return nil, err
					}
					idMap[idStr] = image
				}
			}
		}
	}
	return idMap, nil
}

func (self *SRegion) getPrivateImages() ([]SImage, error) {
	result := []SImage{}
	images := []SImage{}
	err := self.client.list("Microsoft.Compute/images", url.Values{}, &images)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(images); i++ {
		if images[i].Location == self.Name {
			images[i].ImageType = cloudprovider.ImageTypeCustomized
			result = append(result, images[i])
		}
	}
	return result, nil
}

func toLowerStringArray(input []string) []string {
	output := make([]string, len(input))
	for i := range input {
		output[i] = strings.ToLower(input[i])
	}
	return output
}

func (self *SRegion) GetImages(imageType cloudprovider.TImageType) ([]SImage, error) {
	images := make([]SImage, 0)
	if len(imageType) == 0 {
		ret, _ := self.getPrivateImages()
		if len(ret) > 0 {
			images = append(images, ret...)
		}
		ret, _ = self.getOfferedImages(knownPublishers, nil, nil, nil, cloudprovider.ImageTypeSystem, true)
		if len(ret) > 0 {
			images = append(images, ret...)
		}
		return images, nil
	}
	switch imageType {
	case cloudprovider.ImageTypeCustomized:
		return self.getPrivateImages()
	case cloudprovider.ImageTypeSystem:
		return self.getOfferedImages(knownPublishers, nil, nil, nil, cloudprovider.ImageTypeSystem, true)
	default:
		return self.getOfferedImages(nil, nil, nil, nil, cloudprovider.ImageTypeMarket, true)
	}
}

func (self *SRegion) DeleteImage(imageId string) error {
	return self.del(imageId)
}

func (self *SImage) GetBlobUri() string {
	return self.Properties.StorageProfile.OsDisk.BlobURI
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.storageCache.region.DeleteImage(self.ID)
}

type SOsDiskImage struct {
	OperatingSystem string `json:"operatingSystem"`
	SizeInGb        int    `json:"sizeInGb"`
}

type SAzureImageResourceProperties struct {
	ReplicaType      string       `json:"replicaType"`
	OsDiskImage      SOsDiskImage `json:"osDiskImage"`
	HyperVGeneration string       `json:"hyperVGeneration,omitempty"`
}

type SAzureImageResource struct {
	Id         string
	Name       string
	Location   string
	Properties SAzureImageResourceProperties
}

func (region *SRegion) GetImagePublishers(filter []string) ([]string, error) {
	publishers := make([]SAzureImageResource, 0)
	// TODO
	err := region.client.list(fmt.Sprintf("Microsoft.Compute/locations/%s/publishers", region.Name), url.Values{}, &publishers)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	for i := range publishers {
		if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(publishers[i].Name), filter) {
			ret = append(ret, publishers[i].Name)
		}
	}
	return ret, nil
}

func (region *SRegion) getImageOffers(publisher string, filter []string) ([]string, error) {
	ret := make([]string, 0)
	if driver, ok := publisherDrivers[strings.ToLower(publisher)]; ok {
		offers := driver.GetOffers()
		if len(offers) > 0 {
			for _, offer := range offers {
				if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(offer), filter) {
					ret = append(ret, offer)
				}
			}
			return offers, nil
		}
	} else {
		log.Warningf("failed to get publisher %s driver", publisher)
	}
	offers := make([]SAzureImageResource, 0)
	err := region.client.list(fmt.Sprintf("Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers", region.Name, publisher), url.Values{}, &offers)
	if err != nil {
		return nil, err
	}
	for i := range offers {
		if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(offers[i].Name), filter) {
			ret = append(ret, offers[i].Name)
		}
	}
	return ret, nil
}

func (region *SRegion) getImageSkus(publisher string, offer string, filter []string) ([]string, error) {
	ret := make([]string, 0)
	if driver, ok := publisherDrivers[strings.ToLower(publisher)]; ok {
		skus := driver.GetSkus(offer)
		if len(skus) > 0 {
			for _, sku := range skus {
				if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(sku), filter) {
					ret = append(ret, sku)
				}
			}
			return ret, nil
		}
	}
	skus := make([]SAzureImageResource, 0)
	err := region.client.list(fmt.Sprintf("Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers/%s/skus", region.Name, publisher, offer), url.Values{}, &skus)
	if err != nil {
		return nil, err
	}
	for i := range skus {
		if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(skus[i].Name), filter) {
			ret = append(ret, skus[i].Name)
		}
	}
	return ret, nil
}

func (region *SRegion) getImageVersions(publisher string, offer string, sku string, filter []string, latestVer bool) ([]string, error) {
	vers := make([]SAzureImageResource, 0)
	resource := fmt.Sprintf("Microsoft.Compute/locations/%s/publishers/%s/artifacttypes/vmimage/offers/%s/skus/%s/versions", region.Name, publisher, offer, sku)
	params := url.Values{}
	if latestVer {
		params.Set("$top", "1")
		params.Set("orderby", "name desc")
	}
	err := region.client.list(resource, params, &vers)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	for i := range vers {
		if len(filter) == 0 || utils.IsInStringArray(strings.ToLower(vers[i].Name), filter) {
			ret = append(ret, vers[i].Name)
		}
	}
	return ret, nil
}

func (region *SRegion) getImageDetail(publisher string, offer string, sku string, version string) (SAzureImageResource, error) {
	image := SAzureImageResource{}
	id := "/Subscriptions/" + region.client.subscriptionId +
		"/Providers/Microsoft.Compute/locations/" + region.Name +
		"/publishers/" + publisher +
		"/artifacttypes/vmimage/offers/" + offer +
		"/skus/" + sku +
		"/versions/" + version
	return image, region.get(id, url.Values{}, &image)
}

func (region *SRegion) getOfferedImage(offerId string) (SImage, error) {
	image := SImage{}

	parts := strings.Split(offerId, "/")
	if len(parts) < 4 {
		return image, fmt.Errorf("invalid image ID %s", offerId)
	}
	publisher := parts[0]
	offer := parts[1]
	sku := parts[2]
	version := parts[3]
	for _publish := range publisherDrivers {
		if strings.ToLower(_publish) == publisher {
			publisher = _publish
			break
		}
	}
	image.ID = offerId
	image.Location = region.Name
	image.Type = "Microsoft.Compute/vmimage"
	image.Name = publisherGetName(publisher, offer, sku, version)
	image.Publisher = publisher
	image.Offer = offer
	image.Sku = sku
	image.Version = version
	image.Properties.ProvisioningState = ImageStatusAvailable
	_image, err := region.getImageDetail(publisher, offer, sku, version)
	if err == nil {
		image.Properties.StorageProfile.OsDisk.DiskSizeGB = int32(_image.Properties.OsDiskImage.SizeInGb)
		image.Properties.StorageProfile.OsDisk.OperatingSystem = _image.Properties.OsDiskImage.OperatingSystem
		image.Properties.HyperVGeneration = _image.Properties.HyperVGeneration
	}
	return image, nil
}

func (region *SRegion) getOfferedImageId(image *SImage) (string, error) {
	if isPrivateImageID(image.ID) {
		return image.ID, nil
	}
	_image, err := region.getImageDetail(image.Publisher, image.Offer, image.Sku, image.Version)
	if err != nil {
		log.Errorf("failed to get offered image ID from %s error: %v", jsonutils.Marshal(image).PrettyString(), err)
		return "", err
	}
	return _image.Id, nil
}

func (image *SImage) getImageReference() ImageReference {
	if isPrivateImageID(image.ID) {
		return ImageReference{
			ID: image.ID,
		}
	} else {
		return ImageReference{
			Sku:       image.Sku,
			Publisher: image.Publisher,
			Version:   image.Version,
			Offer:     image.Offer,
		}
	}
}
