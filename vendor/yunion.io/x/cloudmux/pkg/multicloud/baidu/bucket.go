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

package baidu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

type SBucket struct {
	multicloud.SBaseBucket
	multicloud.STagBase

	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
}

func (b *SBucket) GetId() string {
	return b.Name
}

func (b *SBucket) GetGlobalId() string {
	return b.Name
}

func (b *SBucket) GetName() string {
	return b.Name
}

func (b *SBucket) GetLocation() string {
	return b.Location
}

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	storageClass, err := b.region.GetBucketStorageClass(b.Name)
	if err != nil {
		return ""
	}
	return storageClass
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl, err := b.region.GetBucketAcl(b.Name)
	if err != nil {
		return cloudprovider.ACLUnknown
	}
	return acl.GetAcl()
}

func (b *SBucket) SetAcl(acl cloudprovider.TBucketACLType) error {
	return b.region.SetBucketAcl(b.Name, acl)
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s.bcebos.com", b.Name, b.region.GetId()),
			Description: "ExtranetEndpoint",
			Primary:     true,
		},
	}
}

func (b *SBucket) GetTags() (map[string]string, error) {
	params := url.Values{}
	params.Set("tagging", "")
	resp, err := b.region.bosRequest(httputils.GET, b.Name, "", params, http.Header{}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetTags")
	}
	_, body, err := httputils.ParseJSONResponse("", resp, nil, b.region.client.debug)
	if err != nil {
		return nil, errors.Wrap(err, "ParseJSONResponse")
	}
	ret := struct {
		Tag []struct {
			TagKey   string `json:"tagKey"`
			TagValue string `json:"tagValue"`
		} `json:"tag"`
	}{}
	err = body.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	res := map[string]string{}
	for _, tag := range ret.Tag {
		res[tag.TagKey] = tag.TagValue
	}
	return res, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	params := url.Values{}
	params.Set("tagging", "")
	_, err := b.region.bosRequest(httputils.DELETE, b.Name, "", params, http.Header{}, nil)
	if err != nil {
		return errors.Wrap(err, "DeleteTagging")
	}
	if len(tags) == 0 {
		return nil
	}
	input := []map[string]string{}
	for k, v := range tags {
		input = append(input, map[string]string{
			"tagKey":   k,
			"tagValue": v,
		})
	}
	body := strings.NewReader(jsonutils.Marshal(map[string]interface{}{"tags": input}).String())
	_, err = b.region.bosRequest(httputils.PUT, b.Name, "", params, http.Header{}, body)
	if err != nil {
		return errors.Wrap(err, "SetTags")
	}
	return nil
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stat, _ := cloudprovider.GetIBucketStats(b)
	return stat
}

func (b *SBucket) SetLimit(limit cloudprovider.SBucketStats) error {
	return b.region.SetBucketLimit(b.Name, limit)
}

func (region *SRegion) SetBucketLimit(bucketName string, limit cloudprovider.SBucketStats) error {
	params := url.Values{}
	params.Set("quota", "")
	body := map[string]interface{}{
		"maxObjectCount":       limit.ObjectCount,
		"maxCapacityMegaBytes": limit.SizeBytes,
	}
	_, err := region.bosUpdate(bucketName, "", params, body)
	return err
}

func (b *SBucket) LimitSupport() cloudprovider.SBucketStats {
	ret, err := b.region.GetBucketLimit(b.Name)
	if err != nil {
		return cloudprovider.SBucketStats{
			ObjectCount: -1,
			SizeBytes:   -1,
		}
	}
	return ret
}

func (region *SRegion) GetBucketLimit(bucketName string) (cloudprovider.SBucketStats, error) {
	params := url.Values{}
	params.Set("quota", "")
	resp, err := region.bosList(bucketName, "", params)
	if err != nil {
		return cloudprovider.SBucketStats{}, err
	}
	ret := struct {
		MaxObjectCount       int
		MaxCapacityMegaBytes int
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return cloudprovider.SBucketStats{
			ObjectCount: -1,
			SizeBytes:   -1,
		}, err
	}
	return cloudprovider.SBucketStats{
		ObjectCount: ret.MaxObjectCount,
		SizeBytes:   int64(ret.MaxCapacityMegaBytes),
	}, nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	header := http.Header{}
	if len(cannedAcl) > 0 {
		header.Set("x-bce-acl", string(cannedAcl))
	}
	if len(storageClassStr) > 0 {
		header.Set("x-bce-storage-class", storageClassStr)
	}

	if len(srcBucket) > 0 {
		header.Set("x-bce-copy-source", fmt.Sprintf("%s/%s", url.PathEscape(srcBucket), url.PathEscape(srcKey)))
	}

	for k := range meta {
		header.Set(k, meta.Get(k))
	}

	resp, err := b.region.bosRequest(httputils.PUT, b.Name, destKey, url.Values{}, header, nil)
	if err != nil {
		return errors.Wrapf(err, "CopyObject %s %s %s %s", b.Name, destKey, srcBucket, srcKey)
	}
	defer httputils.CloseResponse(resp)
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	header := http.Header{}
	if rangeOpt != nil {
		header.Set("Range", rangeOpt.String())
	}
	resp, err := b.region.bosRequest(httputils.GET, b.Name, key, url.Values{}, header, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetObject %s %s", b.Name, key)
	}
	return resp.Body, nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	return b.region.DeleteObject(b.Name, key)
}

func (region *SRegion) DeleteObject(bucketName, key string) error {
	resp, err := region.bosRequest(httputils.DELETE, bucketName, key, url.Values{}, http.Header{}, nil)
	if err != nil {
		return errors.Wrapf(err, "DeleteObject %s %s", bucketName, key)
	}
	httputils.CloseResponse(resp)
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	uri, err := url.Parse(fmt.Sprintf("https://%s.%s.bcebos.com/%s", b.Name, b.region.GetId(), url.PathEscape(key)))
	if err != nil {
		return "", errors.Wrapf(err, "Parse %s %s", b.Name, b.region.GetId())
	}
	header := http.Header{}
	sign, err := b.region.client._sign(uri, method, header, int(expire.Seconds()))
	if err != nil {
		return "", errors.Wrapf(err, "Sign %s %s", b.Name, key)
	}
	query := url.Values{}
	query.Set("authorization", sign)
	return fmt.Sprintf("%s?%s", uri.String(), query.Encode()), nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, input io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	header := http.Header{}
	if len(cannedAcl) > 0 {
		header.Set("x-bce-acl", string(cannedAcl))
	}
	if len(storageClassStr) > 0 {
		header.Set("x-bce-storage-class", storageClassStr)
	}
	for k := range meta {
		header.Set(fmt.Sprintf("x-bce-meta-%s", k), meta.Get(k))
	}
	if sizeBytes > 0 {
		header.Set("Content-Length", strconv.FormatInt(sizeBytes, 10))
	}

	resp, err := b.region.bosRequest(httputils.PUT, b.Name, key, url.Values{}, header, input)
	if err != nil {
		return errors.Wrapf(err, "PutObject %s %s", b.Name, key)
	}
	httputils.CloseResponse(resp)
	return nil
}

const (
	THRESHOLD_100_CONTINUE = 1 << 20
)

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	header := http.Header{}
	if len(cannedAcl) > 0 {
		header.Set("x-bce-acl", string(cannedAcl))
	}
	if len(storageClassStr) > 0 {
		header.Set("x-bce-storage-class", storageClassStr)
	}
	for k := range meta {
		header.Set(fmt.Sprintf("x-bce-meta-%s", k), meta.Get(k))
	}
	params := url.Values{}
	params.Set("uploads", "")
	resp, err := b.region.bosRequest(httputils.POST, b.Name, key, params, header, nil)
	if err != nil {
		return "", errors.Wrapf(err, "NewMultipartUpload %s %s", b.Name, key)
	}
	_, body, err := httputils.ParseJSONResponse("", resp, nil, b.region.client.debug)
	if err != nil {
		return "", errors.Wrapf(err, "ParseJSONResponse %s %s", b.Name, key)
	}
	return body.GetString("uploadId")
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	header := http.Header{}
	header.Set("Content-Length", strconv.FormatInt(partSize, 10))
	params := url.Values{}
	params.Set("uploadId", uploadId)
	params.Set("partNumber", strconv.Itoa(partIndex))
	if partSize > THRESHOLD_100_CONTINUE {
		header.Set("Expect", "100-continue")
	}
	resp, err := b.region.bosRequest(httputils.PUT, b.Name, key, params, header, input)
	if err != nil {
		return "", errors.Wrapf(err, "UploadPart %s %s %s", b.Name, key, uploadId)
	}
	defer httputils.CloseResponse(resp)
	return strings.Trim(resp.Header.Get("ETag"), "\""), nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	header := http.Header{}
	params := url.Values{}
	params.Set("uploadId", uploadId)
	parts := []map[string]string{}
	for i, etag := range partEtags {
		parts = append(parts, map[string]string{
			"partNumber": strconv.Itoa(i + 1),
			"eTag":       etag,
		})
	}
	body := strings.NewReader(jsonutils.Marshal(map[string]interface{}{"parts": parts}).String())
	resp, err := b.region.bosRequest(httputils.POST, b.Name, key, params, header, body)
	if err != nil {
		return errors.Wrapf(err, "CompleteMultipartUpload %s %s %s", b.Name, key, uploadId)
	}
	httputils.CloseResponse(resp)
	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	params := url.Values{}
	params.Set("uploadId", uploadId)
	resp, err := b.region.bosRequest(httputils.DELETE, b.Name, key, params, http.Header{}, nil)
	if err != nil {
		return errors.Wrapf(err, "AbortMultipartUpload %s %s %s", b.Name, key, uploadId)
	}
	httputils.CloseResponse(resp)
	return nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	header := http.Header{}
	params := url.Values{}
	params.Set("uploadId", uploadId)
	params.Set("partNumber", strconv.Itoa(partIndex))
	header.Set("x-bce-copy-source", fmt.Sprintf("%s/%s", url.PathEscape(srcBucket), url.PathEscape(srcKey)))
	if srcLength > 0 {
		header.Set("x-bce-copy-source-range", fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1))
	}
	resp, err := b.region.bosRequest(httputils.PUT, b.Name, key, params, header, nil)
	if err != nil {
		return "", errors.Wrapf(err, "CopyPart %s %s %s %s %d %d", b.Name, key, uploadId, srcBucket, srcOffset, srcLength)
	}
	_, body, err := httputils.ParseJSONResponse("", resp, nil, b.region.client.debug)
	if err != nil {
		return "", errors.Wrapf(err, "ParseJSONResponse %s %s %s", b.Name, key, uploadId)
	}
	return body.GetString("eTag")
}

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetProjectId() string {
	return ""
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	params := url.Values{}
	if len(delimiter) > 0 {
		params.Set("delimiter", delimiter)
	}
	if len(prefix) > 0 {
		params.Set("prefix", prefix)
	}
	if maxCount > 0 {
		params.Set("maxKeys", strconv.Itoa(maxCount))
	}
	if len(marker) > 0 {
		params.Set("marker", marker)
	}
	resp, err := b.region.bosList(b.Name, "", params)
	if err != nil {
		return cloudprovider.SListObjectResult{}, err
	}
	ret := struct {
		Name           string
		Prefix         string
		IsTruncated    bool
		Marker         string
		CommonPrefixes []struct {
			Prefix string
		}
		Contents []SObject
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return cloudprovider.SListObjectResult{}, err
	}
	result := cloudprovider.SListObjectResult{
		Objects:        make([]cloudprovider.ICloudObject, 0),
		CommonPrefixes: make([]cloudprovider.ICloudObject, 0),
	}
	for _, content := range ret.Contents {
		content.bucket = b
		result.Objects = append(result.Objects, &content)
	}
	for _, commonPrefix := range ret.CommonPrefixes {
		obj := &SObject{
			bucket: b,
			Key:    commonPrefix.Prefix,
		}
		result.CommonPrefixes = append(result.CommonPrefixes, obj)
	}
	result.IsTruncated = ret.IsTruncated
	result.NextMarker = ret.Marker
	return result, nil
}

func (region *SRegion) ListBuckets() ([]SBucket, error) {
	resp, err := region.bosList("", "/", nil)
	if err != nil {
		return nil, err
	}
	buckets := []SBucket{}
	err = resp.Unmarshal(&buckets, "buckets")
	if err != nil {
		return nil, err
	}
	return buckets, nil
}

func (region *SRegion) CreateBucket(name string, storageClassStr string, aclStr string) error {
	_, err := region.bosUpdate(name, "/", nil, nil)
	if err != nil {
		return err
	}
	if len(storageClassStr) > 0 {
		err = region.SetBucketStorageClass(name, storageClassStr)
		if err != nil {
			return err
		}
	}
	if len(aclStr) > 0 {
		err = region.SetBucketAcl(name, cloudprovider.TBucketACLType(aclStr))
		if err != nil {
			return err
		}
	}
	return nil
}

func (region *SRegion) DeleteBucket(name string) error {
	_, err := region.bosDelete(name, "/", nil)
	return err
}

func (region *SRegion) bosList(bucketName, res string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.bosList(region.GetId(), bucketName, res, params)
}

func (region *SRegion) bosDelete(bucketName, res string, params url.Values) (jsonutils.JSONObject, error) {
	return region.client.bosDelete(region.GetId(), bucketName, res, params)
}

func (region *SRegion) bosUpdate(bucketName, res string, params url.Values, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return region.client.bosUpdate(region.GetId(), bucketName, res, params, body)
}

func (region *SRegion) bosRequest(method httputils.THttpMethod, bucketName, res string, params url.Values, header http.Header, body io.Reader) (*http.Response, error) {
	return region.client.bosRequest(method, region.GetId(), bucketName, res, params, header, body)
}

func (region *SRegion) GetBucketStorageClass(bucketName string) (string, error) {
	params := url.Values{}
	params.Set("storageClass", "")
	resp, err := region.bosList(bucketName, "", params)
	if err != nil {
		return "", err
	}
	return resp.GetString("storageClass")
}

func (region *SRegion) SetBucketStorageClass(bucketName string, storageClass string) error {
	params := url.Values{}
	params.Set("storageClass", "")
	body := map[string]interface{}{
		"storageClass": storageClass,
	}
	_, err := region.bosUpdate(bucketName, "", params, body)
	return err
}

type SAccessControl struct {
	AccessControlList []SAccessControlList
	Owner             struct {
		Id string
	}
}

func (acl *SAccessControl) GetAcl() cloudprovider.TBucketACLType {
	aclType := cloudprovider.ACLUnknown
	switch len(acl.AccessControlList) {
	case 1:
		if acl.AccessControlList[0].Grantee[0].Id == acl.Owner.Id && acl.AccessControlList[0].Permission[0] == "FULL_CONTROL" {
			aclType = cloudprovider.ACLPrivate
		}
	case 2:
		isRead, isWrite := false, false
		for _, g := range acl.AccessControlList {
			if g.Grantee[0].Id == "*" {
				for _, permission := range g.Permission {
					if strings.EqualFold(permission, "READ") {
						isRead = true
					}
					if strings.EqualFold(permission, "WRITE") {
						isWrite = true
					}
				}
			}
		}
		if isRead && isWrite {
			aclType = cloudprovider.ACLPublicReadWrite
		} else if isRead {
			aclType = cloudprovider.ACLPublicRead
		}
	}
	return aclType
}

type SAccessControlList struct {
	Grantee []struct {
		Id string
	}
	Permission []string
}

func (region *SRegion) GetBucketAcl(bucketName string) (*SAccessControl, error) {
	params := url.Values{}
	params.Set("acl", "")
	resp, err := region.bosList(bucketName, "", params)
	if err != nil {
		return nil, err
	}
	ret := &SAccessControl{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) SetBucketAcl(bucketName string, acl cloudprovider.TBucketACLType) error {
	params := url.Values{}
	params.Set("acl", "")
	header := http.Header{}
	header.Set("x-bce-acl", string(acl))
	resp, err := region.bosRequest(httputils.PUT, bucketName, "", params, header, nil)
	if err != nil {
		return errors.Wrapf(err, "SetBucketAcl %s %s", bucketName, acl)
	}
	httputils.CloseResponse(resp)
	return nil
}
