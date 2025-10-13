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
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/s3cli"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SBucket struct {
	multicloud.SBaseBucket
	AwsTags

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

func (b *SBucket) GetCreatedAt() time.Time {
	return b.CreationDate
}

func (b *SBucket) GetStorageClass() string {
	return ""
}

func s3ToCannedAcl(acls []types.Grant) cloudprovider.TBucketACLType {
	switch {
	case len(acls) == 1:
		if acls[0].Grantee.URI == nil && acls[0].Permission == types.Permission(s3cli.PERMISSION_FULL_CONTROL) {
			return cloudprovider.ACLPrivate
		}
	case len(acls) == 2:
		for _, g := range acls {
			if g.Grantee.Type == types.Type(s3cli.GRANTEE_TYPE_GROUP) && g.Grantee.URI != nil && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_AUTH_USERS && g.Permission == types.Permission(s3cli.PERMISSION_READ) {
				return cloudprovider.ACLAuthRead
			}
			if g.Grantee.Type == types.Type(s3cli.GRANTEE_TYPE_GROUP) && g.Grantee.URI != nil && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == types.Permission(s3cli.PERMISSION_READ) {
				return cloudprovider.ACLPublicRead
			}
		}
	case len(acls) == 3:
		for _, g := range acls {
			if g.Grantee.Type == types.Type(s3cli.GRANTEE_TYPE_GROUP) && g.Grantee.URI != nil && *g.Grantee.URI == s3cli.GRANTEE_GROUP_URI_ALL_USERS && g.Permission == types.Permission(s3cli.PERMISSION_WRITE) {
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
		log.Errorf("GetS3Client fail %v", err)
		return acl
	}
	input := &s3.GetBucketAclInput{}
	input.Bucket = &b.Name
	output, err := s3cli.GetBucketAcl(context.Background(), input)
	if err != nil {
		log.Errorf("s3cli.GetBucketAcl fail %s", err)
		return acl
	}
	return s3ToCannedAcl(output.Grants)
}

func (b *SBucket) SetAcl(aclStr cloudprovider.TBucketACLType) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutBucketAclInput{}
	input.Bucket = &b.Name
	input.ACL = types.BucketCannedACL(string(aclStr))
	_, err = s3cli.PutBucketAcl(context.Background(), input)
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

func (b *SBucket) GetWebsiteUrl() string {
	return fmt.Sprintf("http://%s.%s", b.Name, b.region.getS3WebsiteEndpoint())
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
	input.Bucket = &b.Name
	if len(prefix) > 0 {
		input.Prefix = &prefix
	}
	if len(marker) > 0 {
		input.Marker = &marker
	}
	if len(delimiter) > 0 {
		input.Delimiter = &delimiter
	}
	if maxCount > 0 {
		mc := int32(maxCount)
		input.MaxKeys = &mc
	}
	if len(prefix) > 0 {
		input.Prefix = &prefix
	}
	oResult, err := s3cli.ListObjects(context.Background(), input)
	if err != nil {
		return result, errors.Wrap(err, "ListObjects")
	}
	result.Objects = make([]cloudprovider.ICloudObject, 0)
	for _, object := range oResult.Contents {
		obj := &SObject{
			bucket: b,
			SBaseCloudObject: cloudprovider.SBaseCloudObject{
				StorageClass: string(object.StorageClass),
				Key:          *object.Key,
				SizeBytes:    *object.Size,
				ETag:         *object.ETag,
				LastModified: *object.LastModified,
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

func (b *SBucket) PutObject(ctx context.Context, key string, body io.Reader, sizeBytes int64, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	if sizeBytes < 0 {
		return errors.Error("content length expected")
	}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutObjectInput{}
	input.Bucket = &b.Name
	input.Key = &key
	seeker, err := fileutils.NewReadSeeker(body, sizeBytes)
	if err != nil {
		return errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Body = seeker
	input.ContentLength = &sizeBytes
	if meta != nil {
		metaHdr := make(map[string]string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			value := strings.TrimSpace(v[0])
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.CacheControl = &value
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.ContentType = &value
			case cloudprovider.META_HEADER_CONTENT_MD5:
				input.ContentMD5 = &value
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.ContentLanguage = &value
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.ContentEncoding = &value
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.ContentDisposition = &value
			default:
				metaHdr[k] = value
			}
		}
		if len(metaHdr) > 0 {
			input.Metadata = metaHdr
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = types.ObjectCannedACL(string(cannedAcl))
	if len(storageClassStr) > 0 {
		input.StorageClass = types.StorageClass(storageClassStr)
	}
	_, err = s3cli.PutObject(ctx, input)
	if err != nil {
		return errors.Wrap(err, "PutObjectWithContext")
	}
	return nil
}

func (b *SBucket) NewMultipartUpload(ctx context.Context, key string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.CreateMultipartUploadInput{}
	input.Bucket = &b.Name
	input.Key = &key
	if meta != nil {
		metaHdr := make(map[string]string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			value := strings.TrimSpace(v[0])
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.CacheControl = &value
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.ContentType = &value
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.ContentLanguage = &value
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.ContentEncoding = &value
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.ContentDisposition = &value
			default:
				metaHdr[k] = value
			}
		}
		if len(metaHdr) > 0 {
			input.Metadata = metaHdr
		}
	}
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = types.ObjectCannedACL(string(cannedAcl))
	if len(storageClassStr) > 0 {
		input.StorageClass = types.StorageClass(storageClassStr)
	}
	output, err := s3cli.CreateMultipartUpload(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "CreateMultipartUpload")
	}
	return *output.UploadId, nil
}

func (b *SBucket) UploadPart(ctx context.Context, key string, uploadId string, partIndex int, part io.Reader, partSize int64, offset, totalSize int64) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.UploadPartInput{}
	input.Bucket = &b.Name
	input.Key = &key
	input.UploadId = &uploadId
	pn := int32(partIndex)
	input.PartNumber = &pn
	seeker, err := fileutils.NewReadSeeker(part, partSize)
	if err != nil {
		return "", errors.Wrap(err, "newFakeSeeker")
	}
	defer seeker.Close()
	input.Body = seeker
	input.ContentLength = &partSize
	output, err := s3cli.UploadPart(ctx, input)
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
	input.Bucket = &b.Name
	input.Key = &key
	input.UploadId = &uploadId
	uploads := &types.CompletedMultipartUpload{}
	parts := make([]types.CompletedPart, len(partEtags))
	for i := range partEtags {
		parts[i] = types.CompletedPart{}
		number := int32(i + 1)
		parts[i].PartNumber = &number
		parts[i].ETag = &partEtags[i]
	}
	uploads.Parts = parts
	input.MultipartUpload = uploads
	_, err = s3cli.CompleteMultipartUpload(ctx, input)
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
	input.Bucket = &b.Name
	input.Key = &key
	input.UploadId = &uploadId
	_, err = s3cli.AbortMultipartUpload(ctx, input)
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
	input.Bucket = &b.Name
	input.Key = &key
	_, err = s3cli.DeleteObject(ctx, input)
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
	scli := s3.NewPresignClient(s3cli)
	ctx := context.Background()
	var request *v4.PresignedHTTPRequest
	switch method {
	case "GET":
		input := &s3.GetObjectInput{}
		input.Bucket = &b.Name
		input.Key = &key
		request, _ = scli.PresignGetObject(ctx, input)
	case "PUT":
		input := &s3.PutObjectInput{}
		input.Bucket = &b.Name
		input.Key = &key
		request, _ = scli.PresignPutObject(ctx, input)
	case "DELETE":
		input := &s3.DeleteObjectInput{}
		input.Bucket = &b.Name
		input.Key = &key
		request, _ = scli.PresignDeleteObject(ctx, input)
	default:
		return "", errors.Error("unsupported method")
	}
	return request.URL, nil
}

func (b *SBucket) CopyObject(ctx context.Context, destKey string, srcBucket, srcKey string, cannedAcl cloudprovider.TBucketACLType, storageClassStr string, meta http.Header) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	log.Debugf("copy from %s/%s to %s/%s", srcBucket, srcKey, b.Name, destKey)
	input := &s3.CopyObjectInput{}
	input.Bucket = &b.Name
	input.Key = &destKey
	copySource := fmt.Sprintf("%s/%s", srcBucket, url.PathEscape(srcKey))
	input.CopySource = &copySource
	input.StorageClass = types.StorageClass(storageClassStr)
	if len(cannedAcl) == 0 {
		cannedAcl = b.GetAcl()
	}
	input.ACL = types.ObjectCannedACL(string(cannedAcl))
	var metaDir string
	if meta != nil {
		metaHdr := make(map[string]string)
		for k, v := range meta {
			if len(v) == 0 || len(v[0]) == 0 {
				continue
			}
			value := strings.TrimSpace(v[0])
			switch http.CanonicalHeaderKey(k) {
			case cloudprovider.META_HEADER_CACHE_CONTROL:
				input.CacheControl = &value
			case cloudprovider.META_HEADER_CONTENT_TYPE:
				input.ContentType = &value
			case cloudprovider.META_HEADER_CONTENT_LANGUAGE:
				input.ContentLanguage = &value
			case cloudprovider.META_HEADER_CONTENT_ENCODING:
				input.ContentEncoding = &value
			case cloudprovider.META_HEADER_CONTENT_DISPOSITION:
				input.ContentDisposition = &value
			default:
				metaHdr[k] = value
			}
		}
		if len(metaHdr) > 0 {
			input.Metadata = metaHdr
		}
		metaDir = "REPLACE"
	} else {
		metaDir = "COPY"
	}
	input.MetadataDirective = types.MetadataDirective(metaDir)
	_, err = s3cli.CopyObject(ctx, input)
	if err != nil {
		return errors.Wrap(err, "CopyObject")
	}
	return nil
}

func (b *SBucket) GetObject(ctx context.Context, key string, rangeOpt *cloudprovider.SGetObjectRange) (io.ReadCloser, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.GetObjectInput{}
	input.Bucket = &b.Name
	input.Key = &key
	if rangeOpt != nil {
		rangeStr := rangeOpt.String()
		input.Range = &rangeStr
	}
	output, err := s3cli.GetObject(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "GetObject")
	}
	return output.Body, nil
}

func (b *SBucket) CopyPart(ctx context.Context, key string, uploadId string, partNumber int, srcBucket string, srcKey string, srcOffset int64, srcLength int64) (string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return "", errors.Wrap(err, "GetS3Client")
	}
	input := &s3.UploadPartCopyInput{}
	input.Bucket = &b.Name
	input.Key = &key
	input.UploadId = &uploadId
	pn := int32(partNumber)
	input.PartNumber = &pn
	copySource := fmt.Sprintf("/%s/%s", srcBucket, url.PathEscape(srcKey))
	input.CopySource = &copySource
	if srcLength > 0 {
		copySourceRange := fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
		input.CopySourceRange = &copySourceRange
	}
	output, err := s3cli.UploadPartCopy(ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "s3cli.UploadPartCopy")
	}
	return *output.CopyPartResult.ETag, nil
}

func (b *SBucket) SetWebsite(websitConf cloudprovider.SBucketWebsiteConf) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutBucketWebsiteInput{}
	input.WebsiteConfiguration = &types.WebsiteConfiguration{
		IndexDocument: &types.IndexDocument{Suffix: &websitConf.Index},
		ErrorDocument: &types.ErrorDocument{Key: &websitConf.ErrorDocument},
	}
	input.Bucket = &b.Name
	_, err = s3cli.PutBucketWebsite(context.Background(), input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.PutBucketWebsite(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (b *SBucket) GetWebsiteConf() (cloudprovider.SBucketWebsiteConf, error) {
	result := cloudprovider.SBucketWebsiteConf{}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return result, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.GetBucketWebsiteInput{}
	input.Bucket = &b.Name
	webconfResult, err := s3cli.GetBucketWebsite(context.Background(), input)
	if err != nil {
		return result, errors.Wrapf(err, "s3cli.GetBucketWebsite(%s)", b.Name)
	}

	if webconfResult.IndexDocument != nil && webconfResult.IndexDocument.Suffix != nil {
		result.Index = *webconfResult.IndexDocument.Suffix
	}
	if webconfResult.ErrorDocument != nil && webconfResult.ErrorDocument.Key != nil {
		result.ErrorDocument = *webconfResult.ErrorDocument.Key
	}
	result.Url = b.GetWebsiteUrl()
	return result, nil
}

func (b *SBucket) DeleteWebSiteConf() error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := s3.DeleteBucketWebsiteInput{}
	input.Bucket = &b.Name
	_, err = s3cli.DeleteBucketWebsite(context.Background(), &input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.DeleteBucketWebsite(%s)", b.Name)
	}
	return nil
}

func InputToAwsApiSliceString(input []string) []string {
	result := []string{}
	for i := range input {
		result = append(result, input[i])
	}
	return result
}

func InputToAwsApiInt64(input int64) int64 {
	return input
}

func AwsApiSliceStringToOutput(input []*string) []string {
	result := []string{}
	for i := range input {
		if input[i] != nil {
			result = append(result, *input[i])
		} else {
			result = append(result, "")
		}
	}
	return result
}

func AwsApiInt64ToOutput(input *int64) int64 {
	if input == nil {
		return 0
	}
	return *input
}

func (b *SBucket) SetCORS(rules []cloudprovider.SBucketCORSRule) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	opts := []types.CORSRule{}
	for i := range rules {
		maxAgeSeconds := int32(rules[i].MaxAgeSeconds)
		opts = append(opts, types.CORSRule{
			AllowedOrigins: InputToAwsApiSliceString(rules[i].AllowedOrigins),
			AllowedMethods: InputToAwsApiSliceString(rules[i].AllowedMethods),
			AllowedHeaders: InputToAwsApiSliceString(rules[i].AllowedHeaders),
			MaxAgeSeconds:  &maxAgeSeconds,
			ExposeHeaders:  InputToAwsApiSliceString(rules[i].ExposeHeaders),
		})
	}

	input := &s3.PutBucketCorsInput{}
	input.Bucket = &b.Name
	input.CORSConfiguration = &types.CORSConfiguration{CORSRules: opts}
	_, err = s3cli.PutBucketCors(context.Background(), input)
	if err != nil {
		return errors.Wrapf(err, "PutBucketCors %v", err)
	}
	return nil
}

func (b *SBucket) GetCORSRules() ([]cloudprovider.SBucketCORSRule, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.GetBucketCorsInput{}
	input.Bucket = &b.Name
	conf, err := s3cli.GetBucketCors(context.Background(), input)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuchCORSConfiguration") {
			return nil, errors.Wrapf(err, "s3cli.GetBucketCors(%s)", b.Name)
		}
	}
	if conf == nil {
		return []cloudprovider.SBucketCORSRule{}, nil
	}
	result := []cloudprovider.SBucketCORSRule{}
	for i := range conf.CORSRules {
		result = append(result, cloudprovider.SBucketCORSRule{
			AllowedOrigins: conf.CORSRules[i].AllowedOrigins,
			AllowedMethods: conf.CORSRules[i].AllowedMethods,
			AllowedHeaders: conf.CORSRules[i].AllowedHeaders,
			MaxAgeSeconds:  int(*conf.CORSRules[i].MaxAgeSeconds),
			ExposeHeaders:  conf.CORSRules[i].ExposeHeaders,
			Id:             strconv.Itoa(i),
		})
	}
	return result, nil
}

func (b *SBucket) DeleteCORS() error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}

	input := s3.DeleteBucketCorsInput{}
	input.Bucket = &b.Name
	_, err = s3cli.DeleteBucketCors(context.Background(), &input)
	if err != nil {
		return errors.Wrapf(err, "s3cli.DeleteBucketCors(%s)", b.Name)
	}
	return nil
}

func (b *SBucket) GetTags() (map[string]string, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	tagresult, err := s3cli.GetBucketTagging(context.Background(), &s3.GetBucketTaggingInput{Bucket: &b.Name})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "osscli.GetBucketTagging(%s)", b.Name)
	}
	if tagresult == nil {
		return nil, nil
	}
	result := map[string]string{}
	for i := range tagresult.TagSet {
		if tagresult.TagSet[i].Key != nil && tagresult.TagSet[i].Value != nil {
			result[*tagresult.TagSet[i].Key] = *tagresult.TagSet[i].Value
		}

	}
	return result, nil
}

func (b *SBucket) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		return cloudprovider.ErrNotSupported
	}
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}

	_, err = s3cli.DeleteBucketTagging(context.Background(), &s3.DeleteBucketTaggingInput{Bucket: &b.Name})
	if err != nil {
		return errors.Wrapf(err, "DeleteBucketTagging")
	}

	if len(tags) == 0 {
		return nil
	}

	input := &s3.PutBucketTaggingInput{Tagging: &types.Tagging{}}
	input.Bucket = &b.Name
	apiTagKeys := []string{}
	apiTagValues := []string{}
	for k, v := range tags {
		apiTagKeys = append(apiTagKeys, k)
		apiTagValues = append(apiTagValues, v)

	}
	for i := range apiTagKeys {
		input.Tagging.TagSet = append(input.Tagging.TagSet, types.Tag{Key: &apiTagKeys[i], Value: &apiTagValues[i]})
	}

	_, err = s3cli.PutBucketTagging(context.Background(), input)
	if err != nil {
		return errors.Wrapf(err, "obscli.SetBucketTagging(%s)", jsonutils.Marshal(input))
	}
	return nil
}

func (b *SBucket) ListMultipartUploads() ([]cloudprovider.SBucketMultipartUploads, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	result := []cloudprovider.SBucketMultipartUploads{}

	input := &s3.ListMultipartUploadsInput{}
	input.Bucket = &b.Name
	keyMarker := ""
	uploadIDMarker := ""
	for {
		if len(keyMarker) > 0 {
			input.KeyMarker = &keyMarker
		}
		if len(uploadIDMarker) > 0 {
			input.UploadIdMarker = &uploadIDMarker
		}
		output, err := s3cli.ListMultipartUploads(context.Background(), input)
		if err != nil {
			return nil, errors.Wrap(err, " coscli.Bucket.ListMultipartUploads(context.Background(), &input)")
		}
		if output == nil {
			return nil, nil
		}
		for i := range output.Uploads {
			temp := cloudprovider.SBucketMultipartUploads{}
			if output.Uploads[i].Key != nil {
				temp.ObjectName = *output.Uploads[i].Key
			}
			if output.Uploads[i].Initiator != nil {
				temp.Initiator = *output.Uploads[i].Initiator.DisplayName
			}
			if output.Uploads[i].Initiated != nil {
				temp.Initiated = *output.Uploads[i].Initiated
			}
			if output.Uploads[i].UploadId != nil {
				temp.UploadID = *output.Uploads[i].UploadId
			}
			result = append(result, temp)
		}
		if output.NextKeyMarker != nil {
			keyMarker = *output.NextKeyMarker
		}
		if output.NextUploadIdMarker != nil {
			uploadIDMarker = *output.NextUploadIdMarker
		}

		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}
	}

	return result, nil
}

type SBucketPolicyStatement struct {
	Version   string                          `json:"Version"`
	Id        string                          `json:"Id"`
	Statement []SBucketPolicyStatementDetails `json:"Statement"`
}

type SBucketPolicyStatementDetails struct {
	Sid       string                            `json:"Sid"`
	Principal map[string][]string               `json:"Principal"`
	Action    []string                          `json:"Action"`
	Resource  []string                          `json:"Resource"`
	Effect    string                            `json:"Effect"`
	Condition map[string]map[string]interface{} `json:"Condition"`
}

func (b *SBucket) GetPolicy() ([]cloudprovider.SBucketPolicyStatement, error) {
	res := []cloudprovider.SBucketPolicyStatement{}
	policies, err := b.getPolicy()
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return res, nil
		}
		return nil, errors.Wrap(err, "get policy")
	}
	for i, policy := range policies {
		temp := cloudprovider.SBucketPolicyStatement{}
		temp.Action = policy.Action
		temp.Principal = policy.Principal
		temp.PrincipalId = getLocalPrincipalId(policy.Principal["AWS"])
		temp.PrincipalNames = getLocalPrincipalNames(policy.Principal["AWS"])
		temp.Effect = policy.Effect
		temp.Resource = policy.Resource
		temp.ResourcePath = policy.Resource
		temp.CannedAction = b.actionToCannedAction(policy.Action)
		temp.Id = fmt.Sprintf("%d", i)
		temp.Condition = policy.Condition
		res = append(res, temp)
	}
	return res, nil
}

func (b *SBucket) getPolicy() ([]SBucketPolicyStatementDetails, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}
	input := &s3.GetBucketPolicyInput{}
	input.Bucket = &b.Name
	conf, err := s3cli.GetBucketPolicy(context.Background(), input)
	if err != nil {
		if !strings.Contains(err.Error(), "NoSuch") {
			return nil, errors.Wrapf(err, "s3cli.GetBucketCors(%s)", b.Name)
		}
	}
	if conf == nil {
		return []SBucketPolicyStatementDetails{}, nil
	}
	if conf.Policy == nil {
		return nil, errors.ErrNotFound
	}
	obj, err := jsonutils.Parse([]byte(*conf.Policy))
	if err != nil {
		return nil, errors.Wrap(err, "parse policy")
	}
	policies := []SBucketPolicyStatementDetails{}
	err = obj.Unmarshal(&policies, "Statement")
	if err != nil {
		return nil, errors.Wrap(err, "Statement")
	}
	err = obj.Unmarshal(&policies, "Statement")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return policies, nil
}

func (b *SBucket) SetPolicy(policy cloudprovider.SBucketPolicyStatementInput) error {
	old, err := b.getPolicy()
	if err != nil && err != errors.ErrNotFound {
		return errors.Wrap(err, "getPolicy")
	}
	if old == nil {
		old = []SBucketPolicyStatementDetails{}
	}
	ids := []string{}
	for i := range policy.PrincipalId {
		id := strings.Split(policy.PrincipalId[i], ":")
		if len(id) == 1 {
			ids = append(ids, id...)
		}
		if len(id) == 2 {
			// 没有主账号id,设为owner id
			if len(id[0]) == 0 {
				id[0] = b.region.client.GetAccountId()
			}
			// 没有子账号，默认和主账号相同
			if len(id[1]) == 0 {
				id[1] = "*"
			}
			ids = append(ids, id[1])
		}
		if len(id) > 2 {
			return errors.Wrap(cloudprovider.ErrNotSupported, "Invalida PrincipalId Input")
		}
	}

	old = append(old, SBucketPolicyStatementDetails{
		Effect:   policy.Effect,
		Action:   b.getAction(policy.CannedAction),
		Resource: b.getResources(policy.ResourcePath),
		Sid:      utils.GenRequestId(20),
		Principal: map[string][]string{
			// "AWS": b.getPrincipal(policy.PrincipalId, policy.AccountId),
			"AWS": ids,
		},
		Condition: policy.Condition,
	})
	return b.setPolicy(old)
}

func (b *SBucket) setPolicy(policies []SBucketPolicyStatementDetails) error {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return errors.Wrap(err, "GetS3Client")
	}
	input := &s3.PutBucketPolicyInput{}
	input.Bucket = &b.Name
	// example
	// test := `{"Statement":[{"Action":["s3:GetBucketAcl"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6"],"Sid":"AWSConfigBucketPermissionsCheck"},{"Action":["s3:PutObject"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6/AWSLogs/2xxxxx6/Config/*"],"Sid":"AWSConfigBucketDelivery"},{"Action":["s3:PutObject"],"Effect":"Allow","Principal":{"Service":["config.amazonaws.com"]},"Resource":["arn:aws-cn:s3:::config-bucket-2xxxxx6/AWSLogs/2xxxxx6/Config/*"],"Sid":"test"}]}`
	param := SBucketPolicyStatement{}
	param.Statement = policies

	policyStr := jsonutils.Marshal(param).String()
	input.Policy = &policyStr
	_, err = s3cli.PutBucketPolicy(context.Background(), input)
	if err != nil {
		return errors.Wrap(err, "PutBucketPolicy")
	}
	return nil
}

func (b *SBucket) DeletePolicy(id []string) ([]cloudprovider.SBucketPolicyStatement, error) {
	s3cli, err := b.region.GetS3Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetS3Client")
	}

	policies, err := b.getPolicy()
	if err != nil {
		return nil, errors.Wrap(err, "GetPolicy")
	}
	needKeep := []SBucketPolicyStatementDetails{}
	for i, policy := range policies {
		if utils.IsInStringArray(fmt.Sprintf("%d", i), id) {
			continue
		}
		needKeep = append(needKeep, policy)
	}
	_, err = s3cli.DeleteBucketPolicy(context.Background(), &s3.DeleteBucketPolicyInput{Bucket: &b.Name})
	if err != nil {
		return nil, errors.Wrap(err, "DeleteBucketPolicy")
	}

	if len(needKeep) > 0 {
		err = b.setPolicy(needKeep)
		if err != nil {
			return nil, errors.Wrap(err, "setPolicy")
		}
	}
	res := []cloudprovider.SBucketPolicyStatement{}
	for _, policy := range needKeep {
		temp := cloudprovider.SBucketPolicyStatement{}
		temp.Action = policy.Action
		temp.Principal = policy.Principal
		temp.Effect = policy.Effect
		temp.Resource = policy.Resource
		temp.ResourcePath = policy.Resource
		temp.CannedAction = b.actionToCannedAction(policy.Action)
		temp.Id = policy.Sid
		temp.Condition = policy.Condition
		res = append(res, temp)
	}
	return res, nil
}

func (b *SBucket) getResources(paths []string) []string {
	res := []string{}
	for _, path := range paths {
		res = append(res, fmt.Sprintf("arn:%s:s3:::%s%s", b.region.GetARNPartition(), b.Name, path))
	}
	return res
}

func (b *SBucket) getPrincipal(principalIds []string, accountId string) []string {
	res := []string{}
	for _, id := range principalIds {
		res = append(res, fmt.Sprintf("arn:%s:iam::%s:user/%s", b.region.GetARNPartition(), accountId, id))
	}
	return res
}

func (b *SBucket) getAwsAction(actions []string) []string {
	res := []string{}
	for _, action := range actions {
		res = append(res, fmt.Sprintf("s3:%s", action))
	}
	return res
}

var readActions = []string{
	"s3:Get*",
	"s3:List*",
}

var readWriteActions = []string{
	"s3:Get*",
	"s3:List*",
	"s3:Create*",
	"s3:Put*",
	"s3:Delete*",
	"s3:Create*",
	"s3:AbortMultipartUpload",
}

var fullControlActions = []string{
	"s3:*",
}

func (b *SBucket) getAction(s string) []string {
	switch s {
	case "Read":
		return readActions
	case "ReadWrite":
		return readWriteActions
	case "FullControl":
		return fullControlActions
	default:
		return nil
	}
}

func (b *SBucket) actionToCannedAction(actions []string) string {
	if len(actions) == len(readActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readActions) {
				return ""
			}
		}
		return "Read"
	} else if len(actions) == len(fullControlActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, fullControlActions) {
				return ""
			}
		}
		return "FullControl"
	} else if len(actions) == len(readWriteActions) {
		for _, action := range actions {
			if !utils.IsInStringArray(action, readWriteActions) {
				return ""
			}
		}
		return "ReadWrite"
	}
	return ""
}

/*
	example: in:arn:aws-cn:iam::248697896586:user/yunion-test
			out:[248697896586:yunion-test]
*/

func getLocalPrincipalId(principals []string) []string {
	res := []string{}
	for _, principal := range principals {
		temp := strings.Split(principal, "::")
		temp1 := strings.Split(temp[1], ":user/")
		if len(temp1) > 1 {
			if temp1[1] == "*" {
				temp1[1] = temp1[0]
			}
			res = append(res, fmt.Sprintf("%s:%s", temp1[0], temp1[1]))
		} else {
			res = append(res, temp[1])
		}
	}
	return res
}

/*
	example: in:arn:aws-cn:iam::248697896586:user/yunion-test
			 out:["248697896586:yunion-test":"yunion-test"]
*/

func getLocalPrincipalNames(principals []string) map[string]string {
	res := map[string]string{}
	for _, principal := range principals {
		temp := strings.Split(principal, "::")
		temp1 := strings.Split(temp[1], ":user/")
		if len(temp1) > 1 {
			if temp1[1] == "*" {
				temp1[1] = temp1[0]
			}
			res[fmt.Sprintf("%s:%s", temp1[0], temp1[1])] = temp1[1]
		}
	}
	return res
}
