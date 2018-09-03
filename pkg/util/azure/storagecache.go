package azure

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
)

const (
	DefaultStorageAccount string = "storageimagecache"
	DefaultBlobContainer  string = "image-cache"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) fetchImages() error {
	if images, err := self.region.GetImages(); err != nil {
		return err
	} else {
		self.iimages = make([]cloudprovider.ICloudImage, len(images))
		for i := 0; i < len(images); i++ {
			images[i].storageCache = self
			self.iimages[i] = &images[i]
		}
	}
	return nil
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		if err := self.fetchImages(); err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
}

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		status, _ := self.region.GetImageStatus(extId)
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	}
	return self.uploadImage(userCred, imageId, isForce)
}

func (self *SRegion) CreateStorageAccount(resourceGroup, storageAccount string) error {
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	sku := storage.Sku{Name: storage.SkuName("Standard_GRS")}
	params := storage.AccountCreateParameters{Sku: &sku, Location: &self.Name, Kind: storage.Kind("Storage")}
	if len(resourceGroup) == 0 {
		resourceGroup = DefaultResourceGroup["storage"]
	}
	if len(storageAccount) == 0 {
		storageAccount = DefaultStorageAccount
	}
	if _, err := storageClinet.Create(context.Background(), resourceGroup, storageAccount, params); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) CreateStorageBlob(resourceGroup, storageAccount, containerName string) error {
	blobClient := storage.NewBlobContainersClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	blobClient.Authorizer = self.client.authorizer
	if len(resourceGroup) == 0 {
		resourceGroup = DefaultResourceGroup["storage"]
	}
	if len(storageAccount) == 0 {
		storageAccount = DefaultStorageAccount
	}
	if len(containerName) == 0 {
		containerName = DefaultBlobContainer
	}
	properties := storage.ContainerProperties{PublicAccess: storage.PublicAccess("PublicAccessNone"), LeaseDuration: storage.LeaseDuration("Infinite")}
	_type := "Microsoft.Storage/storageAccounts"
	params := storage.BlobContainer{Name: &containerName, ContainerProperties: &properties, Type: &_type}
	if _, err := blobClient.Create(context.Background(), resourceGroup, storageAccount, containerName, params); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) isStorageBlobExist(resourceGroup, storageAccount, containerName string) bool {
	blobClient := storage.NewBlobContainersClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	blobClient.Authorizer = self.client.authorizer
	if len(resourceGroup) == 0 {
		resourceGroup = DefaultResourceGroup["storage"]
	}
	if len(storageAccount) == 0 {
		storageAccount = DefaultStorageAccount
	}
	if len(containerName) == 0 {
		containerName = DefaultBlobContainer
	}
	if _, err := blobClient.Get(context.Background(), resourceGroup, storageAccount, containerName); err != nil {
		return false
	}
	return true
}

func (self *SRegion) isStorageAccountExist(resourceGroup, storageAccount string) bool {
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	if len(resourceGroup) == 0 {
		resourceGroup = DefaultResourceGroup["storage"]
	}
	if len(storageAccount) == 0 {
		storageAccount = DefaultStorageAccount
	}
	if _, err := storageClinet.GetProperties(context.Background(), resourceGroup, storageAccount); err != nil {
		return false
	}
	return true
}

func (self *SRegion) CheckBlob(resourceGroup, storageAccount, blobName string) error {
	if err := self.client.fetchAzueResourceGroup(); err != nil {
		return err
	} else if !self.isStorageAccountExist(resourceGroup, storageAccount) {
		if err := self.CreateStorageAccount(resourceGroup, storageAccount); err != nil {
			return err
		}
	}
	if !self.isStorageBlobExist(resourceGroup, storageAccount, blobName) {
		if err := self.CreateStorageBlob(resourceGroup, storageAccount, blobName); err != nil {
			return err
		}
	}
	return nil
}

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, isForce bool) (string, error) {
	return "", nil
}
