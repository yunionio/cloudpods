package azure

import (
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/common"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/diskstream"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SContainer struct {
	storageaccount *SStorageAccount
	Name           string
}

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

type PrimaryEndpoints struct {
	Blob  string
	Queue string
	Table string
	File  string
}

type AccountProperties struct {
	//classic
	ClassicStorageProperties

	//normal
	PrimaryEndpoints  PrimaryEndpoints `json:"primaryEndpoints,omitempty"`
	ProvisioningState string
	PrimaryLocation   string
	SecondaryLocation string
	//CreationTime           time.Time
	AccessTier               string `json:"accessTier,omitempty"`
	EnableHTTPSTrafficOnly   *bool  `json:"supportsHttpsTrafficOnly,omitempty"`
	IsHnsEnabled             bool   `json:"isHnsEnabled,omitempty"`
	AzureFilesAadIntegration bool   `json:"azureFilesAadIntegration,omitempty"`
}

type SStorageAccount struct {
	region     *SRegion
	accountKey string
	Sku        Sku    `json:"sku,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Identity   *Identity
	Properties AccountProperties
	Location   string
	ID         string
	Name       string
	Type       string
	Tags       map[string]string
}

func (self *SRegion) GetStorageAccounts() ([]SStorageAccount, error) {
	result := []SStorageAccount{}
	accounts := []SStorageAccount{}
	err := self.client.ListAll("Microsoft.Storage/storageAccounts", &accounts)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Location == self.Name {
			accounts[i].region = self
			result = append(result, accounts[i])
		}
	}
	return result, nil
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
	for {
		uniqString := randomString("storage", 8)
		requestBody := fmt.Sprintf(`{"name": "%s", "type": "Microsoft.Storage/storageAccounts"}`, uniqString)
		body, err := self.client.CheckNameAvailability("Microsoft.Storage", requestBody)
		if err != nil {
			continue
		}
		if avaliable, _ := body.Bool("nameAvailable"); avaliable {
			return uniqString
		}
	}
}

func (self *SRegion) CreateStorageAccount(storageAccount string) (*SStorageAccount, error) {
	account, err := self.getStorageAccountID(storageAccount)
	if err == nil {
		return account, nil
	}
	if err == cloudprovider.ErrNotFound {
		uniqName := self.GetUniqStorageAccountName()
		stoargeaccount := SStorageAccount{
			region: self,
			Sku: Sku{
				Name: "Standard_GRS",
			},
			Location: self.Name,
			Kind:     "Storage",
			Properties: AccountProperties{
				IsHnsEnabled:             true,
				AzureFilesAadIntegration: true,
			},
			Name: uniqName,
			Type: "Microsoft.Storage/storageAccounts",
			Tags: map[string]string{"id": storageAccount},
		}
		return &stoargeaccount, self.client.Create(jsonutils.Marshal(stoargeaccount), &stoargeaccount)
	}
	return nil, err
}

func (self *SRegion) getStorageAccountID(storageAccount string) (*SStorageAccount, error) {
	accounts, err := self.GetStorageAccounts()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(accounts); i++ {
		for k, v := range accounts[i].Tags {
			if k == "id" && v == storageAccount {
				accounts[i].region = self
				return &accounts[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetStorageAccountDetail(accountId string) (*SStorageAccount, error) {
	account := SStorageAccount{region: self}
	return &account, self.client.Get(accountId, []string{}, &account)
}

type AccountKeys struct {
	KeyName     string
	Permissions string
	Value       string
}

func (self *SRegion) GetStorageAccountKey(accountId string) (string, error) {
	body, err := self.client.PerformAction(accountId, "listKeys", "")
	if err != nil {
		return "", err
	}
	if body.Contains("keys") {
		keys := []AccountKeys{}
		err = body.Unmarshal(&keys, "keys")
		if err != nil {
			return "", err
		}
		for _, key := range keys {
			if strings.ToLower(key.Permissions) == "full" {
				return key.Value, nil
			}
		}
		return "", fmt.Errorf("not find storageaccount %s key", accountId)
	}
	return body.GetString("primaryKey")
}

func (self *SRegion) DeleteStorageAccount(accountId string) error {
	return self.client.Delete(accountId)
}

func (self *SRegion) GetClassicStorageAccounts() ([]SStorageAccount, error) {
	result := []SStorageAccount{}
	accounts := []SStorageAccount{}
	err := self.client.ListAll("Microsoft.ClassicStorage/storageAccounts", &accounts)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Location == self.Name {
			accounts[i].region = self
			result = append(result, accounts[i])
		}
	}
	return result, nil
}

func (self *SStorageAccount) GetAccountKey() (accountKey string, err error) {
	if len(self.accountKey) > 0 {
		return self.accountKey, nil
	}
	self.accountKey, err = self.region.GetStorageAccountKey(self.ID)
	return self.accountKey, err
}

func (self *SStorageAccount) GetBlobBaseUrl() string {
	if self.Type == "Microsoft.Storage/storageAccounts" {
		return self.Properties.PrimaryEndpoints.Blob
	}
	for _, url := range self.Properties.Endpoints {
		if strings.Contains(url, ".blob.") {
			return url
		}
	}
	return ""
}

func (self *SStorageAccount) CreateContainer(containerName string) (*SContainer, error) {
	accessKey, err := self.GetAccountKey()
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(self.Name, accessKey, self.region.client.env)
	if err != nil {
		return nil, err
	}
	container := SContainer{storageaccount: self}
	blobService := client.GetBlobService()
	containerRef := blobService.GetContainerReference(containerName)
	err = containerRef.Create(&storage.CreateContainerOptions{})
	if err != nil {
		return nil, err
	}
	return &container, jsonutils.Update(&container, containerRef)
}

func (self *SStorageAccount) GetContainers() ([]SContainer, error) {
	accessKey, err := self.GetAccountKey()
	if err != nil {
		return nil, err
	}
	containers := []SContainer{}
	client, err := storage.NewBasicClientOnSovereignCloud(self.Name, accessKey, self.region.client.env)
	if err != nil {
		return nil, err
	}
	result, err := client.GetBlobService().ListContainers(storage.ListContainersParameters{})
	if err != nil {
		return nil, err
	}
	err = jsonutils.Update(&containers, result.Containers)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(containers); i++ {
		containers[i].storageaccount = self
	}
	return containers, nil
}

func (self *SContainer) ListFiles() ([]storage.Blob, error) {
	storageaccount := self.storageaccount
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, storageaccount.accountKey, storageaccount.region.client.env)
	if err != nil {
		return nil, err
	}
	blobService := client.GetBlobService()
	result, err := blobService.GetContainerReference(self.Name).ListBlobs(storage.ListBlobsParameters{Include: &storage.IncludeBlobDataset{Snapshots: true, Metadata: true}})
	if err != nil {
		return nil, err
	}
	return result.Blobs, nil
}

func (self *SContainer) getClient() (storage.Client, error) {
	storageaccount := self.storageaccount
	return storage.NewBasicClientOnSovereignCloud(storageaccount.Name, storageaccount.accountKey, storageaccount.region.client.env)
}

func (self *SContainer) getContainerRef() (*storage.Container, error) {
	client, err := self.getClient()
	if err != nil {
		return nil, err
	}
	blobService := client.GetBlobService()
	return blobService.GetContainerReference(self.Name), nil
}

func (self *SContainer) Delete(fileName string) error {
	containerRef, err := self.getContainerRef()
	if err != nil {
		return err
	}
	blobRef := containerRef.GetBlobReference(fileName)
	_, err = blobRef.DeleteIfExists(&storage.DeleteBlobOptions{})
	return err
}

func (self *SContainer) CopySnapshot(snapshotId, fileName string) (*storage.Blob, error) {
	containerRef, err := self.getContainerRef()
	if err != nil {
		return nil, err
	}
	blobRef := containerRef.GetBlobReference(fileName)
	uri, err := self.storageaccount.region.GrantAccessSnapshot(snapshotId)
	if err != nil {
		return nil, err
	}
	err = blobRef.Copy(uri, &storage.CopyOptions{})
	if err != nil {
		return nil, err
	}
	return blobRef, blobRef.GetProperties(&storage.GetBlobPropertiesOptions{})
}

func (self *SContainer) UploadFile(filePath string) (string, error) {
	storageaccount := self.storageaccount
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, storageaccount.accountKey, storageaccount.region.client.env)
	if err != nil {
		return "", err
	}
	blobService := client.GetBlobService()
	containerRef := blobService.GetContainerReference(self.Name)

	err = ensureVHDSanity(filePath)
	if err != nil {
		return "", err
	}
	diskStream, err := diskstream.CreateNewDiskStream(filePath)
	if err != nil {
		return "", err
	}
	defer diskStream.Close()
	blobName := path.Base(filePath)
	blobRef := containerRef.GetBlobReference(blobName)
	blobRef.Properties.ContentLength = diskStream.GetSize()
	err = blobRef.PutPageBlob(&storage.PutBlobOptions{})
	if err != nil {
		return "", err
	}
	var rangesToSkip []*common.IndexRange
	uploadableRanges, err := LocateUploadableRanges(diskStream, rangesToSkip, DefaultReadBlockSize)
	if err != nil {
		return "", err
	}
	uploadableRanges, err = DetectEmptyRanges(diskStream, uploadableRanges)
	if err != nil {
		return "", err
	}

	cxt := &DiskUploadContext{
		VhdStream:             diskStream,
		UploadableRanges:      uploadableRanges,
		AlreadyProcessedBytes: common.TotalRangeLength(rangesToSkip),
		BlobServiceClient:     blobService,
		ContainerName:         self.Name,
		BlobName:              blobName,
		Parallelism:           3,
		Resume:                false,
		MD5Hash:               []byte(""), //localMetaData.FileMetaData.MD5Hash,
	}

	if err := Upload(cxt); err != nil {
		return "", err
	}
	return blobRef.GetURL(), nil
}

func (self *SStorageAccount) UploadFile(containerName string, filePath string) (string, error) {
	containers, err := self.GetContainers()
	if err != nil {
		return "", err
	}
	container := &SContainer{}
	find := false
	for i := 0; i < len(containers); i++ {
		if containers[i].Name == containerName {
			container = &containers[i]
			find = true
			break
		}
	}
	if !find {
		container, err = self.CreateContainer(containerName)
		if err != nil {
			return "", err
		}
	}
	return container.UploadFile(filePath)
}
