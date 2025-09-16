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

package google

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLifecycleRuleAction struct {
	Type string
}

type SLifecycleRuleCondition struct {
	Age int
}

type SLifecycleRule struct {
	Action    SLifecycleRuleAction
	Condition SLifecycleRuleCondition
}

type SBucketPolicyOnly struct {
	Enabled bool
}

type SUniformBucketLevelAccess struct {
	Enabled bool
}

type SIamConfiguration struct {
	BucketPolicyOnly         SBucketPolicyOnly
	UniformBucketLevelAccess SUniformBucketLevelAccess
}

type SLifecycle struct {
	Rule []SLifecycleRule
}

type SBucket struct {
	multicloud.SBaseBucket
	GoogleTags

	region *SRegion

	Kind             string
	SelfLink         string
	Name             string
	ProjectNumber    string
	Metageneration   string
	Location         string
	StorageClass     string
	Etag             string
	TimeCreated      time.Time
	Updated          time.Time
	Lifecycle        SLifecycle
	IamConfiguration SIamConfiguration
	LocationType     string
}

func (b *SBucket) GetProjectId() string {
	return b.region.GetProjectId()
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	iam, err := b.region.GetBucketIam(b.Name)
	if err != nil {
		return cloudprovider.ACLUnknown
	}
	acl := cloudprovider.ACLPrivate
	allUsers := []SBucketBinding{}
	allAuthUsers := []SBucketBinding{}
	for _, binding := range iam.Bindings {
		if utils.IsInStringArray("allUsers", binding.Members) {
			allUsers = append(allUsers, binding)
		}
		if utils.IsInStringArray("allAuthenticatedUsers", binding.Members) {
			allAuthUsers = append(allAuthUsers, binding)
		}
	}

	for _, binding := range allUsers {
		switch binding.Role {
		case "roles/storage.admin", "roles/storage.objectAdmin":
			acl = cloudprovider.ACLPublicReadWrite
		case "roles/storage.objectViewer":
			if acl != cloudprovider.ACLPublicReadWrite {
				acl = cloudprovider.ACLPublicRead
			}
		}
	}

	for _, binding := range allAuthUsers {
		switch binding.Role {
		case "roles/storage.admin", "roles/storage.objectAdmin", "roles/storage.objectViewer":
			acl = cloudprovider.ACLAuthRead
		}
	}
	return acl
}

func (region *SRegion) SetBucketAcl(bucket string, acl cloudprovider.TBucketACLType) error {
	iam, err := region.GetBucketIam(bucket)
	if err != nil {
		return errors.Wrap(err, "GetBucketIam")
	}
	bindings := []SBucketBinding{}
	for _, binding := range iam.Bindings {
		if !utils.IsInStringArray(string(storage.AllUsers), binding.Members) && !utils.IsInStringArray(string(storage.AllAuthenticatedUsers), binding.Members) {
			bindings = append(bindings, binding)
		}
	}
	switch acl {
	case cloudprovider.ACLPrivate:
		if len(bindings) == len(iam.Bindings) {
			return nil
		}
	case cloudprovider.ACLAuthRead:
		bindings = append(bindings, SBucketBinding{
			Role:    "roles/storage.objectViewer",
			Members: []string{"allAuthenticatedUsers"},
		})
	case cloudprovider.ACLPublicRead:
		bindings = append(bindings, SBucketBinding{
			Role:    "roles/storage.objectViewer",
			Members: []string{"allUsers"},
		})
	case cloudprovider.ACLPublicReadWrite:
		bindings = append(bindings, SBucketBinding{
			Role:    "roles/storage.objectAdmin",
			Members: []string{"allUsers"},
		})
	default:
		return fmt.Errorf("unknown acl %s", acl)
	}
	iam.Bindings = bindings
	_, err = region.SetBucketIam(bucket, iam)
	if err != nil {
		return errors.Wrap(err, "SetBucketIam")
	}
	return nil

}

func (b *SBucket) SetAcl(acl cloudprovider.TBucketACLType) error {
	return b.region.SetBucketAcl(b.Name, acl)
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (b *SBucket) GetLocation() string {
	return strings.ToLower(b.Location)
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreatedAt() time.Time {
	return b.TimeCreated
}

func (b *SBucket) GetStorageClass() string {
	return b.StorageClass
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s", b.Name),
			Description: "bucket domain",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://www.googleapis.com/upload/storage/v1/b/%s/o", b.Name),
			Description: "object upload endpoint",
		},
		{
			Url:         fmt.Sprintf("https://www.googleapis.com/batch/storage/v1/b/%s", b.Name),
			Description: "batch operation",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	resource := fmt.Sprintf("b/%s/o?uploadType=resumable&upload_id=%s", b.Name, uploadId)
	return b.region.client.storageAbortUpload(resource)
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	resource := fmt.Sprintf("b/%s/o/%s", b.Name, url.PathEscape(key))
	err := b.region.StorageGet(resource, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to get object %s", key)
	}
	return nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	resource := fmt.Sprintf("b/%s/o/%s", srcBucket, url.PathEscape(srcKey))
	action := fmt.Sprintf("copyTo/b/%s/o/%s", b.Name, url.PathEscape(destKey))
	err := b.region.StorageDo(resource, action, nil, nil)
	if err != nil {
		return errors.Wrap(err, "CopyObject")
	}
	err = b.region.SetObjectAcl(b.Name, destKey, cannedAcl)
	if err != nil {
		return errors.Wrapf(err, "AddObjectAcl(%s)", cannedAcl)
	}
	err = b.region.SetObjectMeta(b.Name, destKey, meta)
	if err != nil {
		return errors.Wrap(err, "SetObjectMeta")
	}
	return nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (region *SRegion) DeleteObject(bucket, key string) error {
	resource := fmt.Sprintf("b/%s/o/%s", bucket, url.PathEscape(key))
	return region.StorageDelete(resource)
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	return b.region.DeleteObject(b.Name, key)
}

func (region *SRegion) DownloadObjectRange(bucket, object string, start, end int64) (io.ReadCloser, error) {
	resource := fmt.Sprintf("b/%s/o/%s?alt=media", bucket, url.PathEscape(object))
	header := http.Header{}
	if start <= 0 {
		if end > 0 {
			header.Set("Range", fmt.Sprintf("bytes=0-%d", end))
		} else {
			header.Set("Range", "bytes=-1")
		}
	} else {
		if end > start {
			header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
		} else {
			header.Set("Range", fmt.Sprintf("bytes=%d-", start))
		}
	}
	return region.client.storageDownload(resource, header)
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	return b.region.DownloadObjectRange(b.Name, key, rangeOpt.Start, rangeOpt.End)
}

func (region *SRegion) SingedUrl(bucket, key string, method string, expire time.Duration) (string, error) {
	if expire > time.Hour*24*7 {
		return "", fmt.Errorf(`Expiration Time can\'t be longer than 604800 seconds (7 days)`)
	}
	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         method,
		GoogleAccessID: region.client.clientEmail,
		PrivateKey:     []byte(region.client.privateKey),
		Expires:        time.Now().Add(expire),
	}
	switch method {
	case "GET":
	case "PUT":
		opts.Headers = []string{"Content-Type:application/octet-stream"}
	default:
		return "", errors.Wrapf(cloudprovider.ErrNotSupported, "Not support method %s", method)
	}
	return storage.SignedURL(bucket, key, opts)

}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	return b.region.SingedUrl(b.Name, key, method, expire)
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	objs, err := b.region.GetObjects(b.Name, prefix, marker, delimiter, maxCount)
	if err != nil {
		return result, errors.Wrap(err, "GetObjects")
	}
	result.NextMarker = objs.NextPageToken
	log.Errorf("obj count: %d", len(objs.Items))
	result.Objects = []cloudprovider.ICloudObject{}
	result.CommonPrefixes = []cloudprovider.ICloudObject{}
	for i := range objs.Items {
		if strings.HasSuffix(objs.Items[i].Name, "/") {
			continue
		}
		objs.Items[i].bucket = b
		result.Objects = append(result.Objects, &objs.Items[i])
	}
	for i := range objs.Prefixes {
		obj := &SObject{
			bucket: b,
			Name:   objs.Prefixes[i],
		}
		result.CommonPrefixes = append(result.CommonPrefixes, obj)
	}
	return result, nil
}

func (region *SRegion) NewMultipartUpload(bucket, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	body := map[string]string{"name": key}
	if len(storageClassStr) > 0 {
		body["storageClass"] = storageClassStr
	}
	for k := range meta {
		switch k {
		case cloudprovider.META_HEADER_CONTENT_TYPE:
			body["contentType"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_ENCODING:
			body["contentEncoding"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
			body["contentDisposition"] = meta.Get(k)
		case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
			body["contentLanguage"] = meta.Get(k)
		case cloudprovider.META_HEADER_CACHE_CONTROL:
			body["cacheControl"] = meta.Get(k)
		default:
			body[fmt.Sprintf("metadata.%s", k)] = meta.Get(k)
		}
	}
	switch cannedAcl {
	case cloudprovider.ACLPrivate:
	case cloudprovider.ACLAuthRead:
		body["predefinedAcl"] = "authenticatedRead"
	case cloudprovider.ACLPublicRead:
		body["predefinedAcl"] = "publicRead"
	case cloudprovider.ACLPublicReadWrite:
		return "", cloudprovider.ErrNotSupported
	}
	resource := fmt.Sprintf("b/%s/o?uploadType=resumable", bucket)
	input := strings.NewReader(jsonutils.Marshal(body).String())
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=UTF-8")
	header.Set("Content-Length", fmt.Sprintf("%d", input.Len()))
	resp, err := region.client.storageUpload(resource, header, input)
	if err != nil {
		return "", errors.Wrap(err, "storageUpload")
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	query, err := url.ParseQuery(location)
	if err != nil {
		return "", errors.Wrapf(err, "url.ParseQuery(%s)", location)
	}
	return query.Get("upload_id"), nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	return b.region.NewMultipartUpload(b.Name, key, cannedAcl, storageClassStr, meta)
}

func (region *SRegion) UploadPart(bucket, uploadId string, partIndex int, offset int64, part io.Reader, partSize int64, totalSize int64) error {
	resource := fmt.Sprintf("b/%s/o?uploadType=resumable&upload_id=%s", bucket, uploadId)
	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%d", partSize))
	header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, offset+partSize-1, totalSize))
	resp, err := region.client.storageUploadPart(resource, header, part)
	if err != nil {
		return errors.Wrap(err, "storageUploadPart")
	}
	if resp.StatusCode >= 500 {
		content, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("status code: %d %s", resp.StatusCode, content)
	}
	defer resp.Body.Close()
	return nil
}

func (region *SRegion) CheckUploadRange(bucket string, uploadId string) error {
	resource := fmt.Sprintf("b/%s/o?uploadType=resumable&upload_id=%s", bucket, uploadId)
	header := http.Header{}
	header.Set("Content-Range", "bytes */*")
	resp, err := region.client.storageUploadPart(resource, header, nil)
	if err != nil {
		return errors.Wrap(err, "storageUploadPart")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "ReadAll")
	}
	fmt.Println("content: ", string(content))
	for k, v := range resp.Header {
		fmt.Println("k: ", k, "v: ", v)
	}
	fmt.Println("status code: ", resp.StatusCode)
	return nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	return "", b.region.UploadPart(b.Name, uploadId, partIndex, offset, part, partSize, totalSize)
}

func (b *SBucket) PutObject(ctx context.Context, key string, body io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	return b.region.PutObject(b.Name, key, body, sizeBytes, cannedAcl, meta)
}

func (region *SRegion) GetBucket(name string) (*SBucket, error) {
	resource := "b/" + name
	bucket := &SBucket{}
	err := region.StorageGet(resource, bucket)
	if err != nil {
		return nil, errors.Wrap(err, "GetBucket")
	}
	return bucket, nil
}

func (region *SRegion) GetBuckets(maxResults int, pageToken string) ([]SBucket, error) {
	buckets := []SBucket{}
	params := map[string]string{
		"project": region.GetProjectId(),
	}
	err := region.StorageList("b", params, maxResults, pageToken, &buckets)
	if err != nil {
		return nil, err
	}
	return buckets, nil
}

func (region *SRegion) CreateBucket(name string, storageClass string, acl cloudprovider.TBucketACLType) (*SBucket, error) {
	body := map[string]interface{}{
		"name":     name,
		"location": region.Name,
	}
	if len(storageClass) > 0 {
		body["storageClass"] = storageClass
	}
	params := url.Values{}
	params.Set("predefinedDefaultObjectAcl", "private")
	switch acl {
	case cloudprovider.ACLPrivate, cloudprovider.ACLUnknown:
		params.Set("predefinedAcl", "private")
	case cloudprovider.ACLAuthRead:
		params.Set("predefinedAcl", "authenticatedRead")
	case cloudprovider.ACLPublicRead:
		params.Set("predefinedAcl", "publicRead")
	case cloudprovider.ACLPublicReadWrite:
		params.Set("predefinedAcl", "publicReadWrite")
	}
	params.Set("project", region.GetProjectId())
	bucket := &SBucket{}
	resource := fmt.Sprintf("b?%s", params.Encode())
	err := region.StorageInsert(resource, jsonutils.Marshal(body), bucket)
	if err != nil {
		return nil, err
	}
	return bucket, nil
}

func (region *SRegion) UploadObject(bucket string, params url.Values, header http.Header, input io.Reader) error {
	resource := fmt.Sprintf("b/%s/o", bucket)
	if len(params) > 0 {
		resource = fmt.Sprintf("%s?%s", resource, params.Encode())
	}
	resp, err := region.client.storageUpload(resource, header, input)
	if err != nil {
		return errors.Wrap(err, "storageUpload")
	}
	defer resp.Body.Close()
	return nil
}

func (region *SRegion) PutObject(bucket string, name string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, meta http.Header) error {
	params := url.Values{}
	params.Set("name", name)
	params.Set("uploadType", "media")
	header := http.Header{}
	header.Set("Content-Length", fmt.Sprintf("%v", sizeBytes))
	err := region.UploadObject(bucket, params, header, input)
	if err != nil {
		return errors.Wrap(err, "UploadObject")
	}
	err = region.SetObjectAcl(bucket, name, cannedAcl)
	if err != nil {
		return errors.Wrap(err, "SetObjectAcl")
	}
	return region.SetObjectMeta(bucket, name, meta)
}

func (region *SRegion) DeleteBucket(name string) error {
	return region.StorageDelete("b/" + name)
}

type SBucketCORSRule struct {
	Cors []SCORSDetails
}
type SCORSDetails struct {
	Origin         []string `json:"origin"`
	Method         []string `json:"method"`
	ResponseHeader []string `json:"responseHeader"`
	MaxAgeSeconds  int      `json:"maxAgeSeconds"`
}

func (bucket *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	res := []cloudprovider.SBucketCORSRule{}
	corss := SBucketCORSRule{}
	err := bucket.region.StorageGet(fmt.Sprintf("b/%s?fields=cors", bucket.Name), &corss)
	if err != nil {
		return nil, errors.Wrap(err, "StorageGet cors")
	}
	for _, cors := range corss.Cors {
		temp := cloudprovider.SBucketCORSRule{}
		temp.AllowedHeaders = cors.ResponseHeader
		temp.AllowedMethods = cors.Method
		temp.AllowedOrigins = cors.Origin
		temp.MaxAgeSeconds = cors.MaxAgeSeconds
		res = append(res, temp)
	}
	return res, nil
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	params := []map[string]interface{}{}
	for _, rule := range rules {
		params = append(params, map[string]interface{}{
			"origin":         rule.AllowedOrigins,
			"method":         rule.AllowedMethods,
			"responseHeader": rule.AllowedHeaders,
			"maxAgeSeconds":  rule.MaxAgeSeconds,
		})
	}
	return b.region.StoragePut(fmt.Sprintf("b/%s?fields=cors", b.Name), jsonutils.Marshal(map[string]interface{}{"cors": params}), nil)
}

func (b *SBucket) DeleteCORS() error {
	return b.region.StoragePut(fmt.Sprintf("b/%s?fields=cors", b.Name), nil, nil)
}
