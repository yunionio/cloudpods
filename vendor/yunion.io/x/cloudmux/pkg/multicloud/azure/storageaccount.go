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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/common"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/diskstream"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SContainer struct {
	storageaccount *SStorageAccount
	Name           string
	Properties     struct {
		LastModified          time.Time `xml:"Last-Modified"`
		ETag                  string    `xml:"Etag"`
		LeaseStatus           string    `xml:"LeaseStatus"`
		LeaseState            string    `xml:"LeaseState"`
		HasImmutabilityPolicy bool      `xml:"HasImmutabilityPolicy"`
		HasLegalHold          bool      `xml:"HasLegalHold"`
	} `xml:"properties"`
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
	AzureTags

	region *SRegion

	accountKey string

	Sku      SSku   `json:"sku,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Identity *Identity
	Location string `json:"location,omitempty"`
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`

	Properties AccountProperties `json:"properties"`
}

func (region *SRegion) listStorageAccounts() ([]SStorageAccount, error) {
	accounts := []SStorageAccount{}
	err := region.list("Microsoft.Storage/storageAccounts", url.Values{}, &accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	result := []SStorageAccount{}
	for i := range accounts {
		accounts[i].region = region
		result = append(result, accounts[i])
	}
	return result, nil
}

func (region *SRegion) ListStorageAccounts() ([]SStorageAccount, error) {
	return region.listStorageAccounts()
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

func (region *SRegion) GetUniqStorageAccountName() (string, error) {
	for i := 0; i < 20; i++ {
		name := randomString("storage", 8)
		exist, err := region.checkStorageAccountNameExist(name)
		if err == nil && !exist {
			return name, nil
		}
	}
	return "", fmt.Errorf("failed to found uniq storage name")
}

type sNameAvailableOutput struct {
	NameAvailable bool   `json:"nameAvailable"`
	Reason        string `json:"reason"`
	Message       string `json:"message"`
}

func (region *SRegion) checkStorageAccountNameExist(name string) (bool, error) {
	ok, err := region.client.CheckNameAvailability("Microsoft.Storage/storageAccounts", name)
	if err != nil {
		return false, errors.Wrapf(err, "CheckNameAvailability(%s)", name)
	}
	return !ok, nil
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

func (region *SRegion) GetStorageAccountSkus() ([]SStorageAccountSku, error) {
	skus := make([]SStorageAccountSku, 0)
	err := region.client.list("Microsoft.Storage/skus", url.Values{}, &skus)
	if err != nil {
		return nil, errors.Wrap(err, "List")
	}
	ret := make([]SStorageAccountSku, 0)
	for i := range skus {
		if utils.IsInStringArray(region.GetId(), skus[i].Locations) {
			ret = append(ret, skus[i])
		}
	}
	return ret, nil
}

func (region *SRegion) getStorageAccountSkuByName(name string) (*SStorageAccountSku, error) {
	skus, err := region.GetStorageAccountSkus()
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

func (region *SRegion) createStorageAccount(name string, skuName string) (*SStorageAccount, error) {
	storageKind := "Storage"
	if len(skuName) > 0 {
		sku, err := region.getStorageAccountSkuByName(skuName)
		if err != nil {
			return nil, errors.Wrap(err, "getStorageAccountSkuByName")
		}
		skuName = sku.Name
		storageKind = sku.Kind
	} else {
		skuName = "Standard_GRS"
	}
	storageaccount := SStorageAccount{
		region:   region,
		Location: region.Name,
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

	err := region.create("", jsonutils.Marshal(storageaccount), &storageaccount)
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	return &storageaccount, nil
}

func (region *SRegion) CreateStorageAccount(storageAccount string) (*SStorageAccount, error) {
	account, err := region.getStorageAccountID(storageAccount)
	if err == nil {
		return account, nil
	}
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		uniqName, err := region.GetUniqStorageAccountName()
		if err != nil {
			return nil, errors.Wrapf(err, "GetUniqStorageAccountName")
		}
		storageaccount := SStorageAccount{
			region: region,
			Sku: SSku{
				Name: "Standard_GRS",
			},
			Location: region.Name,
			Kind:     "Storage",
			Properties: AccountProperties{
				IsHnsEnabled:             true,
				AzureFilesAadIntegration: true,
			},
			Name: uniqName,
			Type: "Microsoft.Storage/storageAccounts",
		}
		storageaccount.Tags = map[string]string{"id": storageAccount}
		return &storageaccount, region.create("", jsonutils.Marshal(storageaccount), &storageaccount)
	}
	return nil, err
}

func (region *SRegion) getStorageAccountID(storageAccount string) (*SStorageAccount, error) {
	accounts, err := region.ListStorageAccounts()
	if err != nil {
		return nil, errors.Wrapf(err, "ListStorageAccounts")
	}
	for i := 0; i < len(accounts); i++ {
		for k, v := range accounts[i].Tags {
			if k == "id" && v == storageAccount {
				accounts[i].region = region
				return &accounts[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetStorageAccountDetail(accountId string) (*SStorageAccount, error) {
	account := SStorageAccount{region: region}
	err := region.get(accountId, url.Values{}, &account)
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

func (region *SRegion) GetStorageAccountKey(accountId string) (string, error) {
	body, err := region.client.perform(accountId, "listKeys", nil)
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

func (region *SRegion) DeleteStorageAccount(accountId string) error {
	return region.del(accountId)
}

func (sa *SStorageAccount) GetAccountKey() (accountKey string, err error) {
	if len(sa.accountKey) > 0 {
		return sa.accountKey, nil
	}
	sa.accountKey, err = sa.region.GetStorageAccountKey(sa.ID)
	return sa.accountKey, err
}

func (sa *SStorageAccount) CreateContainer(containerName string) (*SContainer, error) {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("restype", "container")
	header := http.Header{}
	err = sa.region.put_storage_v2(accessKey, sa.Name, containerName, header, params, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "put_storage_v2")
	}
	container := SContainer{
		storageaccount: sa,
		Name:           containerName,
	}
	return &container, nil
}

func (sa *SStorageAccount) GetContainers() ([]SContainer, error) {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("comp", "list")
	params.Set("maxresults", "5000")
	ret := []SContainer{}
	for {
		part := struct {
			Containers []SContainer `xml:"Containers>Container"`
			NextMarker string       `xml:"NextMarker"`
		}{}
		err := sa.region.list_storage_v2(accessKey, sa.Name, "", params, &part)
		if err != nil {
			return nil, err
		}
		for i := range part.Containers {
			part.Containers[i].storageaccount = sa
			ret = append(ret, part.Containers[i])
		}
		if len(part.NextMarker) == 0 || len(part.Containers) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (sa *SStorageAccount) GetContainer(name string) (*SContainer, error) {
	containers, err := sa.GetContainers()
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

func (c *SContainer) ListAllFiles() ([]Blob, error) {
	blobs := make([]Blob, 0)
	var marker string
	for {
		result, err := c.ListFiles("", marker, "", 5000)
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

type BlobListResponse struct {
	Xmlns      string `xml:"xmlns,attr"`
	Prefix     string `xml:"Prefix"`
	Marker     string `xml:"Marker"`
	NextMarker string `xml:"NextMarker"`
	MaxResults int64  `xml:"MaxResults"`
	Blobs      []Blob `xml:"Blobs>Blob"`

	// BlobPrefix is used to traverse blobs as if it were a file system.
	// It is returned if ListBlobsParameters.Delimiter is specified.
	// The list here can be thought of as "folders" that may contain
	// other folders or blobs.
	BlobPrefixes []string `xml:"Blobs>BlobPrefix>Name"`

	// Delimiter is used to traverse blobs as if it were a file system.
	// It is returned if ListBlobsParameters.Delimiter is specified.
	Delimiter string `xml:"Delimiter"`
}

type Blob struct {
	Name       string            `xml:"Name"`
	Properties BlobProperties    `xml:"Properties"`
	Metadata   map[string]string `xml:"Metadata"`
}

type BlobProperties struct {
	LastModified       string `xml:"Last-Modified"`
	Etag               string `xml:"Etag"`
	ContentMD5         string `xml:"Content-MD5" header:"x-ms-blob-content-md5"`
	ContentLength      int64  `xml:"Content-Length"`
	ContentType        string `xml:"Content-Type" header:"x-ms-blob-content-type"`
	ContentEncoding    string `xml:"Content-Encoding" header:"x-ms-blob-content-encoding"`
	CacheControl       string `xml:"Cache-Control" header:"x-ms-blob-cache-control"`
	ContentLanguage    string `xml:"Cache-Language" header:"x-ms-blob-content-language"`
	ContentDisposition string `xml:"Content-Disposition" header:"x-ms-blob-content-disposition"`
	//BlobType              BlobType    `xml:"BlobType"`
	SequenceNumber int64  `xml:"x-ms-blob-sequence-number"`
	CopyID         string `xml:"CopyId"`
	CopyStatus     string `xml:"CopyStatus"`
	CopySource     string `xml:"CopySource"`
	CopyProgress   string `xml:"CopyProgress"`
	//CopyCompletionTime    TimeRFC1123 `xml:"CopyCompletionTime"`
	CopyStatusDescription string `xml:"CopyStatusDescription"`
	LeaseStatus           string `xml:"LeaseStatus"`
	LeaseState            string `xml:"LeaseState"`
	LeaseDuration         string `xml:"LeaseDuration"`
	ServerEncrypted       bool   `xml:"ServerEncrypted"`
	IncrementalCopy       bool   `xml:"IncrementalCopy"`
}

func (c *SContainer) ListFiles(prefix string, marker string, delimiter string, maxCount int) (*BlobListResponse, error) {
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("comp", "list")
	params.Set("restype", "container")
	if maxCount > 0 {
		params.Set("maxresults", strconv.Itoa(maxCount))
	}
	if len(prefix) > 0 {
		params.Set("prefix", prefix)
	}
	if len(marker) > 0 {
		params.Set("marker", marker)
	}
	if len(delimiter) > 0 {
		params.Set("delimiter", delimiter)
	}
	result := &BlobListResponse{}
	err = c.storageaccount.region.client.list_storage_v2(accessKey, c.storageaccount.Name, c.Name, params, result)
	if err != nil {
		return nil, errors.Wrap(err, "list_storage_v2")
	}
	return result, nil
}

func (c *SContainer) Delete(key string) error {
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}
	header := http.Header{}
	params := url.Values{}
	params.Set("restype", "container")
	err = c.storageaccount.region.client.delete_storage_v2(accessKey, c.storageaccount.Name, key, header, params)
	if err != nil {
		return errors.Wrap(err, "delete_storage_v2")
	}
	return nil
}

func setBlobRefMeta(meta http.Header, output http.Header) http.Header {
	for k, v := range meta {
		if len(v) == 0 || len(v[0]) == 0 {
			continue
		}
		switch http.CanonicalHeaderKey(k) {
		case cloudprovider.META_HEADER_CACHE_CONTROL:
			output.Set("x-ms-blob-cache-control", v[0])
		case cloudprovider.META_HEADER_CONTENT_TYPE:
			output.Set("x-ms-blob-content-type", v[0])
		case cloudprovider.META_HEADER_CONTENT_MD5:
			output.Set("x-ms-blob-content-md5", v[0])
		case cloudprovider.META_HEADER_CONTENT_ENCODING:
			output.Set("x-ms-blob-content-encoding", v[0])
		case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
			output.Set("x-ms-blob-content-language", v[0])
		case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
			output.Set("x-ms-blob-content-disposition", v[0])
		default:
			output.Set(fmt.Sprintf("x-ms-meta-%s", k), v[0])
		}
	}
	return output
}

func (c *SContainer) UploadStream(key string, reader io.Reader, meta http.Header) error {
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}
	header := http.Header{}
	header.Set("x-ms-blob-type", "BlockBlob")
	header.Set("Content-Length", "0")

	var n int64
	if reader != nil {
		type lener interface {
			Len() int
		}
		// TODO(rjeczalik): handle io.ReadSeeker, in case blob is *os.File etc.
		if l, ok := reader.(lener); ok {
			n = int64(l.Len())
		} else {
			var buf bytes.Buffer
			n, err = io.Copy(&buf, reader)
			if err != nil {
				return err
			}
			reader = &buf
		}
		header.Set("Content-Length", strconv.FormatInt(n, 10))
	}

	header = setBlobRefMeta(meta, header)

	file := fmt.Sprintf("%s/%s", c.Name, key)
	err = c.storageaccount.region.client.put_storage_v2(accessKey, c.storageaccount.Name, file, header, nil, reader, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}
	return nil
}

func (c *SContainer) SignUrl(method string, key string, expire time.Duration) (string, error) {
	format := "2006-01-02T15:04:05.000000Z"

	permission := []string{}
	switch method {
	case "GET":
		permission = append(permission, "r")
	case "PUT":
		permission = append(permission, "r")
		permission = append(permission, "w")
		permission = append(permission, "c")
		permission = append(permission, "a")
	case "DELETE":
		permission = append(permission, "r")
		permission = append(permission, "w")
		permission = append(permission, "d")
	}

	body := map[string]interface{}{
		"signedServices":      "b",
		"signedResourceTypes": "co",
		"signedPermission":    strings.Join(permission, ""),
		"signedProtocol":      "https,http",
		"signedStart":         time.Now().UTC().Format(format),
		"signedExpiry":        time.Now().UTC().Add(expire).Format(format),
	}
	ret, err := c.storageaccount.region.perform(c.storageaccount.ID, "ListAccountSas", jsonutils.Marshal(body))
	if err != nil {
		return "", errors.Wrap(err, "perform")
	}
	sasToken, err := ret.GetString("accountSasToken")
	if err != nil {
		return "", err
	}

	domain := fmt.Sprintf(azServices[SERVICE_STORAGE][c.storageaccount.region.client.envName], c.storageaccount.Name)
	return fmt.Sprintf("%s/%s/%s?%s", domain, c.Name, key, sasToken), nil
}

func (c *SContainer) UploadFile(filePath string, callback func(progress float32)) (string, error) {
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return "", errors.Wrap(err, "GetAccountKey")
	}

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

	contentLength := diskStream.GetSize()
	if contentLength%512 != 0 {
		return "", errors.Errorf("Content length %d must be aligned to a 512-byte boundary", contentLength)
	}

	params := url.Values{}
	headers := http.Header{}
	headers.Set("x-ms-blob-type", "PageBlob")
	headers.Set("x-ms-blob-content-length", fmt.Sprintf("%v", contentLength))
	headers.Set("x-ms-blob-sequence-number", fmt.Sprintf("%v", 0))
	fileName := fmt.Sprintf("%s/%s", c.Name, blobName)
	err = c.storageaccount.region.client.put_storage_v2(accessKey, c.storageaccount.Name, fileName, headers, params, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "put_storage_v2")
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
		StorageAccount:        c.storageaccount,
		VhdStream:             diskStream,
		UploadableRanges:      uploadableRanges,
		AlreadyProcessedBytes: common.TotalRangeLength(rangesToSkip),
		AccessKey:             accessKey,
		ContainerName:         c.Name,
		BlobName:              blobName,
		Parallelism:           3,
		Resume:                false,
		MD5Hash:               []byte(""), //localMetaData.FileMetaData.MD5Hash,
	}

	if err := Upload(cxt, callback); err != nil {
		return "", err
	}

	domain := fmt.Sprintf(azServices[SERVICE_STORAGE][c.storageaccount.region.client.envName], c.storageaccount.Name)

	return fmt.Sprintf("%s/%s/%s", domain, c.Name, blobName), nil
}

type SignedIdentifiers struct {
	SignedIdentifiers []struct {
		Id           string `xml:"Id"`
		AccessPolicy struct {
			Start      time.Time `xml:"Start"`
			Expiry     time.Time `xml:"Expiry"`
			Permission string    `xml:"Permission"`
		} `xml:"AccessPolicy"`
	} `xml:"SignedIdentifiers>SignedIdentifier"`
}

func (c *SContainer) getAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return acl
	}
	ret := SignedIdentifiers{}
	params := url.Values{}
	params.Set("comp", "acl")
	params.Set("restype", "container")
	err = c.storageaccount.region.list_storage_v2(accessKey, c.storageaccount.Name, c.Name, params, &ret)
	if err != nil {
		return acl
	}
	return acl
}

func (c *SContainer) setAcl(aclStr cloudprovider.TBucketACLType) error {
	accessKey, err := c.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("comp", "acl")
	params.Set("restype", "container")
	access := SignedIdentifiers{}
	header := http.Header{}
	header.Set("x-ms-blob-public-access", "container")

	body, length, err := xmlMarshal(access)
	if err != nil {
		return errors.Wrap(err, "xmlMarshal")
	}
	header.Set("Content-Length", strconv.Itoa(length))

	err = c.storageaccount.region.put_storage_v2(accessKey, c.storageaccount.Name, c.Name, header, params, body, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}
	return nil
}

func (sa *SStorageAccount) UploadFile(containerName string, filePath string, callback func(progress float32)) (string, error) {
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	return container.UploadFile(filePath, callback)
}

func (sa *SStorageAccount) getOrCreateContainer(containerName string, create bool) (*SContainer, error) {
	containers, err := sa.GetContainers()
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
		container, err = sa.CreateContainer(containerName)
		if err != nil {
			return nil, errors.Wrap(err, "CreateContainer")
		}
	}
	return container, nil
}

func (sa *SStorageAccount) UploadStream(containerName string, key string, reader io.Reader, meta http.Header) error {
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "getOrCreateContainer")
	}
	return container.UploadStream(key, reader, meta)
}

func (sa *SStorageAccount) MaxPartSizeBytes() int64 {
	return 100 * 1000 * 1000
}

func (sa *SStorageAccount) MaxPartCount() int {
	return 50000
}

func (sa *SStorageAccount) GetTags() (map[string]string, error) {
	return sa.Tags, nil
}

func (sa *SStorageAccount) GetProjectId() string {
	return getResourceGroup(sa.ID)
}

func (sa *SStorageAccount) GetGlobalId() string {
	return sa.Name
}

func (sa *SStorageAccount) GetName() string {
	return sa.Name
}

func (sa *SStorageAccount) GetLocation() string {
	return sa.Location
}

func (sa *SStorageAccount) GetIRegion() cloudprovider.ICloudRegion {
	return sa.region
}

func (sa *SStorageAccount) GetCreatedAt() time.Time {
	return time.Time{}
}

func (sa *SStorageAccount) GetStorageClass() string {
	return sa.Sku.Tier
}

// get the common ACL of all containers
func (sa *SStorageAccount) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	containers, err := sa.GetContainers()
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

func (sa *SStorageAccount) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	containers, err := sa.GetContainers()
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

func (sa *SStorageAccount) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	primary := sa.Properties.PrimaryEndpoints.getUrls("")
	secondary := sa.Properties.SecondaryEndpoints.getUrls("secondary")
	if len(secondary) > 0 {
		primary = append(primary, secondary...)
	}
	return primary
}

func (sa *SStorageAccount) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(sa)
	return stats
}

func (sa *SStorageAccount) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	containers, err := sa.GetContainers()
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
				oResult, err := container.ListFiles(subPrefix, subMarker, delimiter, maxCount)
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
						},
					}
					o.LastModified, _ = timeutils.ParseTimeStr(blob.Properties.LastModified)
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
		return "", "", errors.Wrap(cloudprovider.ErrForbidden, "cannot put object to root")
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

func (sa *SStorageAccount) PutObject(ctx context.Context, key string, reader io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	containerName, blob, err := splitKey(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	if len(blob) > 0 {
		// put blob
		err = sa.UploadStream(containerName, blob, reader, meta)
		if err != nil {
			return errors.Wrap(err, "UploadStream")
		}
	} else {
		// create container
		if sizeBytes > 0 {
			return errors.Wrap(cloudprovider.ErrForbidden, "not allow to create blob outsize of container")
		}
		_, err = sa.getOrCreateContainer(containerName, true)
		if err != nil {
			return errors.Wrap(err, "getOrCreateContainer")
		}
	}
	return nil
}

func (sa *SStorageAccount) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}

	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return "", errors.Wrap(err, "GetAccountKey")
	}
	header := http.Header{}
	header.Set("x-ms-blob-type", "BlockBlob")
	header = setBlobRefMeta(meta, header)
	params := url.Values{}
	//params.Set("comp", "block")
	header.Set("Content-Length", fmt.Sprintf("%v", 0))
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, key, header, params, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "put_storage_v2")
	}

	header = http.Header{}
	header.Set("x-ms-lease-action", "acquire")
	header.Set("x-ms-lease-duration", "-1")
	params = url.Values{}
	params.Set("comp", "lease")
	header, err = container.storageaccount.region.client.put_header_storage_v2(accessKey, container.storageaccount.Name, key, header, params)
	if err != nil {
		return "", errors.Wrap(err, "put_storage_v2")
	}
	uploadId := header.Get("x-ms-lease-id")
	if len(uploadId) == 0 {
		return "", errors.Error("lease id not returned")
	}
	return uploadId, nil
}

func partIndex2BlockId(partIndex int) string {
	return base64.URLEncoding.EncodeToString([]byte(strconv.FormatInt(int64(partIndex), 10)))
}

func (sa *SStorageAccount) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}

	blockId := partIndex2BlockId(partIndex)
	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return "", errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("comp", "block")
	params.Set("blockid", blockId)
	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%v", partSize))
	header.Set("x-ms-lease-id", uploadId)
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, key, header, params, input, nil)
	if err != nil {
		return "", errors.Wrap(err, "put_storage_v2")
	}
	return blockId, nil
}

type Block struct {
	ID     string
	Status string
}

func prepareBlockListRequest(blocks []Block) string {
	s := `<?xml version="1.0" encoding="utf-8"?><BlockList>`
	for _, v := range blocks {
		s += fmt.Sprintf("<%s>%s</%s>", v.Status, v.ID, v.Status)
	}
	s += `</BlockList>`
	return s
}

func (sa *SStorageAccount) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, blockIds []string) error {
	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "getOrCreateContainer")
	}

	blocks := make([]Block, len(blockIds))
	for i := range blockIds {
		blocks[i] = Block{
			ID:     blockIds[i],
			Status: "Latest",
		}
	}

	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}

	params := url.Values{}
	params.Set("comp", "blocklist")
	blockListXML := prepareBlockListRequest(blocks)
	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%v", len(blockListXML)))
	header.Set("x-ms-lease-id", uploadId)
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, key, header, params, strings.NewReader(blockListXML), nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}

	params = url.Values{}
	params.Set("comp", "lease")
	header = http.Header{}
	header.Set("x-ms-lease-action", "release")
	header.Set("x-ms-lease-id", uploadId)
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, key, header, params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}

	return nil
}

func (sa *SStorageAccount) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, false)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "getOrCreateContainer")
	}

	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}

	params := url.Values{}
	params.Set("comp", "lease")
	header := http.Header{}
	header.Set("x-ms-lease-action", "release")
	header.Set("x-ms-lease-id", uploadId)
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, key, header, params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}

	params = url.Values{}
	params.Set("comp", "lease")
	header.Set("x-ms-delete-snapshots", "include")
	err = container.storageaccount.region.client.delete_storage_v2(accessKey, container.storageaccount.Name, key, header, params)
	if err != nil {
		return errors.Wrap(err, "delete_storage_v2")
	}

	return nil
}

func (sa *SStorageAccount) DeleteObject(ctx context.Context, key string) error {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("comp", "delete")
	header := http.Header{}
	header.Set("x-ms-delete-snapshots", "include")
	err = sa.region.client.delete_storage_v2(accessKey, sa.Name, key, header, params)
	if err != nil {
		return errors.Wrap(err, "delete_storage_v2")
	}
	return nil
}

func (sa *SStorageAccount) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	containerName, blob, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, false)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}
	return container.SignUrl(method, blob, expire)
}

func (sa *SStorageAccount) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	srcIBucket, err := sa.region.GetIBucketByName(srcBucket)
	if err != nil {
		return errors.Wrap(err, "GetIBucketByName")
	}
	srcAccount := srcIBucket.(*SStorageAccount)

	srcDomain := fmt.Sprintf(azServices[SERVICE_STORAGE][srcAccount.region.client.envName], srcAccount.Name)
	srcUrl := fmt.Sprintf("%s/%s", srcDomain, srcKey)

	containerName, _, err := splitKeyAndBlob(destKey)
	if err != nil {
		return errors.Wrap(err, "dest splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return errors.Wrap(err, "dest getOrCreateContainer")
	}

	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}

	params := url.Values{}
	params.Set("comp", "copy")
	header := http.Header{}
	header.Set("x-ms-copy-source", srcUrl)
	header = setBlobRefMeta(meta, header)
	err = container.storageaccount.region.client.put_storage_v2(accessKey, container.storageaccount.Name, destKey, header, params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}
	return nil
}

func (sa *SStorageAccount) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return nil, errors.Wrap(err, "splitKey")
	}
	container, err := sa.getOrCreateContainer(containerName, false)
	if err != nil {
		return nil, errors.Wrap(err, "getOrCreateContainer")
	}

	accessKey, err := container.storageaccount.GetAccountKey()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	header := http.Header{}
	header.Set("x-ms-range", fmt.Sprintf("bytes=%d-%d", rangeOpt.Start, rangeOpt.End))

	body, err := container.storageaccount.region.get_body_storage_v2(accessKey, container.storageaccount.Name, key, header, params)
	if err != nil {
		return nil, errors.Wrap(err, "get_body_storage_v2")
	}
	return io.NopCloser(body), nil
}

func (sa *SStorageAccount) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	srcIBucket, err := sa.region.GetIBucketByName(srcBucket)
	if err != nil {
		return "", errors.Wrap(err, "GetIBucketByName")
	}
	srcAccount := srcIBucket.(*SStorageAccount)
	srcDomain := fmt.Sprintf(azServices[SERVICE_STORAGE][srcAccount.region.client.envName], srcAccount.Name)
	srcUrl := fmt.Sprintf("%s/%s", srcDomain, srcKey)
	srcContName, _, err := splitKeyAndBlob(srcKey)
	if err != nil {
		return "", errors.Wrap(err, "src splitKey")
	}
	_, err = srcAccount.getOrCreateContainer(srcContName, false)
	if err != nil {
		return "", errors.Wrap(err, "src getOrCreateContainer")
	}

	containerName, _, err := splitKeyAndBlob(key)
	if err != nil {
		return "", errors.Wrap(err, "splitKey")
	}
	_, err = sa.getOrCreateContainer(containerName, true)
	if err != nil {
		return "", errors.Wrap(err, "getOrCreateContainer")
	}

	blockId := partIndex2BlockId(partIndex)

	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return "", errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("comp", "block")
	params.Set("blockid", blockId)
	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%v", 0))
	header.Set("x-ms-copy-source", srcUrl)
	header.Set("x-ms-lease-id", uploadId)
	header.Set("x-ms-source-range", fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1))
	err = sa.region.client.put_storage_v2(accessKey, sa.Name, key, header, params, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "put_storage_v2")
	}

	return blockId, nil
}

func (sa *SStorageAccount) DeleteCORS() error {
	return sa.SetCORS([]cloudprovider.SBucketCORSRule{})
}

type ServiceProperties struct {
	Cors *Cors
}

func (sa *SStorageAccount) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return nil, errors.Wrap(err, "GetAccountKey")
	}
	params := url.Values{}
	params.Set("restype", "service")
	params.Set("comp", "properties")

	ret := &ServiceProperties{}

	err = sa.region.client.list_storage_v2(accessKey, sa.Name, "", params, ret)
	if err != nil {
		return nil, errors.Wrap(err, "list_storage_v2")
	}

	result := []cloudprovider.SBucketCORSRule{}
	for i := range ret.Cors.CorsRule {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: strings.Split(ret.Cors.CorsRule[i].AllowedOrigins, ","),
			AllowedMethods: strings.Split(ret.Cors.CorsRule[i].AllowedMethods, ","),
			AllowedHeaders: strings.Split(ret.Cors.CorsRule[i].AllowedHeaders, ","),
			MaxAgeSeconds:  ret.Cors.CorsRule[i].MaxAgeInSeconds,
			ExposeHeaders:  strings.Split(ret.Cors.CorsRule[i].ExposedHeaders, ","),
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

type CorsRule struct {
	AllowedOrigins  string
	AllowedMethods  string
	MaxAgeInSeconds int
	ExposedHeaders  string
	AllowedHeaders  string
}

type Cors struct {
	CorsRule []CorsRule
}

func xmlMarshal(v interface{}) (io.Reader, int, error) {
	b, err := xml.Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(b), len(b), nil
}

func (sa *SStorageAccount) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	corsRoles := []CorsRule{}
	for _, rule := range rules {
		corsRoles = append(corsRoles, CorsRule{
			AllowedOrigins:  strings.Join(rule.AllowedOrigins, ","),
			AllowedMethods:  strings.Join(rule.AllowedMethods, ","),
			AllowedHeaders:  strings.Join(rule.AllowedHeaders, ","),
			MaxAgeInSeconds: rule.MaxAgeSeconds,
			ExposedHeaders:  strings.Join(rule.ExposeHeaders, ","),
		})
	}
	cors := &Cors{
		CorsRule: corsRoles,
	}

	params := url.Values{}
	params.Set("restype", "service")
	params.Set("comp", "properties")

	type StorageServiceProperties struct {
		Cors *Cors
	}
	input := StorageServiceProperties{
		Cors: cors,
	}
	body, length, err := xmlMarshal(input)
	if err != nil {
		return err
	}

	accessKey, err := sa.GetAccountKey()
	if err != nil {
		return errors.Wrap(err, "GetAccountKey")
	}

	headers := http.Header{}
	headers.Set("Content-Length", fmt.Sprintf("%v", length))
	err = sa.region.client.put_storage_v2(accessKey, sa.Name, "", headers, params, body, nil)
	if err != nil {
		return errors.Wrap(err, "put_storage_v2")
	}
	return nil
}
