package azure

import (
	"context"
	"fmt"
	"math/rand"
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
	Tags       map[string]string
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

func randomString(prefix string, length int) string {
	bytes := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return prefix + string(result)
}

func (self *SRegion) GetUniqStorageAccountName() string {
	Type := "Microsoft.Storage/storageAccounts"
	for {
		uniqString := randomString("storage", 8)
		storageClinet := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
		storageClinet.Authorizer = self.client.authorizer
		params := storage.AccountCheckNameAvailabilityParameters{
			Name: &uniqString,
			Type: &Type,
		}
		if result, err := storageClinet.CheckNameAvailability(context.Background(), params); err != nil {
			continue
		} else if *result.NameAvailable {
			return uniqString
		}
	}
}

func (self *SRegion) CreateStorageAccount(storageAccount string) (*SStorageAccount, error) {
	if accountId, err := self.getStorageAccountID(storageAccount); err != nil {
		if err == cloudprovider.ErrNotFound {
			result := SStorageAccount{}
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
				Tags: map[string]*string{"id": &storageAccount},
			}
			_, resourceGroup, _ := pareResourceGroupWithName(storageAccount, STORAGE_RESOURCE)
			storageName := self.GetUniqStorageAccountName()
			if resp, err := storageClinet.Create(context.Background(), resourceGroup, storageName, params); err != nil {
				return nil, err
			} else if err := resp.WaitForCompletion(context.Background(), storageClinet.Client); err != nil {
				return nil, err
			} else if account, err := resp.Result(storageClinet); err != nil {
				return nil, err
			} else if err := jsonutils.Update(&result, account); err != nil {
				return nil, err
			}
			return &result, nil
		}
		return nil, err
	} else {
		return self.GetStorageAccountDetail(accountId)
	}
}

func (self *SRegion) getStorageAccountID(storageAccount string) (string, error) {
	if accounts, err := self.GetStorageAccounts(); err != nil {
		return "", err
	} else {
		for i := 0; i < len(accounts); i++ {
			if accounts[i].Location != self.Name {
				continue
			}
			for k, v := range accounts[i].Tags {
				if k == "id" && v == storageAccount {
					return accounts[i].ID, nil
				}
			}
		}
	}
	return "", cloudprovider.ErrNotFound
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
