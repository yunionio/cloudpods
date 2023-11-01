// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	tos "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
)

type SBucket struct {
	multicloud.SBaseBucket
	VolcEngineTags
	region *SRegion

	Name         string
	Location     string
	CreationDate time.Time
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

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return ""
	}
	input := &tos.HeadBucketInput{
		Bucket: b.Name,
	}
	output, err := toscli.HeadBucket(context.Background(), input)
	if err != nil {
		return ""
	}
	return string(output.StorageClass)
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	ret := []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, b.region.getTOSExternalDomain()),
			Description: "ExtranetEndpoint",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("%s.%s", b.Name, b.region.getTOSInternalDomain()),
			Description: "IntranetEndpoint",
		},
	}
	return ret
}

func grantToCannedAcl(acls []tos.GrantV2) cloudprovider.TBucketACLType {
	isWrite, isRead := false, false
	for _, acl := range acls {
		if acl.GranteeV2.Type != enum.GranteeGroup || acl.GranteeV2.Canned != enum.CannedAllUsers {
			continue
		}
		switch acl.Permission {
		case enum.PermissionWrite:
			isWrite = true
		case enum.PermissionRead:
			isRead = true
		}
	}
	if isWrite && isRead {
		return cloudprovider.ACLPublicReadWrite
	}
	if isRead {
		return cloudprovider.ACLPublicRead
	}
	return cloudprovider.ACLPrivate
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	toscli, err := b.region.GetTosClient()
	if err != nil {
		log.Errorf("GetTosClient fail %s", err)
		return acl
	}
	input := tos.GetBucketACLInput{}
	input.Bucket = b.Name
	output, err := toscli.GetBucketACL(context.Background(), &input)
	if err != nil {
		log.Errorf("toscli.GetBucketAcl fail %s", err)
		return acl
	}
	return grantToCannedAcl(output.Grants)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrapf(err, "Get TosClient")
	}
	input := tos.PutBucketACLInput{}
	input.Bucket = b.Name
	input.ACLType = enum.ACLType(aclStr)
	_, err = toscli.PutBucketACL(context.Background(), &input)
	if err != nil {
		return errors.Wrapf(err, "PutBucketAcl")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return "", errors.Wrapf(err, "GetTosClient")
	}
	input := &tos.CreateMultipartUploadV2Input{
		Bucket:       b.Name,
		Key:          key,
		ACL:          enum.ACLType(cannedAcl),
		StorageClass: enum.StorageClassType(storageClassStr),
		Meta:         map[string]string{},
	}
	for k := range meta {
		input.Meta[k] = meta.Get(k)
	}

	output, err := toscli.CreateMultipartUploadV2(ctx, input)
	if err != nil {
		return "", err
	}
	return output.UploadID, nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrapf(err, "GetTosClient")
	}
	_, err = toscli.AbortMultipartUpload(ctx, &tos.AbortMultipartUploadInput{Bucket: b.Name, Key: key, UploadID: uploadId})
	if err != nil {
		return errors.Wrapf(err, "AbortMultipartUploadWithContext")
	}
	return nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrap(err, "GetTosClient")
	}
	parts := make([]tos.UploadedPartV2, len(partEtags))
	for i := range partEtags {
		parts[i].PartNumber = int(i + 1)
		parts[i].ETag = partEtags[i]
	}
	_, err = toscli.CompleteMultipartUploadV2(ctx, &tos.CompleteMultipartUploadV2Input{Bucket: b.Name, Key: key, UploadID: uploadId, Parts: parts})
	if err != nil {
		return errors.Wrapf(err, "CompleteMultipartUploadV2")
	}
	return nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrap(err, "GetTosClient")
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	var metaDir string
	metaHdr := make(map[string]string)
	cacheControl := ""
	if meta != nil {
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				cacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				cacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				cacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				cacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				cacheControl = v[0]
			default:
				metaHdr[k] = v[0]
			}
		}
		metaDir = "REPLACE"
	} else {
		metaDir = "COPY"
	}
	input := tos.CopyObjectInput{SrcBucket: srcBucket, Bucket: b.Name, Key: destKey, SrcKey: url.PathEscape(srcKey), StorageClass: enum.StorageClassType(storageClassStr), ACL: enum.ACLType(cannedAcl), MetadataDirective: enum.MetadataDirectiveType(metaDir)}
	if len(cacheControl) > 0 {
		input.CacheControl = cacheControl
	}
	if len(metaHdr) > 0 {
		input.Meta = metaHdr
	}
	_, err = toscli.CopyObject(ctx, &input)
	if err != nil {
		return errors.Wrapf(err, "CopyObject")
	}
	return nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return "", errors.Wrap(err, "GetTosClient")
	}
	input := tos.UploadPartCopyV2Input{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadID = uploadId
	input.PartNumber = partNumber
	input.SrcBucket = srcBucket
	input.SrcKey = srcKey
	if srcLength > 0 {
		input.CopySourceRange = fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
	}
	output, err := toscli.UploadPartCopyV2(ctx, &input)
	if err != nil {
		return "", errors.Wrapf(err, "CopyPart")
	}
	return output.ETag, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return "", errors.Wrap(err, "GetTosClient")
	}
	input := tos.UploadPartV2Input{}
	input.Bucket = b.Name
	input.Key = key
	input.UploadID = uploadId
	input.PartNumber = int(partIndex)
	seeker, err := fileutils.NewReadSeeker(part, partSize)
	if err != nil {
		return "", errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Content = seeker
	input.ContentLength = partSize
	output, err := toscli.UploadPartV2(ctx, &input)
	if err != nil {
		return "", errors.Wrapf(err, "UploadPart")
	}
	return output.ETag, nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrap(err, "GetTosClient")
	}
	input := tos.DeleteObjectV2Input{
		Bucket: b.Name,
		Key:    key,
	}
	_, err = toscli.DeleteObjectV2(ctx, &input)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetTosClient")
	}
	input := tos.GetObjectV2Input{
		Bucket:     b.Name,
		Key:        key,
		RangeStart: rangeOpt.Start,
		RangeEnd:   rangeOpt.End,
	}
	output, err := toscli.GetObjectV2(ctx, &input)
	if err != nil {
		return nil, errors.Wrap(err, "DeleteObject")
	}
	return output.Content, nil
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return result, errors.Wrap(err, "GetTosClient")
	}
	input := tos.ListObjectsV2Input{}
	input.Bucket = b.Name
	if len(prefix) > 0 {
		input.Prefix = prefix
	}
	if len(marker) > 0 {
		input.Marker = marker
	}
	if len(delimiter) > 0 {
		input.Delimiter = delimiter
	}
	if maxCount > 0 {
		input.MaxKeys = maxCount
	}
	output, err := toscli.ListObjectsV2(context.Background(), &input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	for _, object := range output.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          object.Key,
				SizeBytes:    object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified,
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if output.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, len(output.CommonPrefixes))
		for i, commonPrefix := range output.CommonPrefixes {
			result.CommonPrefixes[i] = &SObject{
				bucket:           b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{Key: commonPrefix.Prefix},
			}
		}
	}
	result.IsTruncated = output.IsTruncated
	result.NextMarker = output.NextMarker
	return result, nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return "", errors.Wrapf(err, "GetTosClient")
	}
	input := &tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     b.Name,
		Key:        key,
		Expires:    int64(expire.Seconds()),
	}
	output, err := toscli.PreSignedURL(input)
	if err != nil {
		return "", err
	}
	return output.SignedUrl, nil
}

func (b *SBucket) PutObject(ctx context.Context, key string, body io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	if sizeBytes < 0 {
		return errors.Error("context length expected")
	}
	toscli, err := b.region.GetTosClient()
	if err != nil {
		return errors.Wrapf(err, "GetTosClient")
	}
	input := tos.PutObjectV2Input{}
	input.Bucket = b.Name
	input.Key = key
	seeker, err := fileutils.NewReadSeeker(body, sizeBytes)
	if err != nil {
		return errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Content = body
	input.ContentLength = sizeBytes

	if meta != nil {
		metaHdr := make(map[string]string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.CacheControl = v[0]
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.ContentType = v[0]
			case cloudprovider.META_HEADER_CONTENT_MD5:
				input.ContentMD5 = v[0]
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.ContentEncoding = v[0]
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.ContentDisposition = v[0]
			default:
				metaHdr[k] = v[0]
			}
		}
		if len(metaHdr) > 0 {
			input.Meta = metaHdr
		}
	}

	if len(cannedAcl) > 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = enum.ACLType(cannedAcl)
	if len(storageClassStr) > 0 {
		input.StorageClass = enum.StorageClassType(storageClassStr)
	}
	_, err = toscli.PutObjectV2(ctx, &input)
	if err != nil {
		return errors.Wrapf(err, "PutObject")
	}
	return nil
}
