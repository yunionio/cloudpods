package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type Sku struct {
	Name         string
	Tier         string
	Kind         string
	ResourceType string
}

type Identity struct {
	PrincipalID string
	TenantID    string
	Type        string
}

type AccountProperties struct {
	ProvisioningState      string
	PrimaryLocation        string
	SecondaryLocation      string
	CreationTime           time.Time
	AccessTier             string
	EnableHTTPSTrafficOnly bool
	IsHnsEnabled           bool
}

type SStorageAccount struct {
	Sku        Sku
	Kind       string
	Identity   Identity
	Properties AccountProperties
	Location   string
	ID         string
	Name       string
	Type       string
}

func (self *SRegion) GetStorageAccounts() ([]SStorageAccount, error) {
	accounts := []SStorageAccount{}
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	if result, err := storageClinet.List(context.Background()); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&accounts, result.Value); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (self *SRegion) CreateStorageAccount(storageAccount string) (*SStorageAccount, error) {
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	sku := storage.Sku{Name: storage.StandardLRS}
	params := storage.AccountCreateParameters{
		Sku:      &sku,
		Location: &self.Name,
		Kind:     storage.StorageV2,
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
			AccessTier: storage.Hot,
		},
	}
	globalId, resourceGroup, accountName := pareResourceGroupWithName(storageAccount, STORAGE_RESOURCE)
	//log.Debugf("Create Storage Account: %s", jsonutils.Marshal(params).PrettyString())
	if result, err := storageClinet.Create(context.Background(), resourceGroup, accountName, params); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), storageClinet.Client); err != nil {
		return nil, err
	}
	return self.GetStorageAccountDetail(globalId)
}

func (self *SRegion) GetStorageAccountDetail(accountId string) (*SStorageAccount, error) {
	account := SStorageAccount{}
	_, resourceGroup, accountName := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	if len(accountName) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if result, err := storageClinet.GetProperties(context.Background(), resourceGroup, accountName); err != nil {
		if result.Response.StatusCode == 404 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	} else if err := jsonutils.Update(&account, result); err != nil {
		return nil, err
	}
	return &account, nil
}

func (self *SRegion) GetStorageAccountKey(accountId string) (string, error) {
	_, resourceGroup, accountName := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	if len(accountName) == 0 {
		return "", cloudprovider.ErrNotFound
	}
	if result, err := storageClinet.ListKeys(context.Background(), resourceGroup, accountName); err != nil {
		return "", err
	} else {
		for _, key := range *result.Keys {
			permission := strings.ToLower(string(key.Permissions))
			if permission == "full" {
				return *key.Value, nil
			}
		}
	}
	return "", fmt.Errorf("not find storage account accessKey")
}

func (self *SRegion) DeleteStorageAccount(accountId string) error {
	_, resourceGroup, accountName := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	storageClinet.Authorizer = self.client.authorizer
	_, err := storageClinet.Delete(context.Background(), resourceGroup, accountName)
	return err
}
