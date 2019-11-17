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

package qcloud

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket

	region *SRegion

	Name       string
	Location   string
	CreateDate time.Time
}

func (b *SBucket) GetProjectId() string {
	return ""
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

func (b *SBucket) GetIRegion() cloudprovider.ICloudRegion {
	return b.region
}

func (b *SBucket) GetCreateAt() time.Time {
	return b.CreateDate
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

const (
	ACL_GROUP_URI_ALL_USERS  = "http://cam.qcloud.com/groups/global/AllUsers"
	ACL_GROUP_URI_AUTH_USERS = "http://cam.qcloud.com/groups/global/AuthenticatedUsers"
)

func cosAcl2CannedAcl(acls []cos.ACLGrant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == "" && acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if g.Grantee.URI == ACL_GROUP_URI_AUTH_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if g.Grantee.URI == ACL_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if g.Grantee.URI == ACL_GROUP_URI_ALL_USERS && g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		log.Errorf("GetCosClient fail %s", err)
		return acl
	}
	result, _, err := coscli.Bucket.GetACL(context.Background())
	if err != nil {
		log.Errorf("coscli.Bucket.GetACL fail %s", err)
		return acl
	}
	return cosAcl2CannedAcl(result.AccessControlList)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "b.region.GetCosClient")
	}
	opts := &cos.BucketPutACLOptions{}
	opts.Header = &cos.ACLHeaderOptions{}
	opts.Header.XCosACL = string(aclStr)
	_, err = coscli.Bucket.PutACL(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "PutACL")
	}
	return nil
}

func (b *SBucket) getFullName() string {
	return fmt.Sprintf("%s-%s", b.Name, b.region.client.AppID)
}

func (b *SBucket) getBucketUrlHost() string {
	return fmt.Sprintf("%s.%s", b.getFullName(), b.region.getCosEndpoint())
}

func (b *SBucket) getBucketUrl() string {
	return fmt.Sprintf("https://%s", b.getBucketUrlHost())
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         b.getBucketUrl(),
			Description: "bucket domain",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getCosEndpoint(), b.getFullName()),
			Description: "cos domain",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	return cloudprovider.GetIObjects(b, prefix, isRecursive)
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return result, errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.BucketGetOptions{}
	if len(prefix) > 0 {
		opts.Prefix = prefix
	}
	if len(marker) > 0 {
		opts.Marker = marker
	}
	if len(delimiter) > 0 {
		opts.Delimiter = delimiter
	}
	if maxCount > 0 {
		opts.MaxKeys = maxCount
	}
	oResult, _, err := coscli.Bucket.Get(context.Background(), opts)
	if err != nil {
		return result, errors.Wrap(err, "coscli.Bucket.Get")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		lastModified, _ := timeutils.ParseTimeStr(object.LastModified)
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          object.Key,
				SizeBytes:    int64(object.Size),
				ETag:         object.ETag,
				LastModified: lastModified,
				ContentType:  "",
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, 0)
		for _, commPrefix := range oResult.CommonPrefixes {
			obj := &SObject{
				bucket: b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{
					Key: commPrefix,
				},
			}
			result.CommonPrefixes = append(result.CommonPrefixes, obj)
		}
	}
	result.IsTruncated = oResult.IsTruncated
	result.NextMarker = oResult.NextMarker
	return result, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, reader io.Reader, sizeBytes int64, contType string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectPutOptions{
		ACLHeaderOptions:       &cos.ACLHeaderOptions{},
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{},
	}
	if sizeBytes > 0 {
		opts.ContentLength = int(sizeBytes)
	}
	if len(contType) > 0 {
		opts.ContentType = contType
	}
	if len(cannedAcl) > 0 {
		opts.XCosACL = string(cannedAcl)
	}
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	_, err = coscli.Object.Put(ctx, key, reader, opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Put")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, contType string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.InitiateMultipartUploadOptions{
		ACLHeaderOptions:       &cos.ACLHeaderOptions{},
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{},
	}
	if len(contType) > 0 {
		opts.ContentType = contType
	}
	if len(cannedAcl) > 0 {
		opts.XCosACL = string(cannedAcl)
	}
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	result, _, err := coscli.Object.InitiateMultipartUpload(ctx, key, opts)
	if err != nil {
		return "", errors.Wrap(err, "InitiateMultipartUpload")
	}

	return result.UploadID, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, input io.Reader, partSize int64) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectUploadPartOptions{}
	opts.ContentLength = int(partSize)
	resp, err := coscli.Object.UploadPart(ctx, key, uploadId, partIndex, input, opts)
	if err != nil {
		return "", errors.Wrap(err, "UploadPart")
	}

	return resp.Header.Get("Etag"), nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.CompleteMultipartUploadOptions{}
	parts := make([]cos.Object, len(partEtags))
	for i := range partEtags {
		parts[i] = cos.Object{
			PartNumber: i + 1,
			ETag:       partEtags[i],
		}
	}
	opts.Parts = parts
	_, _, err = coscli.Object.CompleteMultipartUpload(ctx, key, uploadId, opts)

	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUpload")
	}

	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}

	_, err = coscli.Object.AbortMultipartUpload(ctx, key, uploadId)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUpload")
	}

	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Object.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Delete")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return "", errors.Error("unsupported method")
	}
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	url, err := coscli.Object.GetPresignedURL(context.Background(), method, key,
		b.region.client.SecretID,
		b.region.client.SecretKey,
		expire, nil)
	if err != nil {
		return "", errors.Wrap(err, "coscli.Object.GetPresignedURL")
	}
	return url.String(), nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucketName, srcKey string, contType string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string) error {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectCopyOptions{
		ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{},
		ACLHeaderOptions:        &cos.ACLHeaderOptions{},
	}
	if len(cannedAcl) > 0 {
		opts.XCosACL = string(cannedAcl)
	}
	if len(storageClassStr) > 0 {
		opts.XCosStorageClass = storageClassStr
	}
	if len(contType) > 0 {
		opts.ContentType = contType
	}
	srcBucket := SBucket{
		region: b.region,
		Name:   srcBucketName,
	}
	srcUrl := fmt.Sprintf("%s/%s", srcBucket.getBucketUrlHost(), srcKey)
	log.Debugf("source url: %s", srcUrl)
	_, _, err = coscli.Object.Copy(ctx, destKey, srcUrl, opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Object.Copy")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return nil, errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.ObjectGetOptions{}
	if rangeOpt != nil {
		opts.Range = rangeOpt.String()
	}
	resp, err := coscli.Object.Get(ctx, key, opts)
	if err != nil {
		return nil, errors.Wrap(err, "coscli.Object.Get")
	}
	return resp.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partIndex int, srcBucketName string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	coscli, err := b.region.GetCosClient(b)
	if err != nil {
		return "", errors.Wrap(err, "GetCosClient")
	}
	srcBucket := SBucket{
		region: b.region,
		Name:   srcBucketName,
	}
	opts := cos.ObjectCopyPartOptions{}
	srcUrl := fmt.Sprintf("%s/%s", srcBucket.getBucketUrlHost(), srcKey)
	opts.XCosCopySourceRange = fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
	result, _, err := coscli.Object.CopyPart(ctx, key, uploadId, partIndex, srcUrl, &opts)
	if err != nil {
		return "", errors.Wrap(err, "coscli.Object.CopyPart")
	}
	return result.ETag, nil
}
