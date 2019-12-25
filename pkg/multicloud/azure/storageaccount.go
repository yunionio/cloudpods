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
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/common"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/diskstream"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SContainer struct {
	storageaccount *SStorageAccount
	Name           string
}

type SSku struct {
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

type SStorageEndpoints struct {
	Blob  string
	Queue string
	Table string
	File  string
}

type AccountProperties struct {
	//classic
	ClassicStorageProperties

	//normal
	PrimaryEndpoints   SStorageEndpoints `json:"primaryEndpoints,omitempty"`
	ProvisioningState  string
	PrimaryLocation    string
	SecondaryEndpoints SStorageEndpoints `json:"secondaryEndpoints,omitempty"`
	SecondaryLocation  string
	//CreationTime           time.Time
	AccessTier               string `json:"accessTier,omitempty"`
	EnableHTTPSTrafficOnly   *bool  `json:"supportsHttpsTrafficOnly,omitempty"`
	IsHnsEnabled             bool   `json:"isHnsEnabled,omitempty"`
	AzureFilesAadIntegration bool   `json:"azureFilesAadIntegration,omitempty"`
}

type SStorageAccount struct {
	multicloud.SBaseBucket

	region *SRegion

	accountKey string

	Sku      SSku   `json:"sku,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Identity *Identity
	Location string `json:"location,omitempty"`
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Tags     map[string]string

	Properties AccountProperties `json:"properties"`
}

func (self *SRegion) GetStorageAccounts() ([]*SStorageAccount, error) {
	iBuckets, err := self.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]*SStorageAccount, 0)
	for i := range iBuckets {
		if iBuckets[i].GetLocation() != self.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i].(*SStorageAccount))
	}
	return ret, nil
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

type sStorageAccountCheckNameAvailabilityInput struct {
	Name string
	Type string
}

type sStorageAccountCheckNameAvailabilityOutput struct {
	NameAvailable bool   `json:"nameAvailable"`
	Reason        string `json:"reason"`
	Message       string `json:"message"`
}

func (self *SRegion) checkStorageAccountNameExist(name string) (bool, error) {
	url := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage/checkNameAvailability?api-version=2019-04-01", self.client.subscriptionId)
	body := jsonutils.Marshal(sStorageAccountCheckNameAvailabilityInput{
		Name: name,
		Type: "Microsoft.Storage/storageAccounts",
	})
	resp, err := self.client.jsonRequest("POST", url, body.String())
	if err != nil {
		return false, errors.Wrap(err, "jsonRequest")
	}
	output := sStorageAccountCheckNameAvailabilityOutput{}
	err = resp.Unmarshal(&output)
	if err != nil {
		return false, errors.Wrap(err, "Unmarshal")
	}
	if output.NameAvailable {
		return false, nil
	} else {
		if output.Reason == "AlreadyExists" {
			return true, nil
		} else {
			return false, errors.Error(output.Reason)
		}
	}
}

type SStorageAccountSku struct {
	ResourceType string   `json:"resourceType"`
	Name         string   `json:"name"`
	Tier         string   `json:"tier"`
	Kind         string   `json:"kind"`
	Locations    []string `json:"locations"`
	Capabilities []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"capabilities"`
	Restrictions []struct {
		Type       string   `json:"type"`
		Values     []string `json:"values"`
		ReasonCode string   `json:"reasonCode"`
	} `json:"restrictions"`
}

func (self *SRegion) GetStorageAccountSkus() ([]SStorageAccountSku, error) {
	skus := make([]SStorageAccountSku, 0)
	err := self.client.List("providers/Microsoft.Storage/skus?api-version=2019-04-01", &skus)
	if err != nil {
		return nil, errors.Wrap(err, "List")
	}
	ret := make([]SStorageAccountSku, 0)
	for i := range skus {
		if utils.IsInStringArray(self.GetId(), skus[i].Locations) {
			ret = append(ret, skus[i])
		}
	}
	return ret, nil
}

func (self *SRegion) getStorageAccountSkuByName(name string) (*SStorageAccountSku, error) {
	skus, err := self.GetStorageAccountSkus()
	if err != nil {
		return nil, errors.Wrap(err, "getStorageAccountSkus")
	}
	for _, kind := range []string{
		"StorageV2",
		"Storage",
	} {
		for i := range skus {
			if skus[i].Name == name && skus[i].Kind == kind {
				return &skus[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) createStorageAccount(name string, skuName string) (*SStorageAccount, error) {
	storageKind := "Storage"
	if len(skuName) > 0 {
		sku, err := self.getStorageAccountSkuByName(skuName)
		if err != nil {
			return nil, errors.Wrap(err, "getStorageAccountSkuByName")
		}
		skuName = sku.Name
		storageKind = sku.Kind
	} else {
		skuName = "Standard_GRS"
	}
	storageaccount := SStorageAccount{
		region:   self,
		Location: self.Name,
		Sku: SSku{
			Name: skuName,
		},
		Kind: storageKind,
		Properties: AccountProperties{
			IsHnsEnabled:             true,
			AzureFilesAadIntegration: true,
		},
		Name: name,
		Type: "Microsoft.Storage/storageAccounts",
	}

	err := self.client.Create(jsonutils.Marshal(storageaccount), &storageaccount)
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	self.client.invalidateIBuckets()
	return &storageaccount, nil
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
			Sku: SSku{
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
				return accounts[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetStorageAccountDetail(accountId string) (*SStorageAccount, error) {
	account := SStorageAccount{region: self}
	err := self.client.Get(accountId, []string{}, &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
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

func (self *SRegion) GetClassicStorageAccounts() ([]*SStorageAccount, error) {
	result := make([]*SStorageAccount, 0)
	accounts := make([]SStorageAccount, 0)
	err := self.client.ListAll("Microsoft.ClassicStorage/storageAccounts", &accounts)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(accounts); i++ {
		if accounts[i].Location == self.Name {
			accounts[i].region = self
			result = append(result, &accounts[i])
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

func (self *SStorageAccount) getBlobServiceClient() (*storage.BlobStorageClient, error) {
	accessKey, err := self.GetAccountKey()
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(self.Name, accessKey, self.region.client.env)
	if err != nil {
		return nil, err
	}
	srv := client.GetBlobService()
	return &srv, nil
}

func (self *SStorageAccount) CreateContainer(containerName string) (*SContainer, error) {
	blobService, err := self.getBlobServiceClient()
	if err != nil {
		return nil, errors.Wrap(err, "getBlobServiceClient")
	}
	containerRef := blobService.GetContainerReference(containerName)
	err = containerRef.Create(&storage.CreateContainerOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	container := SContainer{
		storageaccount: self,
		Name:           containerName,
	}
	return &container, nil
}

func (self *SStorageAccount) GetContainers() ([]SContainer, error) {
	blobService, err := self.getBlobServiceClient()
	if err != nil {
		return nil, errors.Wrap(err, "getBlobServiceClient")
	}
	result, err := blobService.ListContainers(storage.ListContainersParameters{})
	if err != nil {
		return nil, err
	}
	containers := []SContainer{}
	err = jsonutils.Update(&containers, result.Containers)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(containers); i++ {
		containers[i].storageaccount = self
	}
	return containers, nil
}

func (self *SStorageAccount) GetContainer(name string) (*SContainer, error) {
	containers, err := self.GetContainers()
	if err != nil {
		return nil, err
	}
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SContainer) ListAllFiles(include *storage.IncludeBlobDataset) ([]storage.Blob, error) {
	blobs := make([]storage.Blob, 0)
	var marker string
	for {
		result, err := self.ListFiles("", marker, "", 5000, include)
		if err != nil {
			return nil, errors.Wrap(err, "ListFiles")
		}
		if len(result.Blobs) > 0 {
			blobs = append(blobs, result.Blobs...)
		}
		if len(result.NextMarker) == 0 {
			break
		} else {
			marker = result.NextMarker
		}
	}
	return blobs, nil
}

func (self *SContainer) ListFiles(prefix string, marker string, delimiter string, maxCount int, include *storage.IncludeBlobDataset) (storage.BlobListResponse, error) {
	var result storage.BlobListResponse
	blobService, err := self.storageaccount.getBlobServiceClient()
	if err != nil {
		return result, errors.Wrap(err, "getBlobServiceClient")
	}
	params := storage.ListBlobsParameters{Include: include}
	if len(prefix) > 0 {
		params.Prefix = prefix
	}
	if len(marker) > 0 {
		params.Marker = marker
	}
	if len(delimiter) > 0 {
		params.Delimiter = delimiter
	}
	if maxCount > 0 {
		params.MaxResults = uint(maxCount)
	}
	result, err = blobService.GetContainerReference(self.Name).ListBlobs(params)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (self *SContainer) getContainerRef() (*storage.Container, error) {
	blobService, err := self.storageaccount.getBlobServiceClient()
	if err != nil {
		return nil, errors.Wrap(err, "getBlobServiceClient")
	}
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

func setBlobRefMeta(blobRef *storage.Blob, meta http.Header) (bool, bool) {
	propChanged := false
	metaChanged := false
	for k, v := range meta {
		if len(v) == 0 || len(v[0]) == 0 {
			continue
		}
		switch http.CanonicalHeaderKey(k) {
		case cloudprovider.META_HEADER_CACHE_CONTROL:
			blobRef.Properties.CacheControl = v[0]
			propChanged = true
		case cloudprovider.META_HEADER_CONTENT_TYPE:
			blobRef.Properties.ContentType = v[0]
			propChanged = true
		case cloudprovider.META_HEADER_CONTENT_MD5:
			blobRef.Properties.ContentMD5 = v[0]
			propChanged = true
		case cloudprovider.META_HEADER_CONTENT_ENCODING:
			blobRef.Properties.ContentEncoding = v[0]
			propChanged = true
		case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
			blobRef.Properties.ContentLanguage = v[0]
			propChanged = true
		case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
			blobRef.Properties.ContentDisposition = v[0]
			propChanged = true
		default:
			if blobRef.Metadata == nil {
				blobRef.Metadata = storage.BlobMetadata{}
			}
			blobRef.Metadata[k] = v[0]
			metaChanged = true
		}
	}
	return propChanged, metaChanged
}

func (self *SContainer) UploadStream(key string, reader io.Reader, meta http.Header) error {
	blobService, err := self.storageaccount.getBlobServiceClient()
	if err != nil {
		return errors.Wrap(err, "getBlobServiceClient")
	}
	containerRef := blobService.GetContainerReference(self.Name)
	blobRef := containerRef.GetBlobReference(key)
	blobRef.Properties.BlobType = storage.BlobTypeBlock
	if meta != nil {
		setBlobRefMeta(blobRef, meta)
	}
	return blobRef.CreateBlockBlobFromReader(reader, &storage.PutBlobOptions{})
}

func (self *SContainer) SignUrl(method string, key string, expire time.Duration) (string, error) {
	blobService, err := self.storageaccount.getBlobServiceClient()
	if err != nil {
		return "", errors.Wrap(err, "getBlobServiceClient")
	}
	containerRef := blobService.GetContainerReference(self.Name)
	sas := storage.ContainerSASOptions{}
	sas.Start = time.Now()
	sas.Expiry = sas.Start.Add(expire)
	sas.UseHTTPS = true
	sas.Identifier = key
	switch method {
	case "GET":
		sas.Read = true
	case "PUT":
		sas.Read = true
		sas.Add = true
		sas.Create = true
		sas.Write = true
	case "DELETE":
		sas.Read = true
		sas.Write = true
		sas.Delete = true
	default:
		return "", errors.Error("unsupport method")
	}
	return containerRef.GetSASURI(sas)
}

func (self *SContainer) UploadFile(filePath string) (string, error) {
	blobService, err := self.storageaccount.getBlobServiceClient()
	if err != nil {
		return "", errors.Wrap(err, "getBlobServiceClient")
	}
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
		BlobServiceClient:     *blobService,
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

func (self *SContainer) getAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	containerRef, err := self.getContainerRef()
	if err != nil {
		log.Errorf("getContainerRef fail %s", err)
		return acl
	}
	output, err := containerRef.GetPermissions(nil)
	if err != nil {
		log.Errorf("containerRef.GetPermissions fail %s", err)
		return acl
	}
	switch output.AccessType {
	case storage.ContainerAccessTypePrivate:
		acl = cloudprovider.ACLPrivate
	case storage.ContainerAccessTypeContainer:
		acl = cloudprovider.ACLPublicRead
	}
	return acl
}

func (self *SContainer) setAcl(aclStr cloudprovider.TBucketACLType) error {
	perm := storage.ContainerPermissions{}
	switch aclStr {
	case cloudprovider.ACLPublicRead:
		perm.AccessType = storage.ContainerAccessTypeContainer
	case cloudprovider.ACLPrivate:
		perm.AccessType = storage.ContainerAccessTypePrivate
	default:
		return errors.Error("unsupported ACL:" + string(aclStr))
	}
	containerRef, err := self.getContainerRef()
	if err != nil {
		return errors.Wrap(err, "getContainerRef")
	}
	err = containerRef.SetPermissions(perm, nil)
	if err != nil {
		return errors.Wrap(err, "SetPermissions")
	}
	return nil
}

func (self *SStorageAccount) UploadFile(containerName string, filePath string) (string, error) {
	container, err := self.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	return container.UploadFile(filePath)
}

func (self *SStorageAccount) getOrCreateContainer(containerName string, create bool) (*SContainer, error) {
	containers, err := self.GetContainers()
	if err != nil {
		return nil, errors.Wrap(err, "GetContainers")
	}
	var container *SContainer
	find := false
	for i := 0; i < len(containers); i++ {
		if containers[i].Name == containerName {
			container = &containers[i]
			find = true
			break
		}
	}
	if !find {
		if !create {
			return nil, cloudprovider.ErrNotFound
		}
		container, err = self.CreateContainer(containerName)
		if err != nil {
			return nil, errors.Wrap(err, "CreateContainer")
		}
	}
	return container, nil
}

func (self *SStorageAccount) UploadStream(containerName string, key string, reader io.Reader, meta http.Header) error {
	container, err := self.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "getOrCreateContainer")
	}
	return container.UploadStream(key, reader, meta)
}

func (b *SStorageAccount) MaxPartSizeBytes() int64 {
	return 100 * 1000 * 1000
}

func (b *SStorageAccount) MaxPartCount() int {
	return 50000
}

func (b *SStorageAccount) GetProjectId() string {
	return ""
}

func (b *SStorageAccount) GetGlobalId() string {
	return b.Name
}

func (b *SStorageAccount) GetName() string {
	return b.Name
}

func (b *SStorageAccount) GetLocation() string {
	return b.Location
}

func (b *SStorageAccount) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SStorageAccount) GetCreateAt() time.Time {
	return time.Time{}
}

func (b *SStorageAccount) GetStorageClass() string {
	return b.Sku.Tier
}

// get the common ACL of all containers
func (b *SStorageAccount) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	containers, err := b.GetContainers()
	if err != nil {
		log.Errorf("GetContainers error %s", err)
		return acl
	}
	for i := range containers {
		aclC := containers[i].getAcl()
		if i == 0 {
			if aclC != acl {
				acl = aclC
			}
		} else if aclC != acl {
			acl = cloudprovider.ACLPrivate
		}
	}
	return acl
}

func (b *SStorageAccount) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	containers, err := b.GetContainers()
	if err != nil {
		return errors.Wrap(err, "GetContainers")
	}
	for i := range containers {
		err = containers[i].setAcl(aclStr)
		if err != nil {
			return errors.Wrap(err, "containers.setAcl")
		}
	}
	return nil
}

func getDesc(prefix, name string) string {
	if len(prefix) > 0 {
		return prefix + "-" + name
	} else {
		return name
	}
}

func (ep SStorageEndpoints) getUrls(prefix string) []cloudprovider.SBucketAccessUrl {
	ret := make([]cloudprovider.SBucketAccessUrl, 0)
	if len(ep.Blob) > 0 {
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         ep.Blob,
			Description: getDesc(prefix, "blob"),
			Primary:     true,
		})
	}
	if len(ep.Queue) > 0 {
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         ep.Queue,
			Description: getDesc(prefix, "queue"),
		})
	}
	if len(ep.Table) > 0 {
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         ep.Table,
			Description: getDesc(prefix, "table"),
		})
	}
	if len(ep.File) > 0 {
		ret = append(ret, cloudprovider.SBucketAccessUrl{
			Url:         ep.File,
			Description: getDesc(prefix, "file"),
		})
	}
	return ret
}

func (b *SStorageAccount) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	primary := b.Properties.PrimaryEndpoints.getUrls("")
	secondary := b.Properties.SecondaryEndpoints.getUrls("secondary")
	if len(secondary) > 0 {
		primary = append(primary, secondary...)
	}
	return primary
}

func (b *SStorageAccount) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func getBlobRefMeta(blob *storage.Blob) http.Header {
	meta := http.Header{}
	for k, v := range blob.Metadata {
		meta.Add(k, v)
	}
	if len(blob.Properties.CacheControl) > 0 {
		meta.Set(cloudprovider.META_HEADER_CACHE_CONTROL, blob.Properties.CacheControl)
	}
	if len(blob.Properties.ContentType) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_TYPE, blob.Properties.ContentType)
	}
	if len(blob.Properties.ContentDisposition) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_DISPOSITION, blob.Properties.ContentDisposition)
	}
	if len(blob.Properties.ContentLanguage) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_LANGUAGE, blob.Properties.ContentLanguage)
	}
	if len(blob.Properties.ContentEncoding) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_ENCODING, blob.Properties.ContentEncoding)
	}
	if len(blob.Properties.ContentMD5) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_MD5, blob.Properties.ContentMD5)
	}
	return meta
}

func (b *SStorageAccount) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	containers, err := b.GetContainers()
	if err != nil {
		return result, errors.Wrap(err, "GetContainers")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	result.CommonPrefixes = make([]cloudprovider.ICloudObject, 0)
	for i := 0; i < len(containers); i += 1 {
		container := containers[i]
		matchLen := len(container.Name)
		if matchLen > len(prefix) {
			matchLen = len(prefix)
		}
		var subMarker string
		if len(marker) < len(container.Name) {
			if marker > container.Name {
				continue
			}
			subMarker = ""
		} else {
			containerMarker := marker[:len(container.Name)]
			if containerMarker > container.Name {
				continue
			}
			subMarker = marker[len(container.Name)+1:]
		}
		if marker > container.Name {
			continue
		}
		if maxCount <= 0 {
			break
		}
		// container name matches prefix
		if container.Name[:matchLen] == prefix[:matchLen] && (len(prefix) <= len(container.Name) || strings.HasPrefix(prefix, container.Name+"/")) {
			if delimiter == "/" && len(prefix) == 0 {
				// populate CommonPrefixes
				o := &SObject{
					container: &container,
					SBaseCloudObject: cloudprovider.SBaseCloudObject{
						Key: container.Name + "/",
					},
				}
				result.CommonPrefixes = append(result.CommonPrefixes, o)
				maxCount -= 1
			} else if delimiter == "/" && prefix == container.Name+delimiter {
				// do nothing
			} else if len(prefix) <= len(container.Name)+1 {
				// returns contain names only
				o := &SObject{
					container: &container,
					SBaseCloudObject: cloudprovider.SBaseCloudObject{
						Key: container.Name + "/",
					},
				}
				result.Objects = append(result.Objects, o)
				maxCount -= 1
			}
			if delimiter == "" || len(prefix) >= len(container.Name)+1 {
				subPrefix := ""
				if len(prefix) >= len(container.Name)+1 {
					subPrefix = prefix[len(container.Name)+1:]
				}
				oResult, err := container.ListFiles(subPrefix, subMarker, delimiter, maxCount, nil)
				if err != nil {
					return result, errors.Wrap(err, "ListFiles")
				}
				for i := range oResult.Blobs {
					blob := oResult.Blobs[i]
					o := &SObject{
						container: &container,
						SBaseCloudObject: cloudprovider.SBaseCloudObject{
							Key:          container.Name + "/" + blob.Name,
							SizeBytes:    blob.Properties.ContentLength,
							StorageClass: "",
							ETag:         blob.Properties.Etag,
							LastModified: time.Time(blob.Properties.LastModified),
						},
					}
					result.Objects = append(result.Objects, o)
					maxCount -= 1
					if maxCount == 0 {
						break
					}
					result.NextMarker = blob.Name
				}
				for i := range oResult.BlobPrefixes {
					o := &SObject{
						container: &container,
						SBaseCloudObject: cloudprovider.SBaseCloudObject{
							Key: container.Name + "/" + oResult.BlobPrefixes[i],
						},
					}
					result.CommonPrefixes = append(result.CommonPrefixes, o)
					maxCount -= 1
				}
				if len(oResult.NextMarker) > 0 {
					result.NextMarker = container.Name + "/" + oResult.NextMarker
					result.IsTruncated = true
					break
				}
			}
		}
	}
	return result, nil
}

func splitKey(key string) (string, string, error) {
	slashPos := strings.IndexByte(key, '/')
	if slashPos <= 0 {
		return "", "", errors.Wrap(httperrors.ErrForbidden, "cannot put object to root")
	}
	containerName := key[:slashPos]
	key = key[slashPos+1:]
	return containerName, key, nil
}

func splitKeyAndBlob(path string) (string, string, error) {
	containerName, key, err := splitKey(path)
	if err != nil {
		return "", "", errors.Wrapf(err, "splitKey: %s", path)
	}
	if len(key) == 0 {
		return "", "", errors.Error("empty blob path")
	}
	return containerName, key, nil
}

func (b *SStorageAccount) PutObject(ctx context.Context, key string, reader io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	containerName, blob, err := splitKey(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	if len(blob) > 0 {
		// put blob
		err = b.UploadStream(containerName, blob, reader, meta)
		if err != nil {
			return errors.Wrap(err, "UploadStream")
		}
	} else {
		// create container
		if sizeBytes > 0 {
			return errors.Wrap(httperrors.ErrForbidden, "not allow to create blob outsize of container")
		}
		_, err = b.getOrCreateContainer(containerName, true)
		if err != nil {
			return errors.Wrap(err, "getOrCreateContainer")
		}
	}
	return nil
}

func (b *SStorageAccount) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return "", errors.Wrap(err, "getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blob)
	if meta != nil {
		setBlobRefMeta(blobRef, meta)
	}
	err = blobRef.CreateBlockBlob(&storage.PutBlobOptions{})
	if err != nil {
		return "", errors.Wrap(err, "CreateBlockBlob")
	}
	uploadId, err := blobRef.AcquireLease(-1, "", nil)
	if err != nil {
		return "", errors.Wrap(err, "blobRef.AcquireLease")
	}

	return uploadId, nil
}

func partIndex2BlockId(partIndex int) string {
	return base64.URLEncoding.EncodeToString([]byte(strconv.FormatInt(int64(partIndex), 10)))
}

func (b *SStorageAccount) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64) (string, error) {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return "", errors.Wrap(err, "getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blob)

	opts := &storage.PutBlockOptions{}
	opts.LeaseID = uploadId

	blockId := partIndex2BlockId(partIndex)
	err = blobRef.PutBlockWithLength(blockId, uint64(partSize), input, opts)
	if err != nil {
		return "", errors.Wrap(err, "PutBlockWithLength")
	}

	return blockId, nil
}

func (b *SStorageAccount) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, blockIds []string) error {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return errors.Wrap(err, "getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blob)

	blocks := make([]storage.Block, len(blockIds))
	for i := range blockIds {
		blocks[i] = storage.Block{
			ID:     blockIds[i],
			Status: storage.BlockStatusLatest,
		}
	}
	opts := &storage.PutBlockListOptions{}
	opts.LeaseID = uploadId
	err = blobRef.PutBlockList(blocks, opts)
	if err != nil {
		return errors.Wrap(err, "PutBlockList")
	}

	err = blobRef.ReleaseLease(uploadId, nil)
	if err != nil {
		return errors.Wrap(err, "ReleaseLease")
	}

	return nil
}

func (b *SStorageAccount) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, false)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return errors.Wrap(err, "getContainerRef")
	}

	blobRef := containerRef.GetBlobReference(blob)
	err = blobRef.ReleaseLease(uploadId, nil)
	if err != nil {
		return errors.Wrap(err, "ReleaseLease")
	}

	opts := &storage.DeleteBlobOptions{}
	deleteSnapshots := true
	opts.DeleteSnapshots = &deleteSnapshots
	_, err = blobRef.DeleteIfExists(opts)
	if err != nil {
		return errors.Wrap(err, "DeleteIfExists")
	}

	return nil
}

func (b *SStorageAccount) DeleteObject(ctx context.Context, key string) error {
	containerName, blob, err := splitKey(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	blobService, err := b.getBlobServiceClient()
	if err != nil {
		return errors.Wrap(err, "getBlobServiceClient")
	}
	containerRef := blobService.GetContainerReference(containerName)
	if len(blob) > 0 {
		// delete object
		blobRef := containerRef.GetBlobReference(blob)
		_, err = blobRef.DeleteIfExists(nil)
		if err != nil {
			return errors.Wrap(err, "blobRef.DeleteIfExists")
		}
	} else {
		// delete container
		_, err = containerRef.DeleteIfExists(nil)
		if err != nil {
			return errors.Wrap(err, "containerRef.DeleteIfExists")
		}
	}
	return nil
}

func (b *SStorageAccount) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, false)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	return container.SignUrl(method, blob, expire)
}

func (b *SStorageAccount) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	srcIBucket, err := b.region.GetIBucketByName(srcBucket)
	if err != nil {
		return errors.Wrap(err, "GetIBucketByName")
	}
	srcAccount := srcIBucket.(*SStorageAccount)
	srcContName, srcBlob, err := splitKeyAndBlob(srcKey)
	if err != nil {
		return errors.Wrap(err, "src splitKey")
	}
	srcCont, err := srcAccount.getOrCreateContainer(srcContName, false)
	if err != nil {
		return errors.Wrap(err, "src getOrCreateContainer")
	}
	srcContRef, err := srcCont.getContainerRef()
	if err != nil {
		return errors.Wrap(err, "src getContainerRef")
	}
	srcBlobRef := srcContRef.GetBlobReference(srcBlob)

	containerName, blobName, err := splitKeyAndBlob(destKey)
	if err != nil {
		return errors.Wrap(err, "dest splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "dest getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return errors.Wrap(err, "dest getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blobName)
	if meta != nil {
		setBlobRefMeta(blobRef, meta)
	}
	opts := &storage.CopyOptions{}
	err = blobRef.Copy(srcBlobRef.GetURL(), opts)
	if err != nil {
		return errors.Wrap(err, "blboRef.Copy")
	}
	return nil
}

func (b *SStorageAccount) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return nil, errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, false)
	if err != nil {
		return nil, errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return nil, errors.Wrap(err, "container.getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blob)
	if rangeOpt != nil {
		opts := &storage.GetBlobRangeOptions{}
		opts.Range = &storage.BlobRange{
			Start: uint64(rangeOpt.Start),
			End:   uint64(rangeOpt.End),
		}
		return blobRef.GetRange(opts)
	} else {
		opts := &storage.GetBlobOptions{}
		return blobRef.Get(opts)
	}
}

func (b *SStorageAccount) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	srcIBucket, err := b.region.GetIBucketByName(srcBucket)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	srcAccount := srcIBucket.(*SStorageAccount)
	srcContName, srcBlob, err := splitKeyAndBlob(srcKey)
	if err != nil {
		return "", errors.Wrap(err, "src splitKey")
	}
	srcCont, err := srcAccount.getOrCreateContainer(srcContName, false)
	if err != nil {
		return "", errors.Wrap(err, "src getOrCreateContainer")
	}
	srcContRef, err := srcCont.getContainerRef()
	if err != nil {
		return "", errors.Wrap(err, "src getContainerRef")
	}
	srcBlobRef := srcContRef.GetBlobReference(srcBlob)

	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := b.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	containerRef, err := container.getContainerRef()
	if err != nil {
		return "", errors.Wrap(err, "getContainerRef")
	}
	blobRef := containerRef.GetBlobReference(blob)

	opts := &storage.PutBlockFromURLOptions{}
	opts.LeaseID = uploadId

	blockId := partIndex2BlockId(partIndex)
	err = blobRef.PutBlockFromURL(blockId, srcBlobRef.GetURL(), srcOffset, uint64(srcLength), opts)
	if err != nil {
		return "", errors.Wrap(err, "PutBlockFromUrl")
	}

	return blockId, nil
}
