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

package aws

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/s3cli"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SBucket struct {
	multicloud.SBaseBucket

	region *SRegion

	Name         string
	CreationDate time.Time
	Location     string

	acl cloudprovider.TBucketACLType
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
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

func s3ToCannedAcl(acls []*s3.Grant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == nil && *acls[0].Permission == s3cli.PERMISSION_FULL_CONTROL {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_AUTH_USERS && *g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLAuthRead
			}
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && *g.Permission == s3cli.PERMISSION_READ {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if *g.Grantee.Type == s3cli.GRANTEE_TYPE_GROUP && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && *g.Permission == s3cli.PERMISSION_WRITE {
				return cloudprovider.ACLPublicReadWrite
			}
		}
	}
	return cloudprovider.ACLUnknown
}

func (b *SBucket) GetAcl() cloudprovider.TBucketACLType {
	acl := cloudprovider.ACLPrivate
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		log.Errorf("b.region.GetS3Client fail %s", err)
		return acl
	}
	input := &s3.GetBucketAclInput{}
	input.SetBucket(b.Name)
	output, err := s3cli.GetBucketAcl(input)
	if err != nil {
		log.Errorf("s3cli.GetBucketAcl fail %s", err)
		return acl
	}
	return s3ToCannedAcl(output.Grants)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "b.region.GetS3Client")
	}
	input := &s3.PutBucketAclInput{}
	input.SetBucket(b.Name)
	input.SetACL(string(aclStr))
	_, err = s3cli.PutBucketAcl(input)
	if err != nil {
		return errors.Wrap(err, "PutBucketAcl")
	}
	return nil
}

func (b *SBucket) GetAccessUrls() []cloudprovider.SBucketAccessUrl {
	return []cloudprovider.SBucketAccessUrl{
		{
			Url:         fmt.Sprintf("https://%s.%s", b.Name, b.region.getS3Endpoint()),
			Description: "bucket domain",
			Primary:     true,
		},
		{
			Url:         fmt.Sprintf("https://%s/%s", b.region.getS3Endpoint(), b.Name),
			Description: "s3 domain",
		},
	}
}

func (b *SBucket) GetStats() cloudprovider.SBucketStats {
	stats, _ := cloudprovider.GetIBucketStats(b)
	return stats
}

func (b *SBucket) ListObjects(prefix string, marker string, delimiter string, maxCount int) (cloudprovider.SListObjectResult, error) {
	result := cloudprovider.SListObjectResult{}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return result, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.ListObjectsInput{}
	input.SetBucket(b.Name)
	if len(prefix) > 0 {
		input.SetPrefix(prefix)
	}
	if len(marker) > 0 {
		input.SetMarker(marker)
	}
	if len(delimiter) > 0 {
		input.SetDelimiter(delimiter)
	}
	if maxCount > 0 {
		input.SetMaxKeys(int64(maxCount))
	}
	oResult, err := s3cli.ListObjects(input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: *object.StorageClass,
				Key:          *object.Key,
				SizeBytes:    *object.Size,
				ETag:         *object.ETag,
				LastModified: *object.LastModified,
				ContentType:  "",
			},
		}
		result.Objects = append(result.Objects, obj)
	}
	if oResult.CommonPrefixes != nil {
		result.CommonPrefixes = make([]cloudprovider.ICloudObject, len(oResult.CommonPrefixes))
		for i, commPrefix := range oResult.CommonPrefixes {
			result.CommonPrefixes[i] = &SObject{
				bucket:           b,
				SBaseCloudObject: cloudprovider.SBaseCloudObject{Key: *commPrefix.Prefix},
			}
		}
	}
	if oResult.IsTruncated != nil {
		result.IsTruncated = *oResult.IsTruncated
	}
	if oResult.NextMarker != nil {
		result.NextMarker = *oResult.NextMarker
	}
	return result, nil
}

func (b *SBucket) GetIObjects(prefix string, isRecursive bool) ([]cloudprovider.ICloudObject, error) {
	return cloudprovider.GetIObjects(b, prefix, isRecursive)
}

func (b *SBucket) PutObject(ctx context.Context, key string, body io.Reader, sizeBytes int64, contType string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string) error {
	if sizeBytes < 0 {
		return errors.Error("content length expected")
	}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	seeker, err := fileutils2.NewReadSeeker(body, sizeBytes)
	if err != nil {
		return errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.SetBody(seeker)
	input.SetContentLength(sizeBytes)
	if len(contType) > 0 {
		input.SetContentType(contType)
	}
	if len(cannedAcl) > 0 {
		input.SetACL(string(cannedAcl))
	}
	if len(storageClassStr) > 0 {
		input.SetStorageClass(storageClassStr)
	}
	_, err = s3cli.PutObjectWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "PutObjectWithContext")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, contType string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CreateMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	if len(contType) > 0 {
		input.SetContentType(contType)
	}
	if len(cannedAcl) > 0 {
		input.SetACL(string(cannedAcl))
	}
	if len(storageClassStr) > 0 {
		input.SetStorageClass(storageClassStr)
	}
	output, err := s3cli.CreateMultipartUploadWithContext(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "CreateMultipartUpload")
	}
	return *output.UploadId, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.UploadPartInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	input.SetPartNumber(int64(partIndex))
	seeker, err := fileutils2.NewReadSeeker(part, partSize)
	if err != nil {
		return "", errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.SetBody(seeker)
	input.SetContentLength(partSize)
	output, err := s3cli.UploadPartWithContext(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "UploadPartWithContext")
	}
	return *output.ETag, nil
}

func (b *SBucket) CompleteMultipartUpload(ctx context.Context, key string, uploadId string, partEtags []string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CompleteMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	uploads := &s3.CompletedMultipartUpload{}
	parts := make([]*s3.CompletedPart, len(partEtags))
	for i := range partEtags {
		parts[i] = &s3.CompletedPart{}
		parts[i].SetPartNumber(int64(i + 1))
		parts[i].SetETag(partEtags[i])
	}
	uploads.SetParts(parts)
	input.SetMultipartUpload(uploads)
	_, err = s3cli.CompleteMultipartUploadWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "CompleteMultipartUploadWithContext")
	}
	return nil
}

func (b *SBucket) AbortMultipartUpload(ctx context.Context, key string, uploadId string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.AbortMultipartUploadInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	input.SetUploadId(uploadId)
	_, err = s3cli.AbortMultipartUploadWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "AbortMultipartUploadWithContext")
	}
	return nil
}

func (b *SBucket) DeleteObject(ctx context.Context, key string) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.DeleteObjectInput{}
	input.SetBucket(b.Name)
	input.SetKey(key)
	_, err = s3cli.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (b *SBucket) GetTempUrl(method string, key string, expire time.Duration) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	var request *request.Request
	switch method {
	case "GET":
		input := &s3.GetObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.GetObjectRequest(input)
	case "PUT":
		input := &s3.PutObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.PutObjectRequest(input)
	case "DELETE":
		input := &s3.DeleteObjectInput{}
		input.SetBucket(b.Name)
		input.SetKey(key)
		request, _ = s3cli.DeleteObjectRequest(input)
	default:
		return "", errors.Error("unsupported method")
	}
	url, _, err := request.PresignRequest(expire)
	if err != nil {
		return "", errors.Wrap(err, "request.PresignRequest")
	}
	return url, nil
}
